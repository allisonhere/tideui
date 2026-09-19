package tideui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestStatusVocabulary(t *testing.T) {
	kinds := []StatusKind{
		StatusHealthy, StatusWarning, StatusError, StatusStopped,
		StatusActive, StatusStale, StatusUpdating,
	}
	r := chromeRenderer(Compact)
	bg := r.Styles.Workspace.Bg
	seenGlyph := map[string]bool{}
	for _, kind := range kinds {
		if kind.Label() == "" || kind.Label() == "unknown" {
			t.Fatalf("status %d has no label", kind)
		}
		glyph := kind.Glyph(false)
		if glyph == "" || seenGlyph[glyph] {
			t.Fatalf("status %d glyph %q is empty or duplicated", kind, glyph)
		}
		seenGlyph[glyph] = true
		if kind.Glyph(true) == "" {
			t.Fatalf("status %d has no ASCII glyph", kind)
		}
		rendered := ansi.Strip(r.RenderStatus(kind, kind.Label(), bg))
		if !strings.Contains(rendered, kind.Label()) || !strings.Contains(rendered, glyph) {
			t.Fatalf("RenderStatus(%d) = %q", kind, rendered)
		}
	}
	// Colour is never the only signal: words differ across statuses.
	if StatusHealthy.Label() == StatusWarning.Label() {
		t.Fatal("distinct statuses must use distinct words")
	}
}

func TestRenderServicesUsesStatusKind(t *testing.T) {
	r := chromeRenderer(Compact)
	out := ansi.Strip(r.RenderServices([]ServiceStatus{
		{Name: "forgejo", Status: StatusWarning, Age: "6d"},
		{Name: "backup", Status: StatusStopped, Age: "--"},
	}, 40))
	if !strings.Contains(out, "▲") || !strings.Contains(out, "warning") {
		t.Fatalf("warning status glyph/word missing:\n%s", out)
	}
	if !strings.Contains(out, "■") || !strings.Contains(out, "stopped") {
		t.Fatalf("stopped status glyph/word missing:\n%s", out)
	}
	if !strings.Contains(out, "6d") || !strings.Contains(out, "--") {
		t.Fatalf("age column missing:\n%s", out)
	}
}

func TestWeatherHourlyIsGrouped(t *testing.T) {
	r := chromeRenderer(Compact)
	out := ansi.Strip(r.RenderWeather(WeatherData{
		Temperature: 72, Unit: "F", Condition: "Cloudy", High: 76, Low: 61,
		Hourly: []ForecastPoint{{Label: "3PM", Temperature: 74}, {Label: "6PM", Temperature: 70}},
	}, 40))
	grouped := false
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "3PM") && strings.Contains(line, "6PM") {
			grouped = true
		}
	}
	if !grouped {
		t.Fatalf("hourly forecast is not grouped on one line:\n%s", out)
	}
}

func TestFocusTokensAreSofterThanNeon(t *testing.T) {
	styles := BuildStyles(CatppuccinMocha, StyleOptions{})
	ws := styles.Workspace
	if ratio := contrastRatio(ws.FrameActive, ws.Bg); ratio < workspaceFocusMinContrast {
		t.Fatalf("focused frame contrast %.2f below floor %.1f", ratio, workspaceFocusMinContrast)
	}
	if ws.FocusRail == ws.FrameActive {
		t.Fatal("focus rail should be quieter than the focused frame")
	}
	if contrastRatio(ws.FocusRail, ws.Bg) >= contrastRatio(ws.FrameActive, ws.Bg) {
		t.Fatal("focus rail should have lower contrast than the focused frame")
	}
}

func TestWorkspaceStatusHierarchy(t *testing.T) {
	r := chromeRenderer(Compact)
	hints := []KeyHint{Hint("tab", "focus"), Hint("m", "arrange")}

	wide := ansi.Strip(r.RenderWorkspaceStatus("tideDeck  ·  dense", "updated 14:42", "", hints, 100))
	if lipgloss.Width(wide) != 100 {
		t.Fatalf("wide status width = %d, want 100", lipgloss.Width(wide))
	}
	for _, want := range []string{"tideDeck", "updated", "arrange"} {
		if !strings.Contains(wide, want) {
			t.Fatalf("wide status missing %q: %q", want, wide)
		}
	}

	// Hints are dropped before secondary metadata.
	mid := ansi.Strip(r.RenderWorkspaceStatus("tideDeck  ·  dense", "updated 14:42", "", hints, 30))
	if lipgloss.Width(mid) != 30 {
		t.Fatalf("mid status width = %d, want 30", lipgloss.Width(mid))
	}
	if !strings.Contains(mid, "tideDeck") {
		t.Fatalf("mid status lost identity: %q", mid)
	}

	// Primary identity survives the smallest widths.
	tiny := ansi.Strip(r.RenderWorkspaceStatus("tideDeck", "updated 14:42", "", hints, 10))
	if lipgloss.Width(tiny) != 10 {
		t.Fatalf("tiny status width = %d, want 10", lipgloss.Width(tiny))
	}
	if !strings.Contains(tiny, "tide") {
		t.Fatalf("tiny status lost identity: %q", tiny)
	}
}

func TestArrangeModeUsesStatusBarHints(t *testing.T) {
	wr, ws := renderFixture(t)
	if wr.Options.ShowArrangeCard {
		t.Fatal("floating arrange card should be off by default")
	}
	ws.Focus("main")
	ws.Solve(100, 30)
	ws.EnterArrange()
	strip := ansi.Strip(wr.renderStrip(ws, 100))
	for _, want := range []string{"ARRANGE", "move", "stack", "done"} {
		if !strings.Contains(strip, want) {
			t.Fatalf("arrange status strip missing %q: %q", want, strip)
		}
	}
}
