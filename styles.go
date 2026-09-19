package tideui

import "github.com/charmbracelet/lipgloss"

// Density controls spacing across rows, chrome, and overlays. It is a
// first-class look-and-feel choice: dashboards prefer Dense, while reading
// oriented apps prefer Comfortable.
type Density string

const (
	// Comfortable adds breathing room between rows and in modals.
	Comfortable Density = "comfortable"
	// Compact is the default: no optional spacer rows.
	Compact Density = "compact"
	// Dense removes secondary metadata and tightens chrome for dashboards.
	Dense Density = "dense"
)

// IsComfortable reports whether the density adds extra spacing.
func (d Density) IsComfortable() bool { return d == Comfortable }

// IsDense reports whether the density strips secondary metadata.
func (d Density) IsDense() bool { return d == Dense }

// RowStride returns the terminal lines a list row occupies.
func (d Density) RowStride() int {
	if d == Comfortable {
		return 2
	}
	return 1
}

// ShowsSubtitles reports whether panel subtitles and secondary metadata are
// rendered at this density.
func (d Density) ShowsSubtitles() bool { return d != Dense }

// ShowsSecondary reports whether secondary/right-aligned metadata is rendered.
func (d Density) ShowsSecondary() bool { return d != Dense }

// PaneCorners selects the glyph set used for pane border corners.
type PaneCorners string

const (
	// SquareCorners renders pane borders with sharp corners. This is the default.
	SquareCorners PaneCorners = "square"
	// RoundCorners renders pane borders with rounded corners.
	RoundCorners PaneCorners = "round"
)

// GaugeStyle selects the glyph set used by progress bars and metric gauges.
type GaugeStyle string

const (
	GaugeSolid   GaugeStyle = "solid"   // █ fill / ░ track (default)
	GaugeBlocks  GaugeStyle = "blocks"  // ▰ fill / ▱ track
	GaugeCircles GaugeStyle = "circles" // ● fill / ○ track
	GaugeFisheye GaugeStyle = "fisheye" // ◉ fill / ○ track
	GaugeMarker  GaugeStyle = "marker"  // ─ track with a ● marker
	GaugeBars    GaugeStyle = "bars"    // ▮ fill / ▯ track
)

// GaugeStyles lists the available gauge styles in display order.
func GaugeStyles() []GaugeStyle {
	return []GaugeStyle{GaugeSolid, GaugeBlocks, GaugeCircles, GaugeFisheye, GaugeMarker, GaugeBars}
}

// SparklineStyle selects the glyph ramp used by sparklines.
type SparklineStyle string

const (
	SparkBlocks  SparklineStyle = "blocks"  // ▁▂▃▄▅▆▇█ (default)
	SparkDots    SparklineStyle = "dots"    // ·∘○◉●
	SparkBraille SparklineStyle = "braille" // ⡀⡄⡆⡇⣇⣧⣷⣿
	SparkBullets SparklineStyle = "bullets" // ·○●
	SparkTicks   SparklineStyle = "ticks"   // ˌˈ│┃
	SparkShades  SparklineStyle = "shades"  // ░▒▓█
	// SparkHeat sizes and colours each sample by its own absolute value
	// rather than by where it falls within the run, so a quiet stretch stays
	// small and green instead of being stretched to fill the ramp. It reads
	// as a small heatmap: size carries magnitude, colour carries severity.
	SparkHeat SparklineStyle = "heat" // ·∘○◉● graded green → red
	// SparkWeighted bands the same way as SparkHeat but draws weight rather
	// than area: each step is a heavier vertical stroke, so the run reads as
	// heavier and hotter as the value climbs. The weight progression carries
	// the signal on its own, so it still reads on a monochrome terminal.
	SparkWeighted SparklineStyle = "weighted" // ╵╷│┃█ graded green → red
	// SparkStroke bands like the others but grows horizontally: each step is
	// a longer, heavier stroke, so a busy stretch reads as a thicker line.
	SparkStroke SparklineStyle = "stroke" // ╴─━█ graded green → red
)

// StrokeGlyphs is the default ramp for SparkStroke, lightest to heaviest.
const StrokeGlyphs = "╴─━█"

// StrokeGlyphsASCII replaces StrokeGlyphs where the theme asks for ASCII.
const StrokeGlyphsASCII = ".-=#"

// WeightGlyphs is the default ramp for SparkWeighted, lightest to heaviest.
const WeightGlyphs = "╵╷│┃█"

// WeightGlyphsASCII replaces WeightGlyphs where the theme asks for ASCII,
// keeping the sense of increasing weight.
const WeightGlyphsASCII = ".:|#@"

