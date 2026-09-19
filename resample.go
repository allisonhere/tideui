package tideui

import (
	"image"
	"image/color"
	"math"
)

// Resample scales a picture to a new size, bilinear. Something has to interpolate
// when imagery of a fixed resolution meets a pane of whatever size, and a photograph
// of terrain is exactly the case where nearest neighbour shows.
func Resample(source image.Image, width, height int) image.Image {
	if source == nil {
		return nil
	}
	return ResampleInto(source, source.Bounds(), width, height)
}

// ResampleInto scales one rectangle of a picture into a new width by height one: a
// map crop is a resample of the middle of a block of tiles, not of the whole block.
//
// The arithmetic is done on premultiplied channels, which is what every image in Go
// hands out, so a partly transparent pixel stays partly transparent and a composite
// over it comes out right.
func ResampleInto(source image.Image, from image.Rectangle, width, height int) image.Image {
	if source == nil || width <= 0 || height <= 0 || from.Empty() {
		return nil
	}
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	sourceWidth, sourceHeight := float64(from.Dx()), float64(from.Dy())
	for y := 0; y < height; y++ {
		// The sample point is the centre of the destination pixel, mapped back into
		// the source: sampling at the corner is how a half-pixel shift creeps in.
		sourceY := (float64(y)+0.5)*sourceHeight/float64(height) - 0.5
		for x := 0; x < width; x++ {
			sourceX := (float64(x)+0.5)*sourceWidth/float64(width) - 0.5
			result.Set(x, y, bilinear(source, from, sourceX, sourceY))
		}
	}
	return result
}

// bilinear blends the four pixels around a fractional position in the source. At the
// edges it clamps rather than fading to black, because a quarter of a pixel beyond the
// edge is the edge.
func bilinear(source image.Image, from image.Rectangle, x, y float64) color.RGBA {
	left, top := math.Floor(x), math.Floor(y)
	mixX, mixY := x-left, y-top
	lowX, lowY := int(left), int(top)

	at := func(offsetX, offsetY int) (uint32, uint32, uint32, uint32) {
		sampleX := min(max(lowX+offsetX, 0), from.Dx()-1)
		sampleY := min(max(lowY+offsetY, 0), from.Dy()-1)
		return source.At(from.Min.X+sampleX, from.Min.Y+sampleY).RGBA()
	}
	topRed, topGreen, topBlue, topAlpha := at(0, 0)
	topRightRed, topRightGreen, topRightBlue, topRightAlpha := at(1, 0)
	bottomRed, bottomGreen, bottomBlue, bottomAlpha := at(0, 1)
	bottomRightRed, bottomRightGreen, bottomRightBlue, bottomRightAlpha := at(1, 1)

	blend := func(topLeft, topRight, bottomLeft, bottomRight uint32) uint8 {
		topRow := float64(topLeft)*(1-mixX) + float64(topRight)*mixX
		bottomRow := float64(bottomLeft)*(1-mixX) + float64(bottomRight)*mixX
		value := topRow*(1-mixY) + bottomRow*mixY
		return uint8(min(255, math.Round(value/257))) // 16-bit channels to 8
	}
	return color.RGBA{
		R: blend(topRed, topRightRed, bottomRed, bottomRightRed),
		G: blend(topGreen, topRightGreen, bottomGreen, bottomRightGreen),
		B: blend(topBlue, topRightBlue, bottomBlue, bottomRightBlue),
		A: blend(topAlpha, topRightAlpha, bottomAlpha, bottomRightAlpha),
	}
}
