package tideui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// legacyThemeNames are every name a released version accepted. Saved settings
// refer to themes by name, so none may disappear.
var legacyThemeNames = []string{
	"catppuccin-mocha", "catppuccin-latte", "catppuccin-frappe", "catppuccin-macchiato",
	"nord", "dracula", "gruvbox-dark", "gruvbox-light", "tokyo-night", "tokyo-night-day",
	"rose-pine", "rose-pine-moon", "rose-pine-dawn", "one-dark", "magenta-geode",
	"coral-sunset", "lavender-fields-forever", "vt100", "vt52",
}

func TestBuiltinThemeRegistry(t *testing.T) {
	if BuiltinThemes[0].Name != "catppuccin-mocha" {
		t.Fatalf("default theme = %q, want catppuccin-mocha", BuiltinThemes[0].Name)
	}
	seen := map[string]bool{}
	for _, th := range BuiltinThemes {
		if seen[th.Name] {
			t.Errorf("duplicate theme name %q", th.Name)
		}
		seen[th.Name] = true
		if th.Family == "" {
			t.Errorf("%s: no family", th.Name)
		}
	}
	for _, name := range legacyThemeNames {
		if got, ok := ThemeByName(name); !ok || got.Name != name {
			t.Errorf("legacy theme %q no longer resolves", name)
		}
	}
	if got, ok := ThemeByName("no-such-theme"); ok || got.Name != "catppuccin-mocha" {
		t.Errorf("unknown name = %q, %v; want the default and false", got.Name, ok)
	}
}

func TestBuiltinThemeFamiliesAreContiguous(t *testing.T) {
	closed := map[string]bool{}
	prev := ""
	for _, th := range BuiltinThemes {
		if th.Family != prev {
			if closed[th.Family] {
				t.Errorf("family %q is split: %s appears after another family", th.Family, th.Name)
			}
			if prev != "" {
				closed[prev] = true
			}
			prev = th.Family
		}
	}
}

func TestBuiltinThemeFamiliesAndVariants(t *testing.T) {
	want := map[string][]string{
		"Catppuccin":  {"catppuccin-mocha", "catppuccin-macchiato", "catppuccin-frappe", "catppuccin-latte"},
		"Tokyo Night": {"tokyo-night", "tokyo-night-storm", "tokyo-night-moon", "tokyo-night-day"},
		"Rosé Pine":   {"rose-pine", "rose-pine-moon", "rose-pine-dawn"},
		"GitHub":      {"github-dark", "github-dark-dimmed", "github-light"},
		"Gruvbox":     {"gruvbox-dark", "gruvbox-light"},
		"Solarized":   {"solarized-dark", "solarized-light"},
		"Ayu":         {"ayu-dark", "ayu-mirage", "ayu-light"},
		"Kanagawa":    {"kanagawa-wave", "kanagawa-dragon", "kanagawa-lotus"},
		"Nightfox":    {"nightfox", "dayfox", "dawnfox", "duskfox", "nordfox", "terafox", "carbonfox"},
	}
	for family, names := range want {
		for _, name := range names {
			th, ok := ThemeByName(name)
			if !ok || th.Family != family {
				t.Errorf("%s: family = %q, want %q (found %v)", name, th.Family, family, ok)
			}
		}
	}
}

func TestBuiltinThemeColorsAreCompleteHex(t *testing.T) {
	for _, th := range BuiltinThemes {
		fields := map[string]lipgloss.Color{
			"Bg": th.Bg, "Fg": th.Fg, "Border": th.Border, "BorderFocus": th.BorderFocus,
			"Selected": th.Selected, "Unread": th.Unread, "Dimmed": th.Dimmed,
			"StatusBar": th.StatusBar, "StatusFg": th.StatusFg, "Error": th.Error,
			"Overlay": th.Overlay, "OverlayBorder": th.OverlayBorder,
			"Success": th.Success, "Warning": th.Warning, "Link": th.Link,
			"Quote": th.Quote, "Match": th.Match, "Tag": th.Tag, "Cursor": th.Cursor,
		}
		for field, c := range fields {
			if !hexColor.MatchString(string(c)) {
				t.Errorf("%s.%s = %q, want #rrggbb", th.Name, field, c)
			}
		}
	}
}

