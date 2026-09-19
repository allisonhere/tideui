package tideui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// First-party dashboard widgets. Each renders a data model to a bounded,
// themed block; none of them fetch data. Applications supply the model from a
// provider (or a demo feed), so acquisition and rendering stay decoupled.

// RenderWeather renders the compact weather summary.
func (r Renderer) RenderWeather(w WeatherData, width int) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	lines := []string{
		r.weatherHeadline(w, bg),
		lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).
			Render(r.weatherRangeText(w, width)),
	}
	lines = append(lines, r.weatherMetricRows(w, width, bg)...)
	// Hourly detail is useful on a roomy card, but it is the first thing to
	// remove when the workspace has reflowed into narrow columns.
	if len(w.Hourly) > 0 && width >= 28 {
		lines = append(lines, "", r.renderHourly(w.Hourly, width, bg))
	}
	return r.dashBlock(lines, width, bg)
}

// weatherHeadline renders the current temperature with a condition glyph and
// label, colouring the glyph by the coarse kind.
func (r Renderer) weatherHeadline(w WeatherData, bg lipgloss.Color) string {
	ws := r.Styles.Workspace
	kind := w.EffectiveKind()
	headline := r.RenderStatValue(StatValue{Value: fmt.Sprintf("%d°", w.Temperature), Unit: w.Unit, Tone: ToneAccent}, bg)
	if glyph := kind.Glyph(r.Styles.PlainUI); glyph != "" {
		headline += lipgloss.NewStyle().Background(bg).Render(" ") +
			lipgloss.NewStyle().Background(bg).Foreground(ws.WeatherColor(kind)).Bold(true).Render(glyph+" ")
	}
	if w.Condition != "" {
		headline += lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).Render(w.Condition)
	}
	return headline
}

// weatherRangeText renders the high/low line, appending feels-like when known
// and a flame once the feels-like temperature is hot. Facts that do not fit the
// panel are dropped whole, so the line never ends in a half-written figure.
func (r Renderer) weatherRangeText(w WeatherData, width int) string {
	facts := []string{fmt.Sprintf("H %d°", w.High), fmt.Sprintf("L %d°", w.Low)}
	if w.HasFeelsLike {
		feels := fmt.Sprintf("Feels %d°", w.FeelsLike)
		if w.FeelsLike > 90 {
			feels += " " + r.hotGlyph()
		}
		facts = append(facts, feels)
	}
	return joinFit(facts, weatherGap, width)
}

// weatherMetricRows renders the rain and wind readings. They share one row at
// the same gap as the facts line above when the panel is wide enough, and stack
// into the shared label column when it is not.
func (r Renderer) weatherMetricRows(w WeatherData, width int, bg lipgloss.Color) []string {
	rain := r.dashPair(r.rainGlyph()+" Rain", fmt.Sprintf("%d%%", w.RainChance), weatherLabelWidth, bg)
	wind := r.dashPair(r.windGlyph()+" Wind", fmt.Sprintf("%d %s", w.WindSpeed, w.WindUnit), weatherLabelWidth, bg)
	if lipgloss.Width(rain)+weatherGap+lipgloss.Width(wind) <= width {
		gap := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", weatherGap))
		return []string{rain + gap + wind}
	}
	return []string{rain, wind}
}

// hotGlyph marks a hot feels-like temperature. The emoji is two cells wide and
// the width table agrees, so it does not shift the rest of the line.
func (r Renderer) hotGlyph() string {
	if r.Styles.PlainUI {
		return "!"
	}
	return "\U0001F525" // fire
}

// rainGlyph and windGlyph label the rain and wind lines.
func (r Renderer) rainGlyph() string {
	if r.Styles.PlainUI {
		return "*"
	}
	return "\U0001F4A7" // droplet
}

func (r Renderer) windGlyph() string {
	if r.Styles.PlainUI {
		return "~"
	}
	return "\U0001F4A8" // dash
}

// forecastCondition prefixes a condition with its glyph when one is known.
func (r Renderer) forecastCondition(condition string) string {
	if condition == "" {
		return "—"
	}
	kind := WeatherKindFromCondition(condition)
	if kind == WeatherUnknown {
		return condition
	}
	return kind.Glyph(r.Styles.PlainUI) + " " + condition
}

// Weather layout constants: the label column both weather widgets share, and
// the gap between facts on a run-on line.
const (
	weatherLabelWidth = 7 // fits "Updated", the longest label
	weatherGap        = 3
)

// joinFit joins parts with gap spaces, keeping only the leading parts that fit
// in width. The first part is always kept so the line is never empty.
func joinFit(parts []string, gap, width int) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	spacer := strings.Repeat(" ", gap)
	for _, part := range parts[1:] {
		if lipgloss.Width(out)+gap+lipgloss.Width(part) > width {
			break
		}
		out += spacer + part
	}
	return out
}

// renderHourly lays the short forecast out as one grouped strip so it reads as
// a single element rather than a stack of rows. Only whole segments are kept,
// and the slack is shared between them so the strip reads as an even row.
func (r Renderer) renderHourly(points []ForecastPoint, width int, bg lipgloss.Color) string {
	ws := r.Styles.Workspace
	segments := make([]string, 0, len(points))
	for i, point := range points {
		if i >= 5 {
			break
		}
		label := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).Render(point.Label)
		temp := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Bold(true).
			Render(fmt.Sprintf("%d°", point.Temperature))
		segment := label + " " + temp
		if kind := WeatherKindFromCondition(point.Condition); kind != WeatherUnknown {
			segment += lipgloss.NewStyle().Background(bg).Render(" ") +
				lipgloss.NewStyle().Background(bg).Foreground(ws.WeatherColor(kind)).Render(kind.Glyph(r.Styles.PlainUI))
		}
		segments = append(segments, segment)
	}
	const maxGap = 6
	// Drop the segments that would only fit in part.
	used, fit := 0, 0
	for i, segment := range segments {
		next := used + lipgloss.Width(segment)
		if i > 0 {
			next += weatherGap
		}
		if next > width {
			break
		}
		used, fit = next, i+1
	}
	segments = segments[:fit]
	if len(segments) < 2 {
		return strings.Join(segments, "")
	}
	gap := min(weatherGap+(width-used)/(len(segments)-1), maxGap)
	return strings.Join(segments, lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", gap)))
}

// RenderWeatherDetail renders the extended forecast.
func (r Renderer) RenderWeatherDetail(w WeatherData, width int) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	lines := []string{
		r.weatherHeadline(w, bg),
		lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).
			Render(r.weatherRangeText(w, width)),
	}
	lines = append(lines, r.weatherMetricRows(w, width, bg)...)
	if w.Location != "" {
		lines = append(lines, r.dashPair("Place", w.Location, weatherLabelWidth, bg))
	}
	if !w.Updated.IsZero() {
		lines = append(lines, r.dashPair("Updated", w.Updated.Format("15:04"), weatherLabelWidth, bg))
	}
	if len(w.Daily) > 0 {
		lines = append(lines, r.RenderSectionDivider(SectionDivider{Label: "FORECAST", Width: width}, bg))
		lines = append(lines, r.forecastRows(w.Daily, bg)...)
	}
	return r.dashBlock(lines, width, bg)
}

