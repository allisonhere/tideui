package tideui

import (
	"image"
	"image/color"
	"testing"
)

// filled returns a picture of one colour.
func filled(bounds image.Rectangle, paint color.Color) *image.RGBA {
	img := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Set(x, y, paint)
		}
	}
	return img
}

func lumaAt(img image.Image, x, y int) int {
	r, g, b, _ := img.At(x, y).RGBA()
	return int((r>>8 + g>>8 + b>>8) / 3)
}

// A radar frame is nothing at all except where it rains, so what is under it has to
// survive: that is the whole reason for fetching a basemap.
func TestCompositeLeavesWhatShowsThrough(t *testing.T) {
	base := filled(image.Rect(0, 0, 4, 4), color.RGBA{R: 10, G: 20, B: 30, A: 255})
	clear := image.NewRGBA(image.Rect(0, 0, 4, 4))

	got := Composite(base, clear)
	if got.Bounds() != base.Bounds() {
		t.Fatalf("composited bounds = %v, want the base's %v", got.Bounds(), base.Bounds())
	}
	if r, g, b, _ := got.At(1, 1).RGBA(); r>>8 != 10 || g>>8 != 20 || b>>8 != 30 {
		t.Fatalf("a transparent overlay changed the picture under it: %d,%d,%d", r>>8, g>>8, b>>8)
	}
	// And it composes a picture rather than returning the base: a caller that then
	// draws on the result must not damage the frame it composed from.
	if got == image.Image(base) {
		t.Fatal("the composite is the base itself, which the panel draws on")
	}
}

// Where the radar is solid, the radar wins.
func TestCompositeKeepsWhatIsOnTop(t *testing.T) {
	base := filled(image.Rect(0, 0, 4, 4), color.RGBA{G: 255, A: 255})
	got := Composite(base, filled(image.Rect(0, 0, 4, 4), color.RGBA{R: 255, A: 255}))
	if r, g, _, _ := got.At(2, 2).RGBA(); r>>8 != 255 || g>>8 != 0 {
		t.Fatalf("a solid overlay = %d,%d, want the overlay's own colour", r>>8, g>>8)
	}
}

// A half-transparent picture is half of each, which is what makes a light rain echo
// look like rain over a map rather than a hole in it. The translucent colours are
// NRGBA and not RGBA on purpose: RGBA is premultiplied, and "255 of red at half
// alpha" is not a colour that type can hold.
func TestCompositeBlendsHalfTransparency(t *testing.T) {
	black := filled(image.Rect(0, 0, 4, 4), color.RGBA{A: 255})
	halfWhite := filled(image.Rect(0, 0, 4, 4), color.NRGBA{R: 255, G: 255, B: 255, A: 128})
	got := Composite(black, halfWhite)
	if luma := lumaAt(got, 1, 1); luma < 100 || luma > 155 {
		t.Fatalf("half-transparent white over black = %d, want about half of 255", luma)
	}
	// The same overlay over white is white: the base still counts.
	white := filled(image.Rect(0, 0, 4, 4), color.RGBA{R: 255, G: 255, B: 255, A: 255})
	if luma := lumaAt(Composite(white, halfWhite), 1, 1); luma < 240 {
		t.Fatalf("half-transparent white over white = %d, want near white", luma)
	}
}

// No basemap is the common case - the setting is off, or the fetch failed - and it
// has to cost nothing: the picture on top is the picture.
func TestCompositeWithoutABasemapIsTheTopPicture(t *testing.T) {
	radar := filled(image.Rect(0, 0, 4, 4), color.RGBA{A: 255})
	if got := Composite(nil, radar); got != image.Image(radar) {
		t.Fatal("composing onto nothing did not return the picture itself")
	}
	if got := Composite(radar, nil); got != image.Image(radar) {
		t.Fatal("composing nothing onto a picture did not return the picture")
	}
	if got := Composite(nil, nil); got != nil {
		t.Fatalf("composing nothing onto nothing = %v", got)
	}
}

// Two pictures of different sizes are a caller's bug, and the safe way to be wrong is
// to draw the data rather than the backdrop: a panel that briefly shows its picture
// without a map is right, one that briefly shows a map without its picture is not.
func TestCompositeRefusesAMismatchedPair(t *testing.T) {
	base := filled(image.Rect(0, 0, 4, 4), color.RGBA{R: 10, A: 255})
	over := filled(image.Rect(0, 0, 8, 8), color.RGBA{G: 255, A: 255})
	if got := Composite(base, over); got != image.Image(over) {
		t.Fatal("a mismatched pair was composed instead of refused")
	}
}
