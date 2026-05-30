package tideui

// PaneScroller manages a scroll offset for a Pane. The application holds one
// scroller per pane and passes Offset() to Pane.ScrollOffset each render.
type PaneScroller struct {
	offset int
}

// Offset returns the current scroll offset to set on Pane.ScrollOffset.
func (s PaneScroller) Offset() int { return s.offset }

// ScrollDown moves the view down by n lines.
func (s *PaneScroller) ScrollDown(n int) { s.offset += n }

// ScrollUp moves the view up by n lines, stopping at the top.
func (s *PaneScroller) ScrollUp(n int) { s.offset = max(0, s.offset-n) }

// ScrollToTop resets the scroll position to the beginning.
func (s *PaneScroller) ScrollToTop() { s.offset = 0 }

// ClampTo limits the offset so the view never scrolls past the last line.
// totalLines is the number of lines in the pane content; visibleLines is the
// number of lines the pane can display (pane height minus the header row).
func (s *PaneScroller) ClampTo(totalLines, visibleLines int) {
	s.offset = min(s.offset, max(0, totalLines-visibleLines))
}

// CanScrollDown reports whether content is hidden below the current view.
func (s PaneScroller) CanScrollDown(totalLines, visibleLines int) bool {
	return s.offset < max(0, totalLines-visibleLines)
}
