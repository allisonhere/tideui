package tideui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestRenderSoftPanelEmbedsTitleInBorder(t *testing.T) {
	renderer := NewRenderer(Nord, StyleOptions{Density: Compact, PaneCorners: RoundCorners})
	box := renderer.RenderSoftPanel(SoftPanel{Prefix: "tidedock", Title: "theme", Content: "hello", Width: 40})
	plain := ansi.Strip(box)
	if !strings.Contains(plain, "tidedock · theme") {
		t.Fatalf("soft panel did not embed title in border:\n%s", plain)
	}
	for i, line := range strings.Split(box, "\n") {
		if got := lipgloss.Width(line); got != 42 {
			t.Fatalf("line %d width = %d, want 42 (%q)", i, got, ansi.Strip(line))
		}
	}
}

func TestRenderSoftPanelPlainUsesASCIIBorder(t *testing.T) {
	renderer := NewRenderer(VT52, StyleOptions{Density: Compact})
	box := ansi.Strip(renderer.RenderSoftPanel(SoftPanel{Prefix: "td", Title: "help", Content: "hello", Width: 24}))
	lines := strings.Split(box, "\n")
	if strings.ContainsAny(box, "╭╮╰╯") {
		t.Fatalf("plain soft panel used rounded unicode border:\n%s", box)
	}
	if !strings.Contains(box, "td - help") {
		t.Fatalf("plain soft panel missing title:\n%s", box)
	}
	if !strings.Contains(lines[0], "td - help") {
		t.Fatalf("plain soft panel title = %q, want embedded in top border", lines[0])
	}
	if len(lines) > 1 && strings.Contains(lines[1], "td - help") {
		t.Fatalf("plain soft panel title dropped to body line: %q", lines[1])
	}
}

func TestRenderSoftHintsLowercasesAndFitsWidth(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	line := renderer.RenderSoftHints(30, SoftHint{Key: "ENTER", Label: "Confirm"}, SoftHint{Key: "ESC", Label: "Cancel"})
	if got := lipgloss.Width(line); got != 30 {
		t.Fatalf("hint width = %d, want 30", got)
	}
	if plain := ansi.Strip(line); !strings.Contains(plain, "enter confirm") || strings.Contains(plain, "ENTER") {
		t.Fatalf("hint text = %q", plain)
	}
}

func TestRenderSoftRowUsesRailAndFitsWidth(t *testing.T) {
	renderer := NewRenderer(Dracula, StyleOptions{Density: Compact})
	selected := renderer.RenderSoftRow(SoftRow{Text: "Choose theme", Suffix: "T", Selected: true}, 26)
	unselected := renderer.RenderSoftRow(SoftRow{Text: "Help", Suffix: "?", Selected: false}, 26)
	if got := lipgloss.Width(selected); got != 26 {
		t.Fatalf("selected width = %d, want 26", got)
	}
	if got := lipgloss.Width(unselected); got != 26 {
		t.Fatalf("unselected width = %d, want 26", got)
	}
	if !strings.Contains(ansi.Strip(selected), "▌") {
		t.Fatalf("selected row missing rail: %q", ansi.Strip(selected))
	}
}

func TestRenderSoftRowHighlightsSelectedRowBackground(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	renderer := NewRenderer(Dracula, StyleOptions{Density: Compact})
	chrome := newSoftChrome(renderer.Styles)
	if chrome.selectedBg == chrome.baseBg {
		t.Fatalf("selectedBg should differ from baseBg so the focused row is visibly highlighted")
	}

	selected := renderer.RenderSoftRow(SoftRow{Text: "Choose theme", Selected: true}, 26)
	unselected := renderer.RenderSoftRow(SoftRow{Text: "Help", Selected: false}, 26)

	selectedSGR := backgroundSGR(t, chrome.selectedBg)
	if !strings.Contains(selected, selectedSGR) {
		t.Fatalf("selected row does not carry the selected background SGR %q:\n%q", selectedSGR, selected)
	}
	if strings.Contains(unselected, selectedSGR) {
		t.Fatalf("unselected row unexpectedly carries the selected background SGR %q:\n%q", selectedSGR, unselected)
	}
}

func backgroundSGR(t *testing.T, c lipgloss.Color) string {
	t.Helper()
	r, g, b, ok := hexToRGB(c)
	if !ok {
		t.Fatalf("hexToRGB(%q) failed", c)
	}
	return fmt.Sprintf("48;2;%d;%d;%d", int(r*255+0.5), int(g*255+0.5), int(b*255+0.5))
}

func TestRenderSoftBodyDoesNotWrapStyledLongRows(t *testing.T) {
	renderer := NewRenderer(Dracula, StyleOptions{Density: Compact})
	row := renderer.RenderSoftRow(SoftRow{Text: "a-command-name-that-is-too-long-for-this-modal", Selected: true}, 16)
	body := renderer.RenderSoftBody(20, row)
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if got := lipgloss.Width(line); got != 20 {
			t.Fatalf("line %d width = %d, want 20 (%q)", i, got, ansi.Strip(line))
		}
	}
	if len(lines) != 3 {
		t.Fatalf("body lines = %d, want 3:\n%s", len(lines), ansi.Strip(body))
	}
}

func TestRenderSoftBodyThemesRawText(t *testing.T) {
	original := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(original) })

	renderer := NewRenderer(RosePineDawn, StyleOptions{Density: Compact})
	body := renderer.RenderSoftBody(24, "plain help text")
	lines := strings.Split(body, "\n")
	if len(lines) < 2 {
		t.Fatalf("body lines = %d, want raw text line", len(lines))
	}
	if !strings.Contains(lines[1], "mplain help text") {
		t.Fatalf("raw text line does not carry inline theme styling: %q", lines[1])
	}
	if got := lipgloss.Width(lines[1]); got != 24 {
		t.Fatalf("line width = %d, want 24 (%q)", got, ansi.Strip(lines[1]))
	}
}

func TestThemePickerSoftModalLongNamesDoNotWrap(t *testing.T) {
	renderer := NewRenderer(Nord, StyleOptions{Density: Compact})
	assertLongThemeNameDoesNotWrap(t, renderer, 26, 8)
}

func TestThemePickerSoftModalLongNamesDoNotWrapInVT52(t *testing.T) {
	renderer := NewRenderer(VT52, StyleOptions{Density: Compact})
	assertLongThemeNameDoesNotWrap(t, renderer, 26, 9)
}

func assertLongThemeNameDoesNotWrap(t *testing.T, renderer Renderer, wantWidth int, maxLines int) {
	t.Helper()
	picker := NewThemePicker(ThemePickerOptions{
		InitialTheme: "nord",
		Themes: []Theme{
			{Name: "short", Bg: "#000000", Fg: "#ffffff"},
			{Name: "a-theme-name-that-is-way-too-long-for-the-box", Bg: "#000000", Fg: "#ffffff"},
		},
	})
	picker.Open("nord")
	_ = picker.Update(keyMsg("j"))

	overlay := picker.SoftModal(renderer, 24, 10, "td")
	lines := strings.Split(overlay.Content, "\n")
	for i, line := range lines {
		if got := lipgloss.Width(line); got != wantWidth {
			t.Fatalf("line %d width = %d, want %d (%q)", i, got, wantWidth, ansi.Strip(line))
		}
	}
	if got := len(lines); got > maxLines {
		t.Fatalf("soft modal wrapped into %d lines, want at most %d:\n%s", got, maxLines, ansi.Strip(overlay.Content))
	}
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}
