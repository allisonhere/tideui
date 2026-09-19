package tideui

import "testing"

func focusFixture() SolvedLayout {
	regions := []SolvedRegion{
		{Rect: Rect{X: 0, Y: 0, Width: 40, Height: 10}, PanelIDs: []string{"left"}},
		{Rect: Rect{X: 40, Y: 0, Width: 40, Height: 5}, PanelIDs: []string{"topright"}},
		{Rect: Rect{X: 40, Y: 5, Width: 40, Height: 5}, PanelIDs: []string{"botright"}},
	}
	rects := map[string]Rect{}
	for _, region := range regions {
		for _, id := range region.PanelIDs {
			rects[id] = region.Rect
		}
	}
	return SolvedLayout{Width: 80, Height: 10, Regions: regions, Rects: rects}
}

func TestFocusDirectionalPrefersOverlap(t *testing.T) {
	solved := focusFixture()
	all := func(string) bool { return true }

	fm := FocusManager{current: "left"}
	if got := fm.Directional(solved, DirRight, all); got != "topright" {
		t.Fatalf("right from left = %q, want topright", got)
	}
	fm = FocusManager{current: "topright"}
	if got := fm.Directional(solved, DirDown, all); got != "botright" {
		t.Fatalf("down from topright = %q, want botright", got)
	}
	fm = FocusManager{current: "botright"}
	if got := fm.Directional(solved, DirUp, all); got != "topright" {
		t.Fatalf("up from botright = %q, want topright", got)
	}
	fm = FocusManager{current: "botright"}
	if got := fm.Directional(solved, DirLeft, all); got != "left" {
		t.Fatalf("left from botright = %q, want left", got)
	}
}

func TestFocusDirectionalRespectsFocusablePredicate(t *testing.T) {
	solved := focusFixture()
	fm := FocusManager{current: "left"}
	got := fm.Directional(solved, DirRight, func(id string) bool { return id != "topright" })
	if got != "botright" {
		t.Fatalf("right = %q, want botright when topright is not focusable", got)
	}
}

func TestFocusDirectionalReturnsEmptyAtEdge(t *testing.T) {
	solved := focusFixture()
	fm := FocusManager{current: "topright"}
	if got := fm.Directional(solved, DirRight, func(string) bool { return true }); got != "" {
		t.Fatalf("right at edge = %q, want empty", got)
	}
}

func TestFocusCycle(t *testing.T) {
	order := []string{"a", "b", "c"}
	fm := FocusManager{current: "b"}
	if got := fm.Next(order); got != "c" {
		t.Fatalf("next = %q, want c", got)
	}
	if got := fm.Prev(order); got != "a" {
		t.Fatalf("prev = %q, want a", got)
	}
	fm.Set("missing")
	if got := fm.Next(order); got != "b" {
		t.Fatalf("next from unknown = %q, want b (wraps from start)", got)
	}
}
