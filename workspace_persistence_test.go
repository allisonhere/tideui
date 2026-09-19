package tideui

import "testing"

func persistedWorkspace() *Workspace {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("a", Text("a")).Role(RoleNavigation).MinWidth(8).MinHeight(3)
	ws.Panel("b", Text("b")).Role(RolePrimary).MinWidth(10).MinHeight(3)
	return ws
}

func TestPersistenceRoundTripThroughStore(t *testing.T) {
	store := NewMemoryStore()
	ws := NewWorkspace(WithStore(store), WithPersistence("app"), WithGap(0))
	ws.Panel("a", Text("a")).MinWidth(8).MinHeight(3)
	ws.Panel("b", Text("b")).MinWidth(8).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Leaf("b")))
	ws.Hide("b")
	if err := ws.Persist(); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	restored := NewWorkspace(WithStore(store), WithPersistence("app"), WithGap(0))
	restored.Panel("a", Text("a")).MinWidth(8).MinHeight(3)
	restored.Panel("b", Text("b")).MinWidth(8).MinHeight(3)
	// The declared layout is a fallback; the saved layout must win.
	restored.Layout(HStack(Leaf("b"), Leaf("a")))
	restored.Solve(80, 24)

	if !restored.Hidden("b") {
		t.Fatal("hidden state did not survive the round trip")
	}
	solved := restored.Solve(80, 24)
	if _, ok := solved.Rects["b"]; ok {
		t.Fatal("restored layout laid out a hidden panel")
	}
	if solved.Rects["a"].X != 0 {
		t.Fatalf("a should be first in the restored layout: %+v", solved.Rects)
	}
}

func TestPersistenceJSONRoundTrip(t *testing.T) {
	ws := persistedWorkspace()
	ws.Panel("c", Text("c")).MinWidth(8).MinHeight(3)
	ws.Layout(HStack(Leaf("a"), Tabs("b", "c")))
	data, err := ws.PersistedJSON()
	if err != nil {
		t.Fatal(err)
	}
	fresh := persistedWorkspace()
	fresh.Panel("c", Text("c")).MinWidth(8).MinHeight(3)
	if err := fresh.RestoreJSON(data); err != nil {
		t.Fatalf("RestoreJSON: %v", err)
	}
	if !containsTabStack(fresh.RootLayout()) {
		t.Fatalf("tab stack lost: %s", LayoutString(fresh.RootLayout()))
	}
}

func TestPersistenceRejectsUnknownVersion(t *testing.T) {
	ws := persistedWorkspace()
	err := ws.RestoreJSON([]byte(`{"version":99,"root":{"type":"leaf","id":"a"}}`))
	if err == nil {
		t.Fatal("expected unsupported version error")
	}
}

func TestPersistenceRejectsCorruptJSON(t *testing.T) {
	ws := persistedWorkspace()
	if err := ws.RestoreJSON([]byte(`{not json`)); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestPersistenceDropsUnknownPanels(t *testing.T) {
	ws := persistedWorkspace()
	data := []byte(`{"version":1,"root":{"type":"split","orientation":"horizontal","children":[{"type":"leaf","id":"a"},{"type":"leaf","id":"ghost"}]},"hidden":["ghost","a"]}`)
	if err := ws.RestoreJSON(data); err != nil {
		t.Fatalf("RestoreJSON: %v", err)
	}
	if LayoutContainsPanel(ws.RootLayout(), "ghost") {
		t.Fatal("unknown panel survived restore")
	}
	if !ws.Hidden("a") {
		t.Fatal("known hidden panel was not restored")
	}
}

func TestPersistenceFailsWhenNothingUsable(t *testing.T) {
	ws := persistedWorkspace()
	data := []byte(`{"version":1,"root":{"type":"leaf","id":"ghost"}}`)
	if err := ws.RestoreJSON(data); err == nil {
		t.Fatal("expected error when no known panels remain")
	}
}

func TestPersistenceDropsDuplicatePanels(t *testing.T) {
	ws := persistedWorkspace()
	data := []byte(`{"version":1,"root":{"type":"tabs","panels":["a","a","b"]}}`)
	if err := ws.RestoreJSON(data); err != nil {
		t.Fatalf("RestoreJSON: %v", err)
	}
	ids := LayoutPanelIDs(ws.RootLayout())
	if len(ids) != 2 {
		t.Fatalf("panel ids = %v, want 2 unique", ids)
	}
}

func TestPersistenceHiddenNonHideablePanelIsIgnored(t *testing.T) {
	ws := NewWorkspace(WithGap(0))
	ws.Panel("fixed", Text("fixed")).Hideable(false)
	data := []byte(`{"version":1,"root":{"type":"leaf","id":"fixed"},"hidden":["fixed"]}`)
	_ = ws.RestoreJSON(data)
	if ws.Hidden("fixed") {
		t.Fatal("non-hideable panel should not be restorable as hidden")
	}
}