// forecastRows renders the daily forecast as a grid: a shared label column and
// a right-aligned temperature column, so the degrees line up whatever their
// digit count and the conditions all start at the same cell.
func (r Renderer) forecastRows(days []ForecastPoint, bg lipgloss.Color) []string {
	labelWidth, tempWidth := weatherLabelWidth, 0
	temps := make([]string, len(days))
	for i, day := range days {
		temps[i] = fmt.Sprintf("%d°", day.Temperature)
		labelWidth = max(labelWidth, lipgloss.Width(day.Label))
		tempWidth = max(tempWidth, lipgloss.Width(temps[i]))
	}
	rows := make([]string, 0, len(days))
	for i, day := range days {
		value := padLeft(temps[i], tempWidth) + "  " + r.forecastCondition(day.Condition)
		rows = append(rows, r.dashPair(day.Label, value, labelWidth, bg))
	}
	return rows
}

// RenderAgenda renders upcoming events grouped by day with the next event
// emphasised.
func (r Renderer) RenderAgenda(items []AgendaItem, now time.Time, width int) string {
	return r.renderAgenda(items, now, width, false)
}

// RenderAgendaDetail renders the agenda with locations and categories.
func (r Renderer) RenderAgendaDetail(items []AgendaItem, now time.Time, width int) string {
	return r.renderAgenda(items, now, width, true)
}

// RenderNotice renders a muted one-line message, for a panel that has nothing
// to show or whose source failed. Showing the failure beats an empty panel that
// looks like a quiet calendar.
func (r Renderer) RenderNotice(text string, width int) string {
	bg := r.Styles.Workspace.Bg
	line := lipgloss.NewStyle().Background(bg).Foreground(r.Styles.Workspace.HintFg).Render(text)
	return r.dashBlock([]string{line}, width, bg)
}

// RenderCalendar lays a month grid beside the day's agenda, highlighting the
// selected day. The grid covers the day's own month, so stepping the day across
// a month boundary turns the calendar page.
func (r Renderer) RenderCalendar(day time.Time, marked map[int]bool, items []AgendaItem, now time.Time, width int) string {
	return r.renderCalendar(day, marked, items, now, width, false)
}

// RenderCalendarDetail is RenderCalendar with the agenda's locations and
// categories shown.
func (r Renderer) RenderCalendarDetail(day time.Time, marked map[int]bool, items []AgendaItem, now time.Time, width int) string {
	return r.renderCalendar(day, marked, items, now, width, true)
}

const (
	calendarGridWidth = 20
	calendarGap       = 2
)

func (r Renderer) renderCalendar(day time.Time, marked map[int]bool, items []AgendaItem, now time.Time, width int, detail bool) string {
	// The grid needs its full width, and the agenda needs enough room to be
	// worth showing; below that the calendar collapses to the agenda alone.
	agendaWidth := width - calendarGridWidth - calendarGap
	if agendaWidth < 16 {
		return r.renderAgenda(items, now, width, detail)
	}
	bg := r.Styles.Workspace.Bg
	grid := strings.Split(r.RenderMiniCalendar(MiniCalendar{
		Year: day.Year(), Month: day.Month(), Highlight: day.Day(), Width: calendarGridWidth, Marked: marked,
	}, bg), "\n")
	agenda := strings.Split(r.renderAgenda(items, now, agendaWidth, detail), "\n")
	height := max(len(grid), len(agenda))
	padColumn := func(column []string, columnWidth int) []string {
		out := make([]string, height)
		for i := range out {
			if i < len(column) {
				out[i] = padStyled(column[i], columnWidth, bg)
			} else {
				out[i] = padStyled("", columnWidth, bg)
			}
		}
		return out
	}
	grid = padColumn(grid, calendarGridWidth)
	agenda = padColumn(agenda, agendaWidth)
	separator := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", calendarGap))
	for i := range grid {
		grid[i] += separator + agenda[i]
	}
	return strings.Join(grid, "\n")
}

func (r Renderer) renderAgenda(items []AgendaItem, now time.Time, width int, detail bool) string {
	bg := r.Styles.Workspace.Bg
	if len(items) == 0 {
		return r.RenderNotice("No upcoming events", width)
	}
	sorted := append([]AgendaItem(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Start.Before(sorted[j].Start) })

	// Widen the time column to fit "all-day" when the day has one; timed rows
	// right-align to it so every title starts in the same place.
	timeWidth := 0
	for _, item := range sorted {
		timeWidth = max(timeWidth, lipgloss.Width(agendaTime(item)))
	}

	next := ""
	if !detail {
		for _, item := range sorted {
			if agendaUpcoming(item, now) {
				next = agendaItemKey(item)
				break
			}
		}
	}

	maxItems := 10
	if r.Styles.Density.IsDense() {
		maxItems = 8
	}
	var lines []string
	shown := 0
	for _, group := range groupAgenda(sorted, now) {
		lines = append(lines, r.RenderSectionDivider(SectionDivider{Label: group.label, Width: width}, bg))
		for _, item := range group.items {
			if shown >= maxItems {
				break
			}
			lines = append(lines, r.renderAgendaItem(item, agendaItemKey(item) == next, detail, timeWidth, width, bg))
			shown++
		}
	}
	return r.dashBlock(lines, width, bg)
}

// agendaTime is the leading time cell: a clock time, or "all-day" for a
// whole-day event whose midnight start would otherwise read as 00:00.
func agendaTime(item AgendaItem) string {
	if item.AllDay {
		return "all-day"
	}
	return dashTime(item.Start)
}

// agendaEnd is when an event stops occupying the calendar. Timed events finish
// at End (or their start); whole-day events without an End run to next midnight.
func agendaEnd(item AgendaItem) time.Time {
	if !item.End.IsZero() {
		return item.End
	}
	if item.AllDay {
		return item.Start.AddDate(0, 0, 1)
	}
	return item.Start
}

// agendaUpcoming reports whether an event is still relevant, so an all-day or
// in-progress event counts as the next one rather than only future starts.
func agendaUpcoming(item AgendaItem, now time.Time) bool {
	return !item.Done && !agendaEnd(item).Before(now)
}

func agendaItemKey(item AgendaItem) string {
	return item.Start.Format(time.RFC3339) + item.Title
}

