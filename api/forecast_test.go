package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDeriveCeilingCLR(t *testing.T) {
	c := DeriveCeiling([]CloudLevel{{Psl: 850, Cover: 0}, {Psl: 700, Cover: 10}}, 20, false)
	if c.Kind != "CLR" || c.FtAgl != 0 {
		t.Fatalf("DeriveCeiling = %+v, want CLR 0", c)
	}
}

func TestDeriveCeilingEmptyIsCLR(t *testing.T) {
	c := DeriveCeiling(nil, 20, false)
	if c.Kind != "CLR" || c.FtAgl != 0 {
		t.Fatalf("DeriveCeiling = %+v, want CLR 0", c)
	}
}

func TestDeriveCeilingBKNRoundUp(t *testing.T) {
	// 850 hPa level at 1500 m MSL, site at 10 m: (1490)*3.28084 = 4888 ft -> 4900
	c := DeriveCeiling([]CloudLevel{
		{Psl: 925, Cover: 0},
		{Psl: 850, Cover: 60, GeoM: 1500},
	}, 10, false)
	if c.Kind != "BKN" || c.FtAgl != 4900 {
		t.Fatalf("DeriveCeiling = %+v, want BKN 4900", c)
	}
}

func TestDeriveCeilingSCT(t *testing.T) {
	// 700 hPa level at 1000 m MSL, site at 0: 3280.84 ft -> 3300
	c := DeriveCeiling([]CloudLevel{
		{Psl: 925, Cover: 20},
		{Psl: 850, Cover: 10},
		{Psl: 700, Cover: 30, GeoM: 1000},
	}, 0, false)
	if c.Kind != "SCT" || c.FtAgl != 3300 {
		t.Fatalf("DeriveCeiling = %+v, want SCT 3300", c)
	}
}

func TestDeriveCeilingSkipsSCTForBKN(t *testing.T) {
	c := DeriveCeiling([]CloudLevel{
		{Psl: 925, Cover: 25, GeoM: 300},
		{Psl: 850, Cover: 55, GeoM: 950}, // 3116.8 ft -> 3100
	}, 0, false)
	if c.Kind != "BKN" || c.FtAgl != 3100 {
		t.Fatalf("DeriveCeiling = %+v, want BKN 3100", c)
	}
}

func TestDeriveCeilingFog(t *testing.T) {
	c := DeriveCeiling([]CloudLevel{{Psl: 925, Cover: 95, GeoM: 1500}}, 20, true)
	if c.Kind != "FG" || c.FtAgl != 0 {
		t.Fatalf("DeriveCeiling = %+v, want FG 0", c)
	}
}

func levelVars(data map[string][]interface{}, cover, geoM float64) {
	for _, p := range cloudLevels() {
		coverKey := "cloud_cover_" + p + "hPa"
		geoKey := "geopotential_height_" + p + "hPa"
		if data[coverKey] == nil {
			data[coverKey] = []interface{}{cover}
			data[geoKey] = []interface{}{geoM}
		}
	}
}

func sampleHourly() map[string][]interface{} {
	data := map[string][]interface{}{
		"time":                      {1789000000, 1789003600, 1789007200},
		"temperature_2m":            {13.1, 14.2, 15.8},
		"dew_point_2m":              {9.3, 8.3, 7.5},
		"wind_speed_10m":            {3.7, 3.2, 20.1},
		"wind_direction_10m":        {242, 256, 274},
		"wind_gusts_10m":            {3.7, 3.9, 25.0},
		"visibility":                {24140.0, 8000.0, 160934.0},
		"weather_code":              {0, 61, 3},
		"precipitation":             {0.0, 1.2, 0.0},
		"precipitation_probability": {0, 80, 10},
		"pressure_msl":              {1014.7, 1015.4, 1016.0},
	}
	levelVars(data, 0, 1500)
	return data
}

