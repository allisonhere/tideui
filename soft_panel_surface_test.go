package tideui

import (
	"regexp"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

var bgPattern = regexp.MustCompile(`48;2;(\d+);(\d+);(\d+)`)

// backgroundsOf reports every distinct background colour in rendered text.
func backgroundsOf(text string) map[string]bool {
	found := map[string]bool{}
	for _, match := range bgPattern.FindAllStringSubmatch(text, -1) {
		found[match[1]+","+match[2]+","+match[3]] = true
	}
	return found
}

// A soft row drawn into a pane has to sit on the pane's background. Drawn on
// the modal surface it comes out a different shade from the blank space around
// it, and the pane looks banded.
func TestRenderSoftRowOnUsesTheGivenSurface(t *testing.T) {
	trueColor(t)
	for _, theme := range BuiltinThemes[:6] {
		r := NewRenderer(theme, StyleOptions{})
		row := r.RenderSoftRowOn(SoftRow{Text: "gauge style", Suffix: "solid"}, 40, theme.Bg)

		want := rgbOf(theme.Bg)
		got := backgroundsOf(row)
		if !got[want] {
			t.Errorf("%s: row backgrounds %v, want the pane background %s",
				theme.Name, keysOf(got), want)
		}
		if modal := rgbOf(modalSurface(theme)); modal != want && got[modal] {
			t.Errorf("%s: row still drawn on the modal surface %s", theme.Name, modal)
		}
	}
}

// The default stays on the modal surface, which is right inside a modal.
func TestRenderSoftRowKeepsTheModalSurface(t *testing.T) {
	trueColor(t)
	theme := BuiltinThemes[0]
	r := NewRenderer(theme, StyleOptions{})
	row := r.RenderSoftRow(SoftRow{Text: "solid"}, 40)
	if want := rgbOf(modalSurface(theme)); !backgroundsOf(row)[want] {
		t.Errorf("row backgrounds %v, want the modal surface %s",
			keysOf(backgroundsOf(row)), want)
	}
}

// rgbOf renders a swatch in the colour and reads back what was emitted, so the
// comparison runs through the same conversion the rows do. Parsing the hex
// directly disagrees by a unit after termenv's colour conversion.
func rgbOf(c lipgloss.Color) string {
	swatch := lipgloss.NewStyle().Background(c).Render(" ")
	for key := range backgroundsOf(swatch) {
		return key
	}
	return ""
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// A toned divider colours its rule and its label alike, so a line labelled with
// a state reads as one thing rather than as a coloured word on a grey rule.
func TestSectionDividerTone(t *testing.T) {
	trueColor(t)
	theme := BuiltinThemes[0]
	r := NewRenderer(theme, StyleOptions{})

	for _, tone := range []Tone{ToneGood, ToneWarning, ToneDanger} {
		rule := r.RenderSectionDivider(
			SectionDivider{Label: "state", Width: 40, Tone: tone}, theme.Bg)
		want := rgbOf(r.ToneColor(tone))
		got := foregroundsOf(rule)
		if len(got) != 1 || !got[want] {
			t.Errorf("tone %v: colours %v, want only %s", tone, keysOf(got), want)
		}
	}

	// The zero value keeps the quiet default: a separator-coloured rule with a
	// subtitle-coloured label.
	plain := r.RenderSectionDivider(SectionDivider{Label: "general", Width: 40}, theme.Bg)
	if got := foregroundsOf(plain); len(got) < 2 {
		t.Errorf("untoned divider collapsed to %v, want the default two colours", keysOf(got))
	}
}

var fgPattern = regexp.MustCompile(`38;2;(\d+);(\d+);(\d+)`)

func foregroundsOf(text string) map[string]bool {
	found := map[string]bool{}
	for _, match := range fgPattern.FindAllStringSubmatch(text, -1) {
		found[match[1]+","+match[2]+","+match[3]] = true
	}
	return found
}