func (r Renderer) renderAgendaItem(item AgendaItem, isNext, detail bool, timeWidth, width int, bg lipgloss.Color) string {
	ws := r.Styles.Workspace
	timeText := agendaTime(item)
	timeStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).
		Width(timeWidth).Align(lipgloss.Right)
	titleStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg)
	dot := r.RenderStatusDot(StatusDot{Tone: item.Tone}, bg)
	if item.AllDay {
		timeStyle = timeStyle.Foreground(ws.SubtitleFg)
	}
	if item.Done {
		titleStyle = titleStyle.Foreground(ws.BodyDimmedFg)
		timeStyle = timeStyle.Foreground(ws.HintFg)
	}
	if isNext {
		titleStyle = titleStyle.Foreground(ws.FrameActive).Bold(true)
		dot = r.RenderStatusDot(StatusDot{Tone: ToneAccent}, bg)
	}
	left := timeStyle.Render(timeText) + lipgloss.NewStyle().Background(bg).Render("  ") +
		dot + lipgloss.NewStyle().Background(bg).Render(" ") + titleStyle.Render(item.Title)
	suffix := ""
	if isNext {
		suffix = r.RenderBadgeOn(NewBadge("next").WithTone(ToneAccent), bg)
	}
	if detail {
		meta := item.Category
		if item.Location != "" {
			if meta != "" {
				meta += " · "
			}
			meta += item.Location
		}
		if meta != "" {
			left += lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).Render("  " + meta)
		}
	}
	return alignRow(left, "", suffix, width)
}

type agendaGroup struct {
	label string
	items []AgendaItem
}

func groupAgenda(items []AgendaItem, now time.Time) []agendaGroup {
	var groups []agendaGroup
	index := map[string]int{}
	for _, item := range items {
		label := dayLabel(item.Start, now)
		position, ok := index[label]
		if !ok {
			position = len(groups)
			index[label] = position
			groups = append(groups, agendaGroup{label: label})
		}
		groups[position].items = append(groups[position].items, item)
	}
	return groups
}

func dayLabel(t, now time.Time) string {
	switch daysBetween(now, t) {
	case 0:
		return "TODAY"
	case 1:
		return "TOMORROW"
	case -1:
		return "YESTERDAY"
	default:
		return strings.ToUpper(t.Format("Mon Jan 2"))
	}
}

func daysBetween(a, b time.Time) int {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	a0 := time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC)
	b0 := time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	return int(b0.Sub(a0).Hours() / 24)
}

// RenderClock renders local time and world clocks. When the pane is wide enough
// the time is drawn as big digital rows; otherwise it falls back to text. The
// date carries the day-period and a sun/moon glyph, a day-progress gauge sits
// under it, and each world clock carries its own glyph.
func (r Renderer) RenderClock(c ClockData, width int) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	period, glyph := r.dayPeriod(c.Local)
	glyphColor := ws.WeatherSun
	if period == "night" {
		glyphColor = ws.SubtitleFg
	}
	timeStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.FrameActive).Bold(true)
	center := lipgloss.NewStyle().Background(bg).Width(width).Align(lipgloss.Center)
	var lines []string
	bigShown := false
	if big := r.bigTime(clockDigits(c.Local)); lipgloss.Width(big[0]) <= width {
		bigShown = true
		bigStyle := timeStyle.Width(width).Align(lipgloss.Center)
		for _, line := range big {
			lines = append(lines, bigStyle.Render(strings.TrimRight(line, " ")))
		}
		lines = append(lines, "") // spacer below the large clock
	} else {
		lines = append(lines, timeStyle.Render(clockTime(c.Local, c.Hour24)))
	}
	dateLine := lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).
		Render(c.Local.Format("Mon Jan 2")+" · "+period) +
		lipgloss.NewStyle().Background(bg).Foreground(glyphColor).Render(" "+glyph)
	if bigShown {
		lines = append(lines, center.Render(dateLine))
	} else {
		lines = append(lines, dateLine)
	}
	if barWidth := min(width-2, 18); barWidth >= 6 {
		fraction := dayFraction(c.Local)
		bar := r.RenderProgressBar(ProgressBar{Fraction: fraction, Width: barWidth, Tone: ToneAccent}, bg)
		pct := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).
			Render(fmt.Sprintf("  %d%%", int(math.Round(fraction*100))))
		if bigShown {
			lines = append(lines, center.Render(bar+pct))
		} else {
			lines = append(lines, bar+pct)
		}
	}
	if c.Location != "" {
		loc := lipgloss.NewStyle().Background(bg).Foreground(ws.HintFg).Render(c.Location)
		if bigShown {
			loc = center.Render(loc)
		}
		lines = append(lines, loc)
	}
	if len(c.Zones) > 0 {
		lines = append(lines, "")
		for _, zone := range c.Zones {
			label := zone.City
			if zone.Offset != "" {
				label = fmt.Sprintf("%s %s", zone.City, zone.Offset)
			}
			_, zoneGlyph := r.dayPeriod(zone.Time)
			lines = append(lines, r.dashPair(label, clockTime(zone.Time, c.Hour24)+" "+zoneGlyph, 12, bg))
		}
	}
	return r.dashBlock(lines, width, bg)
}

// RenderClockDetail renders the clock as a big digital time over a day-progress
// gauge, an analog face, the world clocks, and a month calendar.
func (r Renderer) RenderClockDetail(c ClockData, width int) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	period, _ := r.dayPeriod(c.Local)
	var parts []string

	timeStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.FrameActive).Bold(true)
	if big := r.bigTime(clockDigits(c.Local)); lipgloss.Width(big[0]) <= width {
		for _, line := range big {
			parts = append(parts, timeStyle.Render(line))
		}
	} else {
		parts = append(parts, timeStyle.Render(clockTime(c.Local, c.Hour24)))
	}
	parts = append(parts, lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).
		Render(c.Local.Format("Monday, Jan 2")+" · "+period))

	if barWidth := min(max(width-14, 0), 24); barWidth >= 6 {
		fraction := dayFraction(c.Local)
		bar := r.RenderProgressBar(ProgressBar{Fraction: fraction, Width: barWidth, Tone: ToneAccent}, bg)
		pct := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).
			Render(fmt.Sprintf("  %d%% of day", int(math.Round(fraction*100))))
		parts = append(parts, bar+pct)
	}

	if width >= 15 {
		for _, line := range r.analogClock(c.Local) {
			parts = append(parts, lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Render(line))
		}
	}

	if len(c.Zones) > 0 {
		parts = append(parts, "")
		for _, zone := range c.Zones {
			label := zone.City
			if zone.Offset != "" {
				label = fmt.Sprintf("%s %s", zone.City, zone.Offset)
			}
			_, zoneGlyph := r.dayPeriod(zone.Time)
			parts = append(parts, r.dashPair(label, clockTime(zone.Time, c.Hour24)+" "+zoneGlyph, 12, bg))
		}
	}

	calendarWidth := min(width, 21)
	if calendarWidth >= 15 {
		parts = append(parts, r.RenderMiniCalendar(MiniCalendar{
			Year: c.Local.Year(), Month: c.Local.Month(), Highlight: c.Local.Day(), Width: calendarWidth,
		}, bg))
	}
	return strings.Join(parts, "\n")
}

// clockTime formats a time as 24-hour ("15:04") or 12-hour ("3:04 PM").
func clockTime(t time.Time, hour24 bool) string {
	if hour24 {
		return t.Format("15:04")
	}
	return t.Format("3:04 PM")
}

// clockDigits formats a time without a meridiem, for the big digital face.
func clockDigits(t time.Time) string {
	hour := t.Hour()
	if hour > 12 {
		hour -= 12
	}
	if hour == 0 {
		hour = 12
	}
	return fmt.Sprintf("%d:%02d", hour, t.Minute())
}

