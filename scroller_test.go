package tideui

import "testing"

func TestPaneScrollerScrollDownUp(t *testing.T) {
	var s PaneScroller
	s.ScrollDown(3)
	if s.Offset() != 3 {
		t.Fatalf("offset = %d, want 3", s.Offset())
	}
	s.ScrollUp(1)
	if s.Offset() != 2 {
		t.Fatalf("offset = %d, want 2", s.Offset())
	}
	s.ScrollUp(100)
	if s.Offset() != 0 {
		t.Fatalf("ScrollUp past top: offset = %d, want 0", s.Offset())
	}
}

func TestPaneScrollerScrollToTop(t *testing.T) {
	var s PaneScroller
	s.ScrollDown(10)
	s.ScrollToTop()
	if s.Offset() != 0 {
		t.Fatalf("offset after ScrollToTop = %d, want 0", s.Offset())
	}
}

func TestPaneScrollerClampTo(t *testing.T) {
	var s PaneScroller
	s.ScrollDown(50)
	s.ClampTo(20, 10) // 20 total lines, 10 visible → max offset = 10
	if s.Offset() != 10 {
		t.Fatalf("offset after ClampTo(20,10) = %d, want 10", s.Offset())
	}
	s.ClampTo(5, 10) // content shorter than view → max offset = 0
	if s.Offset() != 0 {
		t.Fatalf("offset after ClampTo(5,10) = %d, want 0", s.Offset())
	}
}

func TestPaneScrollerCanScrollDown(t *testing.T) {
	var s PaneScroller
	if s.CanScrollDown(5, 10) {
		t.Fatal("CanScrollDown should be false when content fits in view")
	}
	if !s.CanScrollDown(15, 10) {
		t.Fatal("CanScrollDown should be true when content exceeds view")
	}
	s.ScrollDown(5)
	if s.CanScrollDown(15, 10) {
		// offset=5, max=5 → at bottom
		t.Fatal("CanScrollDown should be false when at bottom")
	}
	if !s.CanScrollDown(16, 10) {
		// offset=5, max=6 → one more line available
		t.Fatal("CanScrollDown should be true when one more line is available")
	}
}
