package tideui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestWorkspace(t *testing.T) *Workspace {
	t.Helper()
	ws := NewWorkspace(WithGap(0))
	ws.Panel("nav", Text("nav")).Title("Navigation").Role(RoleNavigation).MinWidth(10).MinHeight(3)
	ws.Panel("main", Text("main")).Title("Main").Role(RolePrimary).MinWidth(20).MinHeight(3)
	ws.Panel("log", Text("log")).Title("Log").Role(RoleTelemetry).MinWidth(10).MinHeight(3)
	ws.Layout(HStack(Leaf("nav"), Leaf("main"), Leaf("log")))
	ws.Focus("main")
	ws.Solve(80, 24)
	return ws
}

// space enters the focused pane - any pane - so its own keys work while the
// tiled layout stays on screen, and esc hands the keyboard back.
func TestSpaceEntersTheFocusedPane(t *testing.T) {
	ws := newTestWorkspace(t)
	if !ws.HandleKey(tea.KeyMsg{Type: tea.KeySpace}) {
		t.Fatal("space was not taken by the workspace")
	}
	if got := ws.EnteredPane(); got != "main" {
		t.Fatalf("entered = %q, want the focused pane", got)
	}
	if !ws.HandleKey(tea.KeyMsg{Type: tea.KeyEsc}) {
		t.Fatal("esc was not taken while a pane had the keyboard")
	}
	if got := ws.EnteredPane(); got != "" {
		t.Fatalf("entered = %q after esc, want none", got)
	}
}

// Moving focus leaves the pane behind: the keyboard belongs to where you are,
// and the next space enters that one. All four ways of moving focus come
// through the same door, so all four are checked.
func TestFocusChangeLeavesTheEnteredPane(t *testing.T) {
	for _, move := range []struct {
		name string
		run  func(*Workspace)
	}{
		{"focus", func(ws *Workspace) { ws.Focus("nav") }},
		{"tab", func(ws *Workspace) { ws.FocusNext() }},
		{"shift+tab", func(ws *Workspace) { ws.FocusPrev() }},
		{"arrows", func(ws *Workspace) { ws.FocusDirection(DirLeft) }},
	} {
		t.Run(move.name, func(t *testing.T) {
			ws := newTestWorkspace(t)
			ws.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
			if ws.EnteredPane() == "" {
				t.Fatal("nothing was entered to leave behind")
			}
			move.run(ws)
			if got := ws.EnteredPane(); got != "" {
				t.Fatalf("entered = %q after %s, want none", got, move.name)
			}
		})
	}
}

// A pane that owns the screen has the keyboard too, so anything asking "who
// reads the keys" has one answer rather than two.
func TestZoomedPaneHasTheKeyboard(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Zoom("main")
	if !ws.PaneHasKeyboard("main") {
		t.Fatal("a zoomed pane does not report the keyboard")
	}
	ws.Unzoom()
	if ws.PaneHasKeyboard("main") {
		t.Fatal("an unzoomed, unentered pane reports the keyboard")
	}
	ws.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	if !ws.PaneHasKeyboard(ws.Focused()) {
		t.Fatal("an entered pane does not report the keyboard")
	}
}

// A pane that binds space itself keeps it: entering is the fallback, not the
// first claim on the key, so a panel cannot be broken by the workspace taking
// the key it uses.
func TestASpaceBoundActionStillFires(t *testing.T) {
	ws := newTestWorkspace(t)
	ran := ""
	ws.Panel("main", Text("main")).Actions(Action("toggle", " ", func(*Workspace) { ran = "toggle" }))
	ws.Focus("main")
	if !ws.HandleKey(tea.KeyMsg{Type: tea.KeySpace}) {
		t.Fatal("space was not taken")
	}
	if ran != "toggle" {
		t.Fatalf("the space-bound action did not run (entered = %q)", ws.EnteredPane())
	}
	if ws.EnteredPane() != "" {
		t.Fatal("space both fired the action and entered the pane")
	}
}

func TestWorkspaceRegistersPanelsOnce(t *testing.T) {
	ws := NewWorkspace()
	first := ws.Panel("a", Text("one")).Title("A")
	second := ws.Panel("a", Text("two"))
	if first != second {
		t.Fatal("re-registering a duplicate id returned a different panel")
	}
	if got := len(ws.PanelIDs()); got != 1 {
		t.Fatalf("panel ids = %d, want 1", got)
	}
	if got := first.Render(PanelContext{}); got != "two" {
		t.Fatalf("view was not updated on re-register: %q", got)
	}
}

