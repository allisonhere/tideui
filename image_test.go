package tideui

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/muesli/termenv"
)

// noPlaceholders points the image transport at a terminal without the protocol, so
// an assertion about half-blocks holds wherever the suite runs. Without this the
// developer's own terminal decides what the test sees: TERM_PROGRAM=ghostty is set
// by Ghostty itself, and the picture then arrives as placeholder cells.
func noPlaceholders(t *testing.T) {
	t.Helper()
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TMUX", "")
}

// The colour profile is a package-level global in lipgloss and every test in this
// package renders through it, so a test that changes it puts the old one back.
func withProfile(t *testing.T, profile termenv.Profile) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(profile)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
}

// solid is a picture of one colour, so a test can say exactly which colour a cell
// should be.
func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// A cell carries two samples: the top half is the glyph's foreground, the bottom
// its background. One column, one row, red over blue.
func TestRenderImagePutsTheTopSampleInTheForeground(t *testing.T) {
	noPlaceholders(t)
	withProfile(t, termenv.TrueColor)
	r := NewRenderer(CatppuccinMocha, StyleOptions{})
	img := image.NewRGBA(image.Rect(0, 0, 1, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})

	got := r.RenderImage(img, 1, 1)
	want := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Background(lipgloss.Color("#0000ff")).Render("▀")
	if !strings.Contains(got, want) {
		t.Fatalf("cell = %q, want it to contain %q", got, want)
	}
}

// The picture keeps its aspect and is centred: a cell is twice as tall as it is
// wide, so a square source at 40 cells is 20 rows, and a tall one is narrower
// than the box rather than stretched into it.
func TestRenderImageFitsInsideTheBoxAndCentres(t *testing.T) {
	noPlaceholders(t)
	withProfile(t, termenv.TrueColor)
	r := NewRenderer(CatppuccinMocha, StyleOptions{})

	square := solid(64, 64, color.RGBA{R: 255, A: 255})
	lines := strings.Split(ansi.Strip(r.RenderImage(square, 40, 24)), "\n")
	if len(lines) != 20 {
		t.Fatalf("a square source drew %d rows, want 20", len(lines))
	}
	if got := ansi.StringWidth(lines[0]); got != 40 {
		t.Fatalf("line width = %d, want the box width 40", got)
	}

	// Four times as tall as it is wide: the box is 40 by 24, and 24 rows is 48
	// samples, so a 1:4 picture is 12 cells wide and centred in the 40.
	tall := solid(16, 64, color.RGBA{G: 255, A: 255})
	lines = strings.Split(ansi.Strip(r.RenderImage(tall, 40, 24)), "\n")
	if len(lines) != 24 {
		t.Fatalf("a tall source drew %d rows, want 24", len(lines))
	}
	if strings.Count(lines[0], "▀") != 12 {
		t.Fatalf("a tall source drew %q, want 12 cells", lines[0])
	}
	if !strings.HasPrefix(lines[0], strings.Repeat(" ", 14)) {
		t.Fatalf("the picture is not centred: %q", lines[0])
	}

	// A wide source is capped by the width, not the height.
	wide := solid(256, 16, color.RGBA{B: 255, A: 255})
	if lines := strings.Split(ansi.Strip(r.RenderImage(wide, 40, 24)), "\n"); len(lines) >= 20 {
		t.Fatalf("a 16:1 source drew %d rows", len(lines))
	}
}

// A transparent sample shows the panel through it: a radar tile is transparent
// where it does not rain, and black there would be a lie about the weather.
func TestRenderImageCompositesTransparencyOntoTheBackground(t *testing.T) {
	noPlaceholders(t)
	withProfile(t, termenv.TrueColor)
	r := NewRenderer(CatppuccinMocha, StyleOptions{})
	bg := r.Styles.Workspace.Bg

	clear := solid(2, 2, color.RGBA{})
	got := r.RenderImage(clear, 1, 1)
	want := lipgloss.NewStyle().Foreground(bg).Background(bg).Render("▀")
	if !strings.Contains(got, want) {
		t.Fatalf("a transparent image drew %q, want the panel background", got)
	}
}

