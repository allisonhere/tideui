//go:build unix

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
