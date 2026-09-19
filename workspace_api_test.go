package tideui

import (
	"strings"
	"testing"
	"time"
)

func TestPanelMetadataAccessors(t *testing.T) {
	ws := NewWorkspace()
	panel := ws.Panel("symbol", Text("body")).
		Title("Symbol").Description("details").Role(RoleInspector).
		MinWidth(12).MinHeight(4).PreferredWidth(30).PreferredHeight(10).
		Grow(3).Shrink(2).Priority(55).Focusable(false).Hideable(false).
		Zoomable(false).Badge("9").Hint("hi").Accent("#ff0000")

	if panel.ID() != "symbol" {
		t.Fatalf("ID = %q", panel.ID())
	}
	if panel.TitleText() != "Symbol" || panel.Desc() != "details" {
		t.Fatalf("title/desc = %q/%q", panel.TitleText(), panel.Desc())
	}
	if panel.SemanticRole() != RoleInspector || panel.BadgeText() != "9" || panel.HintText() != "hi" {
		t.Fatalf("role/badge/hint mismatch")
	}
	if panel.MinWidthValue() != 12 || panel.MinHeightValue() != 4 || panel.GrowValue() != 3 {
		t.Fatalf("size metadata mismatch")
	}
	if panel.PriorityValue() != 55 {
		t.Fatalf("priority = %d, want 55", panel.PriorityValue())
	}
	if panel.CanFocus() || panel.CanHide() || panel.CanZoom() {
		t.Fatal("flags should all be false")
	}
	if got := panel.Render(PanelContext{}); got != "body" {
		t.Fatalf("render = %q", got)
	}
}

func TestPanelRoleDefaultsPriority(t *testing.T) {
	cases := map[PanelRole]int{
		RolePrimary: 100, RoleNavigation: 80, RoleSecondary: 70,
		RoleInspector: 60, RoleTelemetry: 40, RoleOptional: 20,
	}
	for role, want := range cases {
		ws := NewWorkspace()
		panel := ws.Panel("p", nil).Role(role)
		if got := panel.PriorityValue(); got != want {
			t.Fatalf("role %v priority = %d, want %d", role, got, want)
		}
	}
}

func TestDefaultLayoutPrioritizesHighPriorityPanel(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("low", Text("low")).Role(RoleSecondary).Priority(10).MinWidth(5).MinHeight(3)
	ws.Panel("high", Text("high")).Role(RoleSecondary).Priority(90).MinWidth(5).MinHeight(3)
	solved := ws.Solve(60, 20)
	if solved.Rects["high"].Width <= solved.Rects["low"].Width {
		t.Fatalf("high priority panel not given the larger slot: %+v", solved.Rects)
	}
}

func TestWorkspaceZoomToggle(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("main")
	if !ws.ToggleZoom() || ws.Zoomed() != "main" {
		t.Fatal("ToggleZoom did not zoom main")
	}
	if !ws.ToggleZoom() || ws.Zoomed() != "" {
		t.Fatal("second ToggleZoom did not restore")
	}
}

func TestWorkspaceReflowAndAccessors(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.SetFocusPresentation(FocusPresentation{ActiveBorder: true})
	if !ws.FocusPresentation().ActiveBorder || ws.FocusPresentation().DimInactive {
		t.Fatal("focus presentation not applied")
	}
	ws.Reflow(90, 30)
	if ws.Width() != 90 || ws.Height() != 30 {
		t.Fatalf("size = %dx%d, want 90x30", ws.Width(), ws.Height())
	}
	if len(ws.Panels()) != 3 || len(ws.PanelIDs()) != 3 {
		t.Fatal("panel listing mismatch")
	}
	if _, ok := ws.Lookup("main"); !ok {
		t.Fatal("Lookup failed")
	}
	if !ws.CanFocus("main") {
		t.Fatal("main should be focusable")
	}
	if len(ws.VisiblePanels()) == 0 {
		t.Fatal("VisiblePanels empty")
	}
}

func TestWorkspacePresetListingAndCapture(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.SavePreset("mine")
	names := ws.PresetNames()
	found := false
	for _, name := range names {
		if name == "mine" {
			found = true
		}
	}
	if !found {
		t.Fatalf("saved preset missing from %v", names)
	}
	if len(ws.Presets()) != len(names) {
		t.Fatal("Presets and PresetNames disagree")
	}
}

