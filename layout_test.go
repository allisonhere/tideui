package tideui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/muesli/termenv"
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

func TestAlignRowNeverExceedsWidthEvenWithLongSuffix(t *testing.T) {
	cases := []struct{ prefix, text, suffix string }{
		{"  ", "x", "way-too-long-suffix-for-the-width"},
		{"way-too-long-prefix-for-the-width", "x", "y"},
		{"", "", "still-too-long"},
	}
	for width := 0; width <= 10; width++ {
		for _, tc := range cases {
			line := alignRow(tc.prefix, tc.text, tc.suffix, width)
			if got := lipgloss.Width(line); got != width {
				t.Errorf("alignRow(%q,%q,%q,%d) width = %d, want %d", tc.prefix, tc.text, tc.suffix, width, got, width)
			}
		}
	}
}

func TestPaneHeaderWithLongHintStaysOneLine(t *testing.T) {
	// Regression: alignRow used to leave an over-wide Hint untruncated,
	// which made the header's .Width(w).Render(...) word-wrap it into
	// several lines and corrupt the surrounding pane's fixed-height layout.
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	layout := testLayout(StackedRight)
	layout.Width, layout.Height = 20, 10
	layout.Panes[0].Hint = "a-hint-string-much-longer-than-the-pane-is-wide"
	view := renderer.Render(layout)
	assertDimensions(t, view, 20, 10)
}

func TestStatusBarLeftSurvivesWhenRightHintsAreTooLongToFit(t *testing.T) {
	// Regression: renderStatus used to truncate Right (static keyboard hints)
	// last, so a long hint list silently ate all the width and dropped Left
	// (transient, often time-sensitive state like a save confirmation or
	// error) entirely. Left must now win the width contest.
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	layout := testLayout(ThreeColumn)
	layout.Width, layout.Height = 60, 20
	layout.Status = &StatusBar{
		Left:  "settings saved",
		Right: strings.Repeat("a very long static hint list ", 5),
	}
	view := renderer.Render(layout)
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "settings saved") {
		t.Fatalf("status bar dropped Left text when Right didn't fit:\n%s", plain)
	}
}

func TestStatusBarRightStaysThemedWhenLeftCarriesEmbeddedStyling(t *testing.T) {
	// Regression: renderStatus concatenated a pre-styled Left (a host may
	// compose Left from multiple differently-styled segments, e.g. a
	// colored label next to a colored indicator) with plain Right text and
	// gap spaces, then wrapped the whole line in one outer Render call.
	// Once Left's own embedded styling reset, the gap and Right fell
	// through to the raw terminal default instead of the status bar's own
	// background — ANSI resets aren't scoped to "this wrapping style vs.
	// the one before it."
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	styledLeft := lipgloss.NewStyle().Background(lipgloss.Color("#ff0000")).Render(" left ")
	view := renderer.renderStatus(StatusBar{Left: styledLeft, Right: "hint"}, 40)

	bg := renderer.Styles.StatusBar.GetBackground()
	wantSGR := lipgloss.NewStyle().Background(bg).Render(" ")
	if !strings.Contains(view, wantSGR) {
		t.Fatalf("status bar gap/Right lost the bar's own background after a pre-styled Left:\n%q\nwant substring %q", view, wantSGR)
	}
}

func TestRenderRowWithLongSuffixStaysOneLine(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	row := renderer.RenderRow(Row{Prefix: "* ", Text: "x", Suffix: "way-too-long-suffix"}, 6)
	if strings.Contains(row, "\n") {
		t.Fatalf("RenderRow wrapped onto multiple lines: %q", row)
	}
	if got := lipgloss.Width(row); got != 6 {
		t.Fatalf("RenderRow width = %d, want 6", got)
	}
}

