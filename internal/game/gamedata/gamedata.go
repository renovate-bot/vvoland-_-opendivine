// SPDX-License-Identifier: GPL-3.0-only

// Package gamedata resolves the path to a Divine Divinity install.
package gamedata

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// canaryFiles are the files that must exist for a directory to be a usable
// Divine Divinity install.
var canaryFiles = []string{
	"main/startup/data.000",
	"static/objects.000",
	"static/imagelists/CPackedi.0c",
	"static/imagelists/CPackedb.0c",
}

// Validate reports nil if path looks like a Divine Divinity install, or an
// error naming the first missing canary file.
func Validate(path string) error {
	// If the path doesn't exist at all, no need to check every canary file.
	if _, err := os.Stat(path); err != nil {
		return err
	}
	for _, f := range canaryFiles {
		full := filepath.Join(path, f)
		if _, err := os.Stat(full); err != nil {
			return fmt.Errorf("missing %s", f)
		}
	}
	return nil
}

// Candidates returns paths to try when auto-detecting an install.
func Candidates() []string {
	var cands []string
	if env := os.Getenv("OPENDIVINE_GAMEDATA"); env != "" {
		cands = append(cands, env)
	}
	cands = append(cands, "gamedata") // dev shortcut
	home, _ := os.UserHomeDir()

	const steamApp = "steamapps/common/divine_divinity"

	switch runtime.GOOS {
	case "linux":
		if home != "" {
			cands = append(cands,
				filepath.Join(home, ".steam/steam", steamApp),
				filepath.Join(home, ".local/share/Steam", steamApp),
				filepath.Join(home, "GOG Games/Divine Divinity"),
			)
		}
	case "darwin":
		if home != "" {
			cands = append(cands,
				filepath.Join(home, "Library/Application Support/Steam", steamApp),
				filepath.Join(home, "GOG Games/Divine Divinity"),
			)
		}
	case "windows":
		// Steam library defaults on Windows.
		cands = append(cands,
			filepath.Join(`C:\Program Files (x86)\Steam`, steamApp),
			filepath.Join(`C:\Program Files\Steam\`, steamApp),
			`C:\GOG Games\Divine Divinity`,
		)
	}
	return cands
}

// Resolve picks an install directory, validating canary files.
// If explicit is non-empty it's the only candidate (user opted in).
// Otherwise the auto-detect chain runs.
func Resolve(explicit string) (path string, err error) {
	if explicit != "" {
		if err := Validate(explicit); err != nil {
			return "", err
		}
		return explicit, nil
	}
	var tried []string
	for _, c := range Candidates() {
		err := Validate(c)
		if err == nil {
			return c, nil
		}
		tried = append(tried, c)
	}

	for _, c := range tried {
		log.Println("Tried:", c)
	}
	return "", errors.New("could not find a Divine Divinity install")
}
