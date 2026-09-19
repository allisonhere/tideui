package tideui

import (
	"strings"
	"testing"
)

func flatSolver(gap int) LayoutSolver {
	return LayoutSolver{
		HGap:      gap,
		VGap:      gap,
		MinWidth:  func(string) int { return 1 },
		MinHeight: func(string) int { return 1 },
		Weight:    func(LayoutNode) float64 { return 1 },
	}
}

func TestSolverNestedSplits(t *testing.T) {
	root := HStack(Leaf("a"), VStack(Leaf("b"), Leaf("c")))
	solved := flatSolver(0).Solve(root, 90, 24)

	if got, want := solved.Rects["a"], (Rect{X: 0, Y: 0, Width: 45, Height: 24}); got != want {
		t.Fatalf("a = %+v, want %+v", got, want)
	}
	if got, want := solved.Rects["b"], (Rect{X: 45, Y: 0, Width: 45, Height: 12}); got != want {
		t.Fatalf("b = %+v, want %+v", got, want)
	}
	if got, want := solved.Rects["c"], (Rect{X: 45, Y: 12, Width: 45, Height: 12}); got != want {
		t.Fatalf("c = %+v, want %+v", got, want)
	}
	if solved.Overflow {
		t.Fatal("unexpected overflow")
	}
}

func TestSolverWeightedShare(t *testing.T) {
	root := HStack(Leaf("a"), Weighted(Leaf("b"), 3))
	solved := flatSolver(0).Solve(root, 80, 10)
	if got, want := solved.Rects["a"].Width, 20; got != want {
		t.Fatalf("a width = %d, want %d", got, want)
	}
	if got, want := solved.Rects["b"].Width, 60; got != want {
		t.Fatalf("b width = %d, want %d", got, want)
	}
}

func TestSolverGapReducesContent(t *testing.T) {
	root := HStack(Leaf("a"), Leaf("b"))
	solved := flatSolver(1).Solve(root, 21, 5)
	a, b := solved.Rects["a"], solved.Rects["b"]
	if a.Width+b.Width != 20 {
		t.Fatalf("widths %d + %d = %d, want 20", a.Width, b.Width, a.Width+b.Width)
	}
	if b.X != a.X+a.Width+1 {
		t.Fatalf("gap = %d, want 1", b.X-(a.X+a.Width))
	}
}

func TestSolverHonoursMinimums(t *testing.T) {
	solver := LayoutSolver{
		HGap:      0,
		VGap:      0,
		MinWidth:  func(id string) int { return map[string]int{"a": 30, "b": 10}[id] },
		MinHeight: func(string) int { return 1 },
		Weight:    func(LayoutNode) float64 { return 1 },
	}
	solved := solver.Solve(HStack(Leaf("a"), Leaf("b")), 50, 8)
	if got := solved.Rects["a"].Width; got != 30 {
		t.Fatalf("a width = %d, want 30 (its minimum)", got)
	}
	if got := solved.Rects["b"].Width; got != 20 {
		t.Fatalf("b width = %d, want 20", got)
	}
}

func TestSolverOverflowAtTinySizesNeverPanics(t *testing.T) {
	root := HStack(Leaf("a"), VStack(Leaf("b"), Leaf("c"), Tabs("d", "e")))
	for width := 0; width <= 6; width++ {
		for height := 0; height <= 4; height++ {
			solved := flatSolver(1).Solve(root, width, height)
			for id, rect := range solved.Rects {
				if rect.Width < 0 || rect.Height < 0 {
					t.Fatalf("%dx%d: %s negative rect %+v", width, height, id, rect)
				}
				if rect.X < 0 || rect.Y < 0 {
					t.Fatalf("%dx%d: %s out of bounds %+v", width, height, id, rect)
				}
			}
		}
	}
}

func TestDistributeIsDeterministicAndSums(t *testing.T) {
	sizes, overflow := distribute(101, []int{1, 1, 1}, []float64{1, 2, 3})
	if overflow {
		t.Fatal("unexpected overflow")
	}
	sum := 0
	for _, size := range sizes {
		sum += size
	}
	if sum != 101 {
		t.Fatalf("sum = %d, want 101", sum)
	}
	again, _ := distribute(101, []int{1, 1, 1}, []float64{1, 2, 3})
	for i := range sizes {
		if sizes[i] != again[i] {
			t.Fatalf("nondeterministic at %d: %d vs %d", i, sizes[i], again[i])
		}
	}
}