func TestWorkspaceDefaultLayoutUsesRoles(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("nav", Text("nav")).Role(RoleNavigation)
	ws.Panel("primary", Text("p")).Role(RolePrimary)
	ws.Panel("log", Text("log")).Role(RoleTelemetry)
	solved := ws.Solve(120, 30)
	for _, id := range []string{"nav", "primary", "log"} {
		if _, ok := solved.Rects[id]; !ok {
			t.Fatalf("default layout missing %q", id)
		}
	}
}

func TestWorkspaceHideShow(t *testing.T) {
	ws := newTestWorkspace(t)
	if !ws.Hide("log") {
		t.Fatal("Hide returned false")
	}
	if !ws.Hidden("log") {
		t.Fatal("log not marked hidden")
	}
	solved := ws.Solve(80, 24)
	if _, ok := solved.Rects["log"]; ok {
		t.Fatal("hidden panel still laid out")
	}
	if !ws.Show("log") {
		t.Fatal("Show returned false")
	}
	if _, ok := ws.Solve(80, 24).Rects["log"]; !ok {
		t.Fatal("shown panel not laid out")
	}
}

func TestWorkspaceZoomPreservesAndRestoresLayout(t *testing.T) {
	ws := newTestWorkspace(t)
	base := ws.Solve(100, 30)
	baseNav := base.Rects["nav"]

	if !ws.Zoom("main") {
		t.Fatal("Zoom returned false")
	}
	zoomed := ws.Solve(100, 30)
	if len(zoomed.Regions) != 1 || zoomed.Regions[0].ActivePanel() != "main" {
		t.Fatalf("zoomed solve = %+v, want only main", zoomed.Regions)
	}
	if zoomed.Rects["main"].Width != 100 || zoomed.Rects["main"].Height != 30 {
		t.Fatalf("zoomed panel not full area: %+v", zoomed.Rects["main"])
	}
	if !ws.Unzoom() {
		t.Fatal("Unzoom returned false")
	}
	restored := ws.Solve(100, 30)
	if restored.Rects["nav"] != baseNav {
		t.Fatalf("nav rect changed after zoom restore: %+v vs %+v", restored.Rects["nav"], baseNav)
	}
}

func TestWorkspaceZoomIsNotPersisted(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Zoom("main")
	data, err := ws.PersistedJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "zoom") {
		t.Fatalf("persisted layout leaked zoom state: %s", data)
	}
}

func TestWorkspacePeekDoesNotChangeLayout(t *testing.T) {
	ws := newTestWorkspace(t)
	before := ws.Solve(80, 24)
	if !ws.Peek("log") {
		t.Fatal("Peek returned false")
	}
	if ws.Peeked() != "log" {
		t.Fatalf("peeked = %q", ws.Peeked())
	}
	after := ws.Solve(80, 24)
	if after.Rects["log"] != before.Rects["log"] {
		t.Fatalf("peek changed layout: %+v vs %+v", after.Rects["log"], before.Rects["log"])
	}
	ws.Unpeek()
	if ws.Peeked() != "" {
		t.Fatal("peek not dismissed")
	}
}

func TestWorkspaceFocusTraversal(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	ws.FocusNext()
	if got := ws.Focused(); got != "main" {
		t.Fatalf("focus next = %q, want main", got)
	}
	ws.FocusNext()
	if got := ws.Focused(); got != "log" {
		t.Fatalf("focus next = %q, want log", got)
	}
	ws.FocusNext()
	if got := ws.Focused(); got != "nav" {
		t.Fatalf("focus wrapped = %q, want nav", got)
	}
	ws.FocusPrev()
	if got := ws.Focused(); got != "log" {
		t.Fatalf("focus prev wrapped = %q, want log", got)
	}
}

