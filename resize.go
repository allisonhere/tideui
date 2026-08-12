package tideui

// PaneRatioOptions configures a PaneRatio.
type PaneRatioOptions struct {
	// Initial is the starting ratio. Zero uses the midpoint of Min and Max.
	Initial float64
	// Min is the smallest ratio Grow/Shrink will settle at. Zero uses 0.1.
	Min float64
	// Max is the largest ratio Grow/Shrink will settle at. Zero uses 0.9.
	Max float64
	// Step is the amount Grow/Shrink adjusts the ratio by. Zero uses 0.02.
	Step float64
}

// PaneRatio manages one adjustable split ratio — e.g. Layout.SidebarRatio,
// Layout.UpperRightRatio, or a Floating ratio. The application holds one
// PaneRatio per adjustable split, calls Grow/Shrink in response to its own
// key routing (typically shift+arrow keys), and reads Value() into the
// corresponding Layout ratio field each render.
type PaneRatio struct {
	value float64
	min   float64
	max   float64
	step  float64
}

// NewPaneRatio creates a PaneRatio bounded to [Min, Max] and adjusted by
// Step on each Grow/Shrink call.
func NewPaneRatio(options PaneRatioOptions) PaneRatio {
	loBound := options.Min
	if loBound <= 0 {
		loBound = 0.1
	}
	hiBound := options.Max
	if hiBound <= 0 || hiBound > 1 {
		hiBound = 0.9
	}
	step := options.Step
	if step <= 0 {
		step = 0.02
	}
	initial := options.Initial
	if initial <= 0 {
		initial = (loBound + hiBound) / 2
	}
	return PaneRatio{value: min(max(initial, loBound), hiBound), min: loBound, max: hiBound, step: step}
}

// Value returns the current ratio, ready to assign to a Layout ratio field.
func (r PaneRatio) Value() float64 { return r.value }

// Grow increases the ratio by one step, stopping at Max.
func (r *PaneRatio) Grow() { r.value = min(r.value+r.step, r.max) }

// Shrink decreases the ratio by one step, stopping at Min.
func (r *PaneRatio) Shrink() { r.value = max(r.value-r.step, r.min) }
