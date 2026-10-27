package tideui

import "os"

// CellAspectOf is CellSizeOf as the ratio an image is sized by: how many times
// taller a cell is than it is wide. Most fonts land near 2 (a 7x15 cell, an 8x17
// cell), which is what a zero here means: a pty with no emulator behind it, a
// window whose pixel size is not known yet, or a terminal that does not report
// pixels at all.
func CellAspectOf(file *os.File) float64 {
	width, height := CellSizeOf(file)
	if width <= 0 || height <= 0 {
		return 0
	}
	return height / width
}