func TestRenderPaneNeverExceedsAllocatedBox(t *testing.T) {
	// Regression: at width==2 or height==2, turning on both border sides
	// while still guaranteeing >=1 content column/row previously overflowed
	// the allocated box by one cell in each direction.
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	pane := Pane{Title: "X", Content: "y"}
	for width := 1; width <= 6; width++ {
		for height := 1; height <= 6; height++ {
			out := renderer.renderPane(pane, width, height)
			lines := strings.Split(out, "\n")
			if len(lines) != height {
				t.Fatalf("width=%d height=%d: rendered %d lines, want %d", width, height, len(lines), height)
			}
			for _, line := range lines {
				if got := lipgloss.Width(line); got != width {
					t.Fatalf("width=%d height=%d: rendered line width = %d, want %d", width, height, got, width)
				}
			}
		}
	}
}

func TestEveryPaneRendersAFullFourSidedBorder(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	// Side-by-side modes: three (or two, for SidebarOnly) independently
	// bordered panes, each contributing four distinct corner glyphs —
	// including the panes that previously shared or omitted an edge.
	cases := []struct {
		mode        LayoutMode
		wantCorners int
	}{
		{StackedRight, 12},
		{ThreeColumn, 12},
		{SidebarOnly, 8}, // pane 2 is unused in this mode
	}
	for _, tc := range cases {
		plain := ansi.Strip(renderer.Render(testLayout(tc.mode)))
		corners := strings.Count(plain, "┌") + strings.Count(plain, "┐") +
			strings.Count(plain, "└") + strings.Count(plain, "┘")
		if corners != tc.wantCorners {
			t.Errorf("mode %v: corner glyph count = %d, want %d", tc.mode, corners, tc.wantCorners)
		}
	}
}

func TestFloatingBackgroundPaneKeepsItsOwnBorder(t *testing.T) {
	// The background pane in Floating mode previously had no border at all.
	// Its left edge isn't covered by the floating panels (which sit on the
	// right), so its left-side corners should now be visible.
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	lines := strings.Split(ansi.Strip(renderer.Render(testLayout(Floating))), "\n")
	if !strings.HasPrefix(lines[0], "┌") {
		t.Errorf("expected background pane top-left corner, got line 0 = %q", lines[0])
	}
	last := lines[len(lines)-2] // last line is the status bar
	if !strings.HasPrefix(last, "└") {
		t.Errorf("expected background pane bottom-left corner, got last body line = %q", last)
	}
}

func TestPaneCornersRoundUsesRoundedGlyphs(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact, PaneCorners: RoundCorners})
	view := ansi.Strip(renderer.Render(testLayout(StackedRight)))
	for _, glyph := range []string{"╭", "╮", "╰", "╯"} {
		if !strings.Contains(view, glyph) {
			t.Errorf("expected rounded corner glyph %q in output", glyph)
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

func TestRenderDrawsModalShadowOnlyWhenEnabled(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	layout := testLayout(StackedRight)
	layout.Modal = &Overlay{Visible: true, Title: "Confirm", Content: "Proceed?", Footer: "enter apply", Width: 24}

	withShadow := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact, ModalShadow: true})
	viewWithShadow := withShadow.Render(layout)

	withoutShadow := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	viewWithoutShadow := withoutShadow.Render(layout)

	if viewWithShadow == viewWithoutShadow {
		t.Fatal("expected ModalShadow to change the rendered view")
	}
}