// HeatGlyphs is the default ramp for SparkHeat, smallest to largest.
const HeatGlyphs = "·∘○◉●"

// HeatGlyphsASCII replaces HeatGlyphs where the theme asks for ASCII.
const HeatGlyphsASCII = ".oO0@"

// DefaultHeatBands are the upper bounds of each heat band, as fractions of
// full scale: 0-20%, 21-40%, 41-60%, 61-80%, 81-100%.
func DefaultHeatBands() []float64 { return []float64{0.2, 0.4, 0.6, 0.8, 1} }

// DefaultStrokeBands are the upper bounds of each stroke band: 0-25%, 26-50%,
// 51-75%, 76-100%.
func DefaultStrokeBands() []float64 { return []float64{0.25, 0.5, 0.75, 1} }

// BandedRamp is one absolute-banded sparkline style's resolved glyphs and the
// thresholds between them. The two are always the same length.
type BandedRamp struct {
	Glyphs []rune
	Bands  []float64
}

// bandedDefault describes a banded style before any theme override.
type bandedDefault struct {
	unicode string
	ascii   string
	bands   func() []float64
}

// bandedDefaults is the set of sparkline styles that band on absolute value
// rather than scaling to the run. Adding a style here is all it takes for it
// to be configurable and rendered.
var bandedDefaults = map[SparklineStyle]bandedDefault{
	SparkHeat:     {HeatGlyphs, HeatGlyphsASCII, DefaultHeatBands},
	SparkWeighted: {WeightGlyphs, WeightGlyphsASCII, DefaultHeatBands},
	SparkStroke:   {StrokeGlyphs, StrokeGlyphsASCII, DefaultStrokeBands},
}

// SparklineStyles lists the available sparkline styles in display order.
func SparklineStyles() []SparklineStyle {
	return []SparklineStyle{SparkBlocks, SparkDots, SparkBraille, SparkBullets, SparkTicks, SparkShades, SparkHeat, SparkWeighted, SparkStroke}
}

// IconStyle selects the glyph family used for widget icons.
type IconStyle string

const (
	// IconEmoji is the default: the weather widget already draws emoji
	// unconditionally, so icons elsewhere match it rather than looking like a
	// different toolkit. An ASCII theme still falls back to ASCII.
	IconEmoji IconStyle = "emoji" // colour emoji, two cells (default)
	IconPlain IconStyle = "plain" // line-drawing symbols, one cell
	IconNerd  IconStyle = "nerd"  // Nerd Font glyphs, one cell, patched fonts only
)

// IconStyles lists the available icon styles in display order.
func IconStyles() []IconStyle { return []IconStyle{IconEmoji, IconPlain, IconNerd} }

func normalizeIconStyle(i IconStyle) IconStyle {
	for _, known := range IconStyles() {
		if i == known {
			return i
		}
	}
	return IconEmoji
}

// RepoIcons are the marks the git widget draws for each kind of repository
// state. Every field must be exactly one cell wide.
type RepoIcons struct {
	Branch     string // before a branch name
	Ahead      string // commits not pushed
	Behind     string // commits not pulled
	Changed    string // uncommitted files
	Stash      string // stashed entries
	NoUpstream string // branch tracks nothing
	Clean      string // nothing outstanding
	Attention  string // something outstanding
	Conflict   string // rebase or merge in progress
	Missing    string // not a working copy
}

// DefaultRepoIcons uses symbols present in essentially every monospace font.
// Each one was checked against a real terminal font before being chosen: a
// code point that a font lacks renders as a blank cell, and a blank is worse
// than a plainer mark that always draws. Rejected on those grounds: ⎇ and ⑂
// (branch), ✔ and ✘ (ticks), ⇡ and ⇣ (arrows), ✎ (pencil), ⚑ (flag), ⌁ and ⌀
// (no upstream).
func DefaultRepoIcons() RepoIcons {
	return RepoIcons{
		Branch: "", Ahead: "↑", Behind: "↓", Changed: "±", Stash: "≡",
		NoUpstream: "∅", Clean: "✓", Attention: "•", Conflict: "⚠", Missing: "✗",
	}
}

// EmojiRepoIcons uses colour emoji, matching the weather widget. Every glyph
// is two cells wide, which the width table agrees with, so rows still line up.
func EmojiRepoIcons() RepoIcons {
	return RepoIcons{
		Branch:     "\U0001F33F",       // herb
		Ahead:      "\U00002B06\uFE0F", // up arrow
		Behind:     "\U00002B07\uFE0F", // down arrow
		Changed:    "\U0001F4DD",       // memo
		Stash:      "\U0001F4E6",       // package
		NoUpstream: "\U0001F6AB",       // prohibited
		Clean:      "\U00002705",       // check mark button
		Attention:  "\U0001F7E1",       // yellow circle
		Conflict:   "\U000026A0\uFE0F", // warning
		Missing:    "\U0000274C",       // cross mark
	}
}

