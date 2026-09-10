package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const fixture = `[{"icaoId":"KEDU","receiptTime":"2026-09-09T19:58:20.536Z","obsTime":1788983700,"reportTime":"2026-09-09T20:00:00.000Z","temp":34,"dewp":5,"wdir":20,"wspd":7,"visib":"10+","altim":1017,"qcField":6,"metarType":"METAR","rawOb":"METAR KEDU 091955Z AUTO 02007KT 10SM 34/05 A3003 RMK AO1","lat":38.5301,"lon":-121.7883,"elev":20,"name":"Davis/University Arpt, CA, US"}]`

func newServer(t *testing.T, status int, body string, captured *url.URL) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if captured != nil {
			*captured = *r.URL
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGetMetarsCurrent(t *testing.T) {
	var got url.URL
	srv := newServer(t, http.StatusOK, fixture, &got)
	c := NewClient(srv.Client())
	c.BaseURL = srv.URL

	metars, err := c.GetMetars(context.Background(), "kedu", false)
	if err != nil {
		t.Fatalf("GetMetars: %v", err)
	}
	if got.Path != "/api/data/metar" {
		t.Errorf("path = %q", got.Path)
	}
	if q := got.Query().Get("ids"); q != "KEDU" {
		t.Errorf("ids = %q, want KEDU", q)
	}
	if q := got.Query().Get("format"); q != "json" {
		t.Errorf("format = %q, want json", q)
	}
	if q := got.Query().Get("hours"); q != "" {
		t.Errorf("hours = %q, want empty", q)
	}
	if len(metars) != 1 {
		t.Fatalf("len = %d, want 1", len(metars))
	}
	m := metars[0]
	if m.IcaoID != "KEDU" || m.Name != "Davis/University Arpt, CA, US" ||
		m.Temp != 34 || m.Dewp != 5 || m.Wdir != "20" || m.Wspd != 7 ||
		m.Visib != "10+" || m.Altim != 1017 ||
		m.RawOb != "METAR KEDU 091955Z AUTO 02007KT 10SM 34/05 A3003 RMK AO1" ||
		m.ObsTime != 1788983700 {
		t.Errorf("unexpected metar: %+v", m)
	}
}

func TestGetMetarsHistory(t *testing.T) {
	var got url.URL
	srv := newServer(t, http.StatusOK, fixture, &got)
	c := NewClient(srv.Client())
	c.BaseURL = srv.URL

	if _, err := c.GetMetars(context.Background(), "KEDU", true); err != nil {
		t.Fatalf("GetMetars: %v", err)
	}
	if q := got.Query().Get("hours"); q != "720" {
		t.Errorf("hours = %q, want 720", q)
	}
}

func TestGetMetarsNumericVisib(t *testing.T) {
	srv := newServer(t, http.StatusOK, `[{"icaoId":"KEDU","obsTime":1788983700,"visib":10,"rawOb":"METAR KEDU"}]`, nil)
	c := NewClient(srv.Client())
	c.BaseURL = srv.URL

	metars, err := c.GetMetars(context.Background(), "KEDU", false)
	if err != nil {
		t.Fatalf("GetMetars: %v", err)
	}
	if metars[0].Visib != "10" {
		t.Fatalf("visib = %q, want %q", metars[0].Visib, "10")
	}
}

func TestGetMetarsVariableWind(t *testing.T) {
	srv := newServer(t, http.StatusOK, `[{"icaoId":"KGOO","obsTime":1788999300,"wdir":"VRB","wspd":5,"rawOb":"METAR KGOO"}]`, nil)
	c := NewClient(srv.Client())
	c.BaseURL = srv.URL

	metars, err := c.GetMetars(context.Background(), "KGOO", false)
	if err != nil {
		t.Fatalf("GetMetars: %v", err)
	}
	if metars[0].Wdir != "VRB" {
		t.Fatalf("wdir = %q, want %q", metars[0].Wdir, "VRB")
	}
}

func TestGetMetarsNotFound(t *testing.T) {
	srv := newServer(t, http.StatusNoContent, "", nil)
	c := NewClient(srv.Client())
	c.BaseURL = srv.URL

	_, err := c.GetMetars(context.Background(), "KZZZ", false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetMetarsEmptyBodyIsNotFound(t *testing.T) {
	srv := newServer(t, http.StatusOK, "  ", nil)
	c := NewClient(srv.Client())
	c.BaseURL = srv.URL

	_, err := c.GetMetars(context.Background(), "KZZZ", false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetMetarsHTTPError(t *testing.T) {
	srv := newServer(t, http.StatusInternalServerError, "boom", nil)
	c := NewClient(srv.Client())
	c.BaseURL = srv.URL

	_, err := c.GetMetars(context.Background(), "KEDU", false)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("err = %v, want status 500 mentioned", err)
	}
}

func TestGetMetarsInvalidJSON(t *testing.T) {
	srv := newServer(t, http.StatusOK, "not json", nil)
	c := NewClient(srv.Client())
	c.BaseURL = srv.URL

	if _, err := c.GetMetars(context.Background(), "KEDU", false); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
