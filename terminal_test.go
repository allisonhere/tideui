package tideui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestTerminalDefaultBgIsNudgedUnlessTransparent(t *testing.T) {
	cases := map[lipgloss.Color]lipgloss.Color{
		"#000000": "#000001",
		"#0a0f1a": "#0a0f1b",
		"#ffffff": "#fffffe",
		"#1234ff": "#1234fe",
	}
	for in, want := range cases {
		if got := TerminalDefaultBg(in, false); got != want {
			t.Errorf("TerminalDefaultBg(%s) = %s, want %s", in, got, want)
		}
		if got := TerminalDefaultBg(in, true); got != in {
			t.Errorf("transparent TerminalDefaultBg(%s) = %s, want it unchanged", in, got)
		}
	}
	for _, in := range []lipgloss.Color{"", "#fff", "12"} {
		if got := TerminalDefaultBg(in, false); got != in {
			t.Errorf("TerminalDefaultBg(%q) = %q, want it unchanged", in, got)
		}
	}
}

// kitty makes every cell whose color equals the default background
// see-through, so the solid default must never set the exact theme color.
func TestTerminalBackgroundSequencesKeepThemesOpaque(t *testing.T) {
	for _, theme := range BuiltinThemes {
		exact := "\x1b]11;" + string(theme.Bg) + "\x07"
		solid, reset := TerminalBackgroundSequences(theme)
		if solid == "" || reset != "\x1b]111\x07" {
			t.Fatalf("%s: unexpected OSC strings set=%q reset=%q", theme.Name, solid, reset)
		}
		if strings.Contains(solid, exact) {
			t.Errorf("%s: solid mode sets the default background to the exact theme color", theme.Name)
		}
		if clear, _ := TerminalBackgroundSequencesFor(theme, true); clear != exact {
			t.Errorf("%s: transparent mode must set the exact theme color, got %q", theme.Name, clear)
		}
	}
	if set, reset := TerminalBackgroundSequences(Theme{}); set != "" || reset != "" {
		t.Fatalf("a theme without a background must emit nothing: %q %q", set, reset)
	}
}

func TestTransparentBackgroundDropsOnlyTheThemeBackground(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	bg, other := lipgloss.Color("#000000"), lipgloss.Color("#123456")
	view := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#ffffff")).Render("a") +
		lipgloss.NewStyle().Background(bg).Render("b") +
		lipgloss.NewStyle().Background(other).Render("c")
	got := TransparentBackground(view, bg)
	if strings.Contains(got, themeBgParams(bg)) {
		t.Fatalf("theme background left in %q", got)
	}
	if !strings.Contains(got, themeBgParams(other)) {
		t.Fatalf("other surfaces must keep their background: %q", got)
	}
	if !strings.Contains(got, "49") {
		t.Fatalf("expected default-background resets in %q", got)
	}
	if TransparentBackground(view, "") != view {
		t.Fatal("an empty background must leave the frame untouched")
	}
}