// NerdRepoIcons uses Nerd Font glyphs. They are private-use code points, so
// they only draw in a patched font; opt in with StyleOptions.NerdIcons.
func NerdRepoIcons() RepoIcons {
	icons := DefaultRepoIcons()
	icons.Branch = "\ue725"  // git branch
	icons.Changed = "\uf040" // pencil
	icons.Stash = "\uf487"   // box
	return icons
}

// ASCIIRepoIcons is the fallback for themes that ask for ASCII.
func ASCIIRepoIcons() RepoIcons {
	return RepoIcons{
		Branch: "", Ahead: "^", Behind: "v", Changed: "*", Stash: "=",
		NoUpstream: "?", Clean: "ok", Attention: "!", Conflict: "!!", Missing: "x",
	}
}

// ClockFont selects the glyph set used for the large digital clock.
type ClockFont string

const (
	ClockFontDash  ClockFont = "dash"  // light segments (─ │)
	ClockFontBlock ClockFont = "block" // solid blocks (█)
)

// ClockFonts lists the available clock fonts in display order.
func ClockFonts() []ClockFont {
	return []ClockFont{ClockFontDash, ClockFontBlock}
}

// StyleOptions controls density, pane corner style, and optional theme color replacements.
type StyleOptions struct {
	Density     Density
	PaneCorners PaneCorners
	Gauge       GaugeStyle
	Sparkline   SparklineStyle
	ClockFont   ClockFont
	Overrides   ThemeOverrides
	// SparkRamps replaces the glyphs of a banded sparkline style (heat,
	// weighted, stroke), lightest to heaviest. A missing or unusable entry
	// falls back to the built-in ramp, or its ASCII form for an ASCII theme.
	SparkRamps map[SparklineStyle]string
	// IconStyle picks the glyph family for widget icons. Empty means plain,
	// which is safe in any font; emoji and nerd need a colour emoji font or a
	// patched font respectively. An ASCII theme overrides this with ASCII.
	IconStyle IconStyle
	// RepoIcons overrides individual git widget icons. Empty fields keep the
	// resolved default.
	RepoIcons RepoIcons
	// CellAspect is how many times taller a terminal cell is than it is wide -
	// the number an image has to be sized by to keep its shape. Most fonts land
	// near 2 (a 7x15 cell, an 8x17 cell); measure it with CellAspectOf when the
	// terminal reports its pixels, and leave it zero to assume 2.
	CellAspect float64
	// CellWidth is one cell's width in pixels, measured with CellSizeOf. Zero
	// means nobody measured it, and anything that needs a pane's size in pixels
	// assumes 8 - the width a monospace cell has at a sane font size.
	CellWidth float64
	// SparkBands sets where a banded style changes glyph and colour, as
	// ascending fractions of full scale ending at 1, one per glyph in that
	// style's ramp. A missing or malformed entry falls back to the default.
	SparkBands map[SparklineStyle][]float64
	// ModalShadow draws a small drop shadow behind every modal overlay when
	// true. Off by default so existing consumers are unaffected unless they
	// opt in.
	ModalShadow bool
}