func TestOverlayModalMatchesRendersOwnModalShadow(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	// A caller with a bespoke layout builds its own base view and its own
	// already-rendered modal chrome (Raw: true is exactly this case — see
	// Overlay.Raw), rather than going through Render's Panes/Mode system.
	layout := testLayout(StackedRight)
	layout.Modal = nil
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact, ModalShadow: true})
	base := renderer.Render(layout)
	modalChrome := renderer.renderOverlay(Overlay{Content: "Proceed?", Raw: true}, layout.Width)

	got := renderer.OverlayModal(base, modalChrome, layout.Width, layout.Height)

	// What Render itself produces when the same modal is wired in as
	// layout.Modal must be identical: OverlayModal is meant to be the same
	// step, not a second implementation of it.
	layout.Modal = &Overlay{Visible: true, Content: "Proceed?", Raw: true}
	want := renderer.Render(layout)

	if got != want {
		t.Fatal("OverlayModal did not match Render's own modal-shadow compositing")
	}

	withoutShadow := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	plainBase := withoutShadow.Render(testLayout(StackedRight))
	plainChrome := withoutShadow.renderOverlay(Overlay{Content: "Proceed?", Raw: true}, layout.Width)
	unshadowed := withoutShadow.OverlayModal(plainBase, plainChrome, layout.Width, layout.Height)
	if got == unshadowed {
		t.Fatal("expected OverlayModal's shadow to actually change the rendered view")
	}
}

func TestModalShadowRespectsReducedColorProfile(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	layout := testLayout(StackedRight)
	layout.Modal = &Overlay{Visible: true, Title: "Confirm", Content: "Proceed?", Footer: "enter apply", Width: 24}

	withShadow := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact, ModalShadow: true})
	viewWithShadow := withShadow.Render(layout)

	withoutShadow := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	viewWithoutShadow := withoutShadow.Render(layout)

	if viewWithShadow == viewWithoutShadow {
		t.Fatal("expected ModalShadow to still change the rendered view under a reduced (ANSI256) color profile")
	}
	if strings.Contains(viewWithShadow, "48;2;") {
		t.Fatal("shadow emitted a raw truecolor (48;2;) background sequence under an ANSI256 profile — a terminal that doesn't support truecolor would likely ignore it, making the shadow silently not render at all")
	}
}

func TestBlendShadowRectDarkensOnlyTheGivenRectangle(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	page := lipgloss.Color("#1e1e2e")
	shadowBg := lipgloss.Color("#000000")
	pageLine := lipgloss.NewStyle().Background(page).Render(strings.Repeat(" ", 10))
	base := strings.Join([]string{pageLine, pageLine, pageLine, pageLine}, "\n")

	blended := blendShadowRect(base, 2, 1, 4, 2, 10, 4, page, shadowBg)

	buf := cellbuf.NewBuffer(10, 4)
	cellbuf.SetContent(buf, blended)

	pr, pg, pb, _ := hexToRGB(page)
	sr, sg, sb, _ := hexToRGB(shadowBg)

	inside := buf.Cell(3, 1)
	if inside == nil || inside.Style.Bg == nil {
		t.Fatal("expected a resolved background inside the shadow rectangle")
	}
	ir, ig, ib := cellRGB(inside.Style.Bg)
	if !strictlyBetween(ir, pr, sr) || !strictlyBetween(ig, pg, sg) || !strictlyBetween(ib, pb, sb) {
		t.Fatalf("cell inside shadow rect = (%.3f,%.3f,%.3f), want strictly between page (%.3f,%.3f,%.3f) and shadow (%.3f,%.3f,%.3f)",
			ir, ig, ib, pr, pg, pb, sr, sg, sb)
	}

	outside := buf.Cell(0, 0)
	if outside == nil || outside.Style.Bg == nil {
		t.Fatal("expected a resolved background outside the shadow rectangle")
	}
	or, og, ob := cellRGB(outside.Style.Bg)
	if or != pr || og != pg || ob != pb {
		t.Fatalf("cell outside shadow rect = (%.3f,%.3f,%.3f), want unchanged page color (%.3f,%.3f,%.3f)", or, og, ob, pr, pg, pb)
	}
}

