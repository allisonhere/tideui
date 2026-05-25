package tideui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestThemeLookupAndOverrides(t *testing.T) {
	base, ok := ThemeByName(ThemeNameVT100)
	if !ok || base.Name != ThemeNameVT100 {
		t.Fatalf("expected built-in vt100 theme, got %q", base.Name)
	}
	overridden := (ThemeOverrides{
		Background: "#101010",
		Foreground: "#eeeeee",
		Accent:     "#ff00ff",
	}).Apply(base)
	if overridden.Bg != lipgloss.Color("#101010") || overridden.StatusFg != lipgloss.Color("#eeeeee") {
		t.Fatalf("foreground/background overrides were not applied: %#v", overridden)
	}
	if overridden.BorderFocus != lipgloss.Color("#ff00ff") || overridden.OverlayBorder != lipgloss.Color("#ff00ff") {
		t.Fatalf("accent override was not applied: %#v", overridden)
	}
}

func TestVT52BuildsPlainPresentation(t *testing.T) {
	styles := BuildStyles(VT52, StyleOptions{Density: Comfortable})
	if !styles.PlainUI {
		t.Fatal("expected vt52 to use ASCII presentation")
	}
	if styles.ListItemLineStride() != 2 {
		t.Fatalf("comfortable stride = %d, want 2", styles.ListItemLineStride())
	}
	if styles.StatusBarSeparator() != " | " {
		t.Fatalf("plain separator = %q", styles.StatusBarSeparator())
	}
}

func TestBuildStylesUsesConfiguredOverlaySurface(t *testing.T) {
	styles := BuildStyles(CatppuccinMocha, StyleOptions{Density: Compact})
	if got := styles.OverlayBody.GetBackground(); got != CatppuccinMocha.Overlay {
		t.Fatalf("overlay body background = %v, want %v", got, CatppuccinMocha.Overlay)
	}
	if got := styles.OverlayHint.GetBackground(); got != CatppuccinMocha.Overlay {
		t.Fatalf("overlay hint background = %v, want %v", got, CatppuccinMocha.Overlay)
	}
}

func TestStylesMaintainReadableContrast(t *testing.T) {
	for _, theme := range BuiltinThemes {
		styles := BuildStyles(theme, StyleOptions{Density: Compact})
		checks := []struct {
			name string
			fg   lipgloss.TerminalColor
			bg   lipgloss.TerminalColor
		}{
			{"active header", styles.PaneHeaderActive.GetForeground(), styles.PaneHeaderActive.GetBackground()},
			{"inactive header", styles.PaneHeaderInactive.GetForeground(), styles.PaneHeaderInactive.GetBackground()},
			{"status", styles.StatusBar.GetForeground(), styles.StatusBar.GetBackground()},
			{"overlay title", styles.OverlayTitle.GetForeground(), styles.OverlayTitle.GetBackground()},
			{"overlay body", styles.OverlayBody.GetForeground(), styles.OverlayBody.GetBackground()},
		}
		for _, check := range checks {
			fg := lipgloss.Color(check.fg.(lipgloss.Color))
			bg := lipgloss.Color(check.bg.(lipgloss.Color))
			if ratio := contrastRatio(fg, bg); ratio < 4.5 {
				t.Errorf("%s %s contrast %.2f is below 4.5", theme.Name, check.name, ratio)
			}
		}
	}
}
