package tideui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func dashboardNow() time.Time {
	return time.Date(2026, 9, 14, 14, 42, 0, 0, time.UTC)
}

func weatherFixture() WeatherData {
	return WeatherData{
		Location: "Springfield", Temperature: 72, Unit: "F", Condition: "Partly Cloudy",
		High: 76, Low: 61, RainChance: 12, WindSpeed: 9, WindUnit: "mph",
		Hourly:  []ForecastPoint{{Label: "3PM", Temperature: 74}, {Label: "6PM", Temperature: 70}},
		Daily:   []ForecastPoint{{Label: "Tue", Temperature: 71, Condition: "Rain"}},
		Updated: dashboardNow(),
	}
}

func agendaFixture(now time.Time) []AgendaItem {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	at := func(offset, hour, minute int) time.Time {
		return day.AddDate(0, 0, offset).Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
	}
	return []AgendaItem{
		{Title: "Standup", Start: at(1, 8, 0), Category: "Work", Tone: ToneAccent},
		{Title: "Focus block", Start: at(0, 14, 0), Category: "Deep work", Tone: ToneAccent},
		{Title: "Project review", Start: at(0, 15, 30), Location: "Meet", Category: "Work"},
	}
}

func TestRenderWeather(t *testing.T) {
	r := chromeRenderer(Compact)
	plain := ansi.Strip(r.RenderWeather(weatherFixture(), 34))
	for _, want := range []string{"72°F", "Partly Cloudy", "H 76°", "L 61°", "Rain", "Wind", "3PM", "74°"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("weather missing %q:\n%s", want, plain)
		}
	}
	detail := ansi.Strip(r.RenderWeatherDetail(weatherFixture(), 40))
	for _, want := range []string{"FORECAST", "Tue", "Springfield", "Updated"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("weather detail missing %q:\n%s", want, detail)
		}
	}
}

func TestWeatherKindMapping(t *testing.T) {
	cases := map[string]WeatherKind{
		"":              WeatherUnknown,
		"Clear":         WeatherClear,
		"Sunny":         WeatherClear,
		"Partly Cloudy": WeatherPartly,
		"Overcast":      WeatherCloudy,
		"Fog":           WeatherFog,
		"Drizzle":       WeatherRain,
		"Showers":       WeatherRain,
		"Rain":          WeatherRain,
		"Snow":          WeatherSnow,
		"Snow Showers":  WeatherSnow,
		"Thunderstorm":  WeatherStorm,
	}
	for condition, want := range cases {
		if got := WeatherKindFromCondition(condition); got != want {
			t.Fatalf("WeatherKindFromCondition(%q) = %d, want %d", condition, got, want)
		}
	}
}

func TestWeatherKindGlyph(t *testing.T) {
	kinds := []WeatherKind{WeatherClear, WeatherPartly, WeatherCloudy, WeatherFog, WeatherRain, WeatherSnow, WeatherStorm}
	seen := map[string]bool{}
	for _, kind := range kinds {
		glyph := kind.Glyph(false)
		if glyph == "" {
			t.Fatalf("kind %d has no glyph", kind)
		}
		if width := ansi.StringWidth(glyph); width != 2 {
			t.Fatalf("kind %d glyph %q width = %d, want 2 (emoji presentation)", kind, glyph, width)
		}
		if seen[glyph] {
			t.Fatalf("duplicate glyph %q", glyph)
		}
		seen[glyph] = true
		if kind.Glyph(true) == "" {
			t.Fatalf("kind %d has no plain fallback", kind)
		}
	}
}

func TestWeatherFeelsLikeAndGlyphRender(t *testing.T) {
	r := chromeRenderer(Compact)
	w := weatherFixture()
	w.FeelsLike = 70
	w.HasFeelsLike = true
	out := ansi.Strip(r.RenderWeather(w, 40))
	if !strings.Contains(out, "Feels 70°") {
		t.Fatalf("feels-like missing:\n%s", out)
	}
	if !strings.Contains(out, WeatherKindFromCondition(w.Condition).Glyph(false)) {
		t.Fatalf("condition glyph missing:\n%s", out)
	}
	if !strings.Contains(out, r.rainGlyph()+" Rain") || !strings.Contains(out, r.windGlyph()+" Wind") {
		t.Fatalf("rain/wind icons missing:\n%s", out)
	}
	if detail := ansi.Strip(r.RenderWeatherDetail(w, 44)); !strings.Contains(detail, "Feels 70°") {
		t.Fatalf("detail missing feels-like:\n%s", detail)
	}
	// A hot feels-like temperature gets the flame; a mild one does not.
	hot := weatherFixture()
	hot.FeelsLike = 102
	hot.HasFeelsLike = true
	hotOut := ansi.Strip(r.RenderWeather(hot, 44))
	if !strings.Contains(hotOut, "Feels 102° "+r.hotGlyph()) {
		t.Fatalf("hot feels-like missing flame:\n%s", hotOut)
	}
	if strings.Contains(ansi.Strip(r.RenderWeather(w, 44)), r.hotGlyph()) {
		t.Fatal("mild feels-like should not show a flame")
	}
}

func TestWeatherColorsAreDistinct(t *testing.T) {
	ws := chromeRenderer(Compact).Styles.Workspace
	kinds := []WeatherKind{WeatherClear, WeatherCloudy, WeatherFog, WeatherRain, WeatherSnow, WeatherStorm}
	colors := map[WeatherKind]lipgloss.Color{}
	for _, kind := range kinds {
		c := ws.WeatherColor(kind)
		if c == "" {
			t.Fatalf("kind %d has no colour", kind)
		}
		colors[kind] = c
	}
	if colors[WeatherClear] == colors[WeatherCloudy] {
		t.Fatal("clear and cloudy should have distinct colours")
	}
	if colors[WeatherRain] == colors[WeatherSnow] {
		t.Fatal("rain and snow should have distinct colours")
	}
}

