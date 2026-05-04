// SPDX-License-Identifier: GPL-3.0-only

package testutils

import (
	"os"
	"testing"
)

func TestGameData(t testing.TB) string {
	t.Helper()
	p := os.Getenv("TEST_GAMEDATA_PATH")
	if p == "" {
		if _, err := os.Stat("./gamedata"); err == nil {
			p = "./gamedata"
		}
	}
	if p == "" {
		t.Skip("SKIP: Test requires game data")
	}
	return p
}