// dayPeriod returns a short label and a sun/moon glyph for a time of day.
func (r Renderer) dayPeriod(t time.Time) (string, string) {
	label := "night"
	glyph := "\U0001F319\uFE0F" // crescent moon
	switch h := t.Hour(); {
	case h >= 5 && h < 12:
		label, glyph = "morning", "\U0001F31E\uFE0F" // sun with face
	case h >= 12 && h < 17:
		label, glyph = "afternoon", "\U0001F31E\uFE0F"
	case h >= 17 && h < 21:
		label, glyph = "evening", "\U0001F31E\uFE0F"
	}
	if r.Styles.PlainUI {
		if label == "night" {
			glyph = "z"
		} else {
			glyph = "*"
		}
	}
	return label, glyph
}

// dayFraction is how far through the day a time is, 0..1.
func dayFraction(t time.Time) float64 {
	return clamp01(float64(t.Hour()*60+t.Minute()) / (24 * 60))
}

// Two big-clock fonts. "dash" is the original 3-row seven-segment LED look;
// "block" is a heavier 5-row solid font. Both place each segment so digits
// read correctly (a "4" has its top-left vertical).
//
// The dash font is drawn on a connected box-drawing grid: a cell that carries
// both a bar and a vertical uses the matching corner or tee, so the segments
// join instead of floating next to each other. Digits are four cells wide: a
// terminal cell is about twice as tall as it is wide, so three rows of three
// cells renders a digit at half the proportions of a real one, which is what
// made the narrower font look stretched. The colon is narrower than a digit,
// as it is on a real clock face.
var (
	bigClockDash = map[rune][]string{
		'0': {"┌──┐", "│  │", "└──┘"},
		'1': {"   │", "   │", "   │"},
		'2': {"───┐", "┌──┘", "└───"},
		'3': {"───┐", "───┤", "───┘"},
		'4': {"│  │", "└──┤", "   │"},
		'5': {"┌───", "└──┐", "───┘"},
		'6': {"┌───", "├──┐", "└──┘"},
		'7': {"───┐", "   │", "   │"},
		'8': {"┌──┐", "├──┤", "└──┘"},
		'9': {"┌──┐", "└──┤", "───┘"},
		':': {"  ", " ●", " ●"},
	}
	bigClockBlock = map[rune][]string{
		'0': {"███", "█ █", "█ █", "█ █", "███"},
		'1': {"  █", "  █", "  █", "  █", "  █"},
		'2': {"███", "  █", "███", "█  ", "███"},
		'3': {"███", "  █", "███", "  █", "███"},
		'4': {"█ █", "█ █", "███", "  █", "  █"},
		'5': {"███", "█  ", "███", "  █", "███"},
		'6': {"███", "█  ", "███", "█ █", "███"},
		'7': {"███", "  █", "  █", "  █", "  █"},
		'8': {"███", "█ █", "███", "█ █", "███"},
		'9': {"███", "█ █", "███", "  █", "███"},
		':': {"   ", " █ ", "   ", " █ ", "   "},
	}
)

// bigTime renders "14:42" as large digits in the selected font.
func (r Renderer) bigTime(text string) []string {
	font := bigClockDash
	height := 3
	if r.Styles.ClockFont == ClockFontBlock {
		font = bigClockBlock
		height = 5
	}
	rows := make([]string, height)
	for i, ch := range text {
		glyph, ok := font[ch]
		if !ok || len(glyph) != height {
			// Hold a digit's worth of space so the rows stay in step.
			blank := strings.Repeat(" ", lipgloss.Width(font['0'][0]))
			glyph = make([]string, height)
			for j := range glyph {
				glyph[j] = blank
			}
		}
		for row := 0; row < height; row++ {
			if i > 0 {
				rows[row] += " "
			}
			rows[row] += glyph[row]
		}
	}
	return rows
}

// analogClock draws a small clock face: a rim, quarter marks, and hour and
// minute hands. Cells are about twice as tall as wide, so the radius is wider
// horizontally to keep the face round.
func (r Renderer) analogClock(t time.Time) []string {
	const cx, cy = 6, 3
	const rx, ry = 6, 3
	grid := make([][]rune, cy*2+1)
	for i := range grid {
		grid[i] = make([]rune, cx*2+1)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}
	put := func(x, y int, ch rune) {
		if y >= 0 && y < len(grid) && x >= 0 && x < len(grid[0]) {
			grid[y][x] = ch
		}
	}
	// Rim: mark every cell near the ellipse boundary, so it comes out even.
	for y := range grid {
		for x := range grid[y] {
			dx := float64(x-cx) / float64(rx)
			dy := float64(y-cy) / float64(ry)
			if d := math.Sqrt(dx*dx + dy*dy); d > 0.82 && d < 1.18 {
				grid[y][x] = '·'
			}
		}
	}
	for _, hour := range []int{0, 3, 6, 9} {
		a := float64(hour) * math.Pi / 6
		put(cx+int(math.Round(rx*math.Sin(a))), cy-int(math.Round(ry*math.Cos(a))), '•')
	}
	hand := func(angle, length float64, ch rune) {
		for step := 1; step <= 12; step++ {
			f := length * float64(step) / 12
			put(cx+int(math.Round(f*rx*math.Sin(angle))), cy-int(math.Round(f*ry*math.Cos(angle))), ch)
		}
	}
	hourAngle := (float64(t.Hour()%12) + float64(t.Minute())/60) / 12 * 2 * math.Pi
	minuteAngle := float64(t.Minute()) / 60 * 2 * math.Pi
	hand(minuteAngle, 0.85, '•')
	hand(hourAngle, 0.55, '●')
	put(cx, cy, '○')
	lines := make([]string, len(grid))
	for i, row := range grid {
		lines[i] = string(row)
	}
	return lines
}

// RenderSystem renders the system-health summary.
func (r Renderer) RenderSystem(m SystemMetrics, width int) string {
	bg := r.Styles.Workspace.Bg
	compact := width < 28
	total := metricTotal(width, 5, 5, width)
	rows := []MetricRow{
		{Label: "CPU", Value: fmt.Sprintf("%.0f%%", m.CPUPercent), Fraction: m.CPUPercent / 100,
			Spark: m.CPUSpark, Tone: toneForPercent(m.CPUPercent), LabelWidth: 5, ValueWidth: 5, TotalWidth: total},
		{Label: "MEM", Value: fmt.Sprintf("%.0f%%", m.MemoryPercent), Fraction: m.MemoryPercent / 100,
			Bar: true, Tone: toneForPercent(m.MemoryPercent), LabelWidth: 5, ValueWidth: 5, TotalWidth: total},
	}
	lines := make([]string, 0, len(rows)+3)
	for _, row := range rows {
		lines = append(lines, r.RenderMetricRow(row, bg))
	}
	lines = append(lines, r.dashPair("TEMP", fmt.Sprintf("%d°C", m.TemperatureC), 5, bg))
	if !compact {
		lines = append(lines, r.dashPair("LOAD", fmt.Sprintf("%.1f %.1f %.1f", m.Load[0], m.Load[1], m.Load[2]), 5, bg))
		lines = append(lines, r.dashPair("UP", formatUptime(m.Uptime), 5, bg))
	}
	return r.dashBlock(lines, width, bg)
}