func TestRenderAgendaSortsAndGroups(t *testing.T) {
	now := dashboardNow()
	out := ansi.Strip(r2().RenderAgenda(agendaFixture(now), now, 44))
	if !strings.Contains(out, "TODAY") || !strings.Contains(out, "TOMORROW") {
		t.Fatalf("agenda missing day groups:\n%s", out)
	}
	focus := strings.Index(out, "Focus block")
	review := strings.Index(out, "Project review")
	standup := strings.Index(out, "Standup")
	if focus < 0 || review < 0 || standup < 0 {
		t.Fatalf("agenda missing items:\n%s", out)
	}
	if !(focus < review && review < standup) {
		t.Fatalf("agenda not sorted by time:\n%s", out)
	}
	if !strings.Contains(out, "next") {
		t.Fatalf("agenda missing next emphasis:\n%s", out)
	}
}

func TestRenderAgendaAllDay(t *testing.T) {
	now := dashboardNow()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	items := []AgendaItem{
		{Title: "Holiday", Start: day, AllDay: true, Tone: ToneGood},
		{Title: "Project review", Start: day.Add(15*time.Hour + 30*time.Minute), Tone: ToneAccent},
	}
	out := ansi.Strip(r2().RenderAgenda(items, now, 44))
	if !strings.Contains(out, "all-day") {
		t.Fatalf("agenda missing all-day label:\n%s", out)
	}
	if strings.Contains(out, "00:00") {
		t.Fatalf("agenda rendered an all-day event as 00:00:\n%s", out)
	}
	holiday := strings.Index(out, "Holiday")
	review := strings.Index(out, "Project review")
	if holiday < 0 || review < 0 {
		t.Fatalf("agenda missing items:\n%s", out)
	}
	// The timed row right-aligns to the "all-day" column, so both titles sit at
	// the same offset within their line.
	column := func(title int) int {
		return title - (strings.LastIndexByte(out[:title], '\n') + 1)
	}
	if column(holiday) != column(review) {
		t.Fatalf("time columns are not aligned:\n%s", out)
	}
}

func TestRenderCalendarMonthAndAgenda(t *testing.T) {
	now := dashboardNow()
	day := time.Date(now.Year(), now.Month(), 16, 0, 0, 0, 0, now.Location())
	items := []AgendaItem{
		{Title: "Team offsite", Start: day, AllDay: true},
		{Title: "Standup", Start: day.Add(8 * time.Hour)},
	}
	wide := ansi.Strip(r2().RenderCalendar(day, map[int]bool{16: true}, items, now, 64))
	if !strings.Contains(wide, "September 2026") {
		t.Fatalf("calendar missing the month grid:\n%s", wide)
	}
	if !strings.Contains(wide, "Team offsite") || !strings.Contains(wide, "all-day") {
		t.Fatalf("calendar missing the agenda:\n%s", wide)
	}
	if strings.Index(wide, "September") > strings.Index(wide, "Standup") {
		t.Fatalf("month is not laid out before the agenda:\n%s", wide)
	}
	// Too narrow for two columns, the grid gives way to the agenda alone.
	narrow := ansi.Strip(r2().RenderCalendar(day, nil, items, now, 30))
	if strings.Contains(narrow, "September") {
		t.Fatalf("narrow calendar should drop the grid:\n%s", narrow)
	}
	if !strings.Contains(narrow, "Team offsite") {
		t.Fatalf("narrow calendar missing the agenda:\n%s", narrow)
	}
}

func TestRenderAgendaEmpty(t *testing.T) {
	out := r2().RenderAgenda(nil, dashboardNow(), 30)
	if !strings.Contains(ansi.Strip(out), "No upcoming events") {
		t.Fatalf("empty agenda = %q", ansi.Strip(out))
	}
}

func r2() Renderer { return chromeRenderer(Compact) }

func TestRenderClockAndCalendar(t *testing.T) {
	r := chromeRenderer(Compact)
	now := dashboardNow()
	clock := ClockData{Local: now, Location: "Local", Hour24: true, Zones: []WorldClock{{City: "London", Time: now, Offset: "UTC"}, {City: "Tokyo", Time: now.Add(9 * time.Hour), Offset: "+9"}}}
	// At a narrow width the time is plain text.
	plain := ansi.Strip(r.RenderClock(clock, 14))
	for _, want := range []string{"14:42", "Mon Sep 14", "London", "Tokyo"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("clock missing %q:\n%s", want, plain)
		}
	}
	twelve := ansi.Strip(r.RenderClock(ClockData{Local: now, Hour24: false}, 14))
	if !strings.Contains(twelve, "2:42 PM") {
		t.Fatalf("12-hour clock = %q, want 2:42 PM", twelve)
	}
	// At a wide width the time is drawn big, with a day-progress gauge.
	big := ansi.Strip(r.RenderClock(clock, 30))
	if !strings.Contains(big, "afternoon") || !strings.Contains(big, "% of day") && !strings.Contains(big, "61%") {
		t.Fatalf("wide clock should be rich:\n%s", big)
	}
	if !strings.Contains(big, "─") {
		t.Fatalf("wide clock missing big digits:\n%s", big)
	}
	calendar := ansi.Strip(r.RenderMiniCalendar(MiniCalendar{Year: 2026, Month: time.September, Highlight: 14, Width: 21}, r.Styles.Workspace.Bg))
	if !strings.Contains(calendar, "September 2026") || !strings.Contains(calendar, "14") {
		t.Fatalf("calendar missing content:\n%s", calendar)
	}
	for _, line := range strings.Split(calendar, "\n") {
		if lipgloss.Width(line) != 21 {
			t.Fatalf("calendar line width = %d, want 21", lipgloss.Width(line))
		}
	}
}