// A terminal with no colour at all still gets the shape: the brightness as a
// ramp, because a panel that draws nothing looks broken.
func TestRenderImageFallsBackToARampWithoutColour(t *testing.T) {
	// Ascii is what a test binary actually detects, but say so out loud rather
	// than depending on it - and put back whatever the other tests were using,
	// because the profile is one global for the whole package.
	noPlaceholders(t)
	withProfile(t, termenv.Ascii)
	r := NewRenderer(CatppuccinMocha, StyleOptions{})

	got := r.RenderImage(solid(4, 4, color.RGBA{R: 255, A: 255}), 4, 4)
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("a colourless terminal got escape codes: %q", got)
	}
	line := strings.Split(got, "\n")[0]
	if ansi.StringWidth(line) != 4 {
		t.Fatalf("ramp line = %q, want 4 cells", line)
	}
	if strings.TrimSpace(line) == "" {
		t.Fatalf("a bright image drew an empty ramp: %q", line)
	}

	// And nothing is drawn for nothing: no image, no box.
	if got := r.RenderImage(nil, 10, 4); got != "" {
		t.Fatalf("a nil image drew %q", got)
	}
	if got := r.RenderImage(solid(2, 2, color.RGBA{A: 255}), 0, 4); got != "" {
		t.Fatalf("a zero-width box drew %q", got)
	}
}

// A frame is a picture and the time it is about, which is all a radar panel
// needs to draw, and all a provider has to return.
func TestRadarFrameCarriesItsTime(t *testing.T) {
	when := time.Date(2026, 9, 18, 18, 5, 0, 0, time.Local)
	frame := RadarFrame{Time: when, Image: solid(2, 2, color.RGBA{A: 255})}
	if !frame.Time.Equal(when) {
		t.Fatalf("frame time = %v, want %v", frame.Time, when)
	}
	if frame.Image == nil {
		t.Fatal("frame has no image")
	}
}

// The transport is detected, never assumed: tofu in every other terminal would be
// worse than a smaller picture.
func TestPlaceholderCellsNeedsATerminalThatHasThem(t *testing.T) {
	env := func(kv map[string]string) func(string) string {
		return func(key string) string { return kv[key] }
	}
	cases := []struct {
		name    string
		env     map[string]string
		profile colorprofile.Profile
		want    bool
	}{
		{"kitty", map[string]string{"TERM": "xterm-kitty"}, colorprofile.TrueColor, true},
		{"kitty by window id", map[string]string{"KITTY_WINDOW_ID": "1"}, colorprofile.TrueColor, true},
		{"kitty inside tmux", map[string]string{"TERM": "xterm-kitty", "TMUX": "/tmp/tmux-1000/default,1,0"}, colorprofile.TrueColor, false},
		{"another terminal", map[string]string{"TERM": "xterm-256color"}, colorprofile.TrueColor, false},
		{"kitty without truecolor", map[string]string{"TERM": "xterm-kitty"}, colorprofile.ANSI256, false},
		{"ghostty", map[string]string{"TERM": "xterm-ghostty", "TERM_PROGRAM": "ghostty"}, colorprofile.TrueColor, true},
		{"ghostty by program name", map[string]string{"TERM_PROGRAM": "ghostty"}, colorprofile.TrueColor, true},
		{"ghostty inside tmux", map[string]string{"TERM": "xterm-ghostty", "TERM_PROGRAM": "ghostty", "TMUX": "/tmp/tmux-1000/default,1,0"}, colorprofile.TrueColor, false},
		{"ghostty without truecolor", map[string]string{"TERM": "xterm-ghostty"}, colorprofile.ANSI256, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := placeholderCells(env(tc.env), tc.profile); got != tc.want {
				t.Fatalf("placeholderCells = %v, want %v", got, tc.want)
			}
		})
	}
}

