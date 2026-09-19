package tideui

import (
	"image"
	"image/color"
	"math"
)

// Mix blends one colour into another: t of 0 is all of a, t of 1 is all of b. It is how
// a colour is derived from the theme rather than written into the code - a map's line
// colour is the theme's background lifted towards its text, not a grey somebody picked.
func Mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Min(1, math.Max(0, t))
	mix := func(from, to uint8) uint8 {
		return uint8(math.Round(float64(from)*(1-t) + float64(to)*t))
	}
	return color.RGBA{
		R: mix(a.R, b.R),
		G: mix(a.G, b.G),
		B: mix(a.B, b.B),
		A: mix(a.A, b.A),
	}
}

// Duotone re-inks a picture in two colours by luminance, which is how a paper map is
// drawn on dark glass: the paper becomes the background, the ink becomes the light, and
// the map ends up in the theme's own colours rather than in a sheet of white.
//
// It is for maps made of ink on paper - a topographic sheet, a street map - and not for
// imagery, which is already its own colours.
func Duotone(source image.Image, paper, ink color.RGBA) image.Image {
	if source == nil {
		return nil
	}
	bounds := source.Bounds()
	if bounds.Empty() {
		return nil
	}
	result := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			red, green, blue, alpha := source.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if alpha == 0 {
				// Ground the map does not cover stays uncovered: a map has an edge.
				continue
			}
			// Rec. 601 luma, because what reads as light and dark is not the average of
			// the channels: green paper is bright to the eye and blue ink is not.
			luma := (299*uint64(red>>8) + 587*uint64(green>>8) + 114*uint64(blue>>8)) / 1000
			mix := float64(luma) / 255.0
			result.Set(x, y, color.RGBA{
				R: uint8(float64(ink.R)*(1-mix) + float64(paper.R)*mix),
				G: uint8(float64(ink.G)*(1-mix) + float64(paper.G)*mix),
				B: uint8(float64(ink.B)*(1-mix) + float64(paper.B)*mix),
				A: uint8(alpha >> 8),
			})
		}
	}
	return result
}
