package tideui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// containsColor reports whether a rendered view carries the given colour as a
// 24-bit foreground or background SGR sequence.
func containsColor(view string, color lipgloss.Color) bool {
	r, g, b, ok := hexToRGB(color)
	if !ok {
		return false
	}
	component := fmt.Sprintf("%d;%d;%d", int(r*255+0.5), int(g*255+0.5), int(b*255+0.5))
	return strings.Contains(view, "48;2;"+component) || strings.Contains(view, "38;2;"+component)
}

func themedRenderer() WorkspaceRenderer {
	return NewWorkspaceRenderer(NewRenderer(CatppuccinMocha, StyleOptions{
		Density: Compact, PaneCorners: RoundCorners,
	}))
}

func TestPanelThemeAccessorsAndClear(t *testing.T) {
	ws := NewWorkspace()
	panel := ws.Panel("a", Text("a"))
	if panel.HasPanelTheme() {
		t.Fatal("new panel should not have a panel theme")
	}
	if _, ok := panel.PanelTheme(); ok {
		t.Fatal("PanelTheme reported a theme when none was set")
	}

	panel.Theme(GruvboxLight)
	theme, ok := panel.PanelTheme()
	if !ok || theme.Name != GruvboxLight.Name {
		t.Fatalf("PanelTheme = %q, %v", theme.Name, ok)
	}
	panel.Overrides(ThemeOverrides{Accent: "#ff00ff"})
	if panel.PanelOverrides().Accent != "#ff00ff" {
		t.Fatal("PanelOverrides did not round-trip")
	}
	if !panel.HasPanelTheme() {
		t.Fatal("HasPanelTheme should be true after Overrides")
	}

	panel.ClearTheme()
	if panel.HasPanelTheme() {
		t.Fatal("ClearTheme left panel theming in place")
	}
	if _, ok := panel.PanelTheme(); ok {
		t.Fatal("ClearTheme left a theme set")
	}
}

func TestPanelRendererResolution(t *testing.T) {
	wr := themedRenderer()

	plain := &Panel{}
	if got := wr.panelRenderer(plain).Styles.Theme.Name; got != CatppuccinMocha.Name {
		t.Fatalf("unset panel renderer theme = %q, want workspace theme", got)
	}

	themed := &Panel{}
	themed.Theme(GruvboxLight)
	styles := wr.panelRenderer(themed).Styles
	if styles.Theme.Name != GruvboxLight.Name {
		t.Fatalf("panel theme = %q, want %q", styles.Theme.Name, GruvboxLight.Name)
	}
	if styles.Density != Compact || styles.PaneCorners != RoundCorners {
		t.Fatalf("global density/corners not preserved: %q/%q", styles.Density, styles.PaneCorners)
	}

	overridden := &Panel{}
	overridden.Overrides(ThemeOverrides{Accent: "#ff0000"})
	if got := wr.panelRenderer(overridden).Styles.Theme.BorderFocus; got != "#ff0000" {
		t.Fatalf("override accent = %q, want #ff0000", got)
	}

	both := &Panel{}
	both.Theme(Nord)
	both.Overrides(ThemeOverrides{Background: "#101010"})
	bothStyles := wr.panelRenderer(both).Styles
	if bothStyles.Theme.Name != Nord.Name {
		t.Fatalf("layered theme name = %q, want %q", bothStyles.Theme.Name, Nord.Name)
	}
	if bothStyles.Theme.Bg != "#101010" {
		t.Fatalf("layered override bg = %q, want #101010", bothStyles.Theme.Bg)
	}
}

func TestPanelContextCarriesPanelRenderer(t *testing.T) {
	wr := themedRenderer()
	ws := NewWorkspace(WithGap(0))
	captured := ""
	ws.Panel("a", func(ctx PanelContext) string {
		captured = ctx.Renderer.Styles.Theme.Name
		return "a"
	}).Theme(GruvboxLight).MinWidth(5).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(5).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Focus("a")

	wr.Render(ws, 40, 12)
	if captured != GruvboxLight.Name {
		t.Fatalf("PanelContext renderer theme = %q, want %q", captured, GruvboxLight.Name)
	}
}

func TestTabStackUsesActivePanelTheme(t *testing.T) {
	wr := themedRenderer()
	ws := NewWorkspace(WithGap(0))
	captured := ""
	ws.Panel("a", func(ctx PanelContext) string { captured = ctx.Renderer.Styles.Theme.Name; return "a" }).
		Theme(GruvboxLight).MinWidth(5).MinHeight(3)
	ws.Panel("b", func(ctx PanelContext) string { return "b" }).MinWidth(5).MinHeight(3)
	ws.Layout(SetActiveTab(Tabs("a", "b"), "a"))
	ws.Focus("a")

	wr.Render(ws, 40, 12)
	if captured != GruvboxLight.Name {
		t.Fatalf("active tab theme = %q, want %q", captured, GruvboxLight.Name)
	}
}

func TestPeekUsesPanelTheme(t *testing.T) {
	wr := themedRenderer()
	ws := NewWorkspace(WithGap(0))
	captured := ""
	ws.Panel("a", Text("base")).Title("Alpha").MinWidth(5).MinHeight(3)
	ws.Panel("hidden", func(ctx PanelContext) string {
		captured = ctx.Renderer.Styles.Theme.Name
		return "detail"
	}).Title("Hidden").Theme(Nord).MinWidth(5).MinHeight(3).Hide()
	ws.Layout(HStack(Leaf("a"), Leaf("hidden")))
	ws.Solve(80, 24)
	ws.Peek("hidden")

	wr.Render(ws, 80, 24)
	if captured != Nord.Name {
		t.Fatalf("peek renderer theme = %q, want %q", captured, Nord.Name)
	}
}

func TestPerPanelThemeChangesRenderingOnlyForThatPanel(t *testing.T) {
	withTrueColor(t)
	wr := themedRenderer()
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("alpha")).Title("Alpha").Theme(GruvboxLight).MinWidth(5).MinHeight(3)
	ws.Panel("b", Text("beta")).Title("Beta").MinWidth(5).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Focus("b")

	view := wr.Render(ws, 40, 12)
	// The light panel's own background must appear somewhere in the frame,
	// while the workspace theme still owns the page and status bar.
	if !containsColor(view, GruvboxLight.Bg) {
		t.Fatal("themed panel background not present in render")
	}
	if !containsColor(view, CatppuccinMocha.Bg) {
		t.Fatal("workspace background missing from render")
	}
}

func TestThemedPanelRenderBounded(t *testing.T) {
	wr := themedRenderer()
	ws := NewWorkspace(WithGap(1))
	ws.Panel("a", Text("alpha")).Title("Alpha").Theme(GruvboxLight).MinWidth(4).MinHeight(3)
	ws.Panel("b", Text("beta")).Title("Beta").Overrides(ThemeOverrides{Background: "#000000"}).MinWidth(4).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Focus("a")

	for _, size := range [][2]int{{3, 3}, {10, 6}, {40, 12}, {120, 30}} {
		view := wr.Render(ws, size[0], size[1])
		lines := strings.Split(view, "\n")
		if len(lines) != size[1] {
			t.Fatalf("%v: %d lines, want %d", size, len(lines), size[1])
		}
		for i, line := range lines {
			if got := lipgloss.Width(line); got > size[0] {
				t.Fatalf("%v: line %d width %d", size, i, got)
			}
		}
	}
}