func TestBlendShadowRectAlsoDarkensGlyphForeground(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	page := lipgloss.Color("#1e1e2e")
	fg := lipgloss.Color("#e8edf2")
	shadowBg := lipgloss.Color("#000000")
	// A line with real, bright-foreground text spanning the shadow
	// rectangle — this app renders text right up to a pane's edge, exactly
	// where the shadow's sliver sits, so a shadow that only darkens
	// backgrounds leaves any character there at its original brightness.
	textLine := lipgloss.NewStyle().Background(page).Foreground(fg).Render("XXXXXXXXXX")
	base := strings.Join([]string{textLine, textLine, textLine, textLine}, "\n")

	blended := blendShadowRect(base, 2, 1, 4, 2, 10, 4, page, shadowBg)

	buf := cellbuf.NewBuffer(10, 4)
	cellbuf.SetContent(buf, blended)

	fr, fg2, fb, _ := hexToRGB(fg)
	sr, sg, sb, _ := hexToRGB(shadowBg)

	inside := buf.Cell(3, 1)
	if inside == nil || inside.Style.Fg == nil {
		t.Fatal("expected a resolved foreground inside the shadow rectangle")
	}
	ir, ig, ib := cellRGB(inside.Style.Fg)
	if !strictlyBetween(ir, fr, sr) || !strictlyBetween(ig, fg2, sg) || !strictlyBetween(ib, fb, sb) {
		t.Fatalf("glyph foreground inside shadow rect = (%.3f,%.3f,%.3f), want strictly between original fg (%.3f,%.3f,%.3f) and shadow (%.3f,%.3f,%.3f) — the glyph should darken along with its background",
			ir, ig, ib, fr, fg2, fb, sr, sg, sb)
	}

	outside := buf.Cell(0, 0)
	if outside == nil || outside.Style.Fg == nil {
		t.Fatal("expected a resolved foreground outside the shadow rectangle")
	}
	or, og, ob := cellRGB(outside.Style.Fg)
	if or != fr || og != fg2 || ob != fb {
		t.Fatalf("glyph foreground outside shadow rect = (%.3f,%.3f,%.3f), want unchanged original fg (%.3f,%.3f,%.3f)", or, og, ob, fr, fg2, fb)
	}
}

// TestBlendShadowRectLeavesRowsOutsideTheRectangleByteForByte is the
// regression test for scoping blendShadowRect to just rows y..y+h instead
// of the whole totalHeight viewport (a real, measured perf fix — parsing/
// serializing the entire base render every frame to darken a small
// rectangle was costing roughly a third of total render time in a
// consuming app). The earlier shadow tests use identical repeated lines,
// which can't tell a correctly-untouched row apart from one that happened
// to get overwritten with the same content by accident; this uses
// distinct content per row specifically to catch that.
func TestBlendShadowRectLeavesRowsOutsideTheRectangleByteForByte(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	page := lipgloss.Color("#1e1e2e")
	shadowBg := lipgloss.Color("#000000")
	row := func(text string) string {
		return lipgloss.NewStyle().Background(page).Render(text)
	}
	original := []string{row("row-zero-x"), row("row-one--x"), row("row-two--x"), row("row-three-"), row("row-four--")}
	base := strings.Join(original, "\n")

	// Shadow rectangle covers rows 1..3 only (y=1, h=2) out of 5 total.
	blended := blendShadowRect(base, 2, 1, 4, 2, 10, 5, page, shadowBg)
	blendedLines := strings.Split(blended, "\n")

	if len(blendedLines) != len(original) {
		t.Fatalf("blended line count = %d, want %d (row count must be preserved)", len(blendedLines), len(original))
	}
	for _, i := range []int{0, 3, 4} {
		if blendedLines[i] != original[i] {
			t.Fatalf("row %d changed but is outside the shadow rectangle (rows 1-2 only):\ngot  %q\nwant %q", i, blendedLines[i], original[i])
		}
	}
	for _, i := range []int{1, 2} {
		if blendedLines[i] == original[i] {
			t.Fatalf("row %d is inside the shadow rectangle but wasn't darkened at all", i)
		}
	}
}

