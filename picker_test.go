package tideui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestThemePickerDefaultsAndFallbackSelection(t *testing.T) {
	picker := NewThemePicker(ThemePickerOptions{InitialTheme: "missing"})
	if got := picker.ConfirmedTheme().Name; got != BuiltinThemes[0].Name {
		t.Fatalf("confirmed theme = %q, want %q", got, BuiltinThemes[0].Name)
	}

	custom := []Theme{VT100, VT52}
	picker = NewThemePicker(ThemePickerOptions{Themes: custom, InitialTheme: VT52.Name})
	if got := picker.PreviewTheme().Name; got != VT52.Name {
		t.Fatalf("preview theme = %q, want %q", got, VT52.Name)
	}
}

func TestThemePickerNavigatesWithoutWrappingAndConfirms(t *testing.T) {
	picker := NewThemePicker(ThemePickerOptions{Themes: []Theme{CatppuccinMocha, Nord, Dracula}})
	picker.Open(CatppuccinMocha.Name)

	picker.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := picker.PreviewTheme().Name; got != CatppuccinMocha.Name {
		t.Fatalf("up at first theme selected %q", got)
	}
	picker.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	picker.Update(tea.KeyMsg{Type: tea.KeyDown})
	picker.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := picker.PreviewTheme().Name; got != Dracula.Name {
		t.Fatalf("down selection = %q, want %q", got, Dracula.Name)
	}
	if got := picker.Update(tea.KeyMsg{Type: tea.KeyEnter}); got != ThemePickerConfirm {
		t.Fatalf("enter action = %v, want ThemePickerConfirm", got)
	}
	if picker.Opened() || picker.ConfirmedTheme().Name != Dracula.Name {
		t.Fatal("confirm should close picker and commit the previewed theme")
	}
}

func TestThemePickerCancelRestoresConfirmedTheme(t *testing.T) {
	picker := NewThemePicker(ThemePickerOptions{Themes: []Theme{CatppuccinMocha, Nord}, InitialTheme: Nord.Name})
	picker.Open(Nord.Name)
	picker.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if got := picker.PreviewTheme().Name; got != CatppuccinMocha.Name {
		t.Fatalf("preview theme = %q, want %q", got, CatppuccinMocha.Name)
	}
	if got := picker.Update(tea.KeyMsg{Type: tea.KeyEscape}); got != ThemePickerCancel {
		t.Fatalf("escape action = %v, want ThemePickerCancel", got)
	}
	if picker.Opened() || picker.PreviewTheme().Name != Nord.Name {
		t.Fatal("cancel should close picker and restore its confirmed theme")
	}
}

func TestThemePickerModalUsesThemeCursorAndKeepsSelectionVisible(t *testing.T) {
	themes := []Theme{CatppuccinMocha, Nord, Dracula, VT100, VT52}
	picker := NewThemePicker(ThemePickerOptions{Themes: themes, InitialTheme: CatppuccinMocha.Name})
	picker.Open(CatppuccinMocha.Name)
	for range 4 {
		picker.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	renderer := NewRenderer(VT52, StyleOptions{Density: Compact})
	modal := picker.Modal(renderer, 30, 6)
	plain := ansi.Strip(renderer.renderOverlay(modal, 80))
	if !strings.Contains(plain, "> "+VT52.Name) {
		t.Fatalf("ASCII modal does not include selected cursor and theme:\n%s", plain)
	}
	if strings.Contains(plain, CatppuccinMocha.Name) {
		t.Fatalf("small modal should scroll past the first theme:\n%s", plain)
	}

	renderer = NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	box := renderer.renderOverlay(picker.Modal(renderer, 34, 24), 80)
	if !strings.Contains(ansi.Strip(box), "▶ "+VT52.Name) {
		t.Fatalf("Unicode modal does not include selected cursor and theme:\n%s", ansi.Strip(box))
	}
	for i, line := range strings.Split(box, "\n") {
		if got := lipgloss.Width(line); got != 34 {
			t.Fatalf("overlay line %d width = %d, want 34", i, got)
		}
	}
}