func TestWorkspaceFocusDirection(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	if !ws.FocusDirection(DirRight) {
		t.Fatal("right focus failed")
	}
	if got := ws.Focused(); got != "main" {
		t.Fatalf("right focus = %q, want main", got)
	}
	if !ws.FocusDirection(DirRight) {
		t.Fatal("second right focus failed")
	}
	if got := ws.Focused(); got != "log" {
		t.Fatalf("second right focus = %q, want log", got)
	}
	if ws.FocusDirection(DirRight) {
		t.Fatal("right focus should not wrap")
	}
	ws.Focus("log")
	if !ws.FocusDirection(DirLeft) || ws.Focused() != "main" {
		t.Fatalf("left focus = %q, want main", ws.Focused())
	}
}

func TestWorkspaceArrangeMoveIsLiveAndClosesGap(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	ws.Solve(80, 24)
	if !ws.EnterArrange() {
		t.Fatal("EnterArrange failed")
	}
	if !ws.ArrangeMove(DirRight) {
		t.Fatal("ArrangeMove failed")
	}
	// The move is applied immediately; there is no separate preview state.
	solved := ws.Solve(80, 24)
	if solved.Rects["nav"].X <= solved.Rects["main"].X {
		t.Fatalf("nav did not move right of main: %+v", solved.Rects)
	}
	if ws.ArrangeCursor() != "main" {
		t.Fatalf("arrange cursor = %q, want main", ws.ArrangeCursor())
	}
	if !ws.CanUndo() {
		t.Fatal("live move should be recorded in history")
	}
	// Dropping just leaves the mode; the layout is unchanged.
	before := ws.Solve(80, 24).Rects["nav"]
	if !ws.ArrangeDrop() {
		t.Fatal("ArrangeDrop failed")
	}
	if ws.Arranging() {
		t.Fatal("still arranging after drop")
	}
	if got := ws.Solve(80, 24).Rects["nav"]; got != before {
		t.Fatalf("drop changed the layout: %+v -> %+v", before, got)
	}
}

func TestWorkspaceArrangeMerge(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	ws.Solve(80, 24)
	ws.EnterArrange()
	ws.ArrangeMove(DirRight)
	if !ws.ArrangeMerge() {
		t.Fatal("ArrangeMerge failed")
	}
	if !strings.Contains(LayoutString(ws.RootLayout()), "tabs") {
		t.Fatalf("merge did not create tabs: %s", LayoutString(ws.RootLayout()))
	}
}

func TestWorkspaceResizeGrowsTowardNeighbour(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	before := ws.Solve(80, 24)
	if !ws.ResizeEdge(DirRight) {
		t.Fatal("ResizeEdge(DirRight) failed")
	}
	after := ws.Solve(80, 24)
	if after.Rects["nav"].Width <= before.Rects["nav"].Width {
		t.Fatalf("nav did not grow: %d -> %d", before.Rects["nav"].Width, after.Rects["nav"].Width)
	}
	if after.Rects["main"].Width >= before.Rects["main"].Width {
		t.Fatalf("main did not give space: %d -> %d", before.Rects["main"].Width, after.Rects["main"].Width)
	}
}

func TestWorkspaceResizeShrinksAtEdge(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav") // leftmost pane: no left neighbour
	before := ws.Solve(80, 24).Rects["nav"].Width
	if !ws.ResizeEdge(DirLeft) {
		t.Fatal("ResizeEdge(DirLeft) failed")
	}
	after := ws.Solve(80, 24).Rects["nav"].Width
	if after >= before {
		t.Fatalf("left at the edge should shrink nav: %d -> %d", before, after)
	}
}

func TestWorkspaceResizeMiddleGrowsBothWays(t *testing.T) {
	build := func() *Workspace {
		ws := NewWorkspace(WithGap(0))
		ws.Panel("a", Text("a")).MinWidth(5).MinHeight(3)
		ws.Panel("b", Text("b")).MinWidth(5).MinHeight(3)
		ws.Panel("c", Text("c")).MinWidth(5).MinHeight(3)
		ws.Layout(HStack(Leaf("a"), Leaf("b"), Leaf("c")))
		ws.Focus("b")
		ws.Solve(90, 20)
		return ws
	}
	base := build().Solve(90, 20).Rects["b"].Width

	right := build()
	if !right.ResizeEdge(DirRight) {
		t.Fatal("ResizeEdge(DirRight) failed on a middle pane")
	}
	if got := right.Solve(90, 20).Rects["b"].Width; got <= base {
		t.Fatalf("middle pane did not grow right: %d -> %d", base, got)
	}

	left := build()
	if !left.ResizeEdge(DirLeft) {
		t.Fatal("ResizeEdge(DirLeft) failed on a middle pane")
	}
	if got := left.Solve(90, 20).Rects["b"].Width; got <= base {
		t.Fatalf("middle pane did not grow left: %d -> %d", base, got)
	}
}

