package tideui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// TerminalBackgroundSequences returns OSC strings that set and later reset the
// terminal's default background for theme; the consumer decides where to write
// them. The background is kept solid: see TerminalBackgroundSequencesFor.
func TerminalBackgroundSequences(theme Theme) (set string, reset string) {
	return TerminalBackgroundSequencesFor(theme, false)
}

// TerminalBackgroundSequencesFor is TerminalBackgroundSequences with a choice of
// transparency. Terminals such as kitty apply their background opacity to every
// cell whose color equals the default background. Applications paint every cell
// in the theme background, so setting that exact color would make the whole UI
// see-through. With transparent off, the default is set one step away from the
// theme's (see TerminalDefaultBg); with it on, it is the theme's exact color and
// the application should pass its frames through TransparentBackground.
func TerminalBackgroundSequencesFor(theme Theme, transparent bool) (set string, reset string) {
	bg := TerminalDefaultBg(theme.Bg, transparent)
	if bg == "" {
		return "", ""
	}
	return fmt.Sprintf("\x1b]11;%s\x07", string(bg)), "\x1b]111\x07"
}

// TerminalDefaultBg is the color to set as the terminal's default background
// for a theme background. Transparent returns bg itself, so the terminal's
// opacity applies to it; otherwise one channel is nudged by one step, invisible
// to the eye but enough that no painted cell matches it and every cell stays
// opaque. Colors that are not #rrggbb are returned unchanged.
func TerminalDefaultBg(bg lipgloss.Color, transparent bool) lipgloss.Color {
	if bg == "" || transparent {
		return bg
	}
	hex := strings.TrimPrefix(string(bg), "#")
	if len(hex) != 6 {
		return bg
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return bg
	}
	if v&0xff == 0xff {
		v--
	} else {
		v++
	}
	return lipgloss.Color(fmt.Sprintf("#%06x", v))
}

// TransparentBackground rewrites every cell of a rendered frame painted in the
// theme background bg to the terminal's default background (SGR 49), so the
// terminal's own opacity shows through. Use it together with
// TerminalBackgroundSequencesFor(theme, true). Selected rows, overlays and other
// surfaces keep their colors and stay solid.
func TransparentBackground(view string, bg lipgloss.Color) string {
	if bg == "" {
		return view
	}
	params := themeBgParams(bg)
	if params == "" {
		return view
	}
	r := strings.NewReplacer(
		"\x1b["+params+"m", "\x1b[49m",
		"\x1b["+params+";", "\x1b[49;",
		";"+params+"m", ";49m",
		";"+params+";", ";49;",
	)
	return r.Replace(view)
}

var themeBgParamsCache sync.Map // lipgloss.Color + profile → SGR parameters

// themeBgParams returns the SGR parameters ("48;2;r;g;b" or "48;5;n") the
// current color profile uses for bg.
func themeBgParams(bg lipgloss.Color) string {
	key := fmt.Sprintf("%s/%d", bg, lipgloss.ColorProfile())
	if v, ok := themeBgParamsCache.Load(key); ok {
		return v.(string)
	}
	out := lipgloss.NewStyle().Background(bg).Render("x")
	params := ""
	if start := strings.Index(out, "\x1b["); start >= 0 {
		if end := strings.IndexByte(out[start:], 'm'); end > 2 {
			params = out[start+2 : start+end]
		}
	}
	if !strings.HasPrefix(params, "48;") {
		params = ""
	}
	themeBgParamsCache.Store(key, params)
	return params
}
