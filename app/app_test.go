package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"gofly/api"
)

var fixedNow = time.Date(2026, 9, 9, 20, 0, 0, 0, time.UTC)

func unixAt(h, m int) int64 {
	return time.Date(2026, 9, 9, h, m, 0, 0, time.UTC).Unix()
}

type fixtureObs struct {
	IcaoID  string  `json:"icaoId"`
	Name    string  `json:"name"`
	ObsTime int64   `json:"obsTime"`
	Temp    float64 `json:"temp"`
	Dewp    float64 `json:"dewp"`
	Wdir    float64 `json:"wdir"`
	Wspd    float64 `json:"wspd"`
	Visib   string  `json:"visib"`
	Altim   float64 `json:"altim"`
	RawOb   string  `json:"rawOb"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Elev    float64 `json:"elev"`
}

func body(obs ...fixtureObs) string {
	b, _ := json.Marshal(obs)
	return string(b)
}

func appWith(srvURL string, out *strings.Builder) *App {
	c := api.NewClient(&http.Client{})
	c.BaseURL = srvURL
	return &App{
		Client: c,
		Out:    out,
		Now:    func() time.Time { return fixedNow },
	}
}

func appWithForecast(metarURL, forecastURL string, out *strings.Builder) *App {
	mc := api.NewClient(&http.Client{})
	mc.BaseURL = metarURL
	fc := api.NewOpenMeteoClient(&http.Client{})
	fc.BaseURL = forecastURL
	return &App{
		Client:   mc,
		Forecast: fc,
		Out:      out,
		Now:      func() time.Time { return fixedNow },
	}
}

func TestRunNoArgs(t *testing.T) {
	a := &App{}
	err := a.Run(context.Background(), nil)
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("err = %v, want ErrUsage", err)
	}
}

func TestRunTooManyArgs(t *testing.T) {
	a := &App{}
	err := a.Run(context.Background(), []string{"KEDU", "12:00", "extra"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("err = %v, want ErrUsage", err)
	}
}

func TestRunInvalidICAO(t *testing.T) {
	for _, icao := range []string{"12", "TOOLONG", "KE-DU"} {
		a := &App{}
		err := a.Run(context.Background(), []string{icao})
		if err == nil || !strings.Contains(err.Error(), "ICAO") {
			t.Errorf("icao %q: err = %v, want ICAO error", icao, err)
		}
	}
}

func TestRunCurrent(t *testing.T) {
	var pathQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathQuery = r.URL.String()
		io.WriteString(w, body(fixtureObs{
			IcaoID: "KEDU", Name: "Davis/University Arpt, CA, US",
			ObsTime: unixAt(19, 55), RawOb: "METAR KEDU 091955Z AUTO 02007KT 10SM 34/05 A3003 RMK AO1",
			Temp: 34, Dewp: 5, Wdir: 20, Wspd: 7, Visib: "10+", Altim: 1017,
		}))
	}))
	t.Cleanup(srv.Close)

	out := &strings.Builder{}
	a := appWith(srv.URL, out)
	if err := a.Run(context.Background(), []string{"kedu"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.HasPrefix(pathQuery, "/api/data/metar?") {
		t.Fatalf("path = %q", pathQuery)
	}
	u, _ := url.ParseRequestURI(pathQuery)
	if q := u.Query().Get("ids"); q != "KEDU" {
		t.Errorf("ids = %q, want KEDU", q)
	}
	if q := u.Query().Get("hours"); q != "" {
		t.Errorf("hours = %q, want empty for current", q)
	}
	got := out.String()
	for _, want := range []string{"KEDU Davis/University Arpt", "METAR KEDU 091955Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunWithTimePicksClosest(t *testing.T) {
	fixture := []fixtureObs{
		{IcaoID: "KEDU", ObsTime: unixAt(19, 20), RawOb: "METAR KEDU later"},
		{IcaoID: "KEDU", ObsTime: unixAt(19, 0), RawOb: "METAR KEDU closest"},
		{IcaoID: "KEDU", ObsTime: unixAt(18, 40), RawOb: "METAR KEDU older"},
	}
	var pathQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathQuery = r.URL.String()
		io.WriteString(w, body(fixture...))
	}))
	t.Cleanup(srv.Close)

	out := &strings.Builder{}
	a := appWith(srv.URL, out)
	if err := a.Run(context.Background(), []string{"KEDU", "2026-09-09 12:05"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	u, _ := url.ParseRequestURI(pathQuery)
	if q := u.Query().Get("hours"); q != "720" {
		t.Errorf("hours = %q, want 720", q)
	}
	got := out.String()
	if !strings.Contains(got, "METAR KEDU closest") {
		t.Fatalf("output missing closest obs:\n%s", got)
	}
	if strings.Contains(got, "later") || strings.Contains(got, "older") {
		t.Fatalf("output contains wrong obs:\n%s", got)
	}
}

func TestRunFutureTimePicksLatestObservation(t *testing.T) {
	fixture := []fixtureObs{
		{IcaoID: "KEDU", ObsTime: unixAt(19, 55), RawOb: "METAR KEDU latest"},
		{IcaoID: "KEDU", ObsTime: unixAt(18, 55), RawOb: "METAR KEDU older"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body(fixture...))
	}))
	t.Cleanup(srv.Close)

	out := &strings.Builder{}
	a := appWith(srv.URL, out)
	if err := a.Run(context.Background(), []string{"KEDU", "2026-09-09T21:00:00Z"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "METAR KEDU latest") {
		t.Fatalf("output missing latest obs:\n%s", got)
	}
}

func TestRunNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	out := &strings.Builder{}
	a := appWith(srv.URL, out)
	err := a.Run(context.Background(), []string{"KZZZ"})
	if err == nil || !strings.Contains(err.Error(), "no METAR data found for KZZZ") {
		t.Fatalf("err = %v, want not-found for KZZZ", err)
	}
}

func TestRunHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "boom")
	}))
	t.Cleanup(srv.Close)

	out := &strings.Builder{}
	a := appWith(srv.URL, out)
	err := a.Run(context.Background(), []string{"KEDU"})
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("err = %v, want status 500", err)
	}
}

func TestRunInvalidTime(t *testing.T) {
	a := &App{}
	err := a.Run(context.Background(), []string{"KEDU", "banana"})
	if err == nil || !strings.Contains(err.Error(), "invalid time") {
		t.Fatalf("err = %v, want invalid time", err)
	}
}

func TestParseArgs(t *testing.T) {
	for _, tc := range []struct {
		args     []string
		icao, ts string
		forecast bool
		wantErr  bool
	}{
		{nil, "", "", false, true},
		{[]string{"KEDU"}, "KEDU", "", false, false},
		{[]string{"KEDU", "12:00"}, "KEDU", "12:00", false, false},
		{[]string{"KEDU", "--forecast"}, "KEDU", "", true, false},
		{[]string{"--forecast", "KEDU"}, "KEDU", "", true, false},
		{[]string{"KEDU", "-f", "12:00"}, "KEDU", "12:00", true, false},
		{[]string{"KEDU", "12:00", "--forecast"}, "KEDU", "12:00", true, false},
		{[]string{"KEDU", "--forecast", "--forecast"}, "KEDU", "", true, false},
		{[]string{"KEDU", "12:00", "extra"}, "", "", false, true},
		{[]string{"KEDU", "--bogus"}, "", "", false, true},
	} {
		icao, ts, forecast, err := parseArgs(tc.args)
		if tc.wantErr {
			if err == nil || !errors.Is(err, ErrUsage) {
				t.Errorf("parseArgs(%v) err = %v, want ErrUsage", tc.args, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseArgs(%v) unexpected err: %v", tc.args, err)
			continue
		}
		if icao != tc.icao || ts != tc.ts || forecast != tc.forecast {
			t.Errorf("parseArgs(%v) = %q,%q,%v, want %q,%q,%v", tc.args, icao, ts, forecast, tc.icao, tc.ts, tc.forecast)
		}
	}
}

func TestRunUnknownFlag(t *testing.T) {
	a := &App{}
	err := a.Run(context.Background(), []string{"KEDU", "--bogus"})
	if err == nil || !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("err = %v, want unknown flag usage error", err)
	}
}

func TestRunForecastCurrent(t *testing.T) {
	var metarQuery, forecastQuery string
	metarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metarQuery = r.URL.String()
		io.WriteString(w, body(fixtureObs{
			IcaoID: "KEDU", Name: "Davis/University Arpt, CA, US",
			ObsTime: unixAt(19, 55), RawOb: "METAR KEDU 091955Z AUTO 02007KT 10SM 34/05 A3003",
			Temp: 34, Dewp: 5, Wdir: 20, Wspd: 7, Visib: "10+", Altim: 1017,
			Lat: 38.5301, Lon: -121.7883, Elev: 20,
		}))
	}))
	t.Cleanup(metarSrv.Close)
	fx := `{"hourly":{"time":` +
		fmt.Sprintf("[%d,%d,%d]", unixAt(19, 0), unixAt(20, 0), unixAt(21, 0)) +
		`,"temperature_2m":[13.1,14.2,15.8],"dew_point_2m":[9.3,8.3,7.5]}}`
	forecastSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forecastQuery = r.URL.String()
		io.WriteString(w, fx)
	}))
	t.Cleanup(forecastSrv.Close)

	out := &strings.Builder{}
	a := appWithForecast(metarSrv.URL, forecastSrv.URL, out)
	if err := a.Run(context.Background(), []string{"KEDU", "--forecast"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	mu, _ := url.ParseRequestURI(metarQuery)
	if q := mu.Query().Get("hours"); q != "" {
		t.Errorf("metar hours = %q, want empty for forecast", q)
	}
	fu, _ := url.ParseRequestURI(forecastQuery)
	if q := fu.Query().Get("start_hour"); q != "2026-09-09T19:00" {
		t.Errorf("start_hour = %q, want 2026-09-09T19:00", q)
	}
	if q := fu.Query().Get("end_hour"); q != "2026-09-09T21:00" {
		t.Errorf("end_hour = %q, want 2026-09-09T21:00", q)
	}
	if q := fu.Query().Get("latitude"); q != "38.5301" || fu.Query().Get("longitude") != "-121.7883" {
		t.Errorf("coords = %q/%q", fu.Query().Get("latitude"), fu.Query().Get("longitude"))
	}
	got := out.String()
	for _, want := range []string{
		"KEDU Davis/University Arpt",
		"Forecast: 2026-09-09 12:00 PDT (19:00 UTC)",
		"Forecast: 2026-09-09 13:00 PDT (20:00 UTC)",
		"Forecast: 2026-09-09 14:00 PDT (21:00 UTC)",
		"13.1°C/9.3°C",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "METAR") {
		t.Errorf("forecast output should not contain the raw METAR:\n%s", got)
	}
}

func TestRunForecastWithTime(t *testing.T) {
	var metarQuery, forecastQuery string
	metarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metarQuery = r.URL.String()
		io.WriteString(w, body(fixtureObs{
			IcaoID: "KEDU", Name: "Davis/University Arpt, CA, US",
			ObsTime: unixAt(19, 55), RawOb: "METAR KEDU 091955Z AUTO 02007KT 10SM 34/05 A3003",
			Temp: 34, Dewp: 5, Wdir: 20, Wspd: 7, Visib: "10+", Altim: 1017,
			Lat: 38.5301, Lon: -121.7883, Elev: 20,
		}))
	}))
	t.Cleanup(metarSrv.Close)
	day := time.Date(2026, 9, 16, 13, 0, 0, 0, time.UTC)
	fx := `{"hourly":{"time":` +
		fmt.Sprintf("[%d,%d,%d]", day.Unix(), day.Add(time.Hour).Unix(), day.Add(2*time.Hour).Unix()) +
		`,"temperature_2m":[11.2,12.0,12.8]}}`
	forecastSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forecastQuery = r.URL.String()
		io.WriteString(w, fx)
	}))
	t.Cleanup(forecastSrv.Close)

	out := &strings.Builder{}
	a := appWithForecast(metarSrv.URL, forecastSrv.URL, out)
	if err := a.Run(context.Background(), []string{"KEDU", "2026-09-16 07:00", "--forecast"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	mu, _ := url.ParseRequestURI(metarQuery)
	if q := mu.Query().Get("hours"); q != "" {
		t.Errorf("metar hours = %q, want empty for forecast", q)
	}
	fu, _ := url.ParseRequestURI(forecastQuery)
	if q := fu.Query().Get("start_hour"); q != "2026-09-16T13:00" {
		t.Errorf("start_hour = %q, want 2026-09-16T13:00", q)
	}
	if q := fu.Query().Get("end_hour"); q != "2026-09-16T15:00" {
		t.Errorf("end_hour = %q, want 2026-09-16T15:00", q)
	}
	got := out.String()
	for _, want := range []string{
		"Forecast: 2026-09-16 06:00 PDT (13:00 UTC)",
		"Forecast: 2026-09-16 07:00 PDT (14:00 UTC)",
		"Forecast: 2026-09-16 08:00 PDT (15:00 UTC)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunForecastNotFound(t *testing.T) {
	metarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(metarSrv.Close)
	forecastSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"hourly":{}}`)
	}))
	t.Cleanup(forecastSrv.Close)

	out := &strings.Builder{}
	a := appWithForecast(metarSrv.URL, forecastSrv.URL, out)
	err := a.Run(context.Background(), []string{"KZZZ", "--forecast"})
	if err == nil || !strings.Contains(err.Error(), "no METAR data found for KZZZ") {
		t.Fatalf("err = %v, want not-found for KZZZ", err)
	}
}