// Styles exposes resolved Lipgloss styles for composing application content.
// All styles carry the correct theme background so rendering any piece of
// content inside a pane produces a cohesive, uniformly coloured surface.
type Styles struct {
	Theme       Theme // resolved theme after any ThemeOverrides are applied
	PlainUI     bool  // true when the theme uses ASCII borders (e.g. VT52)
	Density     Density
	PaneCorners PaneCorners    // normalized; square unless RoundCorners was requested
	Gauge       GaugeStyle     // normalized; solid unless another style was requested
	Sparkline   SparklineStyle // normalized; blocks unless another style was requested
	ClockFont   ClockFont      // normalized; dash unless another font was requested

	// IconStyle is the normalized icon family, and RepoIcons the git icon set
	// resolved from it.
	IconStyle IconStyle
	RepoIcons RepoIcons

	// Banded holds the resolved configuration of each absolute-banded
	// sparkline style. Every ramp is validated against its own thresholds, so
	// a ramp and its bands are always the same length.
	Banded map[SparklineStyle]BandedRamp

	// ModalShadow and ModalShadowColor control the modal drop shadow drawn
	// by Renderer.Render. ModalShadowColor is resolved once here (from the
	// theme background) rather than recomputed on every render.
	ModalShadow      bool
	ModalShadowColor lipgloss.Color

	// Pane chrome — used internally; available for custom pane-like surfaces.
	Pane               lipgloss.Style // pane background fill
	PaneHeaderActive   lipgloss.Style // focused pane title bar
	PaneHeaderInactive lipgloss.Style // unfocused pane title bar

	// List items — pass to RenderRow or use directly for custom rows.
	Item         lipgloss.Style // default list row
	ItemMuted    lipgloss.Style // de-emphasised row (archived, read, disabled)
	ItemSelected lipgloss.Style // currently highlighted row

	// Inline annotations.
	Badge       lipgloss.Style // unread count or short tag rendered inside a row
	SearchMatch lipgloss.Style // highlighted substring within a search result

	// Detail pane content — compose these to build multi-section detail views.
	DetailTitle     lipgloss.Style // bold accent-background heading (e.g. subject line)
	DetailMeta      lipgloss.Style // dimmed italic secondary line (e.g. author, date)
	DetailBody      lipgloss.Style // plain body text; wraps to Width when set
	DetailFocusLine lipgloss.Style // subtle highlight for the cursor row in a detail list

	// Status bar segments — render text then join with StatusBarSeparator.
	StatusBar     lipgloss.Style // main status bar background and text
	StatusError   lipgloss.Style // error message segment (bold, error colour)
	StatusSuccess lipgloss.Style // success/confirmation segment (bold, Unread colour)
	StatusHint    lipgloss.Style // low-contrast keyboard hint
	StatusNotice  lipgloss.Style // accent-background announcement segment
	// StatusBarJoiner renders the separator returned by StatusBarSeparator.
	StatusBarJoiner lipgloss.Style

	// Overlay / modal content — used by renderOverlay; available for custom modals.
	Overlay      lipgloss.Style // modal border and background
	OverlayTitle lipgloss.Style // accent-background modal heading
	OverlayBody  lipgloss.Style // modal body text
	OverlayHint  lipgloss.Style // dimmed footer hint inside the modal

	// Form inputs — for text fields rendered inside an overlay.
	InputFocused lipgloss.Style // text field with focus border
	InputIdle    lipgloss.Style // text field without focus
	InputLabel   lipgloss.Style // label above a text field

	// Workspace chrome — resolved from the theme so panels, tabs, docking
	// previews, and key hints all stay within one palette.
	Workspace WorkspaceStyles
}

// WorkspaceStyles is TideUI's semantic visual-token set. Every token is
// resolved from the active Theme with safe fallbacks, so any theme — including
// a hand-written or low-colour one — yields a coherent, readable result
// without the widgets naming a single raw colour.
type WorkspaceStyles struct {
	// Surfaces.
	Bg             lipgloss.Color // workspace page background, fills gutters
	SurfaceBg      lipgloss.Color // idle panel body
	FocusSurfaceBg lipgloss.Color // focused panel body (subtle lift)
	RaisedBg       lipgloss.Color // capsules, rails, hovered rows

	// Panel frames.
	FrameActive lipgloss.Color
	FrameIdle   lipgloss.Color
	FrameDimmed lipgloss.Color

	// Titles.
	TitleActiveBg lipgloss.Color
	TitleActiveFg lipgloss.Color
	TitleIdleFg   lipgloss.Color
	TitleDimmedFg lipgloss.Color
	SubtitleFg    lipgloss.Color

	// Bodies.
	BodyFg       lipgloss.Color
	BodyDimmedFg lipgloss.Color
	BodyMutedFg  lipgloss.Color

	// Tabs.
	TabActiveBg lipgloss.Color
	TabActiveFg lipgloss.Color
	TabIdleBg   lipgloss.Color
	TabIdleFg   lipgloss.Color
	TabBadgeBg  lipgloss.Color
	TabBadgeFg  lipgloss.Color

	// Badges.
	BadgeFg        lipgloss.Color
	BadgeBg        lipgloss.Color
	BadgeGoodBg    lipgloss.Color
	BadgeGoodFg    lipgloss.Color
	BadgeWarningBg lipgloss.Color
	BadgeWarningFg lipgloss.Color
	BadgeDangerBg  lipgloss.Color
	BadgeDangerFg  lipgloss.Color

	// Selection.
	SelectionBg         lipgloss.Color
	SelectionFg         lipgloss.Color
	SelectionInactiveBg lipgloss.Color
	SelectionInactiveFg lipgloss.Color
	SelectionBar        lipgloss.Color
	FocusRail           lipgloss.Color
	HoverBg             lipgloss.Color

	// Separators and chrome.
	Separator lipgloss.Color

	// Keyboard hints.
	KeyBg       lipgloss.Color
	KeyFg       lipgloss.Color
	KeyMutedBg  lipgloss.Color
	FooterKeyFg lipgloss.Color
	HintFg      lipgloss.Color

	// Metrics.
	MetricGood    lipgloss.Color
	MetricWarning lipgloss.Color
	MetricBad     lipgloss.Color
	MetricTrack   lipgloss.Color
	MetricYellow  lipgloss.Color
	MetricOrange  lipgloss.Color

	// Weather.
	WeatherSun   lipgloss.Color
	WeatherCloud lipgloss.Color
	WeatherRain  lipgloss.Color
	WeatherSnow  lipgloss.Color
	WeatherStorm lipgloss.Color
	WeatherFog   lipgloss.Color

	// Arrange / docking.
	DockColor lipgloss.Color
	DockFill  lipgloss.Color
}