func TestRemovePanelClosesGap(t *testing.T) {
	root := HStack(Leaf("a"), Leaf("b"), Leaf("c"))
	next, removed := RemovePanel(root, "b")
	if !removed {
		t.Fatal("expected b to be removed")
	}
	if LayoutContainsPanel(next, "b") {
		t.Fatal("b still present")
	}
	ids := LayoutPanelIDs(next)
	if strings.Join(ids, ",") != "a,c" {
		t.Fatalf("ids = %v, want [a c]", ids)
	}
	solved := flatSolver(0).Solve(next, 40, 10)
	if solved.Rects["a"].Width+solved.Rects["c"].Width != 40 {
		t.Fatalf("gap not closed: %+v", solved.Rects)
	}
}

func TestRemovePanelCollapsesOnlyChildSplit(t *testing.T) {
	root := HStack(Leaf("a"), VStack(Leaf("b")))
	next, _ := RemovePanel(root, "b")
	// The now-empty VStack must collapse, leaving a single leaf rather than a
	// split wrapping one child.
	if _, ok := next.(*LeafNode); !ok {
		t.Fatalf("expected collapsed leaf, got %T", next)
	}
}

func TestSwapPanelsExchangesPositions(t *testing.T) {
	root := HStack(Leaf("a"), Leaf("b"))
	next := SwapPanels(root, "a", "b")
	if got := LayoutString(next); !strings.Contains(got, "leaf b") {
		t.Fatalf("unexpected tree: %s", got)
	}
	solved := flatSolver(0).Solve(next, 20, 4)
	if solved.Rects["b"].X >= solved.Rects["a"].X {
		t.Fatalf("a and b not swapped: %+v", solved.Rects)
	}
}

func TestMovePanelDocksOnEachSide(t *testing.T) {
	base := HStack(Leaf("a"), Leaf("b"))
	cases := []struct {
		side    DockSide
		firstID string
	}{
		{DockLeft, "b"},
		{DockRight, "a"},
	}
	for _, tc := range cases {
		next := MovePanel(base, "b", "a", tc.side)
		solved := flatSolver(0).Solve(next, 20, 4)
		if firstID := panelAtX(solved, 0); firstID != tc.firstID {
			t.Fatalf("side %v: leftmost = %q, want %q", tc.side, firstID, tc.firstID)
		}
	}

	stacked := VStack(Leaf("a"), Leaf("b"))
	next := MovePanel(stacked, "b", "a", DockAbove)
	solved := flatSolver(0).Solve(next, 20, 10)
	if top := panelAtY(solved, 0); top != "b" {
		t.Fatalf("above: top = %q, want b", top)
	}

	merged := MovePanel(base, "b", "a", DockCenter)
	if !strings.Contains(LayoutString(merged), "tabs") {
		t.Fatalf("center dock did not create a tab stack: %s", LayoutString(merged))
	}
}

func TestMovePanelIsNoOpForMissingPanels(t *testing.T) {
	root := HStack(Leaf("a"), Leaf("b"))
	if got := LayoutString(MovePanel(root, "x", "a", DockLeft)); got != LayoutString(root) {
		t.Fatalf("missing moving panel changed tree: %s", got)
	}
	if got := LayoutString(MovePanel(root, "a", "x", DockLeft)); got != LayoutString(root) {
		t.Fatalf("missing target changed tree: %s", got)
	}
	if got := LayoutString(MovePanel(root, "a", "a", DockLeft)); got != LayoutString(root) {
		t.Fatalf("self move changed tree: %s", got)
	}
}

func TestNormalizeFlattensAndDropsEmpty(t *testing.T) {
	nested := HStack(Leaf("a"), HStack(Leaf("b"), Leaf("c")))
	normalized := NormalizeLayout(nested)
	if got, want := len(LayoutPanelIDs(normalized)), 3; got != want {
		t.Fatalf("panel count = %d, want %d", got, want)
	}
	empty := NormalizeLayout(Tabs("a"))
	if _, ok := empty.(*LeafNode); !ok {
		t.Fatalf("single-tab stack should normalize to a leaf, got %T", empty)
	}
}

func TestSanitizeLayoutDropsDuplicatePanels(t *testing.T) {
	root := HStack(Leaf("a"), VStack(Leaf("b"), Leaf("a"), Tabs("a", "b", "c")))
	got := SanitizeLayout(root)
	ids := LayoutPanelIDs(got)
	if strings.Join(ids, ",") != "a,b,c" {
		t.Fatalf("ids = %v, want [a b c] once each", ids)
	}
}

func TestWorkspaceLayoutSanitizesDuplicates(t *testing.T) {
	ws := NewWorkspace()
	ws.Panel("a", Text("a"))
	ws.Layout(HStack(Leaf("a"), Leaf("a")))
	if ids := LayoutPanelIDs(ws.RootLayout()); len(ids) != 1 {
		t.Fatalf("layout kept duplicate panels: %v", ids)
	}
}

func TestSetActiveTab(t *testing.T) {
	root := Tabs("a", "b", "c")
	next := SetActiveTab(root, "c")
	stack, ok := next.(*TabStackNode)
	if !ok {
		t.Fatalf("expected tab stack, got %T", next)
	}
	if stack.Active != 2 {
		t.Fatalf("active = %d, want 2", stack.Active)
	}
}

