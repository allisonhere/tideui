package tideui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func entryNames(entries []pickerEntry, themes []Theme) []string {
	var out []string
	for _, entry := range entries {
		if entry.header == "" {
			out = append(out, themes[entry.idx].Name)
		}
	}
	return out
}

func TestPickerEntriesGroupVariantsUnderTheirFamily(t *testing.T) {
	entries := pickerEntries(BuiltinThemes, "")
	if entries[0].header != "Catppuccin" {
		t.Fatalf("first entry = %+v, want the Catppuccin header", entries[0])
	}
	solo := map[string]bool{}
	for _, entry := range entries {
		if entry.solo {
			solo[BuiltinThemes[entry.idx].Name] = true
		}
	}
	for _, want := range []string{"nord", "palenight", "vt52"} {
		if !solo[want] {
			t.Errorf("%s should be a header-less single-theme row (solo = %v)", want, solo)
		}
	}
	if got := len(entryNames(entries, BuiltinThemes)); got != len(BuiltinThemes) {
		t.Fatalf("unfiltered picker lists %d themes, want all %d", got, len(BuiltinThemes))
	}
}

func TestPickerEntriesLeaveFamilylessThemesUngrouped(t *testing.T) {
	themes := []Theme{{Name: "one"}, {Name: "two"}}
	for _, entry := range pickerEntries(themes, "") {
		if entry.header != "" || !entry.solo {
			t.Fatalf("a theme with no family should stand alone, got %+v", entry)
		}
	}
}

func TestPickerEntriesFilter(t *testing.T) {
	names := func(filter string) []string { return entryNames(pickerEntries(BuiltinThemes, filter), BuiltinThemes) }

	if got := names("latte"); len(got) != 1 || got[0] != "catppuccin-latte" {
		t.Errorf(`filter "latte" = %v`, got)
	}
	if got := names("rose dawn"); len(got) != 1 || got[0] != "rose-pine-dawn" {
		t.Errorf(`filter "rose dawn" = %v, want every word to match`, got)
	}
	if got := names("zzzz"); len(got) != 0 {
		t.Errorf(`filter "zzzz" = %v, want none`, got)
	}
	light := names("light")
	if len(light) < 8 {
		t.Fatalf(`filter "light" = %v, want the light themes`, light)
	}
	for _, name := range light {
		theme, _ := ThemeByName(name)
		if !theme.IsLight() {
			t.Errorf(`filter "light" matched dark theme %s`, name)
		}
	}
}

