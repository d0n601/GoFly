package render

import (
	"testing"
	"time"

	"gofly/api"
)

var (
	pdt = time.FixedZone("PDT", -7*3600)
	pst = time.FixedZone("PST", -8*3600)
)

func TestRenderFull(t *testing.T) {
	m := api.Metar{
		IcaoID:  "KEDU",
		Name:    "Davis/University Arpt, CA, US",
		ObsTime: time.Date(2026, 9, 9, 19, 55, 0, 0, time.UTC).Unix(),
		Temp:    34,
		Dewp:    5,
		Wdir:    20,
		Wspd:    7,
		Visib:   "10+",
		Altim:   1017,
		RawOb:   "METAR KEDU 091955Z AUTO 02007KT 10SM 34/05 A3003 RMK AO1",
	}
	got := Render(m, pdt)
	want := "KEDU Davis/University Arpt, CA, US\n" +
		"Observed: 2026-09-09 12:55 PDT (19:55 UTC)\n" +
		"Wind 020° 7kt | Vis 10+ | 34°C/5°C | Altimeter 30.03 inHg\n" +
		"METAR KEDU 091955Z AUTO 02007KT 10SM 34/05 A3003 RMK AO1"
	if got != want {
		t.Fatalf("Render:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderStandardTime(t *testing.T) {
	m := api.Metar{
		IcaoID:  "KEDU",
		ObsTime: time.Date(2026, 1, 15, 19, 55, 0, 0, time.UTC).Unix(),
		Temp:    -2,
		Dewp:    -5,
		Wdir:    0,
		Wspd:    0,
		Visib:   "1/4",
		Altim:   1013,
		RawOb:   "METAR KEDU 151955Z 00000KT 1/4SM 02/M05 A2992",
	}
	got := Render(m, pst)
	want := "KEDU\n" +
		"Observed: 2026-01-15 11:55 PST (19:55 UTC)\n" +
		"Wind 000° 0kt | Vis 1/4 | -2°C/-5°C | Altimeter 29.91 inHg\n" +
		"METAR KEDU 151955Z 00000KT 1/4SM 02/M05 A2992"
	if got != want {
		t.Fatalf("Render:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderMinimal(t *testing.T) {
	m := api.Metar{
		IcaoID:  "KABC",
		ObsTime: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC).Unix(),
	}
	got := Render(m, pst)
	want := "KABC\n" +
		"Observed: 2026-03-01 04:00 PST (12:00 UTC)\n" +
		"Wind 000° 0kt"
	if got != want {
		t.Fatalf("Render:\n got: %q\nwant: %q", got, want)
	}
}
