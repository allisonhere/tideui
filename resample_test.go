package tideui

import (
	"image"
	"image/color"
	"testing"
)

// A picture at the size it already is is the same picture: resampling is for making
// one size out of another, not for touching what fits.
func TestResampleAtTheSameSizeChangesNothing(t *testing.T) {
	source := filled(image.Rect(0, 0, 3, 2), color.RGBA{R: 200, G: 100, B: 50, A: 255})
	got := Resample(source, 3, 2)
	if got.Bounds().Dx() != 3 || got.Bounds().Dy() != 2 {
		t.Fatalf("resampled to %v, want 3x2", got.Bounds())
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			r, g, b, _ := got.At(x, y).RGBA()
			if r>>8 != 200 || g>>8 != 100 || b>>8 != 50 {
				t.Fatalf("pixel %d,%d = %d,%d,%d, want 200,100,50", x, y, r>>8, g>>8, b>>8)
			}
		}
	}
}

// Doubling a two-by-two checkerboard gives four cells of each colour, with the
// boundaries averaged rather than hard: that is the difference between a resampled
// photograph and a screen door.
func TestResampleScalesAndBlends(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 2, 2))
	source.Set(0, 0, color.RGBA{R: 255, A: 255})
	source.Set(1, 0, color.RGBA{G: 255, A: 255})
	source.Set(0, 1, color.RGBA{B: 255, A: 255})
	source.Set(1, 1, color.RGBA{R: 255, G: 255, A: 255})

	got := Resample(source, 4, 4)
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 4 {
		t.Fatalf("resampled to %v, want 4x4", got.Bounds())
	}
	// The corners keep their own colour.
	if r, g, _, _ := got.At(0, 0).RGBA(); r>>8 != 255 || g>>8 != 0 {
		t.Fatalf("the top-left corner = %d,%d, want the red quadrant", r>>8, g>>8)
	}
	// The middle is a blend of all four, which is what bilinear means: no channel is
	// at either extreme there, because the pixel between four colours is not one of
	// the four.
	r, g, b, _ := got.At(2, 2).RGBA()
	if r>>8 == 0 || r>>8 == 255 || g>>8 == 0 || g>>8 == 255 {
		t.Fatalf("the middle = %d,%d,%d, want a blend of the four quadrants", r>>8, g>>8, b>>8)
	}
}

// Transparent ground stays transparent, and see-through stays see-through: a
// resampled backdrop must not become a black box over the panel behind it.
func TestResampleKeepsTransparency(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			source.Set(x, y, color.NRGBA{R: 255, A: 128}) // half-transparent red
		}
	}
	got := Resample(source, 4, 4)
	r, _, _, a := got.At(2, 2).RGBA()
	if a == 0 || a>>8 > 240 {
		t.Fatalf("half-transparent red resampled to alpha %d, want it still see-through", a>>8)
	}
	// Premultiplied: the red channel is scaled by the alpha, which is what makes a
	// later composite over a map come out right.
	if r>>8 > a>>8+2 {
		t.Fatalf("red %d exceeds alpha %d in a premultiplied picture", r>>8, a>>8)
	}
}

// Resampling part of a picture is how a map crop is made: the tiles cover more ground
// than the pane, and what is wanted is the middle of them.
func TestResampleTakesThePartAskedFor(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if x >= 2 {
				source.Set(x, y, color.RGBA{G: 255, A: 255})
			} else {
				source.Set(x, y, color.RGBA{R: 255, A: 255})
			}
		}
	}
	right := ResampleInto(source, image.Rect(2, 0, 4, 4), 2, 2)
	if _, green, _, _ := right.At(0, 0).RGBA(); green>>8 != 255 {
		t.Fatalf("the right half of the picture = %d green, want the green half", green>>8)
	}
	left := ResampleInto(source, image.Rect(0, 0, 2, 4), 2, 2)
	if r, _, _, _ := left.At(1, 1).RGBA(); r>>8 != 255 {
		t.Fatalf("the left half of the picture = %d, want the red half", r>>8)
	}
}

// A picture of nothing is nothing: a caller that asked for a size and got a source it
// did not expect gets no picture rather than a panic.
func TestResampleOfNothingIsNothing(t *testing.T) {
	if got := Resample(nil, 4, 4); got != nil {
		t.Fatalf("resampling nothing gave %v", got)
	}
	empty := image.NewRGBA(image.Rectangle{})
	if got := Resample(empty, 4, 4); got != nil {
		t.Fatalf("resampling an empty picture gave %v", got)
	}
	if got := Resample(filled(image.Rect(0, 0, 2, 2), color.RGBA{A: 255}), 0, 0); got != nil {
		t.Fatalf("resampling to no pixels gave %v", got)
	}
}
