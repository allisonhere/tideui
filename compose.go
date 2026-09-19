package tideui

import (
	"image"
	"image/draw"
)

// Composite lays one picture over another, source over: where the top picture is
// transparent the bottom one shows through, and where it is solid the top one wins.
// It is how a radar frame - which is nothing at all except where it rains - is drawn
// over a map, and it is the whole reason fetching a basemap is worth anything.
//
// The result is the base's size. A nil base is the picture on top, which is the
// everyday case when there is no backdrop. A top picture that is not the base's size
// is a caller's bug, and the *top* picture is returned: it is the one carrying the
// data, and a panel that briefly draws yesterday's backdrop on its own is worse than
// one that briefly draws its data without one.
func Composite(base, over image.Image) image.Image {
	switch {
	case base == nil:
		return over
	case over == nil:
		return base
	}
	baseBounds, overBounds := base.Bounds(), over.Bounds()
	if baseBounds.Dx() != overBounds.Dx() || baseBounds.Dy() != overBounds.Dy() {
		return over
	}
	result := image.NewRGBA(baseBounds)
	draw.Draw(result, baseBounds, base, baseBounds.Min, draw.Src)
	// draw.Over is the standard library's own source-over, which is the same
	// arithmetic every picture in Go is composited with - one fewer formula to get
	// subtly wrong, and it already knows the difference between a picture that
	// carries straight alpha and one that carries premultiplied alpha, which the
	// radar's PNGs and the basemap's JPEGs are on either side of.
	draw.Draw(result, baseBounds, over, overBounds.Min, draw.Over)
	return result
}
