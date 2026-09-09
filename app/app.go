package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"gofly/api"
	"gofly/pick"
	"gofly/render"
	"gofly/timeparse"
)

var ErrUsage = errors.New("usage: gofly <ICAO> [TIME] [--forecast]")

var icaoPattern = regexp.MustCompile(`^[A-Z]{3,4}$`)

type MetarGetter interface {
	GetMetars(ctx context.Context, icao string, history bool) ([]api.Metar, error)
}

type Forecaster interface {
	GetForecast(ctx context.Context, lat, lon, elev float64, start, end time.Time) ([]api.ForecastStep, error)
}

type App struct {
	Client   MetarGetter
	Forecast Forecaster
	Out      io.Writer
	Now      func() time.Time
}

func parseArgs(args []string) (icao, timeStr string, forecast bool, err error) {
	var positionals []string
	for _, a := range args {
		switch a {
		case "--forecast", "-f":
			forecast = true
		default:
			if strings.HasPrefix(a, "-") {
				return "", "", false, fmt.Errorf("unknown flag %q: %w", a, ErrUsage)
			}
			positionals = append(positionals, a)
		}
	}
	if len(positionals) == 0 {
		return "", "", false, fmt.Errorf("no argument(s) given: %w", ErrUsage)
	}
	if len(positionals) > 2 {
		return "", "", false, fmt.Errorf("%d argument(s) given, want 1 or 2: %w", len(positionals), ErrUsage)
	}
	icao = positionals[0]
	if len(positionals) == 2 {
		timeStr = positionals[1]
	}
	return icao, timeStr, forecast, nil
}

func (a *App) Run(ctx context.Context, args []string) error {
	icaoIn, timeStr, doForecast, err := parseArgs(args)
	if err != nil {
		return err
	}
	icao := strings.ToUpper(strings.TrimSpace(icaoIn))
	if !icaoPattern.MatchString(icao) {
		return fmt.Errorf("invalid ICAO ID %q (expected 3-4 letters, e.g. KEDU)", icaoIn)
	}

	now := time.Now
	if a.Now != nil {
		now = a.Now
	}

	local, err := timeparse.DefaultLocation()
	if err != nil {
		return err
	}

	hasTime := timeStr != ""
	history := hasTime && !doForecast

	var target time.Time
	if hasTime {
		target, err = timeparse.ParseTime(timeStr, now())
		if err != nil {
			return err
		}
	} else {
		target = now()
	}

	metars, err := a.Client.GetMetars(ctx, icao, history)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("no METAR data found for %s", icao)
		}
		return err
	}

	var m api.Metar
	if history {
		m, err = pick.Closest(metars, target)
		if err != nil {
			return fmt.Errorf("no METAR data found for %s (no observations in range)", icao)
		}
	} else {
		if len(metars) == 0 {
			return fmt.Errorf("no METAR data found for %s", icao)
		}
		m = metars[0]
	}

	if doForecast {
		if a.Forecast == nil {
			return errors.New("forecast not available")
		}
		window := target.UTC().Truncate(time.Hour)
		steps, err := a.Forecast.GetForecast(ctx, m.Lat, m.Lon, m.Elev,
			window.Add(-time.Hour), window.Add(time.Hour))
		if err != nil {
			return err
		}
		dispLoc := local
		if hasTime {
			dispLoc = target.Location()
		}
		fmt.Fprintln(a.Out, render.RenderForecast(m.IcaoID, m.Name, dispLoc, steps))
		return nil
	}

	fmt.Fprintln(a.Out, render.Render(m, local))
	return nil
}