func typeInto(p *ThemePicker, text string) {
	for _, r := range text {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestThemePickerSearchPreviewsFirstMatchAndEscClears(t *testing.T) {
	picker := NewThemePicker(ThemePickerOptions{InitialTheme: "catppuccin-mocha"})
	picker.Open("catppuccin-mocha")

	picker.Update(keyMsg("/"))
	if !picker.Filtering() {
		t.Fatal("/ should start the search")
	}
	typeInto(&picker, "nord")
	// "nord" also matches Nightfox's Nordfox; the cursor lands on the first match.
	if got := picker.PreviewTheme().Name; got != "nord" {
		t.Fatalf("typing a filter should preview the first match, got %s", got)
	}
	if picker.Filter() != "nord" {
		t.Fatalf("filter = %q", picker.Filter())
	}

	// Letters typed into the search are not navigation keys.
	typeInto(&picker, "k")
	if picker.Filter() != "nordk" {
		t.Fatalf("k should be typed into the search, filter = %q", picker.Filter())
	}
	picker.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if picker.Filter() != "nord" {
		t.Fatalf("backspace left filter %q", picker.Filter())
	}

	picker.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if picker.Filter() != "" || !picker.Filtering() {
		t.Fatalf("first esc clears the filter and keeps searching: filter %q filtering %v", picker.Filter(), picker.Filtering())
	}
	if action := picker.Update(tea.KeyMsg{Type: tea.KeyEsc}); action != ThemePickerNone || !picker.Opened() || picker.Filtering() {
		t.Fatalf("second esc leaves the search, not the picker: action %v opened %v filtering %v", action, picker.Opened(), picker.Filtering())
	}
}

func TestThemePickerNavigationStaysWithinTheFilter(t *testing.T) {
	picker := NewThemePicker(ThemePickerOptions{InitialTheme: "catppuccin-mocha"})
	picker.Open("catppuccin-mocha")
	picker.Update(keyMsg("/"))
	typeInto(&picker, "tokyo")
	picker.Update(tea.KeyMsg{Type: tea.KeyEnter}) // done typing, filter stays applied

	var seen []string
	for range 8 {
		seen = append(seen, picker.PreviewTheme().Name)
		picker.Update(keyMsg("j"))
	}
	for _, name := range seen {
		if !strings.HasPrefix(name, "tokyo-night") {
			t.Fatalf("navigation left the filtered themes: %v", seen)
		}
	}
	if got := picker.PreviewTheme().Name; got != "tokyo-night-day" {
		t.Fatalf("down should stop at the last match, got %s", got)
	}
}

func TestThemePickerEnterIgnoresAThemeTheFilterHides(t *testing.T) {
	picker := NewThemePicker(ThemePickerOptions{InitialTheme: "nord"})
	picker.Open("nord")
	picker.Update(keyMsg("/"))
	typeInto(&picker, "zzzz") // nothing matches, so the highlighted theme is not listed
	picker.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if action := picker.Update(tea.KeyMsg{Type: tea.KeyEnter}); action == ThemePickerConfirm {
		t.Fatal("enter must not confirm a theme the filter hides")
	}
}

func TestThemePickerOpenClearsTheFilter(t *testing.T) {
	picker := NewThemePicker(ThemePickerOptions{})
	picker.Open("nord")
	picker.Update(keyMsg("/"))
	typeInto(&picker, "dark")
	picker.Open("nord")
	if picker.Filter() != "" || picker.Filtering() {
		t.Fatalf("reopening should clear the search, got %q filtering %v", picker.Filter(), picker.Filtering())
	}
}

func TestThemePickerSoftModalShowsGroupsSwatchAndSample(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	picker := NewThemePicker(ThemePickerOptions{InitialTheme: "catppuccin-mocha"})
	picker.Open("catppuccin-mocha")

	view := ansi.Strip(picker.SoftModal(renderer, 46, 30, "tide").Content)
	for _, want := range []string{"/ to search", "Catppuccin", "Mocha ●", "Macchiato", "selected row", "link", "quote", "error", "match", "search", "confirm", "revert"} {
		if !strings.Contains(view, want) {
			t.Errorf("soft modal is missing %q:\n%s", want, view)
		}
	}

	picker.Update(keyMsg("/"))
	typeInto(&picker, "zzzz")
	view = ansi.Strip(picker.SoftModal(renderer, 46, 30, "tide").Content)
	if !strings.Contains(view, "no matching themes") {
		t.Errorf("an empty result should say so:\n%s", view)
	}
	if !strings.Contains(view, "zzzz") || !strings.Contains(view, "clear") || !strings.Contains(view, "done") {
		t.Errorf("search text and the filtering hints should show:\n%s", view)
	}
}

func TestThemePickerSoftModalFitsItsHeightAndScrollsToTheCursor(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	picker := NewThemePicker(ThemePickerOptions{InitialTheme: "catppuccin-mocha"})
	picker.Open("catppuccin-mocha")
	for range len(BuiltinThemes) - 1 {
		picker.Update(keyMsg("j"))
	}
	last := BuiltinThemes[len(BuiltinThemes)-1]

	for _, height := range []int{10, 14, 20, 26, 40} {
		content := picker.SoftModal(renderer, 46, height, "tide").Content
		view := ansi.Strip(content)
		lines := strings.Split(content, "\n")
		if len(lines) > height {
			t.Errorf("height %d: modal is %d lines tall", height, len(lines))
		}
		for i, line := range lines {
			if got := lipgloss.Width(line); got != 48 {
				t.Fatalf("height %d line %d width = %d, want 48", height, i, got)
			}
		}
		if !strings.Contains(view, last.Label()) {
			t.Errorf("height %d: the highlighted last theme %q is not visible:\n%s", height, last.Label(), view)
		}
	}
}

func TestThemePickerHideSwatchOmitsOnlyThatTheme(t *testing.T) {
	renderer := NewRenderer(VT52, StyleOptions{Density: Compact}) // plain UI: swatch cells are "#"
	picker := NewThemePicker(ThemePickerOptions{
		Themes:     []Theme{Nord, Dracula},
		HideSwatch: func(theme Theme) bool { return theme.Name == Dracula.Name },
	})
	picker.Open(Nord.Name)
	view := ansi.Strip(picker.SoftModal(renderer, 40, 12, "tide").Content)
	var nordRow, draculaRow string
	for _, line := range strings.Split(view, "\n") {
		switch {
		case strings.Contains(line, "Nord"):
			nordRow = line
		case strings.Contains(line, "Dracula"):
			draculaRow = line
		}
	}
	if !strings.Contains(nordRow, "#####") {
		t.Errorf("Nord should carry a swatch: %q", nordRow)
	}
	if strings.Contains(draculaRow, "#") {
		t.Errorf("Dracula's swatch should be hidden: %q", draculaRow)
	}
}

func TestLegacyModalHonoursTheFilter(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	picker := NewThemePicker(ThemePickerOptions{Themes: []Theme{CatppuccinMocha, Nord, Dracula}})
	picker.Open(CatppuccinMocha.Name)
	if rows := strings.Count(picker.Modal(renderer, 40, 20).Content, "\n") + 1; rows != 3 {
		t.Fatalf("an idle picker keeps its plain three-row layout, got %d rows", rows)
	}
	picker.Update(keyMsg("/"))
	typeInto(&picker, "nord")
	content := ansi.Strip(picker.Modal(renderer, 40, 20).Content)
	if !strings.Contains(content, "/ nord") || !strings.Contains(content, "Nord") || strings.Contains(content, "Dracula") {
		t.Fatalf("legacy modal should show the search and only the matches:\n%s", content)
	}
}
