package tideui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func renderFixture(t *testing.T) (WorkspaceRenderer, *Workspace) {
	t.Helper()
	theme := CatppuccinMocha
	renderer := NewRenderer(theme, StyleOptions{Density: Compact, PaneCorners: RoundCorners})
	ws := NewWorkspace(WithGap(1))
	ws.Panel("nav", Text("nav body")).Title("Navigation").Role(RoleNavigation).MinWidth(8).MinHeight(3)
	ws.Panel("main", Text("main body")).Title("Editor").Role(RolePrimary).MinWidth(20).MinHeight(3).
		Actions(Action("stage", "s", nil).Labeled("stage"))
	ws.Panel("log", Text("log body")).Title("Logs").Role(RoleTelemetry).MinWidth(8).MinHeight(3)
	ws.Panel("insp", Text("insp body")).Title("Inspector").Role(RoleInspector).MinWidth(8).MinHeight(3)
	ws.Panel("metrics", Text("metrics body")).Title("Metrics").Role(RoleTelemetry).MinWidth(8).MinHeight(3)
	ws.Layout(HStack(Leaf("nav"), Leaf("main"), VStack(Leaf("log"), Tabs("insp", "metrics"))))
	return NewWorkspaceRenderer(renderer), ws
}

func assertBounded(t *testing.T, view string, width, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != height {
		t.Fatalf("rendered %d lines, want %d", len(lines), height)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("line %d width %d exceeds %d: %q", i, got, width, ansi.Strip(line))
		}
	}
}

func TestWorkspaceRendererBoundedAtManySizes(t *testing.T) {
	wr, ws := renderFixture(t)
	for _, size := range [][2]int{{1, 1}, {2, 2}, {3, 3}, {5, 2}, {10, 5}, {40, 12}, {80, 24}, {120, 40}, {200, 60}} {
		view := wr.Render(ws, size[0], size[1])
		assertBounded(t, view, size[0], size[1])
	}
}

func TestWorkspaceRendererShowsFocusedPanelTitle(t *testing.T) {
	wr, ws := renderFixture(t)
	ws.Focus("main")
	ws.Solve(100, 30)
	view := ansi.Strip(wr.Render(ws, 100, 30))
	if !strings.Contains(view, "Editor") {
		t.Fatalf("focused title missing:\n%s", view)
	}
	if !strings.Contains(view, "main body") {
		t.Fatalf("focused body missing:\n%s", view)
	}
}

func TestWorkspaceRendererShowsKeyHintsForFocusedPanel(t *testing.T) {
	wr, ws := renderFixture(t)
	ws.Focus("main")
	view := ansi.Strip(wr.Render(ws, 100, 30))
	if !strings.Contains(view, "stage") || !strings.Contains(view, " s ") {
		t.Fatalf("action hint missing:\n%s", view)
	}
}

func TestWorkspaceRendererTabStackShowsTitles(t *testing.T) {
	wr, ws := renderFixture(t)
	view := ansi.Strip(wr.Render(ws, 120, 30))
	if !strings.Contains(view, "Inspector") || !strings.Contains(view, "Metrics") {
		t.Fatalf("tab stack titles missing:\n%s", view)
	}
}

func TestWorkspaceRendererArrangeMovesPanelLive(t *testing.T) {
	wr, ws := renderFixture(t)
	ws.Focus("nav")
	ws.Solve(100, 30)
	ws.EnterArrange()
	if !ws.ArrangeMove(DirRight) {
		t.Fatal("ArrangeMove failed")
	}
	// The panel really moved: nav now sits to the right of main.
	solved := ws.Solve(100, 30)
	if solved.Rects["nav"].X <= solved.Rects["main"].X {
		t.Fatalf("nav did not move right of main: %+v", solved.Rects)
	}
	view := ansi.Strip(wr.Render(ws, 100, 30))
	if !strings.Contains(view, "nav body") {
		t.Fatalf("moved panel content missing from render:\n%s", view)
	}
}

func TestWorkspaceRendererPeekOverlay(t *testing.T) {
	wr, ws := renderFixture(t)
	ws.Peek("log")
	view := ansi.Strip(wr.Render(ws, 100, 30))
	if !strings.Contains(view, "peek") {
		t.Fatalf("peek overlay missing:\n%s", view)
	}
}

func TestWorkspaceRendererStatusStrip(t *testing.T) {
	wr, ws := renderFixture(t)
	view := ansi.Strip(wr.Render(ws, 100, 30))
	if !strings.Contains(view, "arrange") {
		t.Fatalf("status hints missing:\n%s", view)
	}
}

func TestWorkspaceRendererASCIIDegradesCleanly(t *testing.T) {
	renderer := NewRenderer(VT52, StyleOptions{Density: Compact})
	ws := NewWorkspace(WithGap(1))
	ws.Panel("a", Text("alpha")).Title("Alpha").MinWidth(6).MinHeight(3)
	ws.Panel("b", Text("beta")).Title("Beta").MinWidth(6).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	wr := NewWorkspaceRenderer(renderer)
	view := wr.Render(ws, 40, 12)
	assertBounded(t, view, 40, 12)
	if strings.Contains(view, "╭") {
		t.Fatalf("ASCII theme rendered unicode borders:\n%s", ansi.Strip(view))
	}
}

// A pane that has the keyboard is told so: the panel is what draws its own
// cursor, and it cannot guess. Focused keeps its old meaning - this is the pane
// the workspace is pointed at - which is not the same question.
func TestPanelContextReportsTheKeyboard(t *testing.T) {
	wr, ws := renderFixture(t)
	var entered, focused []bool
	ws.Panel("main", func(ctx PanelContext) string {
		entered = append(entered, ctx.Entered)
		focused = append(focused, ctx.Focused)
		return "main body"
	})
	ws.Focus("main")

	wr.Render(ws, 100, 30)
	if len(entered) == 0 || entered[len(entered)-1] {
		t.Fatalf("Entered = %v before anything was entered", entered)
	}

	ws.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	wr.Render(ws, 100, 30)
	if !entered[len(entered)-1] {
		t.Fatal("an entered pane was not told it has the keyboard")
	}
	if !focused[len(focused)-1] {
		t.Fatal("the focused pane stopped reporting itself as focused")
	}
}
