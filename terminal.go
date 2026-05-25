package tideui

import "fmt"

// TerminalBackgroundSequences returns OSC strings; the consumer decides where to write them.
func TerminalBackgroundSequences(theme Theme) (set string, reset string) {
	if theme.Bg == "" {
		return "", ""
	}
	return fmt.Sprintf("\x1b]11;%s\x07", string(theme.Bg)), "\x1b]111\x07"
}
