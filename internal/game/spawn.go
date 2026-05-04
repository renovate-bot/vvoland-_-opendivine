// SPDX-License-Identifier: GPL-3.0-only

package game

import (
	"errors"
	"os"

	"grono.dev/opendivine/pkg/assets/location"
)

// Spawn is the survivor's starting world position recorded in
// global/location.000.
// Used by callers that want to populate Config.PosX/PosY/Region with
// engine-faithful defaults before flag-parsing.
type Spawn struct {
	X, Y   float64
	Region int
}

// ReadSpawn returns the "stps_hero" record from
// <gamedataDir>/global/location.000.
func ReadSpawn(gamedataDir string) (Spawn, error) {
	fallback := Spawn{X: worldXPx / 2, Y: worldYPx / 2}
	f, err := os.Open(gamedataDir + "/global/location.000")
	if err != nil {
		return fallback, err
	}
	defer f.Close()
	loc, err := location.Decode(f)
	if err != nil {
		return fallback, err
	}
	for _, r := range loc.Records {
		if r.Name == "stps_hero" {
			return Spawn{X: float64(r.V0), Y: float64(r.V1), Region: int(r.V3)}, nil
		}
	}
	return fallback, errors.New("no stps_hero record in location.000")
}
