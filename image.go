package tideui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi/kitty"
)

// MapFrame is one picture of one place: the picture, the moment it describes,
// where in it the reader's own coordinate is, and what a pixel of it is worth on
// the ground. It lives here rather than in provider because it is a rendering
// model, the same way WeatherData does, and because a panel needs it to draw
// without knowing where it came from - but a panel can compute neither of the last
// two from the picture alone, so they travel with it.
type MapFrame struct {
	Time  time.Time
	Image image.Image
	// Centre is the pixel the coordinate sits at. It is the middle of the picture
	// only when the block of tiles is odd-sized, which is why it is carried.
	Centre image.Point
	// KilometresPerPixel is the ground distance one pixel covers.
	KilometresPerPixel float64
}

// RadarFrame is a MapFrame: a radar picture is a map picture that changes every
// few minutes. The basemap under it is the same model with a stiller picture.
type RadarFrame = MapFrame

// imageRamp is what a terminal with no colour gets: brightness, coarse enough to
// read a shape in, because a panel that draws nothing at all looks broken.
const imageRamp = " .:-=+*#%@"

// RenderImage draws an image inside a box of cells. Which way depends on the
// terminal: kitty gets the picture itself, drawn by the terminal at its own pixel
// resolution through placeholder cells; anything else gets coloured half-blocks
// ("▀" with the top sample as foreground and the bottom as background), which is
// one sample across and two down per cell - enough for a radar blob, a heat map
// or a chart; and a terminal with no colour at all gets a brightness ramp.
//
// The picture keeps its aspect and is centred in width, sized by the cell's own
// shape (Renderer.CellAspect) because a stretched picture lies about distance.
// Samples that are
// transparent show the panel background through them, and a picture that cannot
// be drawn is never a hole.
func (r Renderer) RenderImage(img image.Image, width, maxHeight int) string {
	if img == nil || width <= 0 || maxHeight <= 0 {
		return ""
	}
	source := img.Bounds()
	cells, rows := fitCells(source, width, maxHeight, r.CellAspect)
	if cells <= 0 || rows <= 0 {
		return ""
	}
	bg := r.Styles.Workspace.Bg
	switch {
	case placeholderCells(os.Getenv, activeColorProfile()):
		return r.renderImagePlaceholders(img, cells, rows, width, bg)
	case activeColorProfile() == colorprofile.ASCII:
		return r.renderImageRamp(img, source, cells, rows, width, bg)
	default:
		return r.renderImageHalfBlocks(img, cells, rows, width, bg)
	}
}

// renderImageHalfBlocks is the baseline every terminal gets: two samples of
// vertical resolution per cell, in colour.
func (r Renderer) renderImageHalfBlocks(img image.Image, cells, rows, width int, bg lipgloss.Color) string {
	source := img.Bounds()
	bgColour := RGBAOf(bg)
	left := (width - cells) / 2
	pad := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", max(0, left)))
	lines := make([]string, 0, rows)
	for cy := 0; cy < rows; cy++ {
		var line strings.Builder
		line.WriteString(pad)
		for cx := 0; cx < cells; cx++ {
			top := sampleRegion(img, halfCell(source, cells, rows, cx, cy, 0), bgColour)
			bottom := sampleRegion(img, halfCell(source, cells, rows, cx, cy, 1), bgColour)
			line.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(hexColour(top))).
				Background(lipgloss.Color(hexColour(bottom))).
				Render("▀"))
		}
		lines = append(lines, line.String())
	}
	return r.RenderLines(lines, width, bg)
}

// fitCells is the size in cells an image takes inside a box: as large as fits
// while keeping its shape. A cell is cellAspect times taller than it is wide, so
// a picture of aspect sw:sh needs width*sh/(cellAspect*sw) rows at that width.
func fitCells(source image.Rectangle, width, maxHeight int, cellAspect float64) (int, int) {
	sw, sh := source.Dx(), source.Dy()
	aspect := normalisedCellAspect(cellAspect)
	if sw <= 0 || sh <= 0 || width <= 0 || maxHeight <= 0 {
		return 0, 0
	}
	// Rounded rather than ceiled: half a row of the picture is less wrong than a
	// row of distortion, and a square picture is the common case.
	rows := int(math.Round(float64(width) * float64(sh) / (aspect * float64(sw))))
	if rows < 1 {
		rows = 1
	}
	if rows > maxHeight {
		rows = maxHeight
		cells := int(math.Round(aspect * float64(rows) * float64(sw) / float64(sh)))
		width = min(width, max(1, cells))
	}
	return max(1, width), max(1, rows)
}