func TestClockLook(t *testing.T) {
	r := chromeRenderer(Compact)
	morning := time.Date(2026, 9, 14, 8, 5, 0, 0, time.UTC)
	if label, glyph := r.dayPeriod(morning); label != "morning" || glyph == "" {
		t.Fatalf("dayPeriod(morning) = %q,%q", label, glyph)
	}
	if label, _ := r.dayPeriod(time.Date(2026, 9, 14, 2, 0, 0, 0, time.UTC)); label != "night" {
		t.Fatalf("dayPeriod(night) = %q", label)
	}
	if f := dayFraction(morning); f <= 0 || f >= 1 {
		t.Fatalf("dayFraction = %v, want between 0 and 1", f)
	}
	big := r.bigTime("14:42") // default dash font is 3 rows of 4-cell digits
	if len(big) != 3 || lipgloss.Width(big[0]) != 22 {
		t.Fatalf("dash bigTime = %#v", big)
	}
	// The dash "4" carries both top verticals, so it reads as a seven-segment 4.
	if four := bigClockDash['4']; len(four) != 3 || four[0] != "│  │" || four[2] != "   │" {
		t.Fatalf("dash 4 = %#v", four)
	}
	// Digits are four cells wide and square-ish once the cell aspect is taken
	// into account; every row of a digit is the same width, so the rows of a
	// rendered time stay in step.
	for ch, glyph := range bigClockDash {
		if ch == ':' {
			continue
		}
		for _, row := range glyph {
			if lipgloss.Width(row) != 4 {
				t.Fatalf("dash %q row = %q, want 4 cells", ch, row)
			}
		}
	}
	// Row 0 is a full top row: the top bar spans the digit and joins the top
	// verticals, so no digit leaves a gap up there.
	for _, ch := range []rune{'0', '2', '3', '5', '6', '7', '8', '9'} {
		if top := bigClockDash[ch][0]; strings.ContainsRune(top, ' ') || !strings.ContainsRune(top, '─') {
			t.Fatalf("dash %q row 0 = %q, want an unbroken top bar", ch, top)
		}
	}
	// Digits without a top bar still fill row 0 with their verticals.
	for _, ch := range []rune{'1', '4'} {
		if top := bigClockDash[ch][0]; !strings.ContainsRune(top, '│') {
			t.Fatalf("dash %q row 0 = %q, want top verticals", ch, top)
		}
	}
	block := NewRenderer(CatppuccinMocha, StyleOptions{ClockFont: ClockFontBlock}).bigTime("14:42")
	if len(block) != 5 || lipgloss.Width(block[0]) != 19 {
		t.Fatalf("block bigTime = %#v", block)
	}
	face := r.analogClock(morning)
	if len(face) != 7 {
		t.Fatalf("analog face height = %d, want 7", len(face))
	}
	for _, line := range face {
		if lipgloss.Width(line) != 13 {
			t.Fatalf("analog face width = %d, want 13 (%q)", lipgloss.Width(line), line)
		}
	}
	for _, font := range ClockFonts() {
		fr := NewRenderer(CatppuccinMocha, StyleOptions{ClockFont: font})
		if fr.Styles.ClockFont != font {
			t.Fatalf("resolved clock font = %q, want %q", fr.Styles.ClockFont, font)
		}
		// Fonts differ in digit width, but every row of one render must be
		// the same width or the digits shear apart.
		rows := fr.bigTime("14:42")
		for _, row := range rows {
			if lipgloss.Width(row) != lipgloss.Width(rows[0]) {
				t.Fatalf("%s bigTime rows are ragged: %#v", font, rows)
			}
		}
	}
	if NewRenderer(CatppuccinMocha, StyleOptions{ClockFont: ClockFont("x")}).Styles.ClockFont != ClockFontDash {
		t.Fatal("unknown clock font should fall back to dash")
	}

	_, glyph := r.dayPeriod(morning)
	compact := ansi.Strip(r.RenderClock(ClockData{Local: morning, Location: "Local"}, 30))
	if !strings.Contains(compact, "morning") || !strings.Contains(compact, glyph) {
		t.Fatalf("compact clock missing period:\n%s", compact)
	}
	detail := ansi.Strip(r.RenderClockDetail(ClockData{Local: morning, Location: "Local"}, 30))
	if !strings.Contains(detail, "% of day") {
		t.Fatalf("detail clock missing day progress:\n%s", detail)
	}
}

func TestRenderSystemNetworkStorage(t *testing.T) {
	r := chromeRenderer(Compact)
	system := ansi.Strip(r.RenderSystem(SystemMetrics{
		CPUPercent: 18, CPUSpark: []float64{0.1, 0.2, 0.3}, MemoryPercent: 41,
		TemperatureC: 54, Load: [3]float64{1.4, 1.1, 0.9}, Uptime: 3*24*time.Hour + 14*time.Hour,
	}, 30))
	for _, want := range []string{"CPU", "18%", "MEM", "41%", "TEMP", "54°C", "LOAD", "UP", "3d 14h"} {
		if !strings.Contains(system, want) {
			t.Fatalf("system missing %q:\n%s", want, system)
		}
	}

	network := ansi.Strip(r.RenderNetwork(NetworkMetrics{
		Interface: "wlan0", Download: 87, Upload: 14, Unit: "Mbps",
		DownSpark: []float64{0.1, 0.5, 0.9}, LAN: "940 Mbps", WAN: "87/14",
	}, 30))
	for _, want := range []string{"↓", "↑", "87 Mbps", "14 Mbps", "wlan0"} {
		if !strings.Contains(network, want) {
			t.Fatalf("network missing %q:\n%s", want, network)
		}
	}

	storage := ansi.Strip(r.RenderStorage([]StorageMount{
		{Path: "/", UsedPercent: 72}, {Path: "/home", UsedPercent: 48}, {Path: "/media", UsedPercent: 91},
	}, 30))
	for _, want := range []string{"/", "72%", "/home", "48%", "/media", "91%"} {
		if !strings.Contains(storage, want) {
			t.Fatalf("storage missing %q:\n%s", want, storage)
		}
	}
}

