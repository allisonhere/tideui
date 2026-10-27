//go:build !unix

package tideui

import "os"

// CellSizeOf reports 0, 0 where there is no winsize ioctl to ask: the terminal
// cannot say, so callers fall back to their default cell.
func CellSizeOf(file *os.File) (width, height float64) {
	return 0, 0
}
