package pick

import (
	"testing"
	"time"

	"gofly/api"
)

func obsAt(unix int64) api.Metar {
	return api.Metar{IcaoID: "KEDU", ObsTime: unix, RawOb: "METAR KEDU"}
}

func at(hour, min int) int64 {
	return time.Date(2026, 9, 9, hour, min, 0, 0, time.UTC).Unix()
}

func TestClosestBeforeTarget(t *testing.T) {
	metars := []api.Metar{obsAt(at(12, 20)), obsAt(at(11, 50))}
	got, err := Closest(metars, timeUn(at(12, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if got.ObsTime != at(11, 50) {
		t.Fatalf("got obs %d, want %d", got.ObsTime, at(11, 50))
	}
}

func TestClosestAfterTarget(t *testing.T) {
	metars := []api.Metar{obsAt(at(12, 10)), obsAt(at(12, 20))}
	got, err := Closest(metars, timeUn(at(12, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if got.ObsTime != at(12, 10) {
		t.Fatalf("got obs %d, want %d", got.ObsTime, at(12, 10))
	}
}

func TestClosestFutureTargetPicksLatest(t *testing.T) {
	metars := []api.Metar{obsAt(at(12, 0)), obsAt(at(10, 0))}
	got, err := Closest(metars, timeUn(at(15, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if got.ObsTime != at(12, 0) {
		t.Fatalf("got obs %d, want %d", got.ObsTime, at(12, 0))
	}
}

func TestClosestExactMatch(t *testing.T) {
	metars := []api.Metar{obsAt(at(12, 30)), obsAt(at(12, 0))}
	got, err := Closest(metars, timeUn(at(12, 30)))
	if err != nil {
		t.Fatal(err)
	}
	if got.ObsTime != at(12, 30) {
		t.Fatalf("got obs %d, want %d", got.ObsTime, at(12, 30))
	}
}

func TestClosestTiePrefersEarlier(t *testing.T) {
	metars := []api.Metar{obsAt(at(12, 10)), obsAt(at(11, 50))}
	got, err := Closest(metars, timeUn(at(12, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if got.ObsTime != at(11, 50) {
		t.Fatalf("got obs %d, want %d", got.ObsTime, at(11, 50))
	}
}

func TestClosestEmpty(t *testing.T) {
	if _, err := Closest(nil, timeUn(at(12, 0))); err == nil {
		t.Fatal("expected error for empty list")
	}
}

func TestClosestSingle(t *testing.T) {
	metars := []api.Metar{obsAt(at(12, 0))}
	got, err := Closest(metars, timeUn(at(12, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if got.ObsTime != at(12, 0) {
		t.Fatalf("got obs %d, want %d", got.ObsTime, at(12, 0))
	}
}

func timeUn(unix int64) time.Time {
	return time.Unix(unix, 0).UTC()
}