func TestBuildStepsFields(t *testing.T) {
	steps, err := BuildSteps(sampleHourly(), 20)
	if err != nil {
		t.Fatalf("BuildSteps: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("len(steps) = %d, want 3", len(steps))
	}

	t0 := time.Unix(1789000000, 0).UTC()
	s := steps[0]
	if !s.Time.Equal(t0) {
		t.Errorf("steps[0].Time = %v, want %v", s.Time, t0)
	}
	if s.Temp != 13.1 || s.Dewp != 9.3 {
		t.Errorf("temp/dewp = %.1f/%.1f, want 13.1/9.3", s.Temp, s.Dewp)
	}
	if s.WindDir != 242 || s.WindSpd != 3.7 || s.Gust != 3.7 {
		t.Errorf("wind = %.0f/%.1f/%.1f, want 242/3.7/3.7", s.WindDir, s.WindSpd, s.Gust)
	}
	if s.VisibM != 24140 {
		t.Errorf("visib = %.0f, want 24140", s.VisibM)
	}
	if s.WXCode != 0 {
		t.Errorf("wx = %d, want 0", s.WXCode)
	}
	if s.PrecipMm != 0 || s.PrecipProb != 0 {
		t.Errorf("precip = %.1f/%d, want 0/0", s.PrecipMm, s.PrecipProb)
	}
	if s.PressureMsl != 1014.7 {
		t.Errorf("msl = %.1f, want 1014.7", s.PressureMsl)
	}
	if s.Ceiling.Kind != "CLR" {
		t.Errorf("ceiling = %+v, want CLR", s.Ceiling)
	}

	p := steps[1]
	if p.WXCode != 61 || p.PrecipMm != 1.2 || p.PrecipProb != 80 {
		t.Errorf("step1 wx/precip = %d/%.1f/%d, want 61/1.2/80", p.WXCode, p.PrecipMm, p.PrecipProb)
	}
}

func TestBuildStepsCeilingFog(t *testing.T) {
	data := sampleHourly()
	data["weather_code"] = []interface{}{45, 61, 3}
	levelVars(data, 90, 2900)
	data["cloud_cover_925hPa"] = []interface{}{90, 90, 90}
	data["geopotential_height_925hPa"] = []interface{}{2900.0, 2900.0, 2900.0}

	steps, err := BuildSteps(data, 20)
	if err != nil {
		t.Fatalf("BuildSteps: %v", err)
	}
	// Fog step: WMO 45 -> FG despite BKN 925 hPa layer.
	if steps[0].Ceiling.Kind != "FG" {
		t.Errorf("steps[0].ceiling = %+v, want FG", steps[0].Ceiling)
	}
	// Non-fog step with 90% at 925: BKN at (2900-20)*3.28084 = 9448.8 -> 9400.
	if steps[1].Ceiling.Kind != "BKN" || steps[1].Ceiling.FtAgl != 9400 {
		t.Errorf("steps[1].ceiling = %+v, want BKN 9400", steps[1].Ceiling)
	}
}

func TestBuildStepsMissingVariablesAreZero(t *testing.T) {
	steps, err := BuildSteps(map[string][]interface{}{
		"time": {1789000000},
	}, 20)
	if err != nil {
		t.Fatalf("BuildSteps: %v", err)
	}
	s := steps[0]
	if s.Temp != 0 || s.WindSpd != 0 || s.WXCode != 0 || s.VisibM != 0 {
		t.Errorf("missing vars should be zero, got %+v", s)
	}
	if s.Ceiling.Kind != "CLR" {
		t.Errorf("ceiling = %+v, want CLR", s.Ceiling)
	}
}

func TestBuildStepsNoTimeIsError(t *testing.T) {
	if _, err := BuildSteps(map[string][]interface{}{"temperature_2m": {1.0}}, 20); err == nil {
		t.Fatal("expected error when time array missing")
	}
}

func TestOpenMeteoURL(t *testing.T) {
	var got url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = *r.URL
		io.WriteString(w, `{"hourly":{}}`)
	}))
	t.Cleanup(srv.Close)

	c := NewOpenMeteoClient(srv.Client())
	c.BaseURL = srv.URL

	start := time.Date(2026, 9, 16, 13, 0, 0, 0, time.FixedZone("PDT", -7*3600))
	end := time.Date(2026, 9, 16, 15, 0, 0, 0, time.FixedZone("PDT", -7*3600))
	if _, err := c.GetForecast(context.Background(), 38.5301, -121.7883, 20, start, end); err != nil {
		t.Fatalf("GetForecast: %v", err)
	}
	if got.Path != "/v1/forecast" {
		t.Errorf("path = %q, want /v1/forecast", got.Path)
	}
	q := got.Query()
	if q.Get("latitude") != "38.5301" || q.Get("longitude") != "-121.7883" || q.Get("elevation") != "20" {
		t.Errorf("coords = %v/%v/%v", q.Get("latitude"), q.Get("longitude"), q.Get("elevation"))
	}
	if q.Get("start_hour") != "2026-09-16T20:00" {
		t.Errorf("start_hour = %q, want 2026-09-16T20:00 (UTC)", q.Get("start_hour"))
	}
	if q.Get("end_hour") != "2026-09-16T22:00" {
		t.Errorf("end_hour = %q, want 2026-09-16T22:00 (UTC)", q.Get("end_hour"))
	}
	if q.Get("timeformat") != "unixtime" || q.Get("wind_speed_unit") != "kn" {
		t.Errorf("timeformat/wind_speed_unit = %q/%q", q.Get("timeformat"), q.Get("wind_speed_unit"))
	}
	hourly := q.Get("hourly")
	for _, want := range []string{"temperature_2m", "dew_point_2m", "wind_speed_10m", "wind_gusts_10m",
		"visibility", "weather_code", "precipitation_probability", "pressure_msl",
		"cloud_cover_850hPa", "geopotential_height_100hPa"} {
		if !strings.Contains(hourly, want) {
			t.Errorf("hourly missing %q: %s", want, hourly)
		}
	}
}