func TestRenderServicesAndHeadlines(t *testing.T) {
	r := chromeRenderer(Compact)
	services := ansi.Strip(r.RenderServices([]ServiceStatus{
		{Name: "jellyfin", Status: StatusHealthy},
		{Name: "forgejo", Status: StatusWarning},
		{Name: "backup", Status: StatusStopped},
	}, 30))
	for _, want := range []string{"jellyfin", "healthy", "warning", "stopped"} {
		if !strings.Contains(services, want) {
			t.Fatalf("services missing %q:\n%s", want, services)
		}
	}

	headlines := ansi.Strip(r.RenderHeadlines([]Headline{
		{Title: "A very long headline that must be wrapped somewhere", Source: "src", Age: "5m", Unread: true, Tone: ToneAccent},
	}, 20))
	for _, line := range strings.Split(headlines, "\n") {
		if lipgloss.Width(line) > 20 {
			t.Fatalf("headline line too wide: %q", line)
		}
	}
	if !strings.HasPrefix(headlines, "src") {
		t.Fatalf("headline should lead with its source:\n%s", headlines)
	}
	if strings.Contains(headlines, "●") {
		t.Fatalf("headlines should not use a bullet:\n%s", headlines)
	}
	if len(strings.Split(strings.TrimRight(headlines, "\n"), "\n")) < 2 {
		t.Fatalf("a long headline should wrap:\n%s", headlines)
	}
}

// A headline's wrapped tail starts at the left margin, under the source,
// rather than hanging-indented beneath the sentence.
func TestHeadlineWrapsUnderSource(t *testing.T) {
	r := chromeRenderer(Dense)
	out := ansi.Strip(r.RenderHeadlines([]Headline{
		{Title: "alpha beta gamma delta epsilon zeta eta theta", Source: "src", Age: "1m", Unread: true},
	}, 30))
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected a wrapped headline:\n%s", out)
	}
	if !strings.HasPrefix(lines[0], "src · 1m  alpha") {
		t.Fatalf("first line should lead with the source and start the sentence: %q", lines[0])
	}
	if strings.HasPrefix(lines[1], " ") {
		t.Fatalf("wrapped tail should start under the source, got %q", lines[1])
	}
}

// Stories are separated by a blank row so each source-and-sentence block reads
// as one item without a bullet to mark it.
func TestHeadlinesSeparatedByBlankRow(t *testing.T) {
	r := chromeRenderer(Dense)
	out := ansi.Strip(r.RenderHeadlines([]Headline{
		{Title: "One", Source: "a"},
		{Title: "Two", Source: "b"},
	}, 30))
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			return
		}
	}
	t.Fatalf("expected a blank row between stories:\n%s", out)
}

// The row map a panel uses for hit-testing must line up with what the renderer
// drew, at every width.
func TestHeadlineRowsMatchRender(t *testing.T) {
	r := chromeRenderer(Dense)
	items := []Headline{
		{Title: "One long enough to wrap a little", Source: "a", Age: "1m", Unread: true},
		{Title: "Two", Source: "b"},
		{Title: "Three from a much longer source name", Source: "blog.example.org", Age: "4h"},
	}
	for _, width := range []int{12, 20, 40, 60} {
		out, rows := r.RenderHeadlinesRows(items, width, false)
		lines := strings.Split(ansi.Strip(out), "\n")
		if len(lines) != len(rows) {
			t.Fatalf("width %d: %d lines but %d mapped rows", width, len(lines), len(rows))
		}
		if got := HeadlineRows(items, width, false); !reflect.DeepEqual(got, rows) {
			t.Fatalf("width %d: HeadlineRows = %v, renderer rows = %v", width, got, rows)
		}
	}
}

// Selecting a story highlights it without changing the layout or the bounds.
func TestHeadlineSelectionRendersBounded(t *testing.T) {
	r := chromeRenderer(Dense)
	items := []Headline{
		{Title: "One", Source: "a"},
		{Title: "Two is a longer headline that wraps", Source: "b", Selected: true},
	}
	out := ansi.Strip(r.RenderHeadlines(items, 24))
	if !strings.Contains(out, "Two is a longer") {
		t.Fatalf("selected story text missing:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 24 {
			t.Fatalf("line too wide (%d): %q", w, line)
		}
	}
	// The selected story still maps to its own row.
	_, rows := r.RenderHeadlinesRows(items, 24, false)
	if len(HeadlineRows(items, 24, false)) != len(rows) {
		t.Fatal("selection changed the row map")
	}
}

func TestRenderTasksNotesGitMarkets(t *testing.T) {
	r := chromeRenderer(Compact)
	tasks := ansi.Strip(r.RenderTasks([]Task{
		{Title: "Finish TideDeck", Due: "today", Tone: ToneWarning},
		{Title: "Fix spacing", Done: true},
	}, 34))
	for _, want := range []string{"□", "Finish TideDeck", "today", "✓", "Fix spacing"} {
		if !strings.Contains(tasks, want) {
			t.Fatalf("tasks missing %q:\n%s", want, tasks)
		}
	}

	notes := ansi.Strip(r.RenderNotes([]Note{{Title: "Remember", Pinned: true, Body: "- one\n- two"}}, 30))
	for _, want := range []string{"★", "Remember", "• one", "• two"} {
		if !strings.Contains(notes, want) {
			t.Fatalf("notes missing %q:\n%s", want, notes)
		}
	}

	repos := ansi.Strip(r.RenderRepoActivity([]RepoActivity{
		{Name: "tideui", Branch: "main", Commits: 3, Tone: ToneAccent},
	}, 34))
	for _, want := range []string{"tideui", "main", "3 today"} {
		if !strings.Contains(repos, want) {
			t.Fatalf("git missing %q:\n%s", want, repos)
		}
	}

	markets := ansi.Strip(r.RenderMarkets([]MarketQuote{
		{Symbol: "AMD", Price: 162.40, High: 164.20, Low: 158.80, ChangePct: 1.8},
		{Symbol: "NVDA", Price: 214.10, High: 216.30, Low: 211.50, ChangePct: -0.4},
	}, 30))
	for _, want := range []string{"AMD", "162.40", "1.8%", "H 164.20", "L 158.80", "NVDA", "214.10", "0.4%"} {
		if !strings.Contains(markets, want) {
			t.Fatalf("markets missing %q:\n%s", want, markets)
		}
	}
}