func resetTransmissions() {
	transmitted.Lock()
	defer transmitted.Unlock()
	transmitted.next, transmitted.byImage, transmitted.order = 0, map[uintptr]*transmission{}, nil
}

// A picture is transmitted the first time it is drawn and never again: a panel
// that repaints every second must not put a picture on the wire every second.
func TestKittyTransmitsAPictureOnceAndPlacesItWithCells(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("TMUX", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM_PROGRAM", "")
	resetTransmissions()

	r := NewRenderer(CatppuccinMocha, StyleOptions{})
	// Twice as tall as it is wide, so the box is 4 cells by 4 rows.
	picture := solid(8, 16, color.RGBA{R: 255, A: 255})

	first := r.RenderImage(picture, 4, 4)
	second := r.RenderImage(picture, 4, 4)

	if got := strings.Count(first, "\x1b_G"); got != 1 {
		t.Fatalf("first frame transmitted %d times, want 1", got)
	}
	if got := strings.Count(second, "\x1b_G"); got != 0 {
		t.Fatalf("second frame transmitted %d times, want 0", got)
	}
	// a=T with U=1, not a bare transmit: see the comment in renderImagePlaceholders
	// for the terminal that made that load-bearing.
	if !strings.Contains(first, "a=T") || !strings.Contains(first, "U=1") || !strings.Contains(first, "f=100") {
		t.Fatalf("transmit options are wrong: %q", first)
	}
	// The transmit names an id, and the cells that place it name the same one in
	// their foreground colour: that pairing is the whole protocol.
	if !strings.Contains(first, "i=1") {
		t.Fatalf("the transmit has no id: %q", first)
	}
	if !strings.Contains(second, "38;2;0;0;1") {
		t.Fatalf("cells do not carry image 1 as their colour: %q", second)
	}
	lines := strings.Split(ansi.Strip(second), "\n")
	if len(lines) != 4 {
		t.Fatalf("drew %d rows, want 4", len(lines))
	}
	for _, line := range lines {
		if got := strings.Count(line, string(kitty.Placeholder)); got != 4 {
			t.Fatalf("row %q has %d placeholders, want 4", line, got)
		}
	}

	// A different picture is transmitted under its own id.
	other := r.RenderImage(solid(8, 16, color.RGBA{B: 255, A: 255}), 4, 4)
	if !strings.Contains(other, "\x1b_G") {
		t.Fatalf("a new picture was not transmitted: %q", other)
	}
	if !strings.Contains(other, "38;2;0;0;2") {
		t.Fatalf("the second picture did not get its own id: %q", other)
	}
}

// The fallback is the one everybody else runs, so it gets its own test: no
// graphics sequence at all, and half-blocks instead.
func TestRenderImageStaysOnHalfBlocksWithoutKitty(t *testing.T) {
	noPlaceholders(t)
	withProfile(t, termenv.TrueColor)

	r := NewRenderer(CatppuccinMocha, StyleOptions{})
	got := r.RenderImage(solid(2, 2, color.RGBA{R: 255, A: 255}), 1, 1)
	if strings.Contains(got, "\x1b_G") {
		t.Fatalf("a terminal without kitty graphics was transmitted an image: %q", got)
	}
	if !strings.Contains(got, "▀") {
		t.Fatalf("no half-block cell was drawn: %q", got)
	}
}

// The transport only works if the sequence survives the renderer that puts the
// frame on the screen - cellbuf is the blend, not the writer. This is the check
// that found bubbles versus cellbuf disagreeing, kept so a future renderer that
// sanitises escape sequences fails here instead of on someone's desk.
func TestKittySequenceSurvivesTheRenderer(t *testing.T) {
	var out bytes.Buffer
	program := tea.NewProgram(kittyViewModel{}, tea.WithOutput(&out), tea.WithInput(strings.NewReader("")))
	go func() {
		time.Sleep(200 * time.Millisecond)
		program.Quit()
	}()
	if _, err := program.Run(); err != nil {
		t.Fatal(err)
	}
	frame := out.String()
	if !strings.Contains(frame, "\x1b_G") {
		t.Fatalf("the transmit sequence did not survive the renderer: %q", frame)
	}
	if !strings.Contains(frame, string(kitty.Placeholder)) {
		t.Fatalf("the placeholder cells did not survive the renderer: %q", frame)
	}
	if !strings.Contains(frame, string(kitty.Diacritic(0))) {
		t.Fatalf("the diacritics did not survive the renderer: %q", frame)
	}
}

type kittyViewModel struct{}

func (kittyViewModel) Init() tea.Cmd { return nil }

func (kittyViewModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return kittyViewModel{}, nil }

func (kittyViewModel) View() string {
	cells := strings.Repeat(string(kitty.Placeholder)+string(kitty.Diacritic(0))+string(kitty.Diacritic(0)), 2)
	return "\x1b_Ga=t,f=100,i=1,U=1,q=2;AAAA\x1b\\radar" + cells + "\n"
}

// A picture is sized by the shape of a cell, which is not 2:1 on every font: at
// 2.4 a square source in a 40-wide box is 17 rows, not 20, and a panel that
// assumed 2 stretched it. The app measures this from the terminal; a nonsense
// measurement is refused rather than obeyed.
func TestRenderImageSizesToTheCellAspectItIsGiven(t *testing.T) {
	noPlaceholders(t)
	withProfile(t, termenv.TrueColor)

	square := solid(64, 64, color.RGBA{R: 255, A: 255})
	measured := NewRenderer(CatppuccinMocha, StyleOptions{CellAspect: 2.4})
	if got := measured.CellAspect; got != 2.4 {
		t.Fatalf("CellAspect = %v, want 2.4", got)
	}
	if lines := strings.Split(ansi.Strip(measured.RenderImage(square, 40, 24)), "\n"); len(lines) != 17 {
		t.Fatalf("a 2.4 cell drew %d rows, want 17", len(lines))
	}
	if lines := strings.Split(ansi.Strip(NewRenderer(CatppuccinMocha, StyleOptions{}).RenderImage(square, 40, 24)), "\n"); len(lines) != 20 {
		t.Fatalf("an unmeasured cell drew %d rows, want the 2:1 default of 20", len(lines))
	}
	if got := NewRenderer(CatppuccinMocha, StyleOptions{CellAspect: 0.01}).CellAspect; got != defaultCellAspect {
		t.Fatalf("a nonsense measurement became %v, want the default %v", got, defaultCellAspect)
	}
}

// A frame carries where the reader is in the picture and what a pixel is worth on
// the ground, because a panel needs both and can compute neither from the picture.
func TestRadarFrameCarriesItsScale(t *testing.T) {
	frame := RadarFrame{Time: time.Now(), Image: solid(4, 4, color.RGBA{A: 255})}
	if frame.Centre != (image.Point{}) || frame.KilometresPerPixel != 0 {
		t.Fatal("a frame with no scale should have the zero value")
	}
	frame.Centre = image.Pt(3, 4)
	frame.KilometresPerPixel = 0.475
	if frame.Centre.X != 3 || frame.Centre.Y != 4 || frame.KilometresPerPixel != 0.475 {
		t.Fatalf("frame = %+v", frame)
	}
}

// A radar frame is a map frame with a time on it: one type for the radar and the
// basemap that goes under it, so the provider and the panel do not have to care
// which source a picture came from.
func TestRadarFrameIsAMapFrame(t *testing.T) {
	var frame MapFrame = RadarFrame{Time: time.Now(), Image: solid(4, 4, color.RGBA{A: 255})}
	if frame.Time.IsZero() || frame.Image == nil {
		t.Fatal("a radar frame does not satisfy the map frame")
	}
	if frame.Image.Bounds().Dx() != 4 {
		t.Fatalf("the picture did not survive the alias: %v", frame.Image.Bounds())
	}
	// And the other way round, which is what the basemap provider returns.
	var radar RadarFrame = frame
	if radar.KilometresPerPixel != frame.KilometresPerPixel {
		t.Fatal("the alias is not the same type")
	}
}
