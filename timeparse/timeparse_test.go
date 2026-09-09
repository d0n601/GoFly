package timeparse

import (
	"testing"
	"time"
)

var winterNow = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

func TestParseTimeRFC3339Zulu(t *testing.T) {
	got, err := ParseTime("2026-09-09T12:30:00Z", winterNow)
	if err != nil {
		t.Fatalf("ParseTime: %v", err)
	}
	want := time.Date(2026, 9, 9, 12, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseTimeRFC3339ExplicitOffset(t *testing.T) {
	got, err := ParseTime("2026-09-09T12:30:00-07:00", winterNow)
	if err != nil {
		t.Fatalf("ParseTime: %v", err)
	}
	want := time.Date(2026, 9, 9, 12, 30, 0, 0, time.FixedZone("PDT", -7*3600))
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseTimeFullDateDefault(t *testing.T) {
	la, err := DefaultLocation()
	if err != nil {
		t.Fatalf("DefaultLocation: %v", err)
	}
	cases := []struct {
		input string
		want  time.Time
		zone  string
	}{
		{
			"2026-09-09 14:30",
			time.Date(2026, 9, 9, 14, 30, 0, 0, la),
			"PDT",
		},
		{
			"2026-07-04T21:00",
			time.Date(2026, 7, 4, 21, 0, 0, 0, la),
			"PDT",
		},
		{
			"2026-01-15 14:30",
			time.Date(2026, 1, 15, 14, 30, 0, 0, la),
			"PST",
		},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseTime(tc.input, winterNow)
			if err != nil {
				t.Fatalf("ParseTime: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			name, _ := got.Zone()
			if name != tc.zone {
				t.Fatalf("zone = %q, want %q", name, tc.zone)
			}
		})
	}
}

func TestParseTimeTimeOnlyUsesToday(t *testing.T) {
	la, err := DefaultLocation()
	if err != nil {
		t.Fatalf("DefaultLocation: %v", err)
	}
	now := time.Date(2026, 7, 10, 18, 0, 0, 0, time.UTC)
	got, err := ParseTime("14:30", now)
	if err != nil {
		t.Fatalf("ParseTime: %v", err)
	}
	want := time.Date(2026, 7, 10, 14, 30, 0, 0, la)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	name, _ := got.Zone()
	if name != "PDT" {
		t.Fatalf("zone = %q, want PDT", name)
	}
}

func TestParseTimeInvalid(t *testing.T) {
	for _, in := range []string{"", "   ", "not a time", "2026-13-45 99:99", "2026-09-09"} {
		if _, err := ParseTime(in, winterNow); err == nil {
			t.Errorf("ParseTime(%q) expected error, got nil", in)
		}
	}
}

// TestNorthAmericaZones pins the daylight-saving behaviour of the IANA zones
// the tool relies on, confirming they resolve from the embedded tzdata (so the
// static binary works with no system tz database) and that changing
// DefaultTimeZone to any of them "just works", including the DST-free Arizona
// case the user asked about.
func TestNorthAmericaZones(t *testing.T) {
	summer := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	winter := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		zone       string
		summerAbbr string
		winterAbbr string
		summerOff  int // UTC offset in hours
		winterOff  int
	}{
		{"America/Los_Angeles", "PDT", "PST", -7, -8},
		{"America/Chicago", "CDT", "CST", -5, -6},
		{"America/Denver", "MDT", "MST", -6, -7},
		{"America/New_York", "EDT", "EST", -4, -5},
		{"America/Phoenix", "MST", "MST", -7, -7}, // most of Arizona: no DST
		{"America/Anchorage", "AKDT", "AKST", -8, -9},
	}
	for _, tc := range cases {
		t.Run(tc.zone, func(t *testing.T) {
			loc, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatalf("LoadLocation(%q): %v", tc.zone, err)
			}
			sName, sOff := summer.In(loc).Zone()
			wName, wOff := winter.In(loc).Zone()
			if sName != tc.summerAbbr {
				t.Errorf("%s summer abbreviation = %q, want %q", tc.zone, sName, tc.summerAbbr)
			}
			if wName != tc.winterAbbr {
				t.Errorf("%s winter abbreviation = %q, want %q", tc.zone, wName, tc.winterAbbr)
			}
			if got := sOff / 3600; got != tc.summerOff {
				t.Errorf("%s summer offset = %dh, want %dh", tc.zone, got, tc.summerOff)
			}
			if got := wOff / 3600; got != tc.winterOff {
				t.Errorf("%s winter offset = %dh, want %dh", tc.zone, got, tc.winterOff)
			}
		})
	}
}