// RenderSystemDetail renders per-core load, memory totals, and processes.
func (r Renderer) RenderSystemDetail(m SystemMetrics, width int) string {
	bg := r.Styles.Workspace.Bg
	lines := strings.Split(r.RenderSystem(m, width), "\n")
	if len(m.Cores) > 0 && width >= 24 {
		lines = append(lines, r.RenderSectionDivider(SectionDivider{Label: "CORES", Width: width}, bg))
		barWidth := max(4, width-10)
		for i, core := range m.Cores {
			lines = append(lines, r.dashPair(fmt.Sprintf("cpu%d", i), "", 5, bg)+
				r.RenderProgressBar(ProgressBar{Fraction: core / 100, Width: barWidth, Tone: toneForPercent(core)}, bg))
		}
	}
	memory := m.MemoryUsed
	if m.MemoryTotal != "" {
		memory = fmt.Sprintf("%s / %s", m.MemoryUsed, m.MemoryTotal)
	}
	if memory != "" {
		lines = append(lines, r.dashPair("Mem", memory, 5, bg))
	}
	if m.Processes > 0 {
		lines = append(lines, r.dashPair("Proc", fmt.Sprintf("%d", m.Processes), 5, bg))
	}
	return r.dashBlock(lines, width, bg)
}

// RenderGPU renders the GPU summary: utilisation with a short history, then
// memory, temperature, power and clock.
func (r Renderer) RenderGPU(m GPUMetrics, width int) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	if m.Name == "" {
		return r.dashBlock([]string{lipgloss.NewStyle().Background(bg).Foreground(ws.HintFg).Render("No GPU found")}, width, bg)
	}
	total := metricTotal(width, 5, 5, width)
	lines := []string{r.RenderMetricRow(MetricRow{
		Label: "GPU", Value: fmt.Sprintf("%.0f%%", m.BusyPercent), Fraction: m.BusyPercent / 100,
		Spark: m.BusySpark, Tone: toneForPercent(m.BusyPercent),
		LabelWidth: 5, ValueWidth: 5, TotalWidth: total,
	}, bg)}
	lines = append(lines, r.gpuMemoryLine(m, total, bg)...)
	if width < 30 {
		return r.dashBlock(lines, width, bg)
	}
	if m.TemperatureC > 0 {
		lines = append(lines, r.dashPair("TEMP", fmt.Sprintf("%d°C", m.TemperatureC), 5, bg))
	}
	if m.PowerWatts > 0 {
		lines = append(lines, r.dashPair("PWR", fmt.Sprintf("%.1f W", m.PowerWatts), 5, bg))
	}
	if m.ClockMHz > 0 {
		lines = append(lines, r.dashPair("CLK", fmt.Sprintf("%d MHz", m.ClockMHz), 5, bg))
	}
	return r.dashBlock(lines, width, bg)
}

// gpuMemoryLine renders memory as a gauge on a discrete card, where the
// percentage is a real pressure signal, and as a plain figure on an integrated
// one, where the pool is carved out of system RAM and sits near full by
// construction - a gauge there would be permanently and meaninglessly red.
func (r Renderer) gpuMemoryLine(m GPUMetrics, total int, bg lipgloss.Color) []string {
	if m.MemoryTotal == "" {
		return nil
	}
	label := m.MemoryLabel
	if label == "" {
		label = "MEM"
	}
	if m.Integrated {
		return []string{r.dashPair(label, fmt.Sprintf("%s / %s", m.MemoryUsed, m.MemoryTotal), 5, bg)}
	}
	percent := m.MemoryFrac * 100
	return []string{r.RenderMetricRow(MetricRow{
		Label: label, Value: fmt.Sprintf("%.0f%%", percent), Fraction: m.MemoryFrac,
		Bar: true, Tone: toneForPercent(percent),
		LabelWidth: 5, ValueWidth: 5, TotalWidth: total,
	}, bg)}
}

// RenderGPUDetail adds the driver and, on an integrated GPU, a note that the
// memory shown is shared with the system.
func (r Renderer) RenderGPUDetail(m GPUMetrics, width int) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	if m.Name == "" {
		return r.RenderGPU(m, width)
	}
	lines := strings.Split(r.RenderGPU(m, width), "\n")
	lines = append(lines, r.RenderSectionDivider(SectionDivider{Label: "DEVICE", Width: width}, bg))
	lines = append(lines, r.dashPair("Driver", m.Name, 7, bg))
	if m.Integrated {
		lines = append(lines, lipgloss.NewStyle().Background(bg).Foreground(ws.HintFg).
			Render("Shared with system memory"))
	}
	return r.dashBlock(lines, width, bg)
}

// RenderUpdates renders pending system updates: the Omarchy version bump when
// one is waiting, then how many packages are out of date and which.
func (r Renderer) RenderUpdates(u UpdateStatus, width int) string {
	return r.renderUpdates(u, width, false)
}

// RenderUpdatesDetail renders the same with a longer package list.
func (r Renderer) RenderUpdatesDetail(u UpdateStatus, width int) string {
	return r.renderUpdates(u, width, true)
}

func (r Renderer) renderUpdates(u UpdateStatus, width int, detail bool) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	hint := lipgloss.NewStyle().Background(bg).Foreground(ws.HintFg)
	var lines []string

	if u.Omarchy != "" {
		value := u.Omarchy
		tone := ToneMuted
		if u.OmarchyPending != "" {
			value = fmt.Sprintf("%s → %s", u.Omarchy, u.OmarchyPending)
			tone = ToneAccent
		}
		lines = append(lines, r.dashPair("Omarchy", "", 7, bg)+
			lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(tone)).Render(value))
	}

	pending := u.Pending()
	switch {
	case u.Unavailable != "":
		// Say the count is unknown rather than implying a clean system.
		lines = append(lines, hint.Render(u.Unavailable))
	case pending == 0:
		lines = append(lines, hint.Render("Up to date"))
	default:
		lines = append(lines, r.RenderStatValue(StatValue{
			Value: fmt.Sprintf("%d", pending), Label: updatesWord(pending), Tone: ToneWarning,
		}, bg))
	}

	limit := 4
	if detail {
		limit = 14
	}
	for _, pkg := range updatePackages(u, limit) {
		lines = append(lines, r.dashPair(pkg.Name, pkg.To, 16, bg))
	}
	if extra := pending - limit; extra > 0 {
		lines = append(lines, hint.Render(fmt.Sprintf("+%d more", extra)))
	}
	if detail && !u.Checked.IsZero() {
		lines = append(lines, r.dashPair("Checked", u.Checked.Format("15:04"), 7, bg))
	}
	return r.dashBlock(lines, width, bg)
}

// updatePackages returns repository updates followed by AUR ones, capped.
func updatePackages(u UpdateStatus, limit int) []UpdatePackage {
	all := make([]UpdatePackage, 0, len(u.Repo)+len(u.AUR))
	all = append(all, u.Repo...)
	all = append(all, u.AUR...)
	if len(all) > limit {
		all = all[:limit]
	}
	return all
}

