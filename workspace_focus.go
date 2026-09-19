package tideui

// Direction names one of the four spatial directions used by focus traversal
// and keyboard resizing.
type Direction int

const (
	// DirUp moves toward smaller row coordinates.
	DirUp Direction = iota
	// DirDown moves toward larger row coordinates.
	DirDown
	// DirLeft moves toward smaller column coordinates.
	DirLeft
	// DirRight moves toward larger column coordinates.
	DirRight
)

// String returns a stable direction name for hints and debugging.
func (d Direction) String() string {
	switch d {
	case DirUp:
		return "up"
	case DirDown:
		return "down"
	case DirLeft:
		return "left"
	case DirRight:
		return "right"
	default:
		return "none"
	}
}

// Horizontal reports whether the direction runs along the width axis.
func (d Direction) Horizontal() bool { return d == DirLeft || d == DirRight }

// Forward reports whether the direction increases along its axis.
func (d Direction) Forward() bool { return d == DirRight || d == DirDown }

// FocusManager owns the current focus id and the traversal algorithms. It is
// independent of rendering so focus can be tested against solved rectangles.
type FocusManager struct {
	current string
}

// Current returns the focused panel id.
func (f FocusManager) Current() string { return f.current }

// Set focuses a specific panel id, without validation.
func (f *FocusManager) Set(id string) { f.current = id }

// Next returns the next id in order, wrapping around.
func (f FocusManager) Next(order []string) string {
	return cycle(order, f.current, 1)
}

// Prev returns the previous id in order, wrapping around.
func (f FocusManager) Prev(order []string) string {
	return cycle(order, f.current, -1)
}

// Directional returns the nearest focusable panel in a direction from the
// current one, or "" when there is none.
func (f FocusManager) Directional(solved SolvedLayout, dir Direction, focusable func(string) bool) string {
	current, ok := f.currentRect(solved)
	if !ok {
		return ""
	}
	best := ""
	bestScore := -1.0
	bestOverlap := false
	for _, region := range solved.Regions {
		id := region.ActivePanel()
		if id == "" || id == f.current || !focusable(id) {
			continue
		}
		distance, overlap, ok := directionalScore(current, region.Rect, dir)
		if !ok {
			continue
		}
		score := distance
		if overlap {
			score -= 1e6 // strongly prefer candidates sharing the cross axis
		}
		if best == "" || score < bestScore || (score == bestScore && overlap && !bestOverlap) {
			best = id
			bestScore = score
			bestOverlap = overlap
		}
	}
	return best
}

func (f FocusManager) currentRect(solved SolvedLayout) (Rect, bool) {
	for _, region := range solved.Regions {
		for _, id := range region.PanelIDs {
			if id == f.current {
				return region.Rect, true
			}
		}
	}
	return Rect{}, false
}

// directionalScore reports the gap between two rectangles in a direction, and
// whether they overlap on the perpendicular axis. ok is false when the target
// does not lie in the requested direction at all.
func directionalScore(from, to Rect, dir Direction) (distance float64, overlap bool, ok bool) {
	if from.Empty() || to.Empty() {
		return 0, false, false
	}
	switch dir {
	case DirRight:
		if to.X < from.X+from.Width {
			return 0, false, false
		}
		distance = float64(to.X - (from.X + from.Width))
		overlap = to.Y < from.Y+from.Height && to.Y+to.Height > from.Y
	case DirLeft:
		if to.X+to.Width > from.X {
			return 0, false, false
		}
		distance = float64(from.X - (to.X + to.Width))
		overlap = to.Y < from.Y+from.Height && to.Y+to.Height > from.Y
	case DirDown:
		if to.Y < from.Y+from.Height {
			return 0, false, false
		}
		distance = float64(to.Y - (from.Y + from.Height))
		overlap = to.X < from.X+from.Width && to.X+to.Width > from.X
	case DirUp:
		if to.Y+to.Height > from.Y {
			return 0, false, false
		}
		distance = float64(from.Y - (to.Y + to.Height))
		overlap = to.X < from.X+from.Width && to.X+to.Width > from.X
	}
	if !overlap {
		// Penalise candidates that only share a corner so overlapping
		// neighbours always win, while still allowing diagonal movement.
		distance += 1000
	}
	return distance, overlap, true
}

func cycle(order []string, current string, delta int) string {
	if len(order) == 0 {
		return ""
	}
	index := 0
	for i, id := range order {
		if id == current {
			index = i
			break
		}
	}
	index = (index + delta + len(order)) % len(order)
	return order[index]
}

// FocusPresentation controls how the workspace signals which panel is active.
// It is deliberately a set of switches rather than free-form styling so themes
// keep ownership of the actual colours.
type FocusPresentation struct {
	// ActiveBorder highlights the focused panel frame.
	ActiveBorder bool
	// ActiveTitle highlights the focused panel's title and tabs.
	ActiveTitle bool
	// TitleCapsule paints the focused title as an accent capsule rather than
	// bright text; capsule is the stronger, more branded treatment.
	TitleCapsule bool
	// FocusRail draws a thin accent rail down the focused panel's body, an
	// unmistakable but quiet "you are here" marker.
	FocusRail bool
	// DimInactive lowers the contrast of low-priority unfocused panels.
	DimInactive bool
	// MutedSecondary de-emphasises secondary unfocused panels further.
	MutedSecondary bool
	// AccentMarker draws a leading accent marker on the focused title.
	AccentMarker bool
	// InactiveSelection keeps a list selection visible (muted) when its panel
	// is not focused, so context is preserved without competing for attention.
	InactiveSelection bool
	// StatusStrip renders the workspace status strip with active-panel info.
	StatusStrip bool
	// KeyHints renders contextual action hints in the focused panel footer.
	KeyHints bool
}

// DefaultFocusPresentation returns the presentation used when none is given:
// rich, but without animation or excessive motion.
func DefaultFocusPresentation() FocusPresentation {
	return FocusPresentation{
		ActiveBorder:      true,
		ActiveTitle:       true,
		TitleCapsule:      true,
		FocusRail:         true,
		DimInactive:       true,
		MutedSecondary:    true,
		AccentMarker:      true,
		InactiveSelection: true,
		StatusStrip:       true,
		KeyHints:          true,
	}
}
