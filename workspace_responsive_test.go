package tideui

import "testing"

func adaptiveWorkspace(t *testing.T) *Workspace {
	t.Helper()
	ws := NewWorkspace(WithAdaptiveLayout(), WithGap(0))
	ws.Panel("nav", Text("nav")).Role(RoleNavigation).MinWidth(8).MinHeight(3)
	ws.Panel("main", Text("main")).Role(RolePrimary).MinWidth(20).MinHeight(3)
	ws.Panel("log", Text("log")).Role(RoleTelemetry).MinWidth(8).MinHeight(3)
	ws.Panel("insp", Text("insp")).Role(RoleInspector).MinWidth(8).MinHeight(3)
	ws.Layout(HStack(Leaf("nav"), Leaf("main"), Leaf("log"), Leaf("insp")))
	return ws
}

func TestResponsiveHideBelowWithHysteresis(t *testing.T) {
	ws := NewWorkspace(WithAdaptiveLayout(), WithGap(0))
	ws.Panel("main", Text("main")).Role(RolePrimary).MinWidth(20)
	ws.Panel("log", Text("log")).Role(RoleTelemetry).MinWidth(8).HideBelow(100)
	ws.Layout(HStack(Leaf("main"), Leaf("log")))

	ws.Solve(90, 24)
	if ws.IsVisible("log") {
		t.Fatal("log should be hidden at width 90")
	}
	ws.Solve(100, 24)
	if ws.IsVisible("log") {
		t.Fatal("hysteresis should keep log hidden at exactly 100")
	}
	ws.Solve(101, 24)
	if ws.IsVisible("log") {
		t.Fatal("hysteresis should keep log hidden at 101")
	}
	ws.Solve(103, 24)
	if !ws.IsVisible("log") {
		t.Fatal("log should return once width clears threshold + hysteresis")
	}
}

func TestResponsiveStackBelowMergesTabs(t *testing.T) {
	ws := NewWorkspace(WithAdaptiveLayout(), WithGap(0))
	ws.Panel("main", Text("main")).Role(RolePrimary).MinWidth(20)
	ws.Panel("log", Text("log")).Role(RoleTelemetry).MinWidth(8).StackBelow(120, "main")
	ws.Layout(HStack(Leaf("main"), Leaf("log")))

	ws.Solve(100, 24)
	tree := ws.SolvedTree()
	if !containsTabStack(tree) {
		t.Fatalf("expected a tab stack, got %s", LayoutString(tree))
	}
	if !LayoutContainsPanel(tree, "log") {
		t.Fatal("stacked panel missing from tree")
	}
}

func TestResponsiveMoveBelowRepositions(t *testing.T) {
	ws := NewWorkspace(WithAdaptiveLayout(), WithGap(0))
	ws.Panel("main", Text("main")).Role(RolePrimary).MinWidth(20).MinHeight(3)
	ws.Panel("log", Text("log")).Role(RoleTelemetry).MinWidth(8).MinHeight(3).MoveBelow(120, "main")
	ws.Layout(HStack(Leaf("main"), Leaf("log")))

	solved := ws.Solve(100, 24)
	if _, ok := solved.Rects["log"]; !ok {
		t.Fatal("log missing after move below")
	}
	if solved.Rects["log"].Y <= solved.Rects["main"].Y {
		t.Fatalf("log should sit below main: main=%+v log=%+v", solved.Rects["main"], solved.Rects["log"])
	}
}

func TestResponsiveRoleDefaults(t *testing.T) {
	ws := NewWorkspace(WithAdaptiveLayout(), WithGap(0))
	ws.Panel("main", Text("main")).Role(RolePrimary).MinWidth(20).MinHeight(3)
	ws.Panel("log", Text("log")).Role(RoleTelemetry).MinWidth(8).MinHeight(3)
	ws.Panel("opt", Text("opt")).Role(RoleOptional).MinWidth(8).MinHeight(3)
	ws.Layout(HStack(Leaf("main"), Leaf("log"), Leaf("opt")))

	ws.Solve(60, 24)
	if ws.IsVisible("log") {
		t.Fatal("telemetry should hide by default below 70")
	}
	if ws.IsVisible("opt") {
		t.Fatal("optional should hide by default below 80")
	}
	if !ws.IsVisible("main") {
		t.Fatal("primary must survive")
	}
}

func TestResponsiveDisabledLeavesLayoutAlone(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("main", Text("main")).Role(RolePrimary).MinWidth(20).MinHeight(3)
	ws.Panel("opt", Text("opt")).Role(RoleOptional).MinWidth(8).MinHeight(3)
	ws.Layout(HStack(Leaf("main"), Leaf("opt")))
	ws.Solve(40, 24)
	if !ws.IsVisible("opt") {
		t.Fatal("without adaptive layout nothing should be hidden")
	}
}

func containsTabStack(node LayoutNode) bool {
	found := false
	walkLayout(node, func(n LayoutNode) {
		if _, ok := n.(*TabStackNode); ok {
			found = true
		}
	})
	return found
}