func panelAtX(solved SolvedLayout, x int) string {
	for _, region := range solved.Regions {
		if region.Rect.Contains(x, region.Rect.Y) {
			return region.ActivePanel()
		}
	}
	return ""
}

func panelAtY(solved SolvedLayout, y int) string {
	for _, region := range solved.Regions {
		if region.Rect.Contains(region.Rect.X, y) {
			return region.ActivePanel()
		}
	}
	return ""
}

func TestMovePanelDocksIntoMultiPaneRow(t *testing.T) {
	// Three stacked rows, the middle one split into two columns.
	root := VStack(
		HStack(Leaf("a"), Leaf("b")),
		HStack(Leaf("c"), Leaf("d")),
		HStack(Leaf("e"), Leaf("f")),
	)
	// Dock "a" below "c": it must land in c's column, not skip to the bottom.
	next := MovePanel(root, "a", "c", DockBelow)
	solved := flatSolver(0).Solve(next, 40, 30)

	a, c, e := solved.Rects["a"], solved.Rects["c"], solved.Rects["e"]
	if a.Y <= c.Y {
		t.Fatalf("a not below c: a=%+v c=%+v", a, c)
	}
	if a.Y >= e.Y {
		t.Fatalf("a skipped past the row: a=%+v e=%+v", a, e)
	}
	if a.X != c.X {
		t.Fatalf("a not in c's column: a=%+v c=%+v", a, c)
	}
	if a.Width >= 40 {
		t.Fatalf("a spans the full width instead of a column: %+v", a)
	}
	if d := solved.Rects["d"]; d.X < a.X+a.Width {
		t.Fatalf("d overlaps a's column: %+v", solved.Rects)
	}
}

func TestMovePanelDocksBesideTargetInsideColumn(t *testing.T) {
	// A single column: docking right of a pane must create a row at its slot.
	root := VStack(Leaf("x"), Leaf("y"))
	next := MovePanel(root, "y", "x", DockRight)
	solved := flatSolver(0).Solve(next, 40, 20)
	x, y := solved.Rects["x"], solved.Rects["y"]
	if y.X <= x.X {
		t.Fatalf("y not right of x: x=%+v y=%+v", x, y)
	}
	if y.Y != x.Y || y.Height != x.Height {
		t.Fatalf("docking beside should preserve the row: x=%+v y=%+v", x, y)
	}
}

func TestMovePanelRowDockSharesTargetsRow(t *testing.T) {
	root := VStack(
		HStack(Leaf("a"), Leaf("b")),
		HStack(Leaf("c"), Leaf("d")),
	)
	next := MovePanel(root, "a", "c", DockRowBelow)
	solved := flatSolver(0).Solve(next, 40, 20)
	a, c, d := solved.Rects["a"], solved.Rects["c"], solved.Rects["d"]
	if a.Y != c.Y {
		t.Fatalf("a did not join c's row: a=%+v c=%+v", a, c)
	}
	if a.X <= c.X {
		t.Fatalf("a not placed after c: a=%+v c=%+v", a, c)
	}
	if d.X <= a.X {
		t.Fatalf("d not pushed after a: a=%+v d=%+v", a, d)
	}
	if a.Height != c.Height {
		t.Fatalf("row heights diverged: a=%+v c=%+v", a, c)
	}
}

func TestMovePanelRowDockIntoOnePaneRowMakesTwo(t *testing.T) {
	// Two stacked panels: moving one onto the other shares one row of two.
	root := VStack(Leaf("a"), Leaf("b"))
	next := MovePanel(root, "a", "b", DockRowBelow)
	solved := flatSolver(0).Solve(next, 40, 20)
	a, b := solved.Rects["a"], solved.Rects["b"]
	if a.Y != b.Y {
		t.Fatalf("a should share b's row: a=%+v b=%+v", a, b)
	}
	if a.Width == 40 || b.Width == 40 {
		t.Fatalf("row did not become two columns: a=%+v b=%+v", a, b)
	}
	if a.X == b.X {
		t.Fatalf("panels overlap: a=%+v b=%+v", a, b)
	}
}

func TestMovePanelDockBelowStillStacks(t *testing.T) {
	// Directional DockBelow (used by responsive reflow) must stack, not share.
	root := HStack(Leaf("main"), Leaf("log"))
	next := MovePanel(root, "log", "main", DockBelow)
	solved := flatSolver(0).Solve(next, 40, 20)
	if solved.Rects["log"].Y <= solved.Rects["main"].Y {
		t.Fatalf("DockBelow should stack: main=%+v log=%+v", solved.Rects["main"], solved.Rects["log"])
	}
}
