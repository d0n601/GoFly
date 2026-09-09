package render

import (
	"testing"
	"time"

	"gofly/api"
)

func TestRenderForecastClear(t *testing.T) {
	steps := []api.ForecastStep{{
		Time:        time.Date(2026, 9, 9, 19, 0, 0, 0, time.UTC),
		Temp:        13.1,
		Dewp:        9.3,
		WindDir:     242,
		WindSpd:     4,
		Gust:        4,
		VisibM:      160934,
		WXCode:      0,
		PressureMsl: 1014.7,
		Ceiling:     api.Ceiling{Kind: "CLR"},
	}}
	got := RenderForecast("KEDU", "Davis/University Arpt, CA, US", pdt, steps)
	want := "KEDU Davis/University Arpt, CA, US\n" +
		"Forecast: 2026-09-09 12:00 PDT (19:00 UTC)\n" +
		"Wind 242° 04kt | Vis 15+ SM | Sky CLR | 13.1°C/9.3°C | Altimeter 29.96 inHg"
	if got != want {
		t.Fatalf("RenderForecast:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderForecastBKNRain(t *testing.T) {
	steps := []api.ForecastStep{{
		Time:        time.Date(2026, 9, 9, 20, 0, 0, 0, time.UTC),
		Temp:        14.2,
		Dewp:        8.3,
		WindDir:     12,
		WindSpd:     5,
		Gust:        14,
		VisibM:      6437,
		WXCode:      61,
		PrecipMm:    2.1,
		PrecipProb:  80,
		PressureMsl: 1015.2,
		Ceiling:     api.Ceiling{Kind: "BKN", FtAgl: 9400},
	}}
	got := RenderForecast("KEDU", "Davis/University Arpt, CA, US", pdt, steps)
	want := "KEDU Davis/University Arpt, CA, US\n" +
		"Forecast: 2026-09-09 13:00 PDT (20:00 UTC)\n" +
		"Wind 012° 05kt G14 | Vis 4.0 SM | Sky BKN 09400 | Rain | 14.2°C/8.3°C | Altimeter 29.98 inHg | 2.1mm (80%)"
	if got != want {
		t.Fatalf("RenderForecast:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderForecastFogPST(t *testing.T) {
	steps := []api.ForecastStep{{
		Time:        time.Date(2026, 1, 15, 19, 0, 0, 0, time.UTC),
		Temp:        -2.5,
		Dewp:        -2.5,
		VisibM:      500,
		WXCode:      45,
		PressureMsl: 1022.4,
		Ceiling:     api.Ceiling{Kind: "FG"},
	}}
	got := RenderForecast("KEDU", "", pst, steps)
	want := "KEDU\n" +
		"Forecast: 2026-01-15 11:00 PST (19:00 UTC)\n" +
		"Wind 000° 00kt | Vis 0.3 SM | Sky FG | -2.5°C/-2.5°C | Altimeter 30.19 inHg"
	if got != want {
		t.Fatalf("RenderForecast:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderForecastMultipleSteps(t *testing.T) {
	steps := []api.ForecastStep{
		{
			Time:        time.Date(2026, 9, 16, 13, 0, 0, 0, time.UTC),
			Temp:        11.2,
			Dewp:        8.0,
			WindDir:     250,
			WindSpd:     11,
			VisibM:      12000,
			WXCode:      2,
			PressureMsl: 1012.4,
			Ceiling:     api.Ceiling{Kind: "SCT", FtAgl: 3000},
		},
		{
			Time:        time.Date(2026, 9, 16, 14, 0, 0, 0, time.UTC),
			Temp:        12.0,
			Dewp:        8.2,
			WindDir:     255,
			WindSpd:     9,
			VisibM:      13000,
			WXCode:      3,
			PressureMsl: 1012.1,
			Ceiling:     api.Ceiling{Kind: "BKN", FtAgl: 2500},
		},
	}
	got := RenderForecast("KEDU", "Davis/University Arpt, CA, US", pdt, steps)
	want := "KEDU Davis/University Arpt, CA, US\n" +
		"Forecast: 2026-09-16 06:00 PDT (13:00 UTC)\n" +
		"Wind 250° 11kt | Vis 7.5 SM | Sky SCT 03000 | 11.2°C/8°C | Altimeter 29.90 inHg\n" +
		"Forecast: 2026-09-16 07:00 PDT (14:00 UTC)\n" +
		"Wind 255° 09kt | Vis 8.1 SM | Sky BKN 02500 | 12°C/8.2°C | Altimeter 29.89 inHg"
	if got != want {
		t.Fatalf("RenderForecast:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderForecastNoSteps(t *testing.T) {
	got := RenderForecast("KABC", "Anywhere, US", pst, nil)
	want := "KABC Anywhere, US"
	if got != want {
		t.Fatalf("RenderForecast = %q, want %q", got, want)
	}
}
