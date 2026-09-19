package tideui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAnimatorInterpolatesAndDegrades(t *testing.T) {
	anim := NewAnimator(AnimationOptions{Enabled: true})
	anim.Set("focus", 1)
	if got := anim.Value("focus"); got != 0 {
		t.Fatalf("initial value = %v, want 0", got)
	}
	if !anim.Tick() {
		t.Fatal("Tick reported no motion while active")
	}
	if got := anim.Value("focus"); got <= 0 || got >= 1 {
		t.Fatalf("value = %v, want strictly between 0 and 1", got)
	}
	for i := 0; i < 100; i++ {
		anim.Tick()
	}
	if got := anim.Value("focus"); got != 1 {
		t.Fatalf("settled value = %v, want 1", got)
	}
	if anim.Tick() {
		t.Fatal("Tick reported motion after settling")
	}

	reduced := NewAnimator(AnimationOptions{Enabled: true, ReducedMotion: true})
	reduced.Set("focus", 1)
	if reduced.Enabled() {
		t.Fatal("reduced motion should disable interpolation")
	}
	if reduced.Value("focus") != 1 {
		t.Fatal("reduced motion should jump straight to the target")
	}

	off := NewAnimator(AnimationOptions{})
	off.Set("x", 1)
	if off.Value("x") != 1 {
		t.Fatal("disabled animator should expose the target directly")
	}
}

func TestWorkspaceCommandsIncludePanelActions(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Panel("main", nil).Actions(Action("stage", "s", nil).Labeled("stage hunk"))
	commands := ws.Commands()

	byID := map[string]Command{}
	for _, command := range commands {
		byID[command.ID] = command
	}
	for _, want := range []string{"workspace.arrange", "workspace.reset", "action.main.stage", "panel.zoom.main"} {
		if _, ok := byID[want]; !ok {
			t.Fatalf("missing command %q", want)
		}
	}
	if got := byID["action.main.stage"].Key; got != "s" {
		t.Fatalf("action key = %q, want s", got)
	}
}

func TestCommandPaletteFilterAndRun(t *testing.T) {
	ws := newTestWorkspace(t)
	ran := false
	ws.Panel("main", nil).Actions(Action("stage", "s", func(*Workspace) { ran = true }).Labeled("stage hunk"))

	ws.OpenCommandPalette()
	palette := ws.CommandPalette()
	if !palette.Opened() {
		t.Fatal("palette did not open")
	}
	palette.Update(keyMsg("stage"))
	if len(palette.filtered) == 0 {
		t.Fatal("query filtered out the stage action")
	}
	if action := palette.Update(keyMsg("enter")); action != PaletteRun {
		t.Fatalf("enter action = %v, want PaletteRun", action)
	}
	if !ran {
		t.Fatal("selected command did not run")
	}
	if palette.Opened() {
		t.Fatal("palette should close after running")
	}
}

func TestCommandPaletteSearchByKeyAndCategory(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Panel("main", nil).Actions(Action("stage", "s", nil).Labeled("stage hunk").In("Diff"))
	ws.OpenCommandPalette()
	palette := ws.CommandPalette()
	palette.Update(keyMsg("Diff"))
	if len(palette.filtered) != 1 {
		t.Fatalf("category search returned %d commands, want 1", len(palette.filtered))
	}
	palette.Update(keyMsg("z"))
	if len(palette.filtered) != 0 {
		t.Fatalf("nonsense query returned %d commands, want 0", len(palette.filtered))
	}
}

func TestPanelPickerTogglesVisibility(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("log")
	ws.OpenPanelPicker()
	picker := ws.PanelPicker()
	if !picker.Opened() {
		t.Fatal("picker did not open")
	}
	if action := picker.Update(keyMsg(" ")); action != PanelPickerToggle {
		t.Fatalf("space action = %v, want PanelPickerToggle", action)
	}
	if !ws.Hidden("log") {
		t.Fatal("picker did not hide the focused panel")
	}
	if action := picker.Update(keyMsg("esc")); action != PanelPickerClose {
		t.Fatalf("esc action = %v, want PanelPickerClose", action)
	}
	if picker.Opened() {
		t.Fatal("picker should be closed")
	}
}