// TestBlendShadowRectHandlesRectangleOutsideViewport checks the early-out
// for a shadow rectangle that doesn't intersect the viewport at all (e.g.
// a modal positioned such that its shadow offset falls off the bottom
// edge) — must return the input unchanged, not panic on an empty slice.
func TestBlendShadowRectHandlesRectangleOutsideViewport(t *testing.T) {
	page := lipgloss.Color("#1e1e2e")
	shadowBg := lipgloss.Color("#000000")
	pageLine := lipgloss.NewStyle().Background(page).Render(strings.Repeat(" ", 10))
	base := strings.Join([]string{pageLine, pageLine}, "\n")

	got := blendShadowRect(base, 0, 5, 4, 2, 10, 2, page, shadowBg)
	if got != base {
		t.Fatalf("blendShadowRect() with an out-of-viewport rectangle = %q, want the unchanged input %q", got, base)
	}
}

func TestShadowedTextStaysLegible(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	page := lipgloss.Color("#333c46")
	fg := lipgloss.Color("#e8edf2")
	shadowBg := shadowColor(page)
	textLine := lipgloss.NewStyle().Background(page).Foreground(fg).Render("XXXXXXXXXX")
	base := strings.Join([]string{textLine, textLine, textLine, textLine}, "\n")

	blended := blendShadowRect(base, 2, 1, 4, 2, 10, 4, page, shadowBg)
	buf := cellbuf.NewBuffer(10, 4)
	cellbuf.SetContent(buf, blended)

	inside := buf.Cell(3, 1)
	if inside == nil || inside.Style.Fg == nil || inside.Style.Bg == nil {
		t.Fatal("expected resolved fg/bg inside the shadow rectangle")
	}
	fgHex := cellHex(inside.Style.Fg)
	bgHex := cellHex(inside.Style.Bg)
	// Blending fg and bg toward the same near-black target at the same
	// strength collapses their contrast to near zero — shadowed text
	// becomes indistinguishable from its own background instead of just
	// dimming (this is exactly what shadowFgAlpha exists to prevent).
	// WCAG AA's normal-text floor (4.5:1) is a reasonable, well-known bar
	// for "still actually legible."
	if ratio := contrastRatio(fgHex, bgHex); ratio < 4.5 {
		t.Fatalf("shadowed text contrast = %.2f (fg=%s bg=%s), want >= 4.5 — text should dim, not disappear into its background", ratio, fgHex, bgHex)
	}
}

func cellHex(c ansi.Color) lipgloss.Color {
	r, g, b := cellRGB(c)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", uint8(r*255+0.5), uint8(g*255+0.5), uint8(b*255+0.5)))
}

func cellRGB(c ansi.Color) (r, g, b float64) {
	rr, gg, bb, _ := c.RGBA()
	return float64(rr>>8) / 255, float64(gg>>8) / 255, float64(bb>>8) / 255
}

func strictlyBetween(v, a, b float64) bool {
	if a > b {
		a, b = b, a
	}
	return v > a && v < b
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

// BenchmarkBlendShadowRectCompactModal tracks blendShadowRect's cost for a
// realistic case: a compact modal (15 of 34 rows) in a normal-sized
// terminal. Measured live before the row-scoping fix: ~1.17ms/6,783
// allocs; after: ~0.51ms/4,332 allocs — regressing this back toward the
// old numbers means the scoping got lost somewhere.
func BenchmarkBlendShadowRectCompactModal(b *testing.B) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	page := lipgloss.Color("#1e1e2e")
	shadowBg := lipgloss.Color("#000000")
	const totalWidth, totalHeight = 120, 34
	var rows []string
	for i := 0; i < totalHeight; i++ {
		rows = append(rows, lipgloss.NewStyle().Background(page).Render(strings.Repeat("x", totalWidth)))
	}
	base := strings.Join(rows, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = blendShadowRect(base, 30, 8, 50, 15, totalWidth, totalHeight, page, shadowBg)
	}
}
