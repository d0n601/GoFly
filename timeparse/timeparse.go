package timeparse

import (
	"fmt"
	"strings"
	"time"

	_ "time/tzdata"
)

// DefaultTimeZone is the single place that controls the timezone gofly uses for
// two things: interpreting times you type without an explicit offset (e.g.
// "14:30" or "2026-09-09 14:30") and printing the local time in reports. It is
// an IANA zone name, so daylight saving is applied automatically — Los Angeles
// is PDT in summer and PST in winter; Chicago would be CDT/CST; Phoenix (most
// of Arizona) stays on MST all year. To change it, edit this one string to any
// other IANA zone, e.g. "America/Chicago" or "America/Denver".
const DefaultTimeZone = "America/Los_Angeles"

// DefaultLocation resolves DefaultTimeZone to a *time.Location.
func DefaultLocation() (*time.Location, error) {
	return time.LoadLocation(DefaultTimeZone)
}

var dateTimeLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

var timeOnlyLayouts = []string{
	"15:04:05",
	"15:04",
}

func ParseTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	la, err := DefaultLocation()
	if err != nil {
		return time.Time{}, err
	}
	for _, layout := range dateTimeLayouts {
		if t, err := time.ParseInLocation(layout, s, la); err == nil {
			return t, nil
		}
	}
	for _, layout := range timeOnlyLayouts {
		if t, err := time.ParseInLocation(layout, s, la); err == nil {
			today := now.In(la)
			return time.Date(today.Year(), today.Month(), today.Day(), t.Hour(), t.Minute(), t.Second(), 0, la), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q (expected RFC3339, \"2006-01-02 15:04\", or \"15:04\")", s)
}