// WeatherColor maps a coarse weather kind to its themed colour, falling back to
// the muted body colour for unknown conditions.
func (ws WorkspaceStyles) WeatherColor(kind WeatherKind) lipgloss.Color {
	switch kind {
	case WeatherClear:
		return ws.WeatherSun
	case WeatherPartly:
		return ws.WeatherSun
	case WeatherCloudy:
		return ws.WeatherCloud
	case WeatherFog:
		return ws.WeatherFog
	case WeatherRain:
		return ws.WeatherRain
	case WeatherSnow:
		return ws.WeatherSnow
	case WeatherStorm:
		return ws.WeatherStorm
	default:
		return ws.BodyMutedFg
	}
}

// MetricGradient maps a 0..1 fraction onto the conventional metric scale:
// green at the low end, then yellow, orange, and red at the high end. It is
// used to colour sparkline cells by their place in the value range.
func (ws WorkspaceStyles) MetricGradient(fraction float64) lipgloss.Color {
	stops := [...]lipgloss.Color{ws.MetricGood, ws.MetricYellow, ws.MetricOrange, ws.MetricBad}
	f := clamp01(fraction)
	scaled := f * float64(len(stops)-1)
	low := int(scaled)
	if low >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	return MixColors(stops[low], stops[low+1], scaled-float64(low))
}

func buildWorkspaceStyles(t Theme) WorkspaceStyles {
	accent := t.BorderFocus
	if accent == "" {
		accent = t.OverlayBorder
	}
	if accent == "" {
		accent = t.Border
	}
	idle := t.Border
	if idle == "" {
		idle = accent
	}
	dimStep := 0.05
	if !isDark(t.Bg) {
		dimStep = -dimStep
	}
	surface := t.Bg
	focusSurface := focusLineBg(t)
	selection := selectionBgForRatio(t.Bg, selectedBgMinContrast)
	selectionInactive := selectionBgForRatio(t.Bg, 1.8)
	keyBg := selectionBgForRatio(t.Bg, 2.2)
	good := readableText(t.Unread, t.Bg, 3.0)
	bad := readableText(t.Error, t.Bg, 3.0)
	warning := readableText(MixColors(t.Unread, t.Error, 0.5), t.Bg, 3.0)
	if warning == "" {
		warning = accent
	}
	muted := mutedText(t.Fg, t.Bg)
	separator := adjustLightness(idle, dimStep)
	toneBg := func(c lipgloss.Color) lipgloss.Color { return MixColors(t.Bg, c, 0.22) }
	seedColor := func(c lipgloss.Color) lipgloss.Color { return readableText(c, t.Bg, 3.0) }
	return WorkspaceStyles{
		Bg:             t.Bg,
		SurfaceBg:      surface,
		FocusSurfaceBg: focusSurface,
		RaisedBg:       selectionInactive,

		FrameActive: workspaceFocusColor(accent, t.Bg),
		FrameIdle:   idle,
		FrameDimmed: adjustLightness(idle, dimStep),

		TitleActiveBg: accent,
		TitleActiveFg: readableText(t.Fg, accent, 4.5),
		TitleIdleFg:   muted,
		TitleDimmedFg: muted,
		SubtitleFg:    muted,

		BodyFg:       readableText(t.Fg, t.Bg, contrastText),
		BodyDimmedFg: muted,
		BodyMutedFg:  muted,

		TabActiveBg: accent,
		TabActiveFg: readableText(t.Fg, accent, 4.5),
		TabIdleBg:   surface,
		TabIdleFg:   muted,
		TabBadgeBg:  selection,
		TabBadgeFg:  readableText(accent, selection, 4.5),

		BadgeFg:        good,
		BadgeBg:        selection,
		BadgeGoodBg:    toneBg(good),
		BadgeGoodFg:    good,
		BadgeWarningBg: toneBg(warning),
		BadgeWarningFg: warning,
		BadgeDangerBg:  toneBg(bad),
		BadgeDangerFg:  bad,

		SelectionBg:         selection,
		SelectionFg:         readableText(accent, selection, 4.5),
		SelectionInactiveBg: selectionInactive,
		SelectionInactiveFg: readableText(t.Fg, selectionInactive, 4.5),
		SelectionBar:        readableText(accent, t.Bg, 3.0),
		FocusRail:           MixColors(workspaceFocusColor(accent, t.Bg), t.Bg, 0.45),
		HoverBg:             focusSurface,

		Separator: separator,

		KeyBg:       keyBg,
		KeyFg:       readableText(t.Fg, keyBg, 4.5),
		KeyMutedBg:  surface,
		FooterKeyFg: readableText(accent, t.Bg, 3.0),
		HintFg:      muted,

		MetricGood:    good,
		MetricWarning: warning,
		MetricBad:     bad,
		MetricTrack:   separator,
		MetricYellow:  seedColor("#e5c07b"),
		MetricOrange:  seedColor("#d19a66"),

		WeatherSun:   seedColor("#e5c07b"),
		WeatherCloud: seedColor("#8b98a5"),
		WeatherRain:  seedColor("#61afef"),
		WeatherSnow:  seedColor("#b8d4f0"),
		WeatherStorm: seedColor("#c678dd"),
		WeatherFog:   seedColor("#9aa5b1"),

		DockColor: readableText(accent, t.Bg, paneFocusMinContrast),
		DockFill:  MixColors(t.Bg, accent, 0.18),
	}
}

