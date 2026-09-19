package tideui

import "testing"

// A measurement no terminal could produce is treated as no measurement: a pane
// sized from a broken ioctl must not ask for a mosaic it does not need.
func TestNormalisedCellWidthRefusesTheImpossible(t *testing.T) {
	if got := normalisedCellWidth(0); got != defaultCellWidth {
		t.Fatalf("an unmeasured cell = %v, want the default %v", got, defaultCellWidth)
	}
	if got := normalisedCellWidth(-3); got != defaultCellWidth {
		t.Fatalf("a negative cell = %v, want the default %v", got, defaultCellWidth)
	}
	if got := normalisedCellWidth(120); got != defaultCellWidth {
		t.Fatalf("a 120 px cell = %v, want the default %v", got, defaultCellWidth)
	}
	if got := normalisedCellWidth(7.2); got != 7.2 {
		t.Fatalf("a measured cell = %v, want 7.2", got)
	}
}
