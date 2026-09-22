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

func TestPaneRatioNormalizesOptions(t *testing.T) {
	tests := []struct {
		name          string
		options       PaneRatioOptions
		wantValue     float64
		wantAfterGrow float64
	}{
		{
			name:          "valid",
			options:       PaneRatioOptions{Initial: 0.3, Min: 0.2, Max: 0.8, Step: 0.1},
			wantValue:     0.3,
			wantAfterGrow: 0.4,
		},
		{
			name:          "inverted bounds swap",
			options:       PaneRatioOptions{Initial: 0.5, Min: 0.8, Max: 0.2, Step: 0.1},
			wantValue:     0.5,
			wantAfterGrow: 0.6,
		},
		{
			name:          "invalid min",
			options:       PaneRatioOptions{Initial: 0.5, Min: 1, Max: 0.8, Step: 0.1},
			wantValue:     0.5,
			wantAfterGrow: 0.6,
		},
		{
			name:          "invalid max",
			options:       PaneRatioOptions{Initial: 0.5, Min: 0.2, Max: 0, Step: 0.1},
			wantValue:     0.5,
			wantAfterGrow: 0.6,
		},
		{
			name:          "initial below bounds",
			options:       PaneRatioOptions{Initial: 0.1, Min: 0.2, Max: 0.8, Step: 0.1},
			wantValue:     0.2,
			wantAfterGrow: 0.3,
		},
		{
			name:          "initial above bounds",
			options:       PaneRatioOptions{Initial: 0.9, Min: 0.2, Max: 0.8, Step: 0.1},
			wantValue:     0.8,
			wantAfterGrow: 0.8,
		},
		{
			name:          "invalid step",
			options:       PaneRatioOptions{Initial: 0.3, Min: 0.2, Max: 0.8, Step: -1},
			wantValue:     0.3,
			wantAfterGrow: 0.32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := NewPaneRatio(tt.options)
			if got := ratio.Value(); !floatsEqual(got, tt.wantValue) {
				t.Fatalf("initial value = %v, want %v", got, tt.wantValue)
			}
			ratio.Grow()
			if got := ratio.Value(); !floatsEqual(got, tt.wantAfterGrow) {
				t.Fatalf("grown value = %v, want %v", got, tt.wantAfterGrow)
			}
		})
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