func TestDashboardWidgetsBoundedAtTinyWidths(t *testing.T) {
	r := chromeRenderer(Compact)
	now := dashboardNow()
	renderers := map[string]func(int) string{
		"weather":       func(w int) string { return r.RenderWeather(weatherFixture(), w) },
		"weatherDetail": func(w int) string { return r.RenderWeatherDetail(weatherFixture(), w) },
		"agenda":        func(w int) string { return r.RenderAgenda(agendaFixture(now), now, w) },
		"clock": func(w int) string {
			return r.RenderClock(ClockData{Local: now, Zones: []WorldClock{{City: "Tokyo", Time: now}}}, w)
		},
		"system": func(w int) string {
			return r.RenderSystem(SystemMetrics{CPUPercent: 80, CPUSpark: []float64{0.5}, MemoryPercent: 90, Uptime: time.Hour}, w)
		},
		"systemDetail": func(w int) string {
			return r.RenderSystemDetail(SystemMetrics{Cores: []float64{10, 90}, MemoryUsed: "1G", Processes: 3}, w)
		},
		"repos": func(w int) string {
			return r.RenderRepoActivity([]RepoActivity{
				{Name: "threadtide", Branch: "milestone-2.5-account-panel-and-compliance",
					Summary: "no upstream · 6 unpushed · 81 changed", Ahead: 6, Changes: 81, NoUpstream: true},
			}, w)
		},
		"reposDetail": func(w int) string {
			return r.RenderRepoActivityDetail([]RepoActivity{
				{Name: "tidedeck", Branch: "main", Commits: 4, Changes: 25, Ahead: 2, Behind: 1, Summary: "2 unpushed · 25 changed"},
			}, w)
		},
		"gpu": func(w int) string {
			return r.RenderGPU(gpuFixture(), w)
		},
		"gpuDetail": func(w int) string {
			return r.RenderGPUDetail(gpuFixture(), w)
		},
		"gpuDiscrete": func(w int) string {
			return r.RenderGPU(GPUMetrics{Name: "nvidia", BusyPercent: 63, MemoryLabel: "VRAM",
				MemoryUsed: "5.1 GB", MemoryTotal: "8.0 GB", MemoryFrac: 0.64, TemperatureC: 71}, w)
		},
		"gpuEmpty":      func(w int) string { return r.RenderGPU(GPUMetrics{}, w) },
		"updates":       func(w int) string { return r.RenderUpdates(updatesFixture(), w) },
		"updatesDetail": func(w int) string { return r.RenderUpdatesDetail(updatesFixture(), w) },
		"updatesClean":  func(w int) string { return r.RenderUpdates(UpdateStatus{Omarchy: "4.0.4-1"}, w) },
		"network": func(w int) string {
			return r.RenderNetwork(NetworkMetrics{Download: 10, Upload: 2, DownSpark: []float64{0.5}}, w)
		},
		"storage": func(w int) string { return r.RenderStorage([]StorageMount{{Path: "/", UsedPercent: 90}}, w) },
		"services": func(w int) string {
			return r.RenderServices([]ServiceStatus{{Name: "x", State: "healthy", Tone: ToneGood}}, w)
		},
		"headlines": func(w int) string {
			return r.RenderHeadlines([]Headline{{Title: "hello world", Source: "src", Age: "1m", Unread: true}}, w)
		},
		"tasks": func(w int) string { return r.RenderTasks([]Task{{Title: "task", Due: "today"}}, w) },
		"notes": func(w int) string { return r.RenderNotes([]Note{{Title: "n", Body: "- line"}}, w) },
		"git": func(w int) string {
			return r.RenderRepoActivity([]RepoActivity{{Name: "r", Branch: "main", Commits: 1}}, w)
		},
		"markets": func(w int) string {
			return r.RenderMarkets([]MarketQuote{{Symbol: "A", Price: 1.23, ChangePct: 1.5}}, w)
		},
	}
	for name, render := range renderers {
		for _, width := range []int{4, 8, 12, 20, 40} {
			view := render(width)
			for i, line := range strings.Split(view, "\n") {
				if got := lipgloss.Width(line); got > width {
					t.Fatalf("%s width %d: line %d width %d: %q", name, width, i, got, ansi.Strip(line))
				}
			}
		}
	}
}

