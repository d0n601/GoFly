package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultForecastBaseURL = "https://api.open-meteo.com"
	ftPerM                 = 3.28084
)

// CloudLevel is the cloud cover at one pressure surface from a model run.
type CloudLevel struct {
	Psl   int
	Cover int
	GeoM  float64
}

// Ceiling describes the derived sky layer for a forecast step.
type Ceiling struct {
	Kind  string
	FtAgl int
}

// ForecastStep is one hour's worth of a model forecast.
type ForecastStep struct {
	Time        time.Time
	Temp        float64
	Dewp        float64
	WindDir     float64
	WindSpd     float64
	Gust        float64
	VisibM      float64
	WXCode      int
	PrecipMm    float64
	PrecipProb  int
	PressureMsl float64
	Ceiling     Ceiling
}

// baseHourlyVars are the surface-level forecast variables requested.
var baseHourlyVars = []string{
	"temperature_2m",
	"dew_point_2m",
	"wind_speed_10m",
	"wind_direction_10m",
	"wind_gusts_10m",
	"visibility",
	"weather_code",
	"precipitation",
	"precipitation_probability",
	"pressure_msl",
}

// cloudLevels lists the pressure surfaces (hPa) from the surface upward.
func cloudLevels() []string {
	return []string{
		"925", "850", "800", "700", "600", "500",
		"400", "300", "250", "200", "150", "100",
	}
}

// hourlyVars builds the comma-joined hourly variable list for the request.
func hourlyVars() string {
	vars := make([]string, 0, len(baseHourlyVars)+2*len(cloudLevels()))
	vars = append(vars, baseHourlyVars...)
	for _, p := range cloudLevels() {
		vars = append(vars, "cloud_cover_"+p+"hPa", "geopotential_height_"+p+"hPa")
	}
	return strings.Join(vars, ",")
}

func isFog(code int) bool {
	return code == 45 || code == 48
}

// DeriveCeiling derives the sky ceiling from cloud layers, the site elevation,
// and whether the hour has fog. The lowest layer with cover >= 50 is BKN,
// otherwise the lowest with cover >= 20 is SCT, otherwise CLR.
func DeriveCeiling(levels []CloudLevel, elevM float64, fog bool) Ceiling {
	if fog {
		return Ceiling{Kind: "FG"}
	}
	var bknFt, sctFt int
	haveBKN, haveSCT := false, false
	for _, l := range levels {
		if l.GeoM <= 0 {
			continue
		}
		ftAgl := roundTo100FT((l.GeoM - elevM) * ftPerM)
		if !haveBKN && l.Cover >= 50 {
			haveBKN = true
			bknFt = ftAgl
			break
		}
		if !haveSCT && l.Cover >= 20 {
			haveSCT = true
			sctFt = ftAgl
		}
	}
	if haveBKN {
		return Ceiling{Kind: "BKN", FtAgl: bknFt}
	}
	if haveSCT {
		return Ceiling{Kind: "SCT", FtAgl: sctFt}
	}
	return Ceiling{Kind: "CLR"}
}

func roundTo100FT(ft float64) int {
	n := int(ft/100 + 0.5)
	if n < 0 {
		n = 0
	}
	return n * 100
}

// BuildSteps converts a parsed Open-Meteo hourly payload into forecast steps.
func BuildSteps(data map[string][]interface{}, elevM float64) ([]ForecastStep, error) {
	if len(data) == 0 {
		return nil, nil
	}
	times := data["time"]
	if len(times) == 0 {
		return nil, fmt.Errorf("forecast response missing time array")
	}
	steps := make([]ForecastStep, len(times))
	psls := make([]int, len(cloudLevels()))
	for i, p := range cloudLevels() {
		psls[i], _ = strconv.Atoi(p)
	}
	for i := range times {
		step := ForecastStep{
			Time:        time.Unix(epochAt(times, i), 0).UTC(),
			Temp:        numAt(data["temperature_2m"], i),
			Dewp:        numAt(data["dew_point_2m"], i),
			WindDir:     numAt(data["wind_direction_10m"], i),
			WindSpd:     numAt(data["wind_speed_10m"], i),
			Gust:        numAt(data["wind_gusts_10m"], i),
			VisibM:      numAt(data["visibility"], i),
			WXCode:      codeAt(data["weather_code"], i),
			PrecipMm:    numAt(data["precipitation"], i),
			PrecipProb:  codeAt(data["precipitation_probability"], i),
			PressureMsl: numAt(data["pressure_msl"], i),
		}
		levels := make([]CloudLevel, len(cloudLevels()))
		for j, p := range cloudLevels() {
			levels[j] = CloudLevel{
				Psl:   psls[j],
				Cover: codeAt(data["cloud_cover_"+p+"hPa"], i),
				GeoM:  numAt(data["geopotential_height_"+p+"hPa"], i),
			}
		}
		step.Ceiling = DeriveCeiling(levels, elevM, isFog(step.WXCode))
		steps[i] = step
	}
	return steps, nil
}

// OpenMeteoClient fetches model forecasts from the Open-Meteo API.
type OpenMeteoClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewOpenMeteoClient(httpClient *http.Client) *OpenMeteoClient {
	return &OpenMeteoClient{BaseURL: DefaultForecastBaseURL, HTTPClient: httpClient}
}

// GetForecast fetches the hourly forecast for [start, end] at the site.
func (c *OpenMeteoClient) GetForecast(ctx context.Context, lat, lon, elev float64, start, end time.Time) ([]ForecastStep, error) {
	q := url.Values{}
	q.Set("latitude", formatFloat(lat))
	q.Set("longitude", formatFloat(lon))
	q.Set("elevation", formatFloat(elev))
	q.Set("hourly", hourlyVars())
	q.Set("start_hour", start.UTC().Format("2006-01-02T15:04"))
	q.Set("end_hour", end.UTC().Format("2006-01-02T15:04"))
	q.Set("timeformat", "unixtime")
	q.Set("wind_speed_unit", "kn")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/forecast?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Error  bool                     `json:"error"`
		Reason string                   `json:"reason"`
		Hourly map[string][]interface{} `json:"hourly"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		if resp.StatusCode/100 != 2 {
			return nil, fmt.Errorf("api error: status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("decode forecast response: %w", err)
	}
	if payload.Error {
		return nil, fmt.Errorf("forecast error: %s", payload.Reason)
	}
	if resp.StatusCode/100 != 2 {
		if payload.Reason != "" {
			return nil, fmt.Errorf("api error: %s", payload.Reason)
		}
		return nil, fmt.Errorf("api error: status %d", resp.StatusCode)
	}
	return BuildSteps(payload.Hourly, elev)
}

// formatFloat renders a coordinate/elevation without a trailing zero.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func asFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	default:
		return 0
	}
}

func numAt(arr []interface{}, i int) float64 {
	if arr == nil || i < 0 || i >= len(arr) {
		return 0
	}
	return asFloat(arr[i])
}

func codeAt(arr []interface{}, i int) int {
	if arr == nil || i < 0 || i >= len(arr) {
		return 0
	}
	return int(asFloat(arr[i]))
}

func epochAt(arr []interface{}, i int) int64 {
	if arr == nil || i < 0 || i >= len(arr) {
		return 0
	}
	return int64(asFloat(arr[i]))
}