// updatesWord keeps the headline reading naturally at one update.
func updatesWord(count int) string {
	if count == 1 {
		return "update"
	}
	return "updates"
}

// RenderNetwork renders throughput and a short-term graph.
func (r Renderer) RenderNetwork(m NetworkMetrics, width int) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	unit := m.Unit
	if unit == "" {
		unit = "Mbps"
	}
	lines := []string{
		lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).Render("↓ ") +
			lipgloss.NewStyle().Background(bg).Foreground(ws.FrameActive).Bold(true).
				Render(fmt.Sprintf("%.0f %s", m.Download, unit)),
		lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).Render("↑ ") +
			lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Bold(true).
				Render(fmt.Sprintf("%.0f %s", m.Upload, unit)),
	}
	if len(m.DownSpark) > 0 {
		lines = append(lines, "", r.renderTrendBlock("DOWNLOAD", m.DownSpark, width, ToneAccent, bg))
	}
	if m.Interface != "" {
		lines = append(lines, "", lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).
			Render(m.Interface))
	}
	return r.dashBlock(lines, width, bg)
}

// RenderNetworkDetail adds LAN/WAN summaries and an upload graph.
func (r Renderer) RenderNetworkDetail(m NetworkMetrics, width int) string {
	bg := r.Styles.Workspace.Bg
	lines := strings.Split(r.RenderNetwork(m, width), "\n")
	if len(m.UpSpark) > 0 {
		lines = append(lines, "", r.renderTrendBlock("UPLOAD", m.UpSpark, width, ToneGood, bg))
	}
	if m.LAN != "" {
		lines = append(lines, r.dashPair("LAN", m.LAN, 6, bg))
	}
	if m.WAN != "" {
		lines = append(lines, r.dashPair("WAN", m.WAN, 6, bg))
	}
	return r.dashBlock(lines, width, bg)
}

// renderTrendBlock gives a standalone graph enough context to be useful at a
// glance. The plot remains exactly the panel width, while the small range row
// makes a quiet or spiky series legible without relying on colour alone.
func (r Renderer) renderTrendBlock(label string, values []float64, width int, tone Tone, bg lipgloss.Color) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}
	low, high := valueRange(values)
	current := values[len(values)-1]
	stats := fmt.Sprintf("low %.1f   now %.1f   high %.1f", low, current, high)
	if width < 30 {
		stats = fmt.Sprintf("now %.1f", current)
	}
	muted := lipgloss.NewStyle().Background(bg).Foreground(r.Styles.Workspace.BodyMutedFg)
	return strings.Join([]string{
		r.RenderSectionDivider(SectionDivider{Label: label, Width: width}, bg),
		r.RenderSparkline(Sparkline{Values: values, Width: width, Tone: tone}, bg),
		muted.Render(stats),
	}, "\n")
}

// RenderStorage renders mounted filesystems with gauges.
func (r Renderer) RenderStorage(mounts []StorageMount, width int) string {
	bg := r.Styles.Workspace.Bg
	if len(mounts) == 0 {
		return ""
	}
	labelWidth := 0
	for _, mount := range mounts {
		labelWidth = max(labelWidth, lipgloss.Width(mount.Path))
	}
	labelWidth = min(labelWidth, max(4, width/3))
	lines := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		tone := mount.Tone
		if tone == ToneNeutral {
			tone = toneForPercent(mount.UsedPercent)
		}
		lines = append(lines, r.RenderMetricRow(MetricRow{
			Label: mount.Path, Value: fmt.Sprintf("%.0f%%", mount.UsedPercent),
			Fraction: mount.UsedPercent / 100, Bar: true, Tone: tone,
			LabelWidth: labelWidth, ValueWidth: 4, TotalWidth: metricTotal(width, labelWidth, 4, width),
		}, bg))
	}
	return r.dashBlock(lines, width, bg)
}

// RenderServices renders service/container statuses.
func (r Renderer) RenderServices(items []ServiceStatus, width int) string {
	return r.renderServices(items, width, false)
}

// RenderServicesDetail adds each service's detail and uptime.
func (r Renderer) RenderServicesDetail(items []ServiceStatus, width int) string {
	return r.renderServices(items, width, true)
}

func (r Renderer) renderServices(items []ServiceStatus, width int, detail bool) string {
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	if len(items) == 0 {
		return ""
	}
	plain := r.Styles.PlainUI
	nameWidth, stateWidth, ageWidth := 0, 0, 0
	for _, item := range items {
		_, label, _ := item.resolved()
		nameWidth = max(nameWidth, lipgloss.Width(item.Name))
		stateWidth = max(stateWidth, lipgloss.Width(label))
		ageWidth = max(ageWidth, lipgloss.Width(serviceAge(item)))
	}
	nameWidth = min(nameWidth, max(4, width/3))
	stateWidth = min(stateWidth, 9)
	ageWidth = min(ageWidth, 5)
	compact := width < 32

	var lines []string
	for _, item := range items {
		kind, label, tone := item.resolved()
		dot := lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(tone)).
			Render(kind.Glyph(plain))
		name := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).
			Render(padRight(ansi.Truncate(item.Name, nameWidth, "…"), nameWidth))
		state := lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(tone)).
			Render(padRight(ansi.Truncate(label, stateWidth, "…"), stateWidth))
		age := ""
		if !compact {
			age = lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).
				Render(padRight(ansi.Truncate(serviceAge(item), ageWidth, "…"), ageWidth))
		}
		// Keep name, state, and age as one grouped column set rather than
		// flinging the age to the far edge of a wide panel.
		line := dot + " " + name + " " + state
		if age != "" {
			line += "  " + age
		}
		lines = append(lines, line)
		if detail && item.Detail != "" {
			extra := item.Detail
			if item.Uptime != "" {
				extra += "  ·  up " + item.Uptime
			}
			lines = append(lines, lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).
				Render("  "+extra))
		}
	}
	return r.dashBlock(lines, width, bg)
}

// serviceAge returns the compact age column, defaulting to "--".
func serviceAge(item ServiceStatus) string {
	if item.Age != "" {
		return item.Age
	}
	return "--"
}

// RenderHeadlines renders each headline as its source followed by the title
// wrapped as a sentence, with unread stories brightest.
func (r Renderer) RenderHeadlines(items []Headline, width int) string {
	out, _ := r.renderHeadlines(items, width, false)
	return out
}

// RenderHeadlinesDetail renders more headlines than the compact list, in the
// same source-then-wrapped-sentence shape.
func (r Renderer) RenderHeadlinesDetail(items []Headline, width int) string {
	out, _ := r.renderHeadlines(items, width, true)
	return out
}

// RenderHeadlinesRows renders the headline list and also reports the story
// drawn on each content row (-1 for the blank row between stories), so a panel
// can map a mouse click back to a story.
func (r Renderer) RenderHeadlinesRows(items []Headline, width int, detail bool) (string, []int) {
	return r.renderHeadlines(items, width, detail)
}

// headlineRow is one rendered row of the headline list: the story drawn on it
// (negative for a blank separator), and the source and title text that share
// the row.
type headlineRow struct {
	story  int
	source string
	text   string
}