func TestInfoPrimitives(t *testing.T) {
	withTrueColor(t)
	r := chromeRenderer(Compact)
	bg := r.Styles.Workspace.Bg

	dot := ansi.Strip(r.RenderStatusDot(StatusDot{Label: "ok", Tone: ToneGood}, bg))
	if dot != "● ok" {
		t.Fatalf("status dot = %q", dot)
	}
	dot = ansi.Strip(r.RenderStatusDot(StatusDot{Label: "bad", Tone: ToneDanger}, bg))
	if dot != "● bad" {
		t.Fatalf("status dot = %q", dot)
	}

	trend := ansi.Strip(r.RenderTrend(1.8, "%", bg))
	if !strings.Contains(trend, "▲") || !strings.Contains(trend, "1.8%") {
		t.Fatalf("trend = %q", trend)
	}
	down := ansi.Strip(r.RenderTrend(-0.4, "%", bg))
	if !strings.Contains(down, "▼") {
		t.Fatalf("down trend = %q", down)
	}

	stat := ansi.Strip(r.RenderStatValue(StatValue{Value: "72", Unit: "°F", Label: "Partly Cloudy", Tone: ToneAccent}, bg))
	for _, want := range []string{"72°F", "Partly Cloudy"} {
		if !strings.Contains(stat, want) {
			t.Fatalf("stat value missing %q: %q", want, stat)
		}
	}

	gauge := r.RenderBarGauge(BarGauge{Fraction: 0.5, Width: 10, Tone: ToneGood, Label: "x"}, bg)
	if !strings.Contains(ansi.Strip(gauge), "x") || !strings.Contains(ansi.Strip(gauge), "█") {
		t.Fatalf("bar gauge = %q", ansi.Strip(gauge))
	}
}

func TestStatusDotDoesNotRelyOnColourAlone(t *testing.T) {
	r := chromeRenderer(Compact)
	good := ansi.Strip(r.RenderStatusDot(StatusDot{Label: "healthy", Tone: ToneGood}, r.Styles.Workspace.Bg))
	bad := ansi.Strip(r.RenderStatusDot(StatusDot{Label: "stopped", Tone: ToneDanger}, r.Styles.Workspace.Bg))
	if good == bad {
		t.Fatal("status labels must differ even without colour")
	}
}

// weatherLineWith returns the first line containing sub.
func weatherLineWith(lines []string, sub string) string {
	for _, line := range lines {
		if strings.Contains(line, sub) {
			return line
		}
	}
	return ""
}

// weatherColumnOf returns the cell column sub starts at, counting display
// width so emoji do not skew the comparison.
func weatherColumnOf(t *testing.T, line, sub string) int {
	t.Helper()
	at := strings.Index(line, sub)
	if at < 0 {
		t.Fatalf("%q missing from %q", sub, line)
	}
	return lipgloss.Width(line[:at])
}

func TestWeatherDetailIsGridAligned(t *testing.T) {
	r := chromeRenderer(Compact)
	w := weatherFixture()
	w.FeelsLike, w.HasFeelsLike = 70, true
	w.Daily = []ForecastPoint{
		{Label: "Today", Temperature: 8, Condition: "Snow"},
		{Label: "Tue", Temperature: 108, Condition: "Sunny"},
	}
	out := ansi.Strip(r.RenderWeatherDetail(w, 40))
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if lipgloss.Width(line) != 40 {
			t.Fatalf("detail line width = %d, want 40 (%q)", lipgloss.Width(line), line)
		}
	}
	// Rain and wind share one row once there is room for both.
	if rain := weatherLineWith(lines, "Rain"); !strings.Contains(rain, "Wind") {
		t.Fatalf("rain and wind should share a row at width 40:\n%s", out)
	}
	// The label column fits the longest label, so the values line up under
	// each other instead of "Updated" pushing its value a cell right.
	place := weatherColumnOf(t, weatherLineWith(lines, "Place"), "Springfield")
	updated := weatherColumnOf(t, weatherLineWith(lines, "Updated"), "14:42")
	if place != updated {
		t.Fatalf("label column ragged: Place value at %d, Updated value at %d:\n%s", place, updated, out)
	}
	// Temperatures are right-aligned, so 8° and 108° end in the same column
	// and the conditions all start together.
	today := weatherColumnOf(t, weatherLineWith(lines, "Today"), "Snow")
	tue := weatherColumnOf(t, weatherLineWith(lines, "Tue "), "Sunny")
	if today != tue {
		t.Fatalf("forecast conditions ragged: %d vs %d:\n%s", today, tue, out)
	}
}

func TestWeatherNarrowDropsWholeFacts(t *testing.T) {
	r := chromeRenderer(Compact)
	w := weatherFixture()
	w.FeelsLike, w.HasFeelsLike = 70, true
	out := ansi.Strip(r.RenderWeather(w, 24))
	facts := weatherLineWith(strings.Split(out, "\n"), "H 76°")
	// Too narrow for feels-like: it is dropped whole, not cut mid-figure.
	if strings.Contains(facts, "…") || strings.Contains(facts, "Feels") {
		t.Fatalf("narrow facts line should drop feels-like whole, got %q", facts)
	}
	if !strings.Contains(facts, "L 61°") {
		t.Fatalf("narrow facts line lost the low: %q", facts)
	}
	// Rain and wind stack rather than crowding one row.
	if rain := weatherLineWith(strings.Split(out, "\n"), "Rain"); strings.Contains(rain, "Wind") {
		t.Fatalf("rain and wind should stack at width 24:\n%s", out)
	}
}

func TestPanelsUseCompactContentAtNarrowWidths(t *testing.T) {
	r := chromeRenderer(Compact)

	system := ansi.Strip(r.RenderSystem(SystemMetrics{
		CPUPercent: 18, CPUSpark: []float64{0.1, 0.2, 0.3}, MemoryPercent: 41,
		TemperatureC: 54, Load: [3]float64{1.4, 1.1, 0.9}, Uptime: 3 * time.Hour,
	}, 24))
	if strings.Contains(system, "LOAD") || strings.Contains(system, "UP") {
		t.Fatalf("narrow system retained secondary rows:\n%s", system)
	}
	if !strings.Contains(system, "CPU") || !strings.Contains(system, "TEMP") {
		t.Fatalf("narrow system dropped primary rows:\n%s", system)
	}

	gpu := ansi.Strip(r.RenderGPU(gpuFixture(), 24))
	for _, secondary := range []string{"TEMP", "PWR", "CLK"} {
		if strings.Contains(gpu, secondary) {
			t.Fatalf("narrow GPU retained %q:\n%s", secondary, gpu)
		}
	}
	if !strings.Contains(gpu, "GPU") || !strings.Contains(gpu, "MEM") {
		t.Fatalf("narrow GPU dropped primary metrics:\n%s", gpu)
	}
}