// halfCell is the source rectangle one half of a cell covers: the top half of
// cell (cx, cy) when half is 0, the bottom half when it is 1. It is clamped to
// the source, so a box larger than the picture repeats edge samples instead of
// reading outside.
func halfCell(source image.Rectangle, cells, rows, cx, cy, half int) image.Rectangle {
	sw, sh := source.Dx(), source.Dy()
	x0 := source.Min.X + cx*sw/cells
	x1 := source.Min.X + (cx+1)*sw/cells
	y0 := source.Min.Y + (cy*2+half)*sh/(rows*2)
	y1 := source.Min.Y + (cy*2+half+1)*sh/(rows*2)
	clamp := func(v, low, high int) int { return min(max(v, low), high) }
	return image.Rect(
		clamp(x0, source.Min.X, source.Max.X),
		clamp(y0, source.Min.Y, source.Max.Y),
		clamp(max(x1, x0+1), source.Min.X, source.Max.X),
		clamp(max(y1, y0+1), source.Min.Y, source.Max.Y),
	)
}

// sampleRegion is the average colour of the source region one half-cell covers,
// composited onto bg. RGBA() is premultiplied, so the colour is the sum divided
// by the alpha it was multiplied by - averaging the premultiplied values would
// darken every partly transparent pixel, which is what a radar tile is made of.
func sampleRegion(img image.Image, region image.Rectangle, bg color.RGBA) color.RGBA {
	var sumR, sumG, sumB, sumA float64
	count := 0
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			cr, cg, cb, ca := img.At(x, y).RGBA()
			sumR += float64(cr) / 65535
			sumG += float64(cg) / 65535
			sumB += float64(cb) / 65535
			sumA += float64(ca) / 65535
			count++
		}
	}
	if count == 0 {
		return bg
	}
	n := float64(count)
	alpha := sumA / n
	var red, green, blue float64
	if sumA > 0 {
		red, green, blue = sumR/sumA, sumG/sumA, sumB/sumA
	}
	mix := func(front, back float64) uint8 {
		return uint8(255 * clamp01(front*alpha+back*(1-alpha)))
	}
	return color.RGBA{
		R: mix(red, float64(bg.R)/255),
		G: mix(green, float64(bg.G)/255),
		B: mix(blue, float64(bg.B)/255),
		A: 255,
	}
}

// renderImageRamp draws the same cells with no colour at all, which is the one
// case lipgloss cannot downgrade for us.
func (r Renderer) renderImageRamp(img image.Image, source image.Rectangle, cells, rows, width int, bg lipgloss.Color) string {
	style := lipgloss.NewStyle().Background(bg).Foreground(r.Styles.Workspace.BodyFg)
	left := (width - cells) / 2
	lines := make([]string, 0, rows)
	for cy := 0; cy < rows; cy++ {
		var line strings.Builder
		line.WriteString(strings.Repeat(" ", max(0, left)))
		for cx := 0; cx < cells; cx++ {
			top := sampleRegion(img, halfCell(source, cells, rows, cx, cy, 0), color.RGBA{A: 255})
			bottom := sampleRegion(img, halfCell(source, cells, rows, cx, cy, 1), color.RGBA{A: 255})
			luma := (0.2126*float64(top.R) + 0.7152*float64(top.G) + 0.0722*float64(top.B) +
				0.2126*float64(bottom.R) + 0.7152*float64(bottom.G) + 0.0722*float64(bottom.B)) / 2
			// luma is 0..255, and the ramp is shortest at the dark end, so the
			// index is scaled - clamped, because a 255 lands one past the end.
			glyph := imageRamp[min(len(imageRamp)-1, int(luma)*len(imageRamp)/256)]
			line.WriteString(style.Render(string(glyph)))
		}
		lines = append(lines, line.String())
	}
	return r.RenderLines(lines, width, bg)
}

// hexColour is a lipgloss colour for a sampled pixel.
func hexColour(c color.RGBA) string {
	const digits = "0123456789abcdef"
	return string([]byte{
		'#', digits[c.R>>4], digits[c.R&0xF],
		digits[c.G>>4], digits[c.G&0xF],
		digits[c.B>>4], digits[c.B&0xF],
	})
}

// RGBAOf reads a theme colour back as RGBA, so a panel background can be
// composited onto. hexToRGB is the helper the contrast code already uses
// (color.go:45) and it returns 0..1 channels, or ok false for a named colour.
func RGBAOf(c lipgloss.Color) color.RGBA {
	red, green, blue, ok := hexToRGB(c)
	if !ok {
		return color.RGBA{A: 255}
	}
	return color.RGBA{
		R: uint8(math.Round(clamp01(red) * 255)),
		G: uint8(math.Round(clamp01(green) * 255)),
		B: uint8(math.Round(clamp01(blue) * 255)),
		A: 255,
	}
}

// placeholderCells reports whether this terminal can be drawn with Unicode
// placeholder cells: a picture transmitted once and then positioned by ordinary
// cells. Two terminals are known to implement it here - kitty and Ghostty - and
// each is recognised by the environment it sets, because a placeholder in a
// terminal that does not understand one is a screenful of tofu.
//
// A multiplexer is refused even when the terminal underneath supports the
// protocol: the cells mean nothing to tmux, so it would pass the codepoints
// through instead of the picture. A 24-bit foreground is required too, because
// that colour is where the image id travels.
func placeholderCells(getenv func(string) string, profile colorprofile.Profile) bool {
	if profile != colorprofile.TrueColor {
		return false
	}
	if getenv("TMUX") != "" {
		return false
	}
	switch {
	case getenv("KITTY_WINDOW_ID") != "", getenv("TERM") == "xterm-kitty":
		return true
	case getenv("TERM_PROGRAM") == "ghostty", getenv("TERM") == "xterm-ghostty":
		return true
	}
	return false
}

