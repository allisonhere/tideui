package tideui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// This file holds small, reusable information primitives shared by dashboard
// widgets. Like the rest of the chrome layer they are theme-aware, density
// aware, and stay bounded to the width they are given.

// StatusDot is a coloured status marker with an always-present text label, so
// meaning never depends on colour alone.
type StatusDot struct {
	Label string
	Tone  Tone
}

// RenderStatusDot renders a status dot and optional label over bg.
func (r Renderer) RenderStatusDot(d StatusDot, bg lipgloss.Color) string {
	glyph := "●"
	if r.Styles.PlainUI {
		glyph = "o"
	}
	dot := lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(d.Tone)).Render(glyph)
	if d.Label == "" {
		return dot
	}
	label := lipgloss.NewStyle().Background(bg).Foreground(r.Styles.Workspace.BodyFg).Render(" " + d.Label)
	return dot + label
}

// StatusKind is the shared dashboard status vocabulary. Health and liveness
// are shown with a consistent glyph, tone, and word everywhere, so the same
// state reads the same way in Services, System, Network, and future widgets —
// and never depends on colour alone.
type StatusKind int

const (
	// StatusHealthy is nominal.
	StatusHealthy StatusKind = iota
	// StatusWarning needs attention but still works.
	StatusWarning
	// StatusError has failed.
	StatusError
	// StatusStopped is intentionally not running.
	StatusStopped
	// StatusActive is running and doing work.
	StatusActive
	// StatusStale has not reported recently.
	StatusStale
	// StatusUpdating is in a transitional state.
	StatusUpdating
)

// Label returns the lowercase word for a status.
func (s StatusKind) Label() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusWarning:
		return "warning"
	case StatusError:
		return "error"
	case StatusStopped:
		return "stopped"
	case StatusActive:
		return "active"
	case StatusStale:
		return "stale"
	case StatusUpdating:
		return "updating"
	default:
		return "unknown"
	}
}

// Tone maps a status onto the semantic tone palette.
func (s StatusKind) Tone() Tone {
	switch s {
	case StatusHealthy, StatusActive:
		return ToneGood
	case StatusWarning, StatusUpdating:
		return ToneWarning
	case StatusError:
		return ToneDanger
	default:
		return ToneMuted
	}
}

// Glyph returns the status marker, degrading to ASCII on plain terminals.
func (s StatusKind) Glyph(plain bool) string {
	switch s {
	case StatusHealthy:
		if plain {
			return "o"
		}
		return "●"
	case StatusWarning:
		if plain {
			return "!"
		}
		return "▲"
	case StatusError:
		if plain {
			return "x"
		}
		return "✕"
	case StatusStopped:
		if plain {
			return "-"
		}
		return "■"
	case StatusActive:
		if plain {
			return "*"
		}
		return "◉"
	case StatusStale:
		if plain {
			return "."
		}
		return "◌"
	case StatusUpdating:
		if plain {
			return "~"
		}
		return "◐"
	default:
		return "?"
	}
}

// RenderStatus renders a status marker with an optional word.
func (r Renderer) RenderStatus(kind StatusKind, label string, bg lipgloss.Color) string {
	glyph := lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(kind.Tone())).
		Render(kind.Glyph(r.Styles.PlainUI))
	if label == "" {
		return glyph
	}
	return glyph + lipgloss.NewStyle().Background(bg).Foreground(r.Styles.Workspace.BodyFg).
		Render(" "+label)
}

// StatusBadge turns a status into a header badge (label + tone, no glyph so it
// stays quiet next to a title).
func StatusBadge(kind StatusKind, label string) Badge {
	if label == "" {
		label = kind.Label()
	}
	return Badge{Text: label, Tone: kind.Tone()}
}

// StatValue is a headline figure with an optional unit, label, and trend.
type StatValue struct {
	Value string
	Unit  string
	Label string
	Tone  Tone
	Trend float64 // percentage change; 0 hides the indicator
}

// RenderStatValue renders a headline figure over bg.
func (r Renderer) RenderStatValue(s StatValue, bg lipgloss.Color) string {
	ws := r.Styles.Workspace
	value := lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(s.Tone)).Bold(true).
		Render(s.Value + s.Unit)
	if s.Trend != 0 {
		value += lipgloss.NewStyle().Background(bg).Render(" ") + r.RenderTrend(s.Trend, "%", bg)
	}
	if s.Label != "" {
		value += lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).Render("  " + s.Label)
	}
	return value
}