func TestOpenMeteoSuccess(t *testing.T) {
	fx := `{"hourly":{` +
		`"time":[1789752000],` +
		`"temperature_2m":[11.2],"dew_point_2m":[8.0],` +
		`"wind_speed_10m":[5.5],"wind_direction_10m":[250.0],"wind_gusts_10m":[12.3],` +
		`"visibility":[15990.0],"weather_code":[61],"precipitation":[0.8],` +
		`"precipitation_probability":[70],"pressure_msl":[1012.4]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, fx)
	}))
	t.Cleanup(srv.Close)

	c := NewOpenMeteoClient(srv.Client())
	c.BaseURL = srv.URL

	steps, err := c.GetForecast(context.Background(), 38.5, -121.8, 20,
		time.Unix(1789752000, 0), time.Unix(1789752000, 0))
	if err != nil {
		t.Fatalf("GetForecast: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("len(steps) = %d, want 1", len(steps))
	}
	s := steps[0]
	if s.Temp != 11.2 || s.WXCode != 61 || s.PrecipProb != 70 || s.WindSpd != 5.5 {
		t.Errorf("unexpected step: %+v", s)
	}
	if !s.Time.Equal(time.Unix(1789752000, 0).UTC()) {
		t.Errorf("time = %v, want epoch 1789752000 UTC", s.Time)
	}
}

func TestOpenMeteoErrorReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":true,"reason":"Parameter 'start_hour' is out of allowed range from 1940-01-01T00:00 to 2140-12-28T23:00"}`)
	}))
	t.Cleanup(srv.Close)

	c := NewOpenMeteoClient(srv.Client())
	c.BaseURL = srv.URL
	_, err := c.GetForecast(context.Background(), 38.5, -121.8, 20,
		time.Unix(1, 0), time.Unix(2, 0))
	if err == nil || !strings.Contains(err.Error(), "out of allowed range") {
		t.Fatalf("err = %v, want out-of-range reason", err)
	}
}

func TestOpenMeteoHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "boom")
	}))
	t.Cleanup(srv.Close)

	c := NewOpenMeteoClient(srv.Client())
	c.BaseURL = srv.URL
	_, err := c.GetForecast(context.Background(), 38.5, -121.8, 20,
		time.Unix(1789752000, 0), time.Unix(1789752000, 0))
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("err = %v, want status 500", err)
	}
}

func TestOpenMeteoInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "not json")
	}))
	t.Cleanup(srv.Close)

	c := NewOpenMeteoClient(srv.Client())
	c.BaseURL = srv.URL
	if _, err := c.GetForecast(context.Background(), 38.5, -121.8, 20,
		time.Unix(1789752000, 0), time.Unix(1789752000, 0)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestOpenMeteo200WithErrorFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"error":true,"reason":"boom"}`)
	}))
	t.Cleanup(srv.Close)

	c := NewOpenMeteoClient(srv.Client())
	c.BaseURL = srv.URL
	_, err := c.GetForecast(context.Background(), 38.5, -121.8, 20,
		time.Unix(1789752000, 0), time.Unix(1789752000, 0))
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v, want reason boom", err)
	}
}