// Upstream palettes are copied exactly, so a text role that is under AA on its
// own background is lightness-adjusted by the accessor, not edited in the
// palette. Every accessor must therefore clear its threshold on every theme.
func TestBuiltinThemeTextRolesAreLegible(t *testing.T) {
	for _, th := range BuiltinThemes {
		roles := []struct {
			name string
			c    lipgloss.Color
			min  float64
		}{
			{"SuccessColor", th.SuccessColor(), 4.5},
			{"WarningColor", th.WarningColor(), 4.5},
			{"ErrorColor", th.ErrorColor(), 4.5},
			{"LinkColor", th.LinkColor(), 4.5},
			{"QuoteColor", th.QuoteColor(), 4.5},
			{"TagColor", th.TagColor(), 4.5},
			{"UnreadColor", th.UnreadColor(), 4.5},
			{"MatchColor", th.MatchColor(), 3},
			{"CursorColor", th.CursorColor(), 3},
		}
		for _, r := range roles {
			if got := contrastRatio(r.c, th.Bg); got < r.min {
				t.Errorf("%s.%s = %s on %s: %.2f:1, want >= %.1f", th.Name, r.name, r.c, th.Bg, got, r.min)
			}
		}
	}
}

// Body text must clear AA outright; only palettes whose own upstream design
// falls short are listed, with the measured reason.
var bodyTextExceptions = map[string]bool{}

func TestBuiltinThemeBodyTextContrast(t *testing.T) {
	for _, th := range BuiltinThemes {
		if got := contrastRatio(th.Fg, th.Bg); got < 4.5 && !bodyTextExceptions[th.Name] {
			t.Errorf("%s: Fg on Bg = %.2f:1, want >= 4.5", th.Name, got)
		}
		if got := contrastRatio(th.StatusFg, th.StatusBar); got < 4.5 && !bodyTextExceptions[th.Name] {
			t.Errorf("%s: StatusFg on StatusBar = %.2f:1, want >= 4.5", th.Name, got)
		}
	}
}

func TestLegibleKeepsGoodColorsAndMovesBadOnes(t *testing.T) {
	bg := lipgloss.Color("#ffffff")
	if got := Legible("#000000", bg, 4.5); got != "#000000" {
		t.Errorf("already legible color changed to %s", got)
	}
	adjusted := Legible("#ffcc00", bg, 4.5)
	if adjusted == "#ffcc00" || contrastRatio(adjusted, bg) < 4.5 {
		t.Errorf("Legible(#ffcc00) = %s (%.2f:1), want a darker color reaching 4.5", adjusted, contrastRatio(adjusted, bg))
	}
	dark := lipgloss.Color("#101010")
	lighter := Legible("#202020", dark, 4.5)
	if strings.EqualFold(string(lighter), "#202020") || contrastRatio(lighter, dark) < 4.5 {
		t.Errorf("Legible on dark = %s, want a lighter color reaching 4.5", lighter)
	}
}

func TestThemeLabelAndPolarity(t *testing.T) {
	mocha, _ := ThemeByName("catppuccin-mocha")
	if mocha.Label() != "Catppuccin Mocha" || mocha.IsLight() {
		t.Errorf("mocha label/light = %q/%v", mocha.Label(), mocha.IsLight())
	}
	latte, _ := ThemeByName("catppuccin-latte")
	if !latte.IsLight() {
		t.Error("latte should be light")
	}
	if got := (Theme{Name: "custom"}).Label(); got != "custom" {
		t.Errorf("label of a plain theme = %q", got)
	}
	if got := (Theme{Name: "x", Bg: "#000000", Fg: "#ffffff", Unread: "#00ff00"}).SuccessColor(); got != "#00ff00" {
		t.Errorf("role fallback = %s, want Unread", got)
	}
}