func TestWorkspaceResizeVertical(t *testing.T) {
	build := func() *Workspace {
		ws := NewWorkspace(WithGap(0))
		ws.Panel("top", Text("top")).MinWidth(5).MinHeight(3)
		ws.Panel("bottom", Text("bottom")).MinWidth(5).MinHeight(3)
		ws.Layout(VStack(Leaf("top"), Leaf("bottom")))
		ws.Focus("top")
		ws.Solve(40, 24)
		return ws
	}
	base := build().Solve(40, 24).Rects["top"].Height

	down := build()
	if !down.ResizeEdge(DirDown) {
		t.Fatal("ResizeEdge(DirDown) failed")
	}
	if got := down.Solve(40, 24).Rects["top"].Height; got <= base {
		t.Fatalf("down did not grow top: %d -> %d", base, got)
	}

	up := build() // no neighbour above: the bottom edge moves up and top shrinks
	if !up.ResizeEdge(DirUp) {
		t.Fatal("ResizeEdge(DirUp) failed")
	}
	if got := up.Solve(40, 24).Rects["top"].Height; got >= base {
		t.Fatalf("up at the top edge should shrink top: %d -> %d", base, got)
	}
}

func TestWorkspaceResizeRespectsMinimums(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).MinWidth(20).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(20).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Focus("a")
	ws.Solve(80, 10)
	for i := 0; i < 20; i++ {
		ws.ResizeWidth(true)
	}
	solved := ws.Solve(80, 10)
	if solved.Rects["b"].Width < 20 {
		t.Fatalf("b width = %d, want >= its minimum 20", solved.Rects["b"].Width)
	}
	if solved.Rects["a"].Width < 20 {
		t.Fatalf("a width = %d, want >= its minimum 20", solved.Rects["a"].Width)
	}
}

func TestWorkspaceResizeReportsPercentage(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	ws.Solve(80, 24)
	if !ws.ResizeEdge(DirRight) {
		t.Fatal("ResizeEdge failed")
	}
	status := ws.ResizeStatus()
	if !strings.Contains(status, "width") {
		t.Fatalf("resize status = %q, want a width percentage", status)
	}
}

func TestResizeGrowShrinkAPI(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	base := ws.Solve(80, 24).Rects["nav"].Width
	if !ws.ResizeGrow(0) {
		t.Fatal("ResizeGrow failed")
	}
	grown := ws.Solve(80, 24).Rects["nav"].Width
	if grown <= base {
		t.Fatalf("ResizeGrow did not grow: %d -> %d", base, grown)
	}
	if !ws.ResizeShrink(0) {
		t.Fatal("ResizeShrink failed")
	}
	shrunk := ws.Solve(80, 24).Rects["nav"].Width
	if shrunk >= grown {
		t.Fatalf("ResizeShrink did not shrink: %d -> %d", grown, shrunk)
	}
}

func TestHandleKeyResizesWithShiftArrows(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	ws.Solve(80, 24)
	before := ws.Solve(80, 24).Rects["nav"].Width
	if !ws.HandleKey(keyMsg("shift+right")) {
		t.Fatal("shift+right was not handled")
	}
	if got := ws.Solve(80, 24).Rects["nav"].Width; got <= before {
		t.Fatalf("shift+right did not grow nav: %d -> %d", before, got)
	}
}

func TestHandleKeyResizeCtrlAlias(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.Focus("nav")
	ws.Solve(80, 24)
	before := ws.Solve(80, 24).Rects["nav"].Width
	if !ws.HandleKey(keyMsg("ctrl+right")) {
		t.Fatal("ctrl+right was not handled")
	}
	if got := ws.Solve(80, 24).Rects["nav"].Width; got <= before {
		t.Fatalf("ctrl+right did not grow nav: %d -> %d", before, got)
	}
}

