package tideui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func testLayout(mode LayoutMode) Layout {
	return Layout{
		Width: 72, Height: 20, Mode: mode,
		Panes: [3]Pane{
			{Title: "Sidebar", Content: "one\ntwo", Focused: true},
			{Title: "List", Content: "alpha\nbeta"},
			{Title: "Detail", Content: "selected content"},
		},
		Status: &StatusBar{Left: "ready", Right: "? help"},
	}
}

func assertDimensions(t *testing.T, view string, width, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != height {
		t.Fatalf("rendered lines = %d, want %d", len(lines), height)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != width {
			t.Fatalf("line %d width = %d, want %d (%q)", i, got, width, ansi.Strip(line))
		}
	}
}

func TestRenderStackedRightFitsWindow(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	view := renderer.Render(testLayout(StackedRight))
	assertDimensions(t, view, 72, 20)
	plain := ansi.Strip(view)
	for _, title := range []string{"Sidebar", "List", "Detail", "ready", "? help"} {
		if !strings.Contains(plain, title) {
			t.Fatalf("expected output to contain %q in:\n%s", title, plain)
		}
	}
}

func TestRenderThreeColumnFitsWindow(t *testing.T) {
	renderer := NewRenderer(GruvboxLight, StyleOptions{Density: Comfortable})
	layout := testLayout(ThreeColumn)
	layout.ColumnRatios = [3]float64{2, 3, 5}
	view := renderer.Render(layout)
	assertDimensions(t, view, 72, 20)
	if !strings.Contains(ansi.Strip(view), "Detail") {
		t.Fatal("expected third pane to render")
	}
}

func TestFocusedPaneUsesThemeAccentUnlessPaneOverridesIt(t *testing.T) {
	renderer := NewRenderer(TokyoNight, StyleOptions{Density: Compact})
	if got := renderer.paneHeaderStyle(Pane{Title: "Tasks", Focused: true}).GetBackground(); got != TokyoNight.BorderFocus {
		t.Fatalf("focused header background = %v, want theme accent %v", got, TokyoNight.BorderFocus)
	}

	override := lipgloss.Color("#ff00ff")
	if got := renderer.paneHeaderStyle(Pane{Title: "Tasks", Focused: true, Accent: override}).GetBackground(); got != override {
		t.Fatalf("focused header background = %v, want pane accent override %v", got, override)
	}
}

func TestRenderOverlayCoversBaseWithoutChangingDimensions(t *testing.T) {
	renderer := NewRenderer(VT52, StyleOptions{Density: Compact})
	layout := testLayout(StackedRight)
	layout.Modal = &Overlay{Visible: true, Title: "Confirm", Content: "Proceed?", Footer: "enter apply", Width: 24}
	view := renderer.Render(layout)
	assertDimensions(t, view, 72, 20)
	plain := ansi.Strip(view)
	for _, part := range []string{"Confirm", "Proceed?", "enter apply"} {
		if !strings.Contains(plain, part) {
			t.Fatalf("expected overlay to include %q", part)
		}
	}
}

func TestOverlayTitleStaysInsideRequestedModalWidth(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	box := renderer.renderOverlay(Overlay{
		Title:   "TIDEUI",
		Content: "Modal content.",
		Footer:  "esc close",
		Width:   30,
	}, 80)
	for i, line := range strings.Split(box, "\n") {
		if got := lipgloss.Width(line); got != 30 {
			t.Fatalf("overlay line %d width = %d, want 30 (%q)", i, got, ansi.Strip(line))
		}
	}
}

func TestRenderBlockNoBodyMatchesRenderRow(t *testing.T) {
	for _, density := range []Density{Compact, Comfortable} {
		renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: density})
		row := renderer.RenderRow(Row{Prefix: "* ", Text: "Hello", Suffix: "3", Selected: true}, 40)
		block := renderer.RenderBlock(Block{Prefix: "* ", Header: "Hello", Meta: "3", Selected: true}, 40)
		if row != block {
			t.Fatalf("density=%s: block with no body should match row\nrow:   %q\nblock: %q", density, row, block)
		}
	}
}

func TestRenderBlockWithBodyFitsWidth(t *testing.T) {
	renderer := NewRenderer(Nord, StyleOptions{Density: Compact})
	block := renderer.RenderBlock(Block{
		Prefix: "● ", Header: "alice", Meta: "10:02",
		Body: "This is a longer message that should wrap correctly to fit within the block width.",
	}, 40)
	for i, line := range strings.Split(block, "\n") {
		if got := lipgloss.Width(line); got != 40 {
			t.Fatalf("block line %d width = %d, want 40 (%q)", i, got, ansi.Strip(line))
		}
	}
}