// RenderTrend renders a signed change with a direction arrow and semantic tone.
func (r Renderer) RenderTrend(change float64, unit string, bg lipgloss.Color) string {
	arrow, tone := "▲", ToneGood
	switch {
	case change > 0:
	case change < 0:
		arrow, tone = "▼", ToneDanger
	default:
		arrow, tone = "■", ToneMuted
	}
	if r.Styles.PlainUI {
		switch {
		case change > 0:
			arrow = "^"
		case change < 0:
			arrow = "v"
		default:
			arrow = "-"
		}
	}
	text := fmt.Sprintf("%s%.1f%s", arrow, math.Abs(change), unit)
	return lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(tone)).Render(text)
}

// BarGauge is a labelled progress bar.
type BarGauge struct {
	Fraction float64
	Width    int
	Tone     Tone
	Label    string
}

// RenderBarGauge renders an optional label followed by a themed progress bar.
func (r Renderer) RenderBarGauge(g BarGauge, bg lipgloss.Color) string {
	bar := r.RenderProgressBar(ProgressBar{Fraction: g.Fraction, Width: g.Width, Tone: g.Tone}, bg)
	if g.Label == "" {
		return bar
	}
	label := lipgloss.NewStyle().Background(bg).Foreground(r.Styles.Workspace.BodyMutedFg).Render(g.Label + " ")
	return label + bar
}

// MiniCalendar is a compact month grid.
type MiniCalendar struct {
	Year      int
	Month     time.Month
	Highlight int
	Width     int
	// Marked highlights days that have events, so a sparse calendar still
	// shows where to look.
	Marked map[int]bool
}

// RenderMiniCalendar renders a Monday-first month grid, highlighting a day.
func (r Renderer) RenderMiniCalendar(c MiniCalendar, bg lipgloss.Color) string {
	if c.Month < time.January || c.Month > time.December {
		c.Month = time.January
	}
	if c.Year == 0 {
		c.Year = time.Now().Year()
	}
	ws := r.Styles.Workspace
	header := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Bold(true).
		Render(fmt.Sprintf("%s %d", c.Month.String(), c.Year))
	weekday := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	weekHeader := lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).
		Render(strings.Join(weekday, " "))

	first := time.Date(c.Year, c.Month, 1, 0, 0, 0, 0, time.UTC)
	offset := (int(first.Weekday()) + 6) % 7 // Monday-first
	days := time.Date(c.Year, c.Month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	var rows []string
	for week := 0; week < 6; week++ {
		cells := make([]string, 7)
		empty := true
		for slot := 0; slot < 7; slot++ {
			day := week*7 + slot - offset + 1
			if day < 1 || day > days {
				cells[slot] = "  "
				continue
			}
			empty = false
			text := fmt.Sprintf("%2d", day)
			switch {
			case day == c.Highlight:
				cells[slot] = lipgloss.NewStyle().Background(ws.SelectionBg).
					Foreground(ws.SelectionFg).Bold(true).Render(text)
			case c.Marked[day]:
				cells[slot] = lipgloss.NewStyle().Background(bg).
					Foreground(ws.FrameActive).Bold(true).Render(text)
			default:
				cells[slot] = lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).Render(text)
			}
		}
		if empty {
			continue
		}
		rows = append(rows, strings.Join(cells, " "))
	}
	lines := append([]string{header, weekHeader}, rows...)
	return r.dashBlock(lines, c.Width, bg)
}

// RenderLines bounds a block of lines to a width: each line is truncated with
// an ellipsis and padded, keeping its own styling and its background
// continuous. Every panel body ends here, which is what guarantees a panel
// never overflows its pane.
//
// It is exported because a panel does not have to live in this package: one
// rendered from a document supplied by an external program needs the same
// guarantee, and reimplementing it elsewhere would mean two subtly different
// notions of "fits".
func (r Renderer) RenderLines(lines []string, width int, bg lipgloss.Color) string {
	for i, line := range lines {
		lines[i] = padStyled(ansi.Truncate(line, width, "…"), width, bg)
	}
	return strings.Join(lines, "\n")
}

// dashBlock is the in-package spelling of RenderLines.
func (r Renderer) dashBlock(lines []string, width int, bg lipgloss.Color) string {
	return r.RenderLines(lines, width, bg)
}

// dashPair renders a muted label column and a value.
func (r Renderer) dashPair(label, value string, labelWidth int, bg lipgloss.Color) string {
	ws := r.Styles.Workspace
	labelCell := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).
		Render(padRight(label, labelWidth))
	valueCell := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Render(value)
	return labelCell + lipgloss.NewStyle().Background(bg).Render("  ") + valueCell
}

// dashTime formats a time as 24-hour HH:MM.
func dashTime(t time.Time) string { return t.Format("15:04") }