func TestWeatherHourlyKeepsWholeSegments(t *testing.T) {
	r := chromeRenderer(Compact)
	w := weatherFixture()
	w.Hourly = []ForecastPoint{
		{Label: "3PM", Temperature: 74}, {Label: "6PM", Temperature: 70},
		{Label: "9PM", Temperature: 64}, {Label: "12AM", Temperature: 61},
	}
	strip := weatherLineWith(strings.Split(ansi.Strip(r.RenderWeather(w, 28)), "\n"), "3PM")
	if strings.Contains(strip, "…") {
		t.Fatalf("hourly strip was cut mid-segment: %q", strip)
	}
	if !strings.Contains(strip, "9PM") || strings.Contains(strip, "12AM") {
		t.Fatalf("hourly strip should keep whole segments only: %q", strip)
	}
}

func gpuFixture() GPUMetrics {
	return GPUMetrics{
		Name: "amdgpu", BusyPercent: 7, BusySpark: []float64{0.05, 0.4, 0.07},
		MemoryUsed: "29.2 GB", MemoryTotal: "31.0 GB", MemoryLabel: "MEM", MemoryFrac: 0.94,
		TemperatureC: 48, PowerWatts: 16.2, ClockMHz: 1003, Integrated: true,
	}
}

func updatesFixture() UpdateStatus {
	return UpdateStatus{
		Omarchy: "4.0.3-1", OmarchyPending: "4.0.4-1",
		Repo: []UpdatePackage{
			{Name: "omarchy", From: "4.0.3-1", To: "4.0.4-1"},
			{Name: "linux", From: "6.17.2-1", To: "6.17.4-1"},
		},
		AUR:     []UpdatePackage{{Name: "yay", From: "12.4.2-1", To: "12.5.0-1"}},
		Checked: dashboardNow(),
	}
}

func TestRenderGPU(t *testing.T) {
	r := chromeRenderer(Compact)
	out := ansi.Strip(r.RenderGPU(gpuFixture(), 32))
	for _, want := range []string{"GPU", "7%", "MEM", "29.2 GB", "31.0 GB", "48°C", "16.2 W", "1003 MHz"} {
		if !strings.Contains(out, want) {
			t.Fatalf("gpu missing %q:\n%s", want, out)
		}
	}
	// An integrated GPU's memory is near-full by construction, so it is shown
	// as a figure rather than a gauge that would always read as critical.
	if strings.Contains(out, "94%") {
		t.Fatalf("integrated memory should not be gauged as a percentage:\n%s", out)
	}
	detail := ansi.Strip(r.RenderGPUDetail(gpuFixture(), 36))
	for _, want := range []string{"DEVICE", "amdgpu", "Shared with system memory"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("gpu detail missing %q:\n%s", want, detail)
		}
	}
	// A discrete card does get the gauge, where the percentage means something.
	discrete := ansi.Strip(r.RenderGPU(GPUMetrics{Name: "nvidia", BusyPercent: 63,
		MemoryLabel: "VRAM", MemoryUsed: "5.1 GB", MemoryTotal: "8.0 GB", MemoryFrac: 0.64}, 32))
	if !strings.Contains(discrete, "VRAM") || !strings.Contains(discrete, "64%") {
		t.Fatalf("discrete gpu should gauge VRAM:\n%s", discrete)
	}
	// Rows with no reading are left out rather than shown as zero.
	if strings.Contains(discrete, "TEMP") {
		t.Fatalf("absent temperature should be omitted:\n%s", discrete)
	}
	if empty := ansi.Strip(r.RenderGPU(GPUMetrics{}, 24)); !strings.Contains(empty, "No GPU") {
		t.Fatalf("empty gpu = %q", empty)
	}
}

func TestRenderUpdates(t *testing.T) {
	r := chromeRenderer(Compact)
	out := ansi.Strip(r.RenderUpdates(updatesFixture(), 34))
	for _, want := range []string{"Omarchy", "4.0.3-1", "4.0.4-1", "3", "updates", "linux", "yay"} {
		if !strings.Contains(out, want) {
			t.Fatalf("updates missing %q:\n%s", want, out)
		}
	}
	// A clean system says so rather than showing an empty panel.
	clean := ansi.Strip(r.RenderUpdates(UpdateStatus{Omarchy: "4.0.4-1"}, 30))
	if !strings.Contains(clean, "Up to date") {
		t.Fatalf("clean updates = %q", clean)
	}
	if strings.Contains(clean, "→") {
		t.Fatalf("no pending version should mean no arrow:\n%s", clean)
	}
	// A failed check must not read as "nothing to do".
	broken := ansi.Strip(r.RenderUpdates(UpdateStatus{Omarchy: "4.0.4-1", Unavailable: "checkupdates unavailable"}, 34))
	if !strings.Contains(broken, "checkupdates unavailable") {
		t.Fatalf("unavailable updates = %q", broken)
	}
	if strings.Contains(broken, "Up to date") {
		t.Fatalf("an unknown count must not claim the system is current:\n%s", broken)
	}
	// One update reads naturally.
	single := ansi.Strip(r.RenderUpdates(UpdateStatus{Repo: []UpdatePackage{{Name: "linux", To: "6.17.4-1"}}}, 30))
	if !strings.Contains(single, "1  update") || strings.Contains(single, "updates") {
		t.Fatalf("single update wording = %q", single)
	}
	// The list is capped and says how many were left out.
	many := UpdateStatus{}
	for i := 0; i < 9; i++ {
		many.Repo = append(many.Repo, UpdatePackage{Name: fmt.Sprintf("pkg%d", i), To: "1.0"})
	}
	capped := ansi.Strip(r.RenderUpdates(many, 30))
	if !strings.Contains(capped, "+5 more") {
		t.Fatalf("capped list should report the remainder:\n%s", capped)
	}
}