func TestWorkspaceGapsArePerAxis(t *testing.T) {
	// Rows flush, columns guttered.
	ws := NewWorkspace(WithGaps(1, 0))
	ws.Panel("top", Text("t")).MinWidth(4).MinHeight(3)
	ws.Panel("bottom", Text("b")).MinWidth(4).MinHeight(3)
	ws.Layout(VStack(Leaf("top"), Leaf("bottom")))
	solved := ws.Solve(40, 20)
	top, bottom := solved.Rects["top"], solved.Rects["bottom"]
	if bottom.Y != top.Y+top.Height {
		t.Fatalf("vertical gap should be zero: top=%+v bottom=%+v", top, bottom)
	}

	ws2 := NewWorkspace(WithGaps(1, 0))
	ws2.Panel("left", Text("l")).MinWidth(4).MinHeight(3)
	ws2.Panel("right", Text("r")).MinWidth(4).MinHeight(3)
	ws2.Layout(HStack(Leaf("left"), Leaf("right")))
	solved2 := ws2.Solve(40, 10)
	left, right := solved2.Rects["left"], solved2.Rects["right"]
	if right.X != left.X+left.Width+1 {
		t.Fatalf("horizontal gap should be 1: left=%+v right=%+v", left, right)
	}
}

func TestWorkspaceArrangeDownJoinsTargetRow(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).MinWidth(4).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(4).MinHeight(3)
	ws.Panel("c", Text("c")).MinWidth(4).MinHeight(3)
	ws.Panel("d", Text("d")).MinWidth(4).MinHeight(3)
	ws.Layout(VStack(HStack(Leaf("a"), Leaf("b")), HStack(Leaf("c"), Leaf("d"))))
	ws.Focus("a")
	ws.Solve(40, 20)
	ws.EnterArrange()
	if !ws.ArrangeMove(DirDown) {
		t.Fatal("ArrangeMove(DirDown) failed")
	}
	solved := ws.Solve(40, 20)
	if solved.Rects["a"].Y != solved.Rects["c"].Y {
		t.Fatalf("down should join c's row: a=%+v c=%+v", solved.Rects["a"], solved.Rects["c"])
	}
	if solved.Rects["a"].X <= solved.Rects["c"].X {
		t.Fatalf("a not placed beside c: a=%+v c=%+v", solved.Rects["a"], solved.Rects["c"])
	}
}

// Removing a panel takes it out of the workspace entirely, tree included, so an
// uninstalled plugin leaves no empty split behind.
func TestRemovePanelRemovesItEntirely(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).MinWidth(4).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(4).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Solve(40, 10)

	if !ws.RemovePanel("a") {
		t.Fatal("RemovePanel reported nothing removed")
	}
	if _, ok := ws.Lookup("a"); ok {
		t.Fatal("a is still registered")
	}
	solved := ws.Solve(40, 10)
	if _, ok := solved.Rects["a"]; ok {
		t.Fatal("a is still in the layout")
	}
	if _, ok := solved.Rects["b"]; !ok {
		t.Fatal("removing a removed b too")
	}
}

// A panel declared hidden stays hidden in a preset that does not name it, so a
// plugin does not drop into every preset just by being installed. It can still
// be enabled, and once shown it stays shown.
func TestPresetRespectsStartsHidden(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).MinWidth(4).MinHeight(3)
	ws.Panel("plugin", Text("p")).MinWidth(4).MinHeight(3).Hide()
	ws.AddPreset("One", HStack(Leaf("a")))

	ws.ApplyPreset("One")
	if ws.Hidden("a") {
		t.Fatal("a placed panel should be visible")
	}
	if !ws.Hidden("plugin") {
		t.Fatal("a panel declared hidden should stay hidden in a preset that does not name it")
	}
	if !ws.Show("plugin") {
		t.Fatal("a hidden panel should be showable")
	}
	if ws.Hidden("plugin") {
		t.Fatal("show did not reveal the panel")
	}
}

// Revealing a panel gives it focus: a panel that takes typing (a calculator)
// would otherwise get nothing while the previously focused panel kept the
// keyboard.
func TestShowFocusesTheRevealedPanel(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).MinWidth(4).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(4).MinHeight(3).Hide()
	ws.Layout(HStack(Leaf("a")))
	ws.Focus("a")
	ws.Solve(40, 10)

	if !ws.Show("b") {
		t.Fatal("Show reported nothing revealed")
	}
	if got := ws.Focused(); got != "b" {
		t.Fatalf("focus after Show = %q, want b", got)
	}
}