func TestWorkspaceOptionsSmoke(t *testing.T) {
	ws := NewWorkspace(
		WithFocusPresentation(FocusPresentation{ActiveBorder: true}),
		WithAnimation(AnimationOptions{Enabled: true}),
		WithPresets(Preset{Name: "builtin", Root: HStack(Leaf("a"))}),
		WithGap(2),
		WithHistoryLimit(5),
	)
	ws.Panel("a", Text("a"))
	if len(ws.PresetNames()) != 1 {
		t.Fatalf("preset option not applied: %v", ws.PresetNames())
	}
	if ws.Animation() == nil || !ws.Animation().Enabled() {
		t.Fatal("animation option not applied")
	}
}

func TestTabStackSolverAndActivePanel(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).MinWidth(5).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(5).MinHeight(3)
	ws.Layout(Tabs("a", "b"))
	ws.Focus("a")

	solved := ws.Solve(40, 10)
	if len(solved.Regions) != 1 || !solved.Regions[0].TabStack {
		t.Fatalf("expected one tab-stack region, got %+v", solved.Regions)
	}
	if got := solved.Regions[0].ActivePanel(); got != "a" {
		t.Fatalf("active tab = %q, want a", got)
	}
	if solved.Rects["a"] != solved.Rects["b"] {
		t.Fatalf("tab panels should share a rect: %+v", solved.Rects)
	}

	ws.root = SetActiveTab(ws.root, "b")
	solved = ws.Solve(40, 10)
	if got := solved.Regions[0].ActivePanel(); got != "b" {
		t.Fatalf("active tab after switch = %q, want b", got)
	}
}

func TestUndoRedoAvailability(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Hide("log")
	if !ws.CanUndo() {
		t.Fatal("expected undo available")
	}
	ws.Undo()
	if !ws.CanRedo() {
		t.Fatal("expected redo available after undo")
	}
}

func TestCommandDisplayAndPaletteRender(t *testing.T) {
	command := Command{Category: "Panel", Label: "Focus Editor"}
	if got := command.Display(); got != "Panel · Focus Editor" {
		t.Fatalf("Display = %q", got)
	}
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{})
	ws := newTestWorkspace(t)
	ws.OpenCommandPalette()
	overlay := ws.CommandPalette().Render(renderer, 100, 30)
	if !overlay.Visible || !strings.Contains(overlay.Content, "Focus") {
		t.Fatalf("palette overlay missing content")
	}
	ws.CommandPalette().Update(keyMsg("esc"))
}

func TestPanelPickerRender(t *testing.T) {
	renderer := NewRenderer(CatppuccinMocha, StyleOptions{})
	ws := newTestWorkspace(t)
	ws.OpenPanelPicker()
	overlay := ws.PanelPicker().Render(renderer, 100, 30)
	if !overlay.Visible || !strings.Contains(overlay.Content, "Navigation") {
		t.Fatalf("panel picker overlay missing content")
	}
}

func TestAnimationAccessors(t *testing.T) {
	anim := NewAnimator(AnimationOptions{Enabled: true, ReducedMotion: true, TickRate: time.Second})
	if !anim.ReducedMotion() {
		t.Fatal("ReducedMotion not reported")
	}
	if anim.TickRate() != time.Second {
		t.Fatalf("TickRate = %v", anim.TickRate())
	}
	active := NewAnimator(AnimationOptions{Enabled: true})
	active.Set("x", 1)
	active.Tick()
	if got := active.Ease("x"); got <= 0 || got >= 1 {
		t.Fatalf("Ease = %v, want between 0 and 1", got)
	}
}

func TestDirectionAndDockSideStrings(t *testing.T) {
	if DirLeft.String() != "left" || DirDown.String() != "down" {
		t.Fatal("direction strings wrong")
	}
	if !DirRight.Horizontal() || DirRight.Forward() != true || DirUp.Forward() {
		t.Fatal("direction helpers wrong")
	}
	for side, want := range map[DockSide]string{
		DockLeft: "left", DockRight: "right", DockAbove: "up",
		DockBelow: "down", DockCenter: "stack",
	} {
		if got := side.String(); got != want {
			t.Fatalf("side %d = %q, want %q", side, got, want)
		}
	}
}