// headlinePlan lays out the headline list at a width, as the renderer draws it
// but without styling, so the same wrapping can drive hit-testing.
func headlinePlan(items []Headline, width int, detail bool) []headlineRow {
	limit := 6
	if detail {
		limit = 12
	}
	var rows []headlineRow
	for i, item := range items {
		if i >= limit {
			break
		}
		// A blank row between stories keeps each source-and-sentence block
		// readable now that there is no bullet to mark where one begins.
		if i > 0 {
			rows = append(rows, headlineRow{story: -1})
		}
		source := headlineSource(item)
		avail := width - lipgloss.Width(source) - 2
		// A long source with no room beside it gets its own line, so the
		// sentence still has the full width to wrap into.
		if avail < 8 {
			rows = append(rows, headlineRow{story: i, source: source})
			for _, part := range strings.Split(ansi.Wordwrap(item.Title, width, ""), "\n") {
				rows = append(rows, headlineRow{story: i, text: part})
			}
			continue
		}
		head, rest := wrapFirst(item.Title, avail)
		rows = append(rows, headlineRow{story: i, source: source, text: head})
		if rest != "" {
			for _, part := range strings.Split(ansi.Wordwrap(rest, width, ""), "\n") {
				rows = append(rows, headlineRow{story: i, text: part})
			}
		}
	}
	return rows
}

// HeadlineRows reports, for the headline list at a width, the story index on
// each content row (-1 for the blank row between stories). It shares the
// renderer's wrapping so a panel can map a click back to a story without
// rendering.
func HeadlineRows(items []Headline, width int, detail bool) []int {
	plan := headlinePlan(items, width, detail)
	rows := make([]int, len(plan))
	for i, row := range plan {
		rows[i] = row.story
	}
	return rows
}

// headlineSource is the muted attribution drawn before a title: its source and
// age, or a fallback so a story is never unattributed.
func headlineSource(item Headline) string {
	source := item.Source
	if source == "" {
		source = "news"
	}
	if item.Age != "" {
		source += " · " + item.Age
	}
	return source
}

func (r Renderer) renderHeadlines(items []Headline, width int, detail bool) (string, []int) {
	if len(items) == 0 {
		return "", nil
	}
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	// A read headline dims; an unread one stays bright. The source is the
	// item's anchor rather than a right-aligned afterthought: the sentence
	// starts beside it and wraps underneath it at the left margin, so the
	// wrapped lines use the whole width and the panel is not a wall of
	// markers.
	readStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyDimmedFg)
	unreadStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg)
	sourceStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg)
	selectedStyle := lipgloss.NewStyle().Background(ws.SelectionBg).Foreground(ws.SelectionFg).Width(width)

	var lines []string
	var rows []int
	for _, row := range headlinePlan(items, width, detail) {
		rows = append(rows, row.story)
		if row.story < 0 {
			lines = append(lines, "")
			continue
		}
		// The cursor is a highlighted block. The whole row takes the selection
		// colours so it reads as one, and is padded to the pane so the block
		// spans the panel rather than stopping at the last word.
		if items[row.story].Selected {
			text := row.source
			if row.text != "" {
				if text != "" {
					text += "  "
				}
				text += row.text
			}
			lines = append(lines, selectedStyle.Render(ansi.Truncate(text, width, "…")))
			continue
		}
		style := readStyle
		if items[row.story].Unread {
			style = unreadStyle
		}
		rendered := ""
		if row.source != "" {
			rendered = sourceStyle.Render(row.source)
			if row.text != "" {
				rendered += sourceStyle.Render("  ")
			}
		}
		if row.text != "" {
			rendered += style.Render(row.text)
		}
		lines = append(lines, rendered)
	}
	return r.dashBlock(lines, width, bg), rows
}

// wrapFirst takes the words of s that fit in limit as the first line and
// returns the remainder, so a sentence can start beside a source and its tail
// can then wrap at the full width. A single word wider than limit is returned
// whole in the first line, where the caller's own bound truncates it.
func wrapFirst(s string, limit int) (string, string) {
	words := strings.Fields(s)
	if len(words) == 0 {
		return "", ""
	}
	line := ""
	for i, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if line != "" && lipgloss.Width(candidate) > limit {
			return line, strings.Join(words[i:], " ")
		}
		line = candidate
	}
	return line, ""
}

// RenderTasks renders a task list with checkboxes and due dates.
func (r Renderer) RenderTasks(tasks []Task, width int) string {
	if len(tasks) == 0 {
		return ""
	}
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	plain := r.Styles.PlainUI
	var lines []string
	for _, task := range tasks {
		box := "□"
		tone := ToneMuted
		if task.Done {
			box = "✓"
			tone = ToneGood
		} else if task.Tone != ToneNeutral {
			tone = task.Tone
		}
		if plain {
			if task.Done {
				box = "[x]"
			} else {
				box = "[ ]"
			}
		}
		boxStyle := lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(tone))
		titleStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg)
		if task.Done {
			titleStyle = titleStyle.Foreground(ws.BodyDimmedFg)
		}
		left := boxStyle.Render(box) + lipgloss.NewStyle().Background(bg).Render(" ") + titleStyle.Render(task.Title)
		if len(task.Tags) > 0 && !r.Styles.Density.IsDense() {
			left += lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).Render("  " + strings.Join(task.Tags, " "))
		}
		right := ""
		if task.Due != "" {
			dueStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg)
			if !task.Done && strings.EqualFold(task.Due, "today") {
				dueStyle = dueStyle.Foreground(r.ToneColor(ToneWarning))
			}
			right = dueStyle.Render(task.Due)
		}
		lines = append(lines, alignRow(left, "", right, width))
	}
	return r.dashBlock(lines, width, bg)
}

// RenderNotes renders pinned and short notes.
func (r Renderer) RenderNotes(items []Note, width int) string {
	if len(items) == 0 {
		return ""
	}
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	var lines []string
	for i, note := range items {
		if i > 0 {
			lines = append(lines, "")
		}
		title := note.Title
		if note.Pinned {
			title = "★ " + title
		}
		if title != "" {
			lines = append(lines, lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Bold(true).Render(title))
		}
		for _, bodyLine := range strings.Split(note.Body, "\n") {
			lines = append(lines, lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).
				Render("  • "+strings.TrimSpace(strings.TrimPrefix(bodyLine, "- "))))
		}
	}
	return r.dashBlock(lines, width, bg)
}

// RenderRepoActivity renders one line per repository: the name, the branch,
// and what is outstanding.
func (r Renderer) RenderRepoActivity(items []RepoActivity, width int) string {
	return r.renderRepoActivity(items, width, false)
}

// RenderRepoActivityDetail adds the counts the compact line leaves out.
func (r Renderer) RenderRepoActivityDetail(items []RepoActivity, width int) string {
	return r.renderRepoActivity(items, width, true)
}