func normalizeDensity(d Density) Density {
	switch d {
	case Comfortable:
		return Comfortable
	case Dense:
		return Dense
	default:
		return Compact
	}
}

func normalizePaneCorners(c PaneCorners) PaneCorners {
	if c == RoundCorners {
		return RoundCorners
	}
	return SquareCorners
}

func normalizeGaugeStyle(g GaugeStyle) GaugeStyle {
	for _, known := range GaugeStyles() {
		if g == known {
			return g
		}
	}
	return GaugeSolid
}

func normalizeSparklineStyle(s SparklineStyle) SparklineStyle {
	for _, known := range SparklineStyles() {
		if s == known {
			return s
		}
	}
	return SparkBlocks
}

// resolveRepoIcons picks the base icon set for the theme and applies any
// per-field override.
func resolveRepoIcons(overrides RepoIcons, style IconStyle, plain bool) RepoIcons {
	icons := EmojiRepoIcons()
	switch {
	case plain:
		// An ASCII theme cannot show emoji or private-use glyphs whatever was
		// asked for, so it always wins.
		icons = ASCIIRepoIcons()
	case style == IconPlain:
		icons = DefaultRepoIcons()
	case style == IconNerd:
		icons = NerdRepoIcons()
	}
	for _, field := range []struct {
		override string
		target   *string
	}{
		{overrides.Branch, &icons.Branch}, {overrides.Ahead, &icons.Ahead},
		{overrides.Behind, &icons.Behind}, {overrides.Changed, &icons.Changed},
		{overrides.Stash, &icons.Stash}, {overrides.NoUpstream, &icons.NoUpstream},
		{overrides.Clean, &icons.Clean}, {overrides.Attention, &icons.Attention},
		{overrides.Conflict, &icons.Conflict}, {overrides.Missing, &icons.Missing},
	} {
		if field.override != "" {
			*field.target = field.override
		}
	}
	return icons
}

// resolveBanded builds every banded style's ramp, applying any theme override
// and falling back to the built-in defaults.
func resolveBanded(ramps map[SparklineStyle]string, bands map[SparklineStyle][]float64, plain bool) map[SparklineStyle]BandedRamp {
	out := make(map[SparklineStyle]BandedRamp, len(bandedDefaults))
	for style, def := range bandedDefaults {
		glyphs, thresholds := normalizeBanded(ramps[style], bands[style], plain, def)
		out[style] = BandedRamp{Glyphs: glyphs, Bands: thresholds}
	}
	return out
}

// normalizeBanded resolves a ramp and its bands together, because they
// have to stay the same length: a band without a glyph could never be drawn.
// A malformed request falls back to the defaults rather than rendering a
// sparkline with holes in it.
func normalizeBanded(ramp string, bands []float64, plain bool, def bandedDefault) ([]rune, []float64) {
	glyphs := []rune(ramp)
	if len(glyphs) < 2 {
		glyphs = []rune(def.unicode)
		if plain {
			glyphs = []rune(def.ascii)
		}
	}
	resolved := append([]float64(nil), bands...)
	if !validHeatBands(resolved, len(glyphs)) {
		resolved = def.bands()
		if len(resolved) != len(glyphs) {
			// A custom ramp of another length gets evenly spaced bands.
			resolved = evenHeatBands(len(glyphs))
		}
	}
	return glyphs, resolved
}