func TestRenderBlockBodyIsIndentedByPrefix(t *testing.T) {
	renderer := NewRenderer(Dracula, StyleOptions{Density: Compact})
	const prefix = ">> "
	block := renderer.RenderBlock(Block{
		Prefix: prefix, Header: "bob", Body: "Line one\nLine two",
	}, 40)
	lines := strings.Split(ansi.Strip(block), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	wantIndent := strings.Repeat(" ", lipgloss.Width(prefix))
	for _, line := range lines[1:] {
		if !strings.HasPrefix(line, wantIndent) {
			t.Fatalf("body line %q does not start with %q-char indent", line, wantIndent)
		}
	}
}

func TestRenderBlockStateRoutingMatchesRenderRow(t *testing.T) {
	// ANSI codes are stripped in non-TTY test environments, so we verify that
	// RenderBlock routes Selected/Muted identically to RenderRow by comparing
	// their outputs directly (they share the same style-selection code path).
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	for _, tc := range []struct{ selected, muted bool }{
		{false, false}, {true, false}, {false, true},
	} {
		got := renderer.RenderBlock(Block{Prefix: "  ", Header: "x", Selected: tc.selected, Muted: tc.muted}, 30)
		want := renderer.RenderRow(Row{Prefix: "  ", Text: "x", Selected: tc.selected, Muted: tc.muted}, 30)
		if got != want {
			t.Fatalf("selected=%v muted=%v: RenderBlock routing does not match RenderRow\ngot:  %q\nwant: %q",
				tc.selected, tc.muted, got, want)
		}
	}
}

func TestRenderSidebarOnlyFitsWindow(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	view := renderer.Render(testLayout(SidebarOnly))
	assertDimensions(t, view, 72, 20)
	plain := ansi.Strip(view)
	for _, title := range []string{"Sidebar", "List"} {
		if !strings.Contains(plain, title) {
			t.Fatalf("expected output to contain %q in:\n%s", title, plain)
		}
	}
	if strings.Contains(plain, "Detail") {
		t.Fatal("pane 2 title should not appear in SidebarOnly mode")
	}
}

func TestRenderTabbedFitsWindow(t *testing.T) {
	renderer := NewRenderer(Nord, StyleOptions{Density: Compact})
	layout := testLayout(Tabbed)
	view := renderer.Render(layout)
	assertDimensions(t, view, 72, 20)
	plain := ansi.Strip(view)
	for _, title := range []string{"Sidebar", "List", "Detail"} {
		if !strings.Contains(plain, title) {
			t.Fatalf("expected tab bar to contain %q in:\n%s", title, plain)
		}
	}
	if !strings.Contains(plain, "one") {
		t.Fatal("expected active pane content to be visible")
	}
}

func TestRenderFloatingFitsWindow(t *testing.T) {
	renderer := NewRenderer(Dracula, StyleOptions{Density: Compact})
	view := renderer.Render(testLayout(Floating))
	assertDimensions(t, view, 72, 20)
	plain := ansi.Strip(view)
	for _, title := range []string{"Sidebar", "List", "Detail"} {
		if !strings.Contains(plain, title) {
			t.Fatalf("expected output to contain %q in:\n%s", title, plain)
		}
	}
}

func TestRenderConstrainedWindowsNeverExceedsDimensions(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	for _, mode := range []LayoutMode{StackedRight, ThreeColumn, SidebarOnly, Tabbed, Floating} {
		for width := 1; width <= 12; width++ {
			for height := 1; height <= 8; height++ {
				layout := testLayout(mode)
				layout.Width, layout.Height = width, height
				layout.Modal = &Overlay{Visible: true, Title: "Modal", Content: "content", Width: 20}
				view := renderer.Render(layout)
				assertDimensions(t, view, width, height)
			}
		}
	}
}

func TestThreeColumnRatiosAlwaysReserveEachPaneWhenSpaceAllows(t *testing.T) {
	for _, ratios := range [][3]float64{{1000, 1, 1}, {1, 0, 0}, {-1, 2, 3}} {
		widths := columnSizes(20, ratios)
		if widths[0] < 1 || widths[1] < 1 || widths[2] < 1 {
			t.Fatalf("ratios %v produced empty pane widths %v", ratios, widths)
		}
		if got := widths[0] + widths[1] + widths[2]; got != 20 {
			t.Fatalf("ratios %v width total = %d, want 20", ratios, got)
		}
	}
}

func TestRenderTabbedScrollOffsetSkipsLines(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	layout := testLayout(Tabbed)
	layout.Panes[0].Focused = true
	layout.Panes[0].Content = "line-A\nline-B\nline-C\nline-D\nline-E"
	layout.Panes[0].ScrollOffset = 2
	view := renderer.Render(layout)
	assertDimensions(t, view, 72, 20)
	plain := ansi.Strip(view)
	if strings.Contains(plain, "line-A") || strings.Contains(plain, "line-B") {
		t.Fatal("scrolled-past lines should not appear in tabbed view")
	}
	if !strings.Contains(plain, "line-C") {
		t.Fatal("first visible line after scroll offset should appear in tabbed view")
	}
}

func TestRenderPaneScrollOffsetSkipsLines(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	layout := testLayout(StackedRight)
	layout.Panes[0].Content = "line-A\nline-B\nline-C\nline-D\nline-E"
	layout.Panes[0].ScrollOffset = 2
	view := renderer.Render(layout)
	assertDimensions(t, view, 72, 20)
	plain := ansi.Strip(view)
	if strings.Contains(plain, "line-A") || strings.Contains(plain, "line-B") {
		t.Fatal("scrolled-past lines should not appear in the view")
	}
	if !strings.Contains(plain, "line-C") {
		t.Fatal("first visible line after scroll offset should appear")
	}
}

func TestRenderPaneScrollOffsetOutOfRangeDoesNotPanic(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	layout := testLayout(StackedRight)
	layout.Panes[1].Content = "only\ntwo\nlines"
	layout.Panes[1].ScrollOffset = 999
	view := renderer.Render(layout)
	assertDimensions(t, view, 72, 20)
}

func TestTerminalBackgroundSequenceDoesNotWriteTerminal(t *testing.T) {
	set, reset := TerminalBackgroundSequences(CatppuccinMocha)
	if !strings.Contains(set, string(CatppuccinMocha.Bg)) || reset == "" {
		t.Fatalf("unexpected OSC strings: set=%q reset=%q", set, reset)
	}
}
