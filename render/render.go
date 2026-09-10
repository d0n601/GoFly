package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gofly/api"
)

const hpaPerInHg = 3.386389

func windDir(dir api.WindDir) string {
	s := string(dir)
	if d, err := strconv.ParseFloat(s, 64); err == nil {
		return fmt.Sprintf("%03.0f°", d)
	}
	if s != "" {
		return s
	}
	return "000°"
}

func Render(m api.Metar, local *time.Location) string {
	var b strings.Builder

	header := m.IcaoID
	if m.Name != "" {
		header += " " + m.Name
	}
	b.WriteString(header)
	b.WriteByte('\n')

	obs := time.Unix(m.ObsTime, 0)
	fmt.Fprintf(&b, "Observed: %s (%s UTC)\n", obs.In(local).Format("2006-01-02 15:04 MST"), obs.UTC().Format("15:04"))

	parts := []string{fmt.Sprintf("Wind %s %dkt", windDir(m.Wdir), int(m.Wspd))}
	if m.Visib != "" {
		parts = append(parts, "Vis "+string(m.Visib))
	}
	if m.Temp != 0 || m.Dewp != 0 {
		parts = append(parts, fmt.Sprintf("%g°C/%g°C", m.Temp, m.Dewp))
	}
	if m.Altim != 0 {
		parts = append(parts, fmt.Sprintf("Altimeter %.2f inHg", m.Altim/10/hpaPerInHg))
	}
	b.WriteString(strings.Join(parts, " | "))
	if m.RawOb != "" {
		b.WriteByte('\n')
		b.WriteString(m.RawOb)
	}
	return b.String()
}