// transmitted is the bookkeeping behind "transmit once": kitty is told about a
// picture the first time a frame draws it, and every frame after that sends only
// the cells that place it. An entry holds the image itself, so its address cannot
// be handed to a different picture while a placement for it may still be on
// screen; the oldest go once no terminal could still be showing them.
var transmitted = struct {
	sync.Mutex
	next    uint32
	byImage map[uintptr]*transmission
	order   []uintptr
}{byImage: map[uintptr]*transmission{}}

type transmission struct {
	id   uint32
	sent bool
	// image is held so its address is not reused by another picture.
	image image.Image
}

// maxTransmissions bounds the registry. A panel drawing a fresh picture every
// five minutes stays far below it, and everything past it is off screen already.
const maxTransmissions = 32

// transmissionFor is the id a picture is transmitted under, and whether this call
// is the one that has to send it.
func transmissionFor(img image.Image) (id uint32, send bool) {
	if reflect.ValueOf(img).Kind() != reflect.Pointer {
		// A value type has no identity to key on, so it is sent every time rather
		// than sharing an id with a picture it is not.
		transmitted.Lock()
		defer transmitted.Unlock()
		transmitted.next++
		return transmitted.next, true
	}
	key := reflect.ValueOf(img).Pointer()

	transmitted.Lock()
	defer transmitted.Unlock()
	entry, ok := transmitted.byImage[key]
	if !ok {
		transmitted.next++
		entry = &transmission{id: transmitted.next, image: img}
		transmitted.byImage[key] = entry
		transmitted.order = append(transmitted.order, key)
		for len(transmitted.order) > maxTransmissions {
			oldest := transmitted.order[0]
			transmitted.order = transmitted.order[1:]
			delete(transmitted.byImage, oldest)
		}
	}
	if entry.sent {
		return entry.id, false
	}
	entry.sent = true
	return entry.id, true
}

// renderImagePlaceholders transmits the picture once and then draws it as
// placeholder cells: U+10EEEE with the diacritics that name a row and a column
// inside the image, and the image id in the cell's foreground colour. Those cells
// are ordinary characters, so everything the dashboard does to text - panes,
// zoom, padding, the background blend - moves the picture with them.
func (r Renderer) renderImagePlaceholders(img image.Image, cells, rows, width int, bg lipgloss.Color) string {
	id, send := transmissionFor(img)
	var out strings.Builder
	if send {
		options := &kitty.Options{
			// Transmit *and display*, with U=1 so the display is a virtual
			// placement rather than anything drawn at the cursor. That looks
			// redundant next to a bare transmit, and it is not: kitty creates the
			// virtual placement for either action, but Ghostty only does it in the
			// display path - so a=t,U=1 leaves an image stored and unplaceable,
			// and every placeholder cell draws nothing. a=T,U=1 is what kitty's
			// own tools send, and both terminals place it.
			Action:           kitty.TransmitAndPut,
			Quite:            2, // no OK or error replies on the wire
			ID:               int(id),
			Format:           kitty.PNG,
			Transmission:     kitty.Direct,
			Columns:          cells,
			Rows:             rows,
			VirtualPlacement: true, // placed by the cells below, not by the cursor
		}
		if err := kitty.EncodeGraphics(&out, img, options); err != nil {
			// A picture that cannot be transmitted is drawn the way every other
			// terminal gets it, never left as a hole.
			return r.renderImageHalfBlocks(img, cells, rows, width, bg)
		}
	}

	// The id travels as a 24-bit foreground colour: that is how kitty knows which
	// picture a placeholder cell belongs to.
	cell := lipgloss.NewStyle().Foreground(lipgloss.Color(imageIDColour(id)))
	left := (width - cells) / 2
	pad := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", max(0, left)))
	lines := make([]string, 0, rows)
	for row := 0; row < rows; row++ {
		var line strings.Builder
		line.WriteString(pad)
		for col := 0; col < cells; col++ {
			line.WriteString(cell.Render(string(kitty.Placeholder) +
				string(kitty.Diacritic(row)) + string(kitty.Diacritic(col))))
		}
		lines = append(lines, line.String())
	}
	// The transmit goes in front of the cells in the same string. It is an APC
	// sequence, so it is zero cells wide: the terminal reads it and everything
	// that measures or moves text - the blend, the pane padding, the renderer -
	// carries it along with the row it belongs to.
	return out.String() + r.RenderLines(lines, width, bg)
}

// imageIDColour is an image id as the colour a placeholder cell carries.
func imageIDColour(id uint32) string { return fmt.Sprintf("#%06x", id&0xFFFFFF) }