// validHeatBands reports whether bands are usable: one per glyph, ascending,
// inside 0..1, and reaching the top of the scale.
func validHeatBands(bands []float64, glyphs int) bool {
	if len(bands) != glyphs || glyphs == 0 {
		return false
	}
	previous := 0.0
	for _, band := range bands {
		if band <= previous || band > 1 {
			return false
		}
		previous = band
	}
	return bands[len(bands)-1] >= 1
}

// evenHeatBands spaces bands evenly across the scale.
func evenHeatBands(count int) []float64 {
	bands := make([]float64, count)
	for i := range bands {
		bands[i] = float64(i+1) / float64(count)
	}
	return bands
}

func normalizeClockFont(f ClockFont) ClockFont {
	for _, known := range ClockFonts() {
		if f == known {
			return f
		}
	}
	return ClockFontDash
}

// ListItemLineStride returns the terminal-line height expected per rendered row.
func (s Styles) ListItemLineStride() int {
	return s.Density.RowStride()
}

// StatusBarSeparator returns theme-appropriate separator text for footer segments.
func (s Styles) StatusBarSeparator() string {
	if s.PlainUI {
		return " | "
	}
	return "  |  "
}

func paneBorder(plain bool) lipgloss.Border {
	if plain {
		return lipgloss.ASCIIBorder()
	}
	return lipgloss.NormalBorder()
}

func overlayBorder(plain bool) lipgloss.Border {
	if plain {
		return lipgloss.ASCIIBorder()
	}
	return lipgloss.RoundedBorder()
}

// paneFrameBorder picks the pane border glyph set: rounded corners when
// requested, falling back to ASCII for the plain VT52 theme either way.
func paneFrameBorder(plain bool, rounded bool) lipgloss.Border {
	if rounded {
		return overlayBorder(plain)
	}
	return paneBorder(plain)
}

// PaneFrame returns a full 4-sided border box for a pane, colored to signal
// whether that pane currently has focus. accent overrides the theme's
// BorderFocus color when the pane supplies its own (see Pane.Accent).
func (s Styles) PaneFrame(focused bool, accent lipgloss.Color) lipgloss.Style {
	style := s.Pane.Copy().
		Border(paneFrameBorder(s.PlainUI, s.PaneCorners == RoundCorners)).
		AlignVertical(lipgloss.Top)
	if !focused {
		return style.BorderForeground(s.Theme.Border)
	}
	if accent == "" {
		accent = s.Theme.BorderFocus
	}
	return style.BorderForeground(readableText(accent, s.Theme.Bg, paneFocusMinContrast))
}

