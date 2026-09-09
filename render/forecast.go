package render

import (
	"fmt"
	"strings"
	"time"

	"gofly/api"
)

const metersPerSM = 1609.34

// wmoPhrases maps WMO weather codes to short phrases. Sky (0-3) and fog
// (45/48) codes are omitted because the Sky line already conveys them.
var wmoPhrases = map[int]string{
	51: "Drizzle",
	53: "Drizzle",
	55: "Dense drizzle",
	56: "Freezing drizzle",
	57: "Freezing drizzle",
	61: "Rain",
	63: "Rain",
	65: "Heavy rain",
	66: "Freezing rain",
	67: "Freezing rain",
	71: "Snow",
	73: "Snow",
	75: "Heavy snow",
	77: "Snow grains",
	80: "Rain showers",
	81: "Rain showers",
	82: "Heavy rain showers",
	85: "Snow showers",
	86: "Heavy snow showers",
	95: "Thunderstorms",
	96: "Thunderstorms with hail",
	99: "Thunderstorms with heavy hail",
}

// RenderForecast renders one block per forecast step under a shared header.
func RenderForecast(icao, name string, local *time.Location, steps []api.ForecastStep) string {
	var b strings.Builder
	header := icao
	if name != "" {
		header += " " + name
	}
	b.WriteString(header)
	for _, s := range steps {
		b.WriteByte('\n')
		fmt.Fprintf(&b, "Forecast: %s (%s UTC)\n", s.Time.In(local).Format("2006-01-02 15:04 MST"), s.Time.UTC().Format("15:04"))
		parts := []string{fmt.Sprintf("Wind %03d° %02dkt", int(s.WindDir), int(s.WindSpd))}
		if s.Gust > s.WindSpd {
			parts[0] += fmt.Sprintf(" G%d", int(s.Gust))
		}
		parts = append(parts, "Vis "+visibilitySM(s.VisibM))
		parts = append(parts, skyPhrase(s.Ceiling))
		if phrase, ok := wmoPhrases[s.WXCode]; ok {
			parts = append(parts, phrase)
		}
		if s.Temp != 0 || s.Dewp != 0 {
			parts = append(parts, fmt.Sprintf("%g°C/%g°C", s.Temp, s.Dewp))
		}
		if s.PressureMsl != 0 {
			parts = append(parts, fmt.Sprintf("Altimeter %.2f inHg", s.PressureMsl/10/hpaPerInHg))
		}
		if s.PrecipMm > 0 {
			parts = append(parts, fmt.Sprintf("%.1fmm (%d%%)", s.PrecipMm, s.PrecipProb))
		}
		b.WriteString(strings.Join(parts, " | "))
	}
	return b.String()
}

func visibilitySM(m float64) string {
	mi := m / metersPerSM
	if mi >= 9.95 {
		return "15+ SM"
	}
	return fmt.Sprintf("%.1f SM", mi)
}

func skyPhrase(c api.Ceiling) string {
	if c.Kind == "BKN" || c.Kind == "SCT" {
		return fmt.Sprintf("Sky %s %05d", c.Kind, c.FtAgl)
	}
	return "Sky " + c.Kind
}