func (r Renderer) renderRepoActivity(items []RepoActivity, width int, detail bool) string {
	if len(items) == 0 {
		return ""
	}
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	labelWidth := 0
	for _, item := range items {
		labelWidth = max(labelWidth, lipgloss.Width(item.Name))
	}
	labelWidth = min(labelWidth, max(4, width/3))
	var lines []string
	for _, item := range items {
		tone := item.Tone
		if tone == ToneNeutral {
			tone = ToneAccent
		}
		left := lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(tone)).Bold(true).
			Render(padRight(item.Name, labelWidth))

		// What is outstanding matters more than which branch it is on, so the
		// branch is the first thing dropped when the row is tight. The state
		// mark stays either way.
		state := r.repoState(item)
		summary := state
		if !r.Styles.Density.IsDense() && item.Branch != "" {
			// Branch names run long ("milestone-2.5-account-panel-and-..."),
			// so they are elided from the middle, where the distinguishing
			// part of a branch name usually is not.
			branch := r.Styles.RepoIcons.Branch + elideMiddle(item.Branch, max(6, width/3))
			withBranch := strings.TrimSpace(branch + " " + state)
			if labelWidth+1+lipgloss.Width(withBranch) <= width {
				summary = withBranch
			}
		}
		left += lipgloss.NewStyle().Background(bg).Render(" ") +
			lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).Render(summary)

		// Only offer the count when it fits whole: alignRow would otherwise
		// truncate it to a stray digit hanging off the summary.
		right := ""
		if item.Commits > 0 {
			text := fmt.Sprintf("%d today", item.Commits)
			if lipgloss.Width(left)+1+lipgloss.Width(text) <= width {
				right = lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Render(text)
			}
		}
		lines = append(lines, alignRow(left, "", right, width))
		if detail {
			lines = append(lines, r.repoDetailLines(item, bg)...)
		}
	}
	return r.dashBlock(lines, width, bg)
}

// repoMark shows at a glance whether a repository needs attention. The tick is
// reserved for a repository with nothing outstanding; it used to be shown
// unconditionally, which marked a dirty tree as good.
func (r Renderer) repoMark(item RepoActivity) string {
	icons := r.Styles.RepoIcons
	switch {
	case item.State != "":
		return icons.Conflict
	case item.Branch == "":
		return icons.Missing
	case item.Clean():
		return icons.Clean
	default:
		return icons.Attention
	}
}

// repoState renders the mark plus what is outstanding, as icons and counts.
// Icons carry this better than words: "↑2 ±25 ≡1" says the same as
// "2 unpushed · 25 changed · 1 stashed" in a third of the cells, which is what
// lets a narrow panel keep the branch name as well.
func (r Renderer) repoState(item RepoActivity) string {
	icons := r.Styles.RepoIcons
	parts := []string{r.repoMark(item)}
	if item.State != "" {
		parts = append(parts, icons.Conflict+strings.ToUpper(item.State))
	}
	if item.NoUpstream && item.Branch != "" {
		parts = append(parts, icons.NoUpstream)
	}
	for _, count := range []struct {
		icon string
		n    int
	}{
		{icons.Ahead, item.Ahead}, {icons.Behind, item.Behind},
		{icons.Changed, item.Changes}, {icons.Stash, item.Stashes},
	} {
		if count.n > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", count.icon, count.n))
		}
	}
	if len(parts) == 1 {
		// Nothing countable: fall back to whatever the source said, which
		// carries "clean", "not a repo" and "remote URL — clone it first".
		return strings.TrimSpace(parts[0] + "  " + item.Summary)
	}
	return strings.Join(parts, " ")
}

// repoDetailLines spell out what the one-line summary compresses, including
// commits made today, which the compact line drops whenever there is
// outstanding work more worth the space.
func (r Renderer) repoDetailLines(item RepoActivity, bg lipgloss.Color) []string {
	var facts []string
	if item.Commits > 0 {
		facts = append(facts, fmt.Sprintf("%d commit(s) today", item.Commits))
	}
	if item.Ahead > 0 || item.Behind > 0 {
		facts = append(facts, fmt.Sprintf("%d ahead, %d behind", item.Ahead, item.Behind))
	}
	if item.NoUpstream && item.Branch != "" {
		facts = append(facts, "no upstream")
	}
	if len(facts) == 0 {
		return nil
	}
	ws := r.Styles.Workspace
	return []string{lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).
		Render("   " + strings.Join(facts, " · "))}
}

// elideMiddle shortens a string to width cells, cutting the middle so both
// ends stay readable.
func elideMiddle(value string, width int) string {
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	if width <= 3 {
		return ansi.Truncate(value, width, "")
	}
	head := (width - 1) / 2
	tail := width - 1 - head
	runes := []rune(value)
	return string(runes[:head]) + "…" + string(runes[len(runes)-tail:])
}

// RenderMarkets renders an aligned watchlist.
func (r Renderer) RenderMarkets(items []MarketQuote, width int) string {
	return r.renderMarkets(items, width, false)
}

// RenderMarketsDetail renders the watchlist with high/low facts. The compact
// view keeps those facts on a second line when needed; the detail view always
// gives them their own readable line.
func (r Renderer) RenderMarketsDetail(items []MarketQuote, width int) string {
	return r.renderMarkets(items, width, true)
}

func (r Renderer) renderMarkets(items []MarketQuote, width int, detail bool) string {
	if len(items) == 0 {
		return ""
	}
	bg := r.Styles.Workspace.Bg
	ws := r.Styles.Workspace
	symbolWidth := 0
	for _, item := range items {
		symbolWidth = max(symbolWidth, lipgloss.Width(item.Symbol))
	}
	var lines []string
	for _, item := range items {
		price := fmt.Sprintf("%.2f", item.Price)
		if item.Currency != "" {
			price += " " + item.Currency
		}
		left := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Bold(true).
			Render(padRight(item.Symbol, symbolWidth))
		right := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg).Render(price) +
			lipgloss.NewStyle().Background(bg).Render("  ") + r.RenderTrend(item.ChangePct, "%", bg)
		lines = append(lines, alignRow(left, "", right, width))
		if item.High != 0 || item.Low != 0 {
			facts := ""
			if item.High != 0 {
				facts = fmt.Sprintf("H %.2f", item.High)
			}
			if item.Low != 0 {
				if facts != "" {
					facts += "  "
				}
				facts += fmt.Sprintf("L %.2f", item.Low)
			}
			if detail || width < 52 {
				lines = append(lines, lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).
					Render("  "+facts))
			} else {
				lines[len(lines)-1] = alignRow(left, "", right+"  "+
					lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).Render(facts), width)
			}
		}
	}
	return r.dashBlock(lines, width, bg)
}

// metricTotal reserves the label/value columns and gives the plot all
// remaining width, bounded by the panel width.
func metricTotal(width, labelWidth, valueWidth, maxPlot int) int {
	return min(width, labelWidth+valueWidth+4+maxPlot)
}

// toneForPercent maps a utilisation percentage to a metric tone.
func toneForPercent(percent float64) Tone {
	switch {
	case percent >= 85:
		return ToneDanger
	case percent >= 70:
		return ToneWarning
	default:
		return ToneGood
	}
}

// formatUptime renders a duration in a compact dashboard form.
func formatUptime(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}
