package tideui

import "testing"

func TestWorkspaceResizeIsUndoableAndRedoable(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).MinWidth(5).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(5).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Focus("a")
	ws.Solve(80, 10)

	before := ws.Solve(80, 10).Rects["a"].Width
	if !ws.ResizeEdge(DirRight) {
		t.Fatal("ResizeEdge failed")
	}
	after := ws.Solve(80, 10).Rects["a"].Width
	if after <= before {
		t.Fatalf("resize width = %d, want greater than %d", after, before)
	}
	if !ws.CanUndo() {
		t.Fatal("weight-only resize was not recorded")
	}

	if !ws.Undo() {
		t.Fatal("Undo failed")
	}
	if got := ws.Solve(80, 10).Rects["a"].Width; got != before {
		t.Fatalf("undo width = %d, want %d", got, before)
	}
	if !ws.Redo() {
		t.Fatal("Redo failed")
	}
	if got := ws.Solve(80, 10).Rects["a"].Width; got != after {
		t.Fatalf("redo width = %d, want %d", got, after)
	}
}

func TestSameSnapshotIncludesStructuralLayoutState(t *testing.T) {
	base := workspaceSnapshot{root: HStack(Leaf("a"), Weighted(Tabs("b", "c"), 2))}
	if !sameSnapshot(base, base.clone()) {
		t.Fatal("identical snapshots were not equal")
	}

	weight := base.clone()
	weight.root.(*SplitNode).Children[0].(*LeafNode).Weight = 2
	if sameSnapshot(base, weight) {
		t.Fatal("weight-only difference was ignored")
	}

	active := base.clone()
	active.root.(*SplitNode).Children[1].(*TabStackNode).Active = 1
	if sameSnapshot(base, active) {
		t.Fatal("active-tab difference was ignored")
	}
}

func TestLayoutHistoryDeduplicatesIdenticalSnapshots(t *testing.T) {
	history := NewLayoutHistory(8)
	state := workspaceSnapshot{root: HStack(Leaf("a"), Leaf("b"))}
	history.Reset(state)
	history.Push(state.clone())
	if got := history.Len(); got != 1 {
		t.Fatalf("history length = %d, want 1", got)
	}
}
