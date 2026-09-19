package tideui

import (
	"image"
	"image/color"
	"testing"
)

// A theme's own colours, derived rather than written down: half of the background mixed
// into the text colour is a line colour that belongs to the theme.
func TestMixDerivesAColourBetweenTwo(t *testing.T) {
	bg := color.RGBA{R: 30, G: 30, B: 46, A: 255}
	fg := color.RGBA{R: 205, G: 214, B: 244, A: 255}
	if got := Mix(bg, fg, 0); got != bg {
		t.Fatalf("nothing of the second colour = %v, want the first", got)
	}
	if got := Mix(bg, fg, 1); got != fg {
		t.Fatalf("all of the second colour = %v, want the second", got)
	}
	half := Mix(bg, fg, 0.5)
	if half.R != 118 || half.G != 122 || half.B != 145 {
		t.Fatalf("half way = %v, want 118,122,145", half)
	}
	// Out of range is clamped rather than wrapping a colour.
	if got := Mix(bg, fg, 4); got != fg {
		t.Fatalf("more than all of it = %v, want the second colour", got)
	}
	if got := Mix(bg, fg, -1); got != bg {
		t.Fatalf("less than none of it = %v, want the first colour", got)
	}
}

// A paper map drawn on dark glass: white paper becomes the background colour, dark ink
// becomes the light one. This is the whole point - the map ends up in the theme's own
// colours instead of in a lit sheet of white.
func TestDuotoneInksAPaperMapInTwoColours(t *testing.T) {
	paper := color.RGBA{R: 16, G: 16, B: 22, A: 255} // the panel's background
	ink := color.RGBA{R: 205, G: 214, B: 244, A: 255}
	source := image.NewRGBA(image.Rect(0, 0, 2, 1))
	source.Set(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255}) // white paper
	source.Set(1, 0, color.RGBA{R: 0, G: 0, B: 0, A: 255})       // black ink

	got := Duotone(source, paper, ink)
	if r, g, b, _ := got.At(0, 0).RGBA(); r>>8 != 16 || g>>8 != 16 || b>>8 != 22 {
		t.Fatalf("the paper came out %d,%d,%d, want the background's own colour", r>>8, g>>8, b>>8)
	}
	if r, g, b, _ := got.At(1, 0).RGBA(); r>>8 != 205 || g>>8 != 214 || b>>8 != 244 {
		t.Fatalf("the ink came out %d,%d,%d, want the theme's own light colour", r>>8, g>>8, b>>8)
	}
}

// A mid-grey feature lands between the two: the map keeps its gradations rather than
// becoming two flat colours.
func TestDuotoneKeepsGradations(t *testing.T) {
	paper := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	ink := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	source := image.NewRGBA(image.Rect(0, 0, 1, 1))
	source.Set(0, 0, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	r, _, _, _ := Duotone(source, paper, ink).At(0, 0).RGBA()
	if r>>8 < 110 || r>>8 > 145 {
		t.Fatalf("a mid-grey pixel came out %d, want about half way", r>>8)
	}
}

// Green reads as light and blue as dark to the eye, and a map full of both must not be
// greyed by their average. The map is re-inked, so the colour that is *lighter on the
// paper* is the *darker pixel* afterwards - that is the inversion, not a bug in it.
func TestDuotoneWeighsColourTheWayAnEyeDoes(t *testing.T) {
	paper := color.RGBA{A: 255}
	ink := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	green := image.NewRGBA(image.Rect(0, 0, 1, 1))
	green.Set(0, 0, color.RGBA{G: 255, A: 255})
	blue := image.NewRGBA(image.Rect(0, 0, 1, 1))
	blue.Set(0, 0, color.RGBA{B: 255, A: 255})

	greenLight, _, _, _ := Duotone(green, paper, ink).At(0, 0).RGBA()
	blueLight, _, _, _ := Duotone(blue, paper, ink).At(0, 0).RGBA()
	if blueLight>>8 <= greenLight>>8 {
		t.Fatalf("blue came out %d and green %d, want blue - the darker ink - the lighter line", blueLight>>8, greenLight>>8)
	}
}

// Ground the map does not cover stays uncovered: a re-inked map must not become a solid
// rectangle over whatever is behind it.
func TestDuotoneLeavesTransparencyAlone(t *testing.T) {
	got := Duotone(image.NewRGBA(image.Rect(0, 0, 2, 2)), color.RGBA{A: 255}, color.RGBA{R: 255, A: 255})
	if got == nil {
		t.Fatal("a transparent map gave no picture")
	}
	if _, _, _, alpha := got.At(1, 1).RGBA(); alpha != 0 {
		t.Fatalf("transparent ground came out with alpha %d", alpha>>8)
	}
	if Duotone(nil, color.RGBA{}, color.RGBA{}) != nil {
		t.Fatal("re-inking nothing gave a picture")
	}
}