func TestRunForecastClientError(t *testing.T) {
	metarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body(fixtureObs{IcaoID: "KEDU", ObsTime: unixAt(19, 55), Lat: 38.5, Lon: -121.8, Elev: 20}))
	}))
	t.Cleanup(metarSrv.Close)
	forecastSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":true,"reason":"Parameter 'start_hour' is out of allowed range from 1940-01-01T00:00 to 2140-12-28T23:00"}`)
	}))
	t.Cleanup(forecastSrv.Close)

	out := &strings.Builder{}
	a := appWithForecast(metarSrv.URL, forecastSrv.URL, out)
	err := a.Run(context.Background(), []string{"KEDU", "--forecast"})
	if err == nil || !strings.Contains(err.Error(), "out of allowed range") {
		t.Fatalf("err = %v, want out-of-range reason", err)
	}
}

func TestRunForecastWithoutClient(t *testing.T) {
	metarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body(fixtureObs{IcaoID: "KEDU", ObsTime: unixAt(19, 55), Lat: 38.5, Lon: -121.8, Elev: 20}))
	}))
	t.Cleanup(metarSrv.Close)

	out := &strings.Builder{}
	a := appWith(metarSrv.URL, out)
	err := a.Run(context.Background(), []string{"KEDU", "--forecast"})
	if err == nil || !strings.Contains(err.Error(), "forecast not available") {
		t.Fatalf("err = %v, want forecast not available", err)
	}
}
