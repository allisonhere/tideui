package tideui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// pickerEntry is one line of the grouped picker: a family header or a theme.
type pickerEntry struct {
	header string // family name, for a header line
	idx    int    // index into the picker's themes, for a theme row (-1 for headers)
	solo   bool   // the only theme of its family: shown without a header
}

// themeSearchText is what a filter word is matched against: the family and
// variant, the stored name, and whether the theme is light or dark.
func themeSearchText(t Theme) string {
	polarity := "dark"
	if t.IsLight() {
		polarity = "light"
	}
	return strings.ToLower(t.Family + " " + t.Variant + " " + t.Name + " " + polarity)
}

// pickerEntries lists the themes matching every word of filter, grouped under a
// header per family. Themes are grouped by consecutive run, so the order the
// caller gave is kept. A theme with no Family is listed on its own, as is a
// family with a single match.
func pickerEntries(themes []Theme, filter string) []pickerEntry {
	words := strings.Fields(strings.ToLower(filter))
	var matched []int
	for i, theme := range themes {
		text := themeSearchText(theme)
		ok := true
		for _, word := range words {
			if !strings.Contains(text, word) {
				ok = false
				break
			}
		}
		if ok {
			matched = append(matched, i)
		}
	}
	var out []pickerEntry
	for start := 0; start < len(matched); {
		family := themes[matched[start]].Family
		end := start + 1
		for family != "" && end < len(matched) && themes[matched[end]].Family == family {
			end++
		}
		if end-start == 1 {
			out = append(out, pickerEntry{idx: matched[start], solo: true})
		} else {
			out = append(out, pickerEntry{header: family, idx: -1})
			for _, i := range matched[start:end] {
				out = append(out, pickerEntry{idx: i})
			}
		}
		start = end
	}
	return out
}

// entryPos is the position of theme idx in entries, or -1.
func entryPos(entries []pickerEntry, idx int) int {
	for i, entry := range entries {
		if entry.header == "" && entry.idx == idx {
			return i
		}
	}
	return -1
}

// ThemeSwatch draws a theme's accent, success, warning, error and tag colors as
// five colored cells, so variants can be told apart before they are previewed.
// With plain set the cells are "#" characters for terminals without color
// backgrounds.
func ThemeSwatch(theme Theme, plain bool) string {
	cell := " "
	if plain {
		cell = "#"
	}
	var b strings.Builder
	for _, c := range []lipgloss.Color{
		theme.BorderFocus, theme.SuccessColor(), theme.WarningColor(), theme.Error, theme.TagColor(),
	} {
		b.WriteString(lipgloss.NewStyle().Background(c).Foreground(c).Render(cell))
	}
	return b.String()
}

// renderPickerSample shows a theme's colors in use on the modal surface: a
// selected row, body, muted, link and quote text, and the status and tag
// colors. It is drawn from the previewed theme's own roles, so it is right even
// when the application has not yet rebuilt its renderer for the preview.
func renderPickerSample(theme Theme, plain bool, surface lipgloss.Color, width int) []string {
	chrome := newSoftChromeOn(Styles{Theme: theme, PlainUI: plain}, surface)
	on := func(fg lipgloss.Color, s string) string {
		return lipgloss.NewStyle().Background(surface).Foreground(Legible(fg, surface, 4.5)).Render(s)
	}
	pill := func(c lipgloss.Color, s string) string {
		return lipgloss.NewStyle().Background(c).Foreground(contrastFg(c)).Render(" " + s + " ")
	}
	gap := on(chrome.text, "  ")

	selected := padStyled(
		lipgloss.NewStyle().Background(chrome.selectedBg).Foreground(chrome.selectedText).Bold(true).Render(" selected row"),
		width, chrome.selectedBg)
	text := strings.Join([]string{
		on(chrome.text, "text"), on(chrome.muted, "muted"), on(theme.LinkColor(), "link"), on(theme.QuoteColor(), "│ quote"),
	}, gap)
	status := strings.Join([]string{
		on(theme.SuccessColor(), "ok"), on(theme.WarningColor(), "warn"), on(theme.ErrorColor(), "error"),
		pill(theme.TagColor(), "tag"), pill(theme.MatchColor(), "match"),
	}, gap)
	return []string{selected, padStyled(text, width, surface), padStyled(status, width, surface)}
}
