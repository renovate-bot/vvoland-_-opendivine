// SPDX-License-Identifier: GPL-3.0-only

// Package assets embeds bundled game resources into the binary.
package assets

import _ "embed"

// UncialAntiquaTTF is the main-menu display font (SIL OFL 1.1, see
// fonts/OFL.txt), embedded so the binary is self-contained.
//
//go:embed fonts/UncialAntiqua-Regular.ttf
var UncialAntiquaTTF []byte
