package tideui

import "testing"

func TestPaneRatioGrowShrink(t *testing.T) {
	r := NewPaneRatio(PaneRatioOptions{Initial: 0.28, Min: 0.15, Max: 0.5, Step: 0.02})
	r.Grow()
	if got, want := r.Value(), 0.30; !floatsEqual(got, want) {
		t.Fatalf("Value() = %v, want %v", got, want)
	}
	r.Shrink()
	r.Shrink()
	if got, want := r.Value(), 0.26; !floatsEqual(got, want) {
		t.Fatalf("Value() = %v, want %v", got, want)
	}
}

func TestPaneRatioClampsAtBounds(t *testing.T) {
	r := NewPaneRatio(PaneRatioOptions{Initial: 0.48, Min: 0.15, Max: 0.5, Step: 0.1})
	r.Grow()
	r.Grow()
	if got, want := r.Value(), 0.5; !floatsEqual(got, want) {
		t.Fatalf("Value() at Max = %v, want %v", got, want)
	}
	for i := 0; i < 10; i++ {
		r.Shrink()
	}
	if got, want := r.Value(), 0.15; !floatsEqual(got, want) {
		t.Fatalf("Value() at Min = %v, want %v", got, want)
	}
}

func TestPaneRatioDefaults(t *testing.T) {
	r := NewPaneRatio(PaneRatioOptions{})
	if got, want := r.Value(), 0.5; !floatsEqual(got, want) {
		t.Fatalf("default Value() = %v, want %v (midpoint of 0.1/0.9)", got, want)
	}
	r.Shrink()
	if got, want := r.Value(), 0.48; !floatsEqual(got, want) {
		t.Fatalf("Value() after default Shrink = %v, want %v", got, want)
	}
}

func floatsEqual(a, b float64) bool {
	const epsilon = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < epsilon
}