// BuildStyles resolves a theme and options into reusable Lipgloss styles.
func BuildStyles(base Theme, options StyleOptions) Styles {
	t := options.Overrides.Apply(base)
	density := normalizeDensity(options.Density)
	paneCorners := normalizePaneCorners(options.PaneCorners)
	gauge := normalizeGaugeStyle(options.Gauge)
	sparkline := normalizeSparklineStyle(options.Sparkline)
	clockFont := normalizeClockFont(options.ClockFont)
	plain := t.UsesASCII()
	banded := resolveBanded(options.SparkRamps, options.SparkBands, plain)
	iconStyle := normalizeIconStyle(options.IconStyle)
	repoIcons := resolveRepoIcons(options.RepoIcons, iconStyle, plain)
	itemPadding := func(style lipgloss.Style) lipgloss.Style {
		if density == Comfortable {
			return style.Padding(0, 0, 1, 0)
		}
		return style
	}

	modalBG := modalSurface(t)
	modalBorder := t.OverlayBorder
	if modalBorder == "" {
		modalBorder = t.Border
	}
	modalAccent := t.BorderFocus
	if modalAccent == "" {
		modalAccent = modalBorder
	}
	modalPadTop, modalPadRight, modalPadBottom, modalPadLeft := 1, 2, 1, 2
	titleBottomMargin := 1
	if density == Compact {
		modalPadTop, modalPadRight, modalPadBottom, modalPadLeft = 0, 1, 0, 1
		titleBottomMargin = 0
	}

	selectedBG := selectionBgForRatio(t.Bg, selectedBgMinContrast)
	shadowBG := shadowColor(t.Bg)
	focusBG := focusLineBg(t)
	modalFG := readableText(t.Fg, modalBG, 4.5)
	modalMuted := mutedText(modalFG, modalBG)

	return Styles{
		Theme: t, PlainUI: plain, Density: density, PaneCorners: paneCorners,
		Gauge: gauge, Sparkline: sparkline, ClockFont: clockFont,
		Banded:      banded,
		IconStyle:   iconStyle,
		RepoIcons:   repoIcons,
		ModalShadow: options.ModalShadow, ModalShadowColor: shadowBG,
		Pane: lipgloss.NewStyle().Background(t.Bg).BorderBackground(t.Bg),
		PaneHeaderActive: lipgloss.NewStyle().Background(t.BorderFocus).
			Foreground(readableText(t.Fg, t.BorderFocus, 4.5)).Bold(true),
		PaneHeaderInactive: lipgloss.NewStyle().Background(t.Border).
			Foreground(readableText(t.Fg, t.Border, 4.5)),
		Item: itemPadding(lipgloss.NewStyle().Background(t.Bg).Foreground(t.Fg)),
		ItemMuted: itemPadding(lipgloss.NewStyle().Background(t.Bg).
			Foreground(readableText(t.Dimmed, t.Bg, 3.0))),
		ItemSelected: itemPadding(lipgloss.NewStyle().Background(selectedBG).
			Foreground(readableText(t.BorderFocus, selectedBG, 4.5)).Bold(true)),
		Badge: lipgloss.NewStyle().Foreground(t.Unread).Bold(true),
		DetailTitle: lipgloss.NewStyle().Background(t.BorderFocus).
			Foreground(readableText(t.Fg, t.BorderFocus, 4.5)).Bold(true).Padding(0, 1),
		DetailMeta: lipgloss.NewStyle().Background(t.Bg).
			Foreground(readableText(t.Dimmed, t.Bg, 3.0)).Italic(true),
		DetailBody: lipgloss.NewStyle().Background(t.Bg).Foreground(t.Fg),
		DetailFocusLine: lipgloss.NewStyle().Background(focusBG).
			Foreground(readableText(t.Fg, focusBG, 4.5)),
		SearchMatch: lipgloss.NewStyle().Background(t.BorderFocus).
			Foreground(readableText(t.Fg, t.BorderFocus, 4.5)),
		StatusBar: lipgloss.NewStyle().Background(t.StatusBar).
			Foreground(readableText(t.StatusFg, t.StatusBar, 4.5)).Padding(0, 1),
		StatusError: lipgloss.NewStyle().Background(t.StatusBar).
			Foreground(readableText(t.Error, t.StatusBar, 4.5)).Bold(true).Padding(0, 1),
		StatusSuccess: lipgloss.NewStyle().Background(t.StatusBar).
			Foreground(readableText(t.Unread, t.StatusBar, 4.5)).Bold(true).Padding(0, 1),
		StatusHint: lipgloss.NewStyle().Background(t.StatusBar).
			Foreground(readableText(t.StatusFg, t.StatusBar, 3.0)),
		StatusBarJoiner: lipgloss.NewStyle().Background(t.StatusBar).
			Foreground(readableText(t.StatusFg, t.StatusBar, 4.5)),
		StatusNotice: lipgloss.NewStyle().Background(t.BorderFocus).
			Foreground(readableText(t.Fg, t.BorderFocus, 4.5)).Bold(true).Padding(0, 1),
		Overlay: lipgloss.NewStyle().Background(modalBG).Border(overlayBorder(plain)).
			BorderForeground(modalBorder).
			BorderBackground(modalBG).
			Padding(modalPadTop, modalPadRight, modalPadBottom, modalPadLeft),
		OverlayTitle: lipgloss.NewStyle().Background(modalAccent).
			Foreground(readableText(t.Fg, modalAccent, 4.5)).Bold(true).Padding(0, 1).
			MarginBottom(titleBottomMargin),
		OverlayBody: lipgloss.NewStyle().Background(modalBG).Foreground(modalFG),
		OverlayHint: lipgloss.NewStyle().Background(modalBG).Foreground(modalMuted),
		InputFocused: lipgloss.NewStyle().Background(modalBG).Foreground(modalFG).
			Border(paneBorder(plain)).BorderForeground(modalAccent).BorderBackground(modalBG).Padding(0, 1),
		InputIdle: lipgloss.NewStyle().Background(modalBG).Foreground(modalFG).
			Border(paneBorder(plain)).BorderForeground(modalBorder).BorderBackground(modalBG).Padding(0, 1),
		InputLabel: lipgloss.NewStyle().Foreground(modalMuted),
		Workspace:  buildWorkspaceStyles(t),
	}
}
