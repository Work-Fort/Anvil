// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"charm.land/lipgloss/v2"
)

const omarchyColorsPath = ".config/omarchy/current/theme/colors.toml"

// LoadOmarchyTheme reads the Omarchy system theme and applies it to CurrentTheme.
// Returns an error if the file cannot be found or parsed; callers should ignore
// the error to fall back to the default theme on non-Omarchy systems.
func LoadOmarchyTheme() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	path := filepath.Join(home, omarchyColorsPath)

	// Resolve symlinks — Omarchy's "current" directory may be a symlink.
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("read colors.toml: %w", err)
	}

	var colors map[string]string
	if err := toml.Unmarshal(data, &colors); err != nil {
		return fmt.Errorf("parse colors.toml: %w", err)
	}

	// Map Omarchy ANSI palette to Anvil Theme.
	// Positions follow the standard ANSI 16-color convention:
	//   color0=black, color1=red, color2=green, color3=yellow,
	//   color4=blue, color5=magenta, color6=cyan, color7=white,
	//   color8-15=bright variants.
	theme := CurrentTheme // start from defaults

	if v, ok := colors["accent"]; ok {
		theme.Primary = lipgloss.Color(v)
		theme.Accent = lipgloss.Color(v)
		theme.Success = lipgloss.Color(v) // success tracks primary
	}
	if v, ok := colors["foreground"]; ok {
		theme.Text = lipgloss.Color(v)
	}
	if v, ok := colors["background"]; ok {
		theme.BgDark = lipgloss.Color(v)
	}
	if v, ok := colors["color8"]; ok {
		theme.TextDim = lipgloss.Color(v)
	}
	if v, ok := colors["color4"]; ok {
		theme.Secondary = lipgloss.Color(v)
		theme.Info = lipgloss.Color(v) // info tracks secondary
	}
	if v, ok := colors["color1"]; ok {
		theme.Error = lipgloss.Color(v)
	}
	if v, ok := colors["color2"]; ok {
		theme.Success = lipgloss.Color(v)
	}
	if v, ok := colors["color3"]; ok {
		theme.Warning = lipgloss.Color(v)
	}

	// Muted is derived: 60% background + 40% color8 (bright black).
	bg := colors["background"]
	dim := colors["color8"]
	if bg != "" && dim != "" {
		if blended, err := blendHex(bg, dim, 0.4); err == nil {
			theme.Muted = lipgloss.Color(blended)
		}
	}

	CurrentTheme = theme
	return nil
}

// blendHex linearly interpolates between two "#RRGGBB" hex colors.
// t=0 returns a, t=1 returns b.
func blendHex(a, b string, t float64) (string, error) {
	ar, ag, ab, err := parseHex(a)
	if err != nil {
		return "", err
	}
	br, bg, bb, err := parseHex(b)
	if err != nil {
		return "", err
	}

	lerp := func(x, y uint8, t float64) uint8 {
		return uint8(float64(x)*(1-t) + float64(y)*t)
	}

	r := lerp(ar, br, t)
	g := lerp(ag, bg, t)
	bl := lerp(ab, bb, t)

	return fmt.Sprintf("#%02X%02X%02X", r, g, bl), nil
}

// parseHex extracts RGB components from a "#RRGGBB" string.
func parseHex(hex string) (r, g, b uint8, err error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color: %q", hex)
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	return uint8(rv), uint8(gv), uint8(bv), nil
}