func TestRenderRepoActivityShowsWhatIsOutstanding(t *testing.T) {
	r := chromeRenderer(Compact)

	// A repository with unpushed commits must not read as clean: this is the
	// state a dashboard is best placed to catch.
	icons := r.Styles.RepoIcons
	unpushed := ansi.Strip(r.RenderRepoActivity([]RepoActivity{
		{Name: "z13control", Branch: "main", Ahead: 6, Summary: "6 unpushed", Tone: ToneWarning},
	}, 40))
	if !strings.Contains(unpushed, icons.Ahead+"6") {
		t.Fatalf("unpushed work missing:\n%s", unpushed)
	}
	if strings.Contains(unpushed, icons.Clean) {
		t.Fatalf("a repo with unpushed commits should not be ticked:\n%s", unpushed)
	}

	// The tick belongs only to a repository with nothing outstanding.
	clean := ansi.Strip(r.RenderRepoActivity([]RepoActivity{
		{Name: "tide", Branch: "main", Summary: "clean", Tone: ToneGood},
	}, 40))
	if !strings.Contains(clean, icons.Clean) {
		t.Fatalf("a clean repo should be ticked:\n%s", clean)
	}

	// An interrupted rebase is flagged rather than blending in.
	mid := ansi.Strip(r.RenderRepoActivity([]RepoActivity{
		{Name: "olympus", Branch: "summer", State: "rebase", Summary: "REBASE", Tone: ToneDanger},
	}, 40))
	if !strings.Contains(mid, icons.Conflict) || !strings.Contains(mid, "REBASE") {
		t.Fatalf("an interrupted rebase should be flagged:\n%s", mid)
	}

	// Long branch names are elided from the middle, not left to overflow.
	long := ansi.Strip(r.RenderRepoActivity([]RepoActivity{
		{Name: "threadtide", Branch: "milestone-2.5-account-panel-and-compliance", Summary: "clean"},
	}, 40))
	for _, line := range strings.Split(long, "\n") {
		if lipgloss.Width(line) != 40 {
			t.Fatalf("line width = %d, want 40: %q", lipgloss.Width(line), line)
		}
	}
	if strings.Contains(long, "milestone-2.5-account-panel-and-compliance") {
		t.Fatalf("a 42-cell branch name should be elided at width 40:\n%s", long)
	}
	if !strings.Contains(long, "…") {
		t.Fatalf("elision should be marked:\n%s", long)
	}

	// The detail view restores what the compact line drops for space.
	detail := ansi.Strip(r.RenderRepoActivityDetail([]RepoActivity{
		{Name: "tidedeck", Branch: "main", Commits: 4, Changes: 25, Ahead: 2, Behind: 1,
			Summary: "2 unpushed · 25 changed", Tone: ToneWarning},
	}, 44))
	for _, want := range []string{icons.Changed + "25", "4 commit(s) today", "2 ahead, 1 behind"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("repo detail missing %q:\n%s", want, detail)
		}
	}
}

func TestElideMiddle(t *testing.T) {
	// Short enough to fit is returned untouched.
	for _, value := range []string{"main", "feat/x"} {
		if got := elideMiddle(value, 10); got != value {
			t.Fatalf("elideMiddle(%q, 10) = %q, want it unchanged", value, got)
		}
	}
	// A long branch is cut to exactly the width, keeping both ends so the
	// start and the distinguishing suffix both survive.
	const branch = "milestone-2.5-account-panel-and-compliance"
	for _, width := range []int{8, 12, 20, 30} {
		got := elideMiddle(branch, width)
		if lipgloss.Width(got) != width {
			t.Fatalf("elideMiddle(branch, %d) = %q, width %d", width, got, lipgloss.Width(got))
		}
		if !strings.Contains(got, "…") {
			t.Fatalf("elideMiddle(branch, %d) = %q, want an ellipsis", width, got)
		}
		if !strings.HasPrefix(got, "m") || !strings.HasSuffix(got, "e") {
			t.Fatalf("elideMiddle(branch, %d) = %q, want both ends kept", width, got)
		}
	}
	// Too narrow for an ellipsis to be worth a cell: hard clamp, still bounded.
	if got := elideMiddle("abcdef", 3); lipgloss.Width(got) > 3 {
		t.Fatalf("elideMiddle(abcdef, 3) = %q, wider than 3", got)
	}
	// Multi-byte text must be cut on rune boundaries, not bytes.
	if got := elideMiddle("ünïcödé-brånch-nåme", 9); lipgloss.Width(got) != 9 {
		t.Fatalf("elideMiddle(unicode, 9) = %q, width %d", got, lipgloss.Width(got))
	}
}

func TestGraphsUsePanelWidth(t *testing.T) {
	r := NewRenderer(CatppuccinMocha, StyleOptions{Density: Compact})
	const width = 48

	network := r.RenderNetwork(NetworkMetrics{
		DownSpark: []float64{0.1, 0.4, 0.2, 0.8},
	}, width)
	if got := lipgloss.Width(strings.Split(network, "\n")[2]); got != width {
		t.Fatalf("network sparkline width = %d, want %d", got, width)
	}

	system := r.RenderSystem(SystemMetrics{CPUSpark: []float64{0.1, 0.4, 0.8}}, width)
	if got := lipgloss.Width(strings.Split(system, "\n")[0]); got != width {
		t.Fatalf("system sparkline row width = %d, want %d", got, width)
	}
}
