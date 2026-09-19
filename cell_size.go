package tideui

import (
	"os"

	"golang.org/x/sys/unix"
)

// CellSizeOf measures a terminal's cell in pixels, and reports 0, 0 when the
// terminal will not say: the ioctl reports the window in pixels in the same call
// that reports its size in cells, and a pty with no emulator behind it reports
// zeros.
func CellSizeOf(file *os.File) (width, height float64) {
	if file == nil {
		return 0, 0
	}
	size, err := unix.IoctlGetWinsize(int(file.Fd()), unix.TIOCGWINSZ)
	if err != nil || size == nil || size.Col == 0 || size.Row == 0 {
		return 0, 0
	}
	if size.Xpixel == 0 || size.Ypixel == 0 {
		return 0, 0
	}
	return float64(size.Xpixel) / float64(size.Col), float64(size.Ypixel) / float64(size.Row)
}

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