func TestMouseClickFocusesPanel(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	solved := ws.Solve(80, 24)
	main := solved.Rects["main"]
	msg := tea.MouseMsg{X: main.X + 1, Y: main.Y + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	if !ws.HandleMouse(msg) {
		t.Fatal("mouse click not handled")
	}
	if got := ws.Focused(); got != "main" {
		t.Fatalf("clicked focus = %q, want main", got)
	}
}

func TestMouseTabClickSwitchesTab(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{PaneCorners: RoundCorners})
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).Title("Alpha").MinWidth(6).MinHeight(3)
	ws.Panel("b", Text("b")).Title("Beta").MinWidth(6).MinHeight(3)
	ws.Layout(Tabs("a", "b"))
	ws.Focus("a")
	wr := NewWorkspaceRenderer(renderer)
	wr.Render(ws, 40, 12)

	if len(ws.tabHits) < 2 {
		t.Fatalf("expected two tab hit boxes, got %d", len(ws.tabHits))
	}
	second := ws.tabHits[1]
	msg := tea.MouseMsg{X: second.rect.X, Y: second.rect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	ws.HandleMouse(msg)
	stack, ok := ws.RootLayout().(*TabStackNode)
	if !ok || stack.Active != 1 {
		t.Fatalf("tab click did not switch tab: %+v", ws.RootLayout())
	}
}

func TestMouseSeparatorDragResizes(t *testing.T) {
	ws := NewWorkspace(WithGap(1))
	ws.Panel("a", Text("a")).MinWidth(5).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(5).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Focus("a")
	before := ws.Solve(80, 10).Rects["a"].Width

	gapX := ws.Solved().Rects["a"].X + ws.Solved().Rects["a"].Width
	press := tea.MouseMsg{X: gapX, Y: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	if !ws.HandleMouse(press) {
		t.Fatal("separator press not handled")
	}
	motion := tea.MouseMsg{X: gapX + 3, Y: 2, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft}
	ws.HandleMouse(motion)
	release := tea.MouseMsg{X: gapX + 3, Y: 2, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft}
	ws.HandleMouse(release)

	after := ws.Solve(80, 10).Rects["a"].Width
	if after <= before {
		t.Fatalf("drag did not grow panel a: %d -> %d", before, after)
	}
}

func TestOverlayRoutesToOpenPicker(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{})
	ws := newTestWorkspace(t)
	if ws.Overlay(renderer) != nil {
		t.Fatal("no overlay expected when nothing is open")
	}
	ws.OpenPanelPicker()
	overlay := ws.Overlay(renderer)
	if overlay == nil || !overlay.Visible {
		t.Fatal("panel picker overlay missing")
	}
}

func TestWorkspaceDispatchesFocusedPanelActions(t *testing.T) {
	ws := NewWorkspace()
	ran := ""
	ws.Panel("a", Text("a")).Actions(Action("toggle", "space", func(*Workspace) { ran = "toggle" }))
	ws.Panel("b", Text("b")).Actions(Action("other", "x", func(*Workspace) { ran = "other" }))
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Focus("a")

	if !ws.RunFocusedAction(" ") || ran != "toggle" {
		t.Fatalf("space dispatch = %q, want toggle", ran)
	}
	ran = ""
	if ws.RunFocusedAction("x") {
		t.Fatal("action on an unfocused panel should not run")
	}
	ws.Focus("b")
	if !ws.RunFocusedAction("x") || ran != "other" {
		t.Fatalf("x dispatch = %q, want other", ran)
	}

	ws.Focus("a")
	if !ws.HandleKey(keyMsg(" ")) {
		t.Fatal("HandleKey did not dispatch the focused action")
	}
	if ran != "toggle" {
		t.Fatalf("HandleKey dispatch = %q, want toggle", ran)
	}
}
