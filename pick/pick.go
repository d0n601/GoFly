package pick

import (
	"errors"
	"time"

	"gofly/api"
)

var ErrEmpty = errors.New("no observations available")

func Closest(obs []api.Metar, target time.Time) (api.Metar, error) {
	if len(obs) == 0 {
		return api.Metar{}, ErrEmpty
	}
	best := obs[0]
	for _, o := range obs[1:] {
		if closer(o, best, target) {
			best = o
		}
	}
	return best, nil
}

func closer(a, b api.Metar, target time.Time) bool {
	da := abs(a.Time().Sub(target))
	db := abs(b.Time().Sub(target))
	if da != db {
		return da < db
	}
	return a.ObsTime < b.ObsTime
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
