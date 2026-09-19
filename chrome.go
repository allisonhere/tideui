package tideui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Tone is a semantic intent used by badges, metrics, and gauges. Every tone
// resolves to a theme token, so meaning never depends on a literal colour and
// survives low-colour themes.
type Tone int

const (
	// ToneNeutral is ordinary, non-emphasised content.
	ToneNeutral Tone = iota
	// ToneAccent marks the current/active thing.
	ToneAccent
	// ToneGood is success, healthy, or in-range.
	ToneGood
	// ToneWarning is degraded or approaching a limit.
	ToneWarning
	// ToneDanger is an error or out-of-range value.
	ToneDanger
	// ToneMuted is deliberately de-emphasised.
	ToneMuted
)

// KeyHint is one keyboard hint, rendered as a compact key capsule plus label.
type KeyHint struct {
	Key   string
	Label string
}

// Hint builds a KeyHint.
func Hint(key, label string) KeyHint { return KeyHint{Key: key, Label: label} }

// Badge is a short inline status label such as "saved", "3", or "warning".
type Badge struct {
	Text string
	Tone Tone
}

// NewBadge creates a neutral badge.
func NewBadge(text string) Badge { return Badge{Text: text} }

// WithTone returns the badge with a semantic tone.
func (b Badge) WithTone(tone Tone) Badge { b.Tone = tone; return b }

// TabItem is one tab in a tab strip.
type TabItem struct {
	Title string
	Badge string
	Muted bool
}

// ListItem is one polished list row with an optional icon, right-aligned meta,
// counter, indentation, and selection/hover state.
type ListItem struct {
	Icon     string
	Text     string
	Meta     string
	Counter  string
	Indent   int
	Selected bool
	Active   bool // the owning panel has focus
	Muted    bool
	Hover    bool
}

// MetricRow is one aligned metric line: label, value, optional gauge or
// sparkline, and a trend arrow. TotalWidth, when non-zero, is the full cell
// budget; the plot takes whatever remains after the label and value columns.
type MetricRow struct {
	Label      string
	Value      string
	Unit       string
	Fraction   float64 // 0..1
	Bar        bool    // render a gauge
	Spark      []float64
	Trend      int // -1 down, 0 flat, 1 up
	Tone       Tone
	LabelWidth int
	ValueWidth int
	TotalWidth int
}

// PlotWidth returns the cells available to a gauge or sparkline.
func (m MetricRow) PlotWidth() int {
	return max(0, m.TotalWidth-m.LabelWidth-m.ValueWidth-4)
}

// ProgressBar describes a standalone gauge.
type ProgressBar struct {
	Fraction float64
	Width    int
	Tone     Tone
}

// Sparkline describes a compact trend glyph run.
type Sparkline struct {
	Values []float64
	Width  int
	Tone   Tone
}

// SectionDivider is a labelled rule used to group content inside a panel.
type SectionDivider struct {
	Label string
	Width int
	// Tone colours the rule and its label. ToneNeutral, the zero value, keeps
	// the quiet separator colouring a divider has by default; a tone makes the
	// rule itself carry meaning, which is what a divider labelled with a state
	// wants - the words and the line should agree.
	Tone Tone
}

// FocusChrome centralises the decisions a themed panel makes about its focus
// state, so renderers and widgets agree on what "focused" means.
type FocusChrome struct {
	Presentation FocusPresentation
	Density      Density
}

// FrameColor returns the border colour for a panel's state.
func (fc FocusChrome) FrameColor(styles Styles, focused, dimmed bool, accent lipgloss.Color) lipgloss.Color {
	ws := styles.Workspace
	if focused {
		if accent != "" {
			return readableText(accent, ws.Bg, paneFocusMinContrast)
		}
		return ws.FrameActive
	}
	if dimmed {
		return ws.FrameDimmed
	}
	return ws.FrameIdle
}

// SurfaceBg returns the body background for a panel's state. Content rendered
// by applications uses Styles.Theme.Bg, so this stays aligned with it.
func (fc FocusChrome) SurfaceBg(styles Styles) lipgloss.Color {
	return styles.Workspace.Bg
}

// ShowRail reports whether a focused panel should draw an inner accent rail.
func (fc FocusChrome) ShowRail(focused bool) bool {
	return focused && fc.Presentation.FocusRail
}

// --- Badges ---------------------------------------------------------------

func (r Renderer) badgeStyle(tone Tone, bg lipgloss.Color) lipgloss.Style {
	ws := r.Styles.Workspace
	switch tone {
	case ToneAccent:
		return lipgloss.NewStyle().Background(ws.SelectionBg).Foreground(ws.SelectionFg).Bold(true)
	case ToneGood:
		return lipgloss.NewStyle().Background(ws.BadgeGoodBg).Foreground(ws.BadgeGoodFg).Bold(true)
	case ToneWarning:
		return lipgloss.NewStyle().Background(ws.BadgeWarningBg).Foreground(ws.BadgeWarningFg).Bold(true)
	case ToneDanger:
		return lipgloss.NewStyle().Background(ws.BadgeDangerBg).Foreground(ws.BadgeDangerFg).Bold(true)
	case ToneMuted:
		return lipgloss.NewStyle().Background(bg).Foreground(ws.HintFg)
	default:
		return lipgloss.NewStyle().Background(bg).Foreground(ws.BadgeFg).Bold(true)
	}
}

// RenderBadge renders a badge inline at the active page background.
func (r Renderer) RenderBadge(badge Badge) string {
	return r.RenderBadgeOn(badge, r.Styles.Workspace.Bg)
}

// RenderBadgeOn renders a badge over a specific background, for use inside
// headers and status bars.
func (r Renderer) RenderBadgeOn(badge Badge, bg lipgloss.Color) string {
	if badge.Text == "" {
		return ""
	}
	return r.badgeStyle(badge.Tone, bg).Render(" " + badge.Text + " ")
}

// --- Keyboard hints -------------------------------------------------------

// RenderKeyHints renders compact key capsules for a panel footer. Labels are
// included while they fit and dropped per-hint as space runs out, so a footer
// that shows only "c" is the fallback for a narrow panel, not the rule even in
// Dense mode: a key without its verb is not discoverable.
func (r Renderer) RenderKeyHints(hints []KeyHint, maxWidth int) string {
	ws := r.Styles.Workspace
	return r.renderHints(ws.Bg, ws.KeyBg, ws.KeyFg, ws.HintFg, maxWidth, hints, true)
}

// RenderStatusKeyHints renders key capsules for the workspace status strip.
// Labels are always attempted because discoverability matters more than the
// density setting in the global status bar.
func (r Renderer) RenderStatusKeyHints(hints []KeyHint, maxWidth int) string {
	s := r.Styles
	ws := s.Workspace
	labelFg := readableText(s.Theme.StatusFg, s.Theme.StatusBar, 3.0)
	return r.renderHints(s.Theme.StatusBar, ws.KeyBg, ws.KeyFg, labelFg, maxWidth, hints, true)
}

// keyGlyphs is one family of key symbols, so a hint can draw a key as the mark
// the keyboard carries rather than its name.
type keyGlyphs struct {
	enter, esc, tab, backtab, space, backspace, del string
	up, down, left, right                           string
	shift, ctrl                                     string
}

var (
	// plainKeyGlyphs uses symbols present in essentially every monospace font.
	plainKeyGlyphs = keyGlyphs{
		enter: "↵", esc: "␛", tab: "↹", backtab: "⇤", space: "␣",
		backspace: "⌫", del: "⌦",
		up: "▲", down: "▼", left: "◀", right: "▶",
		shift: "⇧", ctrl: "⌃",
	}
	// emojiKeyGlyphs uses colour emoji where one exists and the plain symbol
	// where it does not. Each emoji is two cells in the width table, so hints
	// still line up.
	emojiKeyGlyphs = keyGlyphs{
		enter: "\u21a9\ufe0f", esc: "␛", tab: "↹", backtab: "⇤", space: "␣",
		backspace: "⌫", del: "⌦",
		up: "\u2b06\ufe0f", down: "\u2b07\ufe0f", left: "\u2b05\ufe0f", right: "\u27a1\ufe0f",
		shift: "⇧", ctrl: "⌃",
	}
	// nerdKeyGlyphs uses Nerd Font key icons, verified against the patched
	// font's own glyph names. Private-use code points: patched fonts only.
	nerdKeyGlyphs = keyGlyphs{
		enter: "\uf0311", esc: "\uf12b7", tab: "\uf0312", backtab: "\uf0325",
		space: "\uf1050", backspace: "\uf030d", del: "\uf01b4",
		up: "\uf005e", down: "\uf0046", left: "\uf004e", right: "\uf0055",
		shift: "\uf0636", ctrl: "\uf0634",
	}
)

// glyph maps a key name to its symbol. Modifiers compose and a "/" list maps
// each part; an unknown key passes through, so letters and composite hints
// like "⇧arrows" are unchanged.
func (g keyGlyphs) glyph(key string) string {
	switch strings.ToLower(key) {
	case "enter", "return":
		return g.enter
	case "esc", "escape":
		return g.esc
	case "tab":
		return g.tab
	case "shift+tab":
		return g.backtab
	case "space":
		return g.space
	case "backspace":
		return g.backspace
	case "delete":
		return g.del
	case "up":
		return g.up
	case "down":
		return g.down
	case "left":
		return g.left
	case "right":
		return g.right
	}
	lower := strings.ToLower(key)
	switch {
	case strings.HasPrefix(lower, "shift+"):
		return g.shift + g.glyph(key[len("shift+"):])
	case strings.HasPrefix(lower, "ctrl+"):
		return g.ctrl + g.glyph(key[len("ctrl+"):])
	case strings.Contains(key, "/"):
		parts := strings.Split(key, "/")
		for i, part := range parts {
			parts[i] = g.glyph(part)
		}
		return strings.Join(parts, "/")
	}
	return key
}

// keyGlyph resolves a key's symbol under the renderer's icon style, so hints
// match the rest of the UI. An ASCII theme keeps the key's name.
func (r Renderer) keyGlyph(key string) string {
	if r.Styles.PlainUI {
		return key
	}
	switch r.Styles.IconStyle {
	case IconNerd:
		return nerdKeyGlyphs.glyph(key)
	case IconEmoji:
		return emojiKeyGlyphs.glyph(key)
	default:
		return plainKeyGlyphs.glyph(key)
	}
}

func (r Renderer) renderHints(bg, keyBg, keyFg, labelFg lipgloss.Color, maxWidth int, hints []KeyHint, tryLabels bool) string {
	if maxWidth <= 0 || len(hints) == 0 {
		return ""
	}
	keyStyle := lipgloss.NewStyle().Background(keyBg).Foreground(keyFg).Bold(true)
	labelStyle := lipgloss.NewStyle().Background(bg).Foreground(labelFg)
	gap := 2
	sep := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", gap))

	var b strings.Builder
	used := 0
	for _, hint := range hints {
		if hint.Key == "" && hint.Label == "" {
			continue
		}
		capsule := ""
		if hint.Key != "" {
			capsule = keyStyle.Render(" " + r.keyGlyph(hint.Key) + " ")
		}
		candidates := make([]string, 0, 2)
		if tryLabels && hint.Label != "" {
			candidates = append(candidates, capsule+labelStyle.Render(" "+strings.ToLower(hint.Label)))
		}
		candidates = append(candidates, capsule)

		placed := false
		for _, candidate := range candidates {
			cost := lipgloss.Width(candidate)
			if used > 0 {
				cost += gap
			}
			if used+cost > maxWidth {
				continue
			}
			if used > 0 {
				b.WriteString(sep)
			}
			b.WriteString(candidate)
			used += cost
			placed = true
			break
		}
		if !placed {
			break
		}
	}
	return b.String()
}

func joinWithin(parts []string, sep string, sepWidth, maxWidth int) (string, bool) {
	var b strings.Builder
	used := 0
	count := 0
	for i, part := range parts {
		cost := lipgloss.Width(part)
		if i > 0 {
			cost += sepWidth
		}
		if used+cost > maxWidth {
			break
		}
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(part)
		used += cost
		count++
	}
	return b.String(), count == len(parts)
}

func mustJoinWithin(parts []string, sep string, sepWidth, maxWidth int) string {
	joined, _ := joinWithin(parts, sep, sepWidth, maxWidth)
	return joined
}

// --- Tabs -----------------------------------------------------------------

type tabSegment struct {
	item  TabItem
	start int
	width int
}

// tabLabel is the plain text of a tab including its optional badge.
func tabLabel(item TabItem) string {
	if item.Badge != "" {
		return item.Title + " " + item.Badge
	}
	return item.Title
}

// tabSegments greedily fits tabs into innerWidth, returning placement and the
// total width consumed including separators.
func tabSegments(items []TabItem, innerWidth int) ([]tabSegment, int) {
	if innerWidth <= 0 || len(items) == 0 {
		return nil, 0
	}
	var segments []tabSegment
	used := 0
	for i, item := range items {
		if used >= innerWidth {
			break
		}
		available := innerWidth - used
		if i > 0 {
			if available <= 2 {
				break
			}
			available--
		}
		labelWidth := lipgloss.Width(tabLabel(item)) + 2
		if labelWidth > available {
			labelWidth = available
		}
		if labelWidth < 3 {
			break
		}
		start := used
		if i > 0 {
			start = used + 1
		}
		segments = append(segments, tabSegment{item: item, start: start, width: labelWidth})
		used = start + labelWidth
	}
	return segments, used
}

// RenderTabStrip renders a themed tab strip of up to width cells. It returns
// the rendered strip and the width consumed; callers pad the remainder with
// panel border chrome. Mouse hit boxes can be derived from the same segments.
func (r Renderer) RenderTabStrip(items []TabItem, active int, focused bool, width int) (string, int) {
	segments, used := tabSegments(items, width)
	if len(segments) == 0 {
		return "", 0
	}
	ws := r.Styles.Workspace
	plain := r.Styles.PlainUI
	var b strings.Builder
	for i, segment := range segments {
		if i > 0 {
			b.WriteString(lipgloss.NewStyle().Background(ws.Bg).Foreground(ws.FrameIdle).Render(separatorGlyph(plain)))
		}
		style := lipgloss.NewStyle().Background(ws.TabIdleBg).Foreground(ws.TabIdleFg)
		switch {
		case i == active && focused:
			style = lipgloss.NewStyle().Background(ws.TabActiveBg).Foreground(ws.TabActiveFg).Bold(true)
		case i == active:
			style = lipgloss.NewStyle().Background(ws.SelectionInactiveBg).Foreground(ws.SelectionInactiveFg).Bold(true)
		case segment.item.Muted:
			style = lipgloss.NewStyle().Background(ws.TabIdleBg).Foreground(ws.HintFg)
		}
		text := ansi.Truncate(tabLabel(segment.item), max(1, segment.width-2), "…")
		b.WriteString(style.Render(padRight(" "+text+" ", segment.width)))
	}
	return b.String(), used
}

func separatorGlyph(plain bool) string {
	if plain {
		return "|"
	}
	return "│"
}

// --- Panel header / footer ------------------------------------------------

// PanelHeader describes the top chrome of a panel.
type PanelHeader struct {
	Title        string
	Subtitle     string
	Badge        *Badge
	Right        string
	Tabs         []TabItem
	ActiveTab    int
	Focused      bool
	Dimmed       bool
	AccentMarker bool
	Capsule      bool
	Accent       lipgloss.Color
	Density      Density
	Border       lipgloss.Border
}

// RenderPanelHeader renders the inner content of a panel's top border line,
// exactly width cells wide (excluding the corner glyphs).
func (r Renderer) RenderPanelHeader(h PanelHeader, width int) string {
	if width <= 0 {
		return ""
	}
	if h.Border.Top == "" {
		h.Border = lipgloss.NormalBorder()
	}
	styles := r.Styles
	ws := styles.Workspace
	bg := ws.Bg
	borderStyle := lipgloss.NewStyle().Background(bg).Foreground(panelFrameColor(styles, h.Focused, h.Dimmed, h.Accent))

	if len(h.Tabs) > 0 {
		if content, used := r.RenderTabStrip(h.Tabs, h.ActiveTab, h.Focused, width); used > 0 {
			if pad := width - used; pad > 0 {
				content += borderStyle.Render(strings.Repeat(h.Border.Top, pad))
			}
			return clampStyled(content, width, ws.Bg)
		}
	}

	right := ""
	if h.Right != "" && h.Density.ShowsSecondary() {
		right = lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).Render(" " + h.Right + " ")
	}
	badge := ""
	if h.Badge != nil && h.Badge.Text != "" {
		badge = borderStyle.Render(h.Border.Top) + r.badgeStyle(h.Badge.Tone, bg).Render(" "+h.Badge.Text+" ")
	}

	titleStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.TitleIdleFg)
	switch {
	case h.Focused && h.Capsule:
		titleStyle = lipgloss.NewStyle().Background(ws.TitleActiveBg).Foreground(ws.TitleActiveFg).Bold(true)
	case h.Focused:
		titleStyle = lipgloss.NewStyle().Background(bg).Foreground(ws.FrameActive).Bold(true)
	case h.Dimmed:
		titleStyle = lipgloss.NewStyle().Background(bg).Foreground(ws.TitleDimmedFg)
	}

	marker := ""
	if h.Focused && h.AccentMarker {
		marker = markerGlyph(styles.PlainUI)
	}
	markerStr := ""
	if marker != "" {
		markerStr = marker + " "
	}
	subtitle := ""
	if h.Subtitle != "" && h.Density.ShowsSubtitles() {
		subtitle = h.Subtitle
	}

	avail := width - lipgloss.Width(right) - lipgloss.Width(badge) - 1
	if avail < 3 {
		right = ""
		avail = width - lipgloss.Width(badge) - 1
	}
	if avail < 3 {
		badge = ""
		avail = width - 1
	}
	if avail < 1 {
		avail = 1
	}

	subStr := ""
	if subtitle != "" {
		candidate := " " + subtitle + " "
		if lipgloss.Width(markerStr)+lipgloss.Width(h.Title)+lipgloss.Width(candidate)+4 <= avail {
			subStr = candidate
		}
	}
	titleBudget := avail - lipgloss.Width(markerStr) - lipgloss.Width(subStr) - 2
	if titleBudget < 1 {
		titleBudget = 1
	}
	title := ansi.Truncate(h.Title, titleBudget, "…")

	left := titleStyle.Render(" " + markerStr + title + " ")
	if subStr != "" {
		left += borderStyle.Render(h.Border.Top) + lipgloss.NewStyle().Background(bg).Foreground(ws.SubtitleFg).Render(subStr)
	}
	used := 1 + lipgloss.Width(left) + lipgloss.Width(badge) + lipgloss.Width(right)
	fill := max(0, width-used)
	result := borderStyle.Render(h.Border.Top) + left + borderStyle.Render(strings.Repeat(h.Border.Top, fill)) + badge + right
	return clampStyled(result, width, bg)
}

// PanelFooter describes the bottom chrome of a panel.
type PanelFooter struct {
	Hints   []KeyHint
	Mode    string
	Focused bool
	Dimmed  bool
	Accent  lipgloss.Color
	Density Density
	Border  lipgloss.Border
}

// RenderPanelFooter renders the inner content of a panel's bottom border line,
// exactly width cells wide (excluding the corner glyphs).
func (r Renderer) RenderPanelFooter(f PanelFooter, width int) string {
	if width <= 0 {
		return ""
	}
	if f.Border.Bottom == "" {
		f.Border = lipgloss.NormalBorder()
	}
	styles := r.Styles
	bg := styles.Workspace.Bg
	borderStyle := lipgloss.NewStyle().Background(bg).Foreground(panelFrameColor(styles, f.Focused, f.Dimmed, f.Accent))

	content := ""
	if f.Mode != "" {
		content = lipgloss.NewStyle().Background(styles.Workspace.SelectionBg).
			Foreground(styles.Workspace.SelectionFg).Bold(true).
			Render(" " + strings.ToUpper(f.Mode) + " ")
	} else if f.Focused && len(f.Hints) > 0 {
		if hints := r.RenderKeyHints(f.Hints, width-3); hints != "" {
			content = " " + hints + " "
		}
	}
	if lipgloss.Width(content) > width {
		content = ansi.Truncate(content, width, "…")
	}
	if content == "" {
		return borderStyle.Render(strings.Repeat(f.Border.Bottom, width))
	}
	fill := max(0, width-lipgloss.Width(content)-1)
	return borderStyle.Render(f.Border.Bottom) + content + borderStyle.Render(strings.Repeat(f.Border.Bottom, fill))
}

// --- Dividers, gauges, sparklines, metrics --------------------------------

// RenderSectionDivider renders a labelled rule of exactly width cells.
func (r Renderer) RenderSectionDivider(d SectionDivider, bg lipgloss.Color) string {
	if d.Width <= 0 {
		return ""
	}
	ws := r.Styles.Workspace
	ruleGlyph := "─"
	if r.Styles.PlainUI {
		ruleGlyph = "-"
	}
	ruleFg, labelFg := ws.Separator, ws.SubtitleFg
	if d.Tone != ToneNeutral {
		ruleFg = r.ToneColor(d.Tone)
		labelFg = ruleFg
	}
	rule := lipgloss.NewStyle().Background(bg).Foreground(ruleFg)
	if d.Label == "" {
		return rule.Render(strings.Repeat(ruleGlyph, d.Width))
	}
	label := lipgloss.NewStyle().Background(bg).Foreground(labelFg).Bold(true).Render(" " + d.Label + " ")
	prefixWidth := lipgloss.Width(label) + 1
	if prefixWidth >= d.Width {
		return clampStyled(rule.Render(ruleGlyph)+label, d.Width, bg)
	}
	return rule.Render(ruleGlyph) + label + rule.Render(strings.Repeat(ruleGlyph, max(0, d.Width-prefixWidth)))
}

// RenderProgressBar renders a gauge of exactly width cells using theme tokens.
// The glyph set follows Styles.Gauge; the marker style draws a track with a
// single dot at the fill position instead of filling a run of cells.
func (r Renderer) RenderProgressBar(bar ProgressBar, bg lipgloss.Color) string {
	width := max(1, bar.Width)
	fraction := clamp01(bar.Fraction)
	filled := int(math.Round(fraction * float64(width)))
	filled = min(width, max(0, filled))
	fullStyle := lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(bar.Tone))
	emptyStyle := lipgloss.NewStyle().Background(bg).Foreground(r.Styles.Workspace.MetricTrack)
	if r.Styles.PlainUI {
		return fullStyle.Render(strings.Repeat("#", filled)) + emptyStyle.Render(strings.Repeat("-", width-filled))
	}
	full, empty, marker, track := gaugeGlyphs(r.Styles.Gauge)
	if marker != "" {
		if fraction <= 0 {
			return emptyStyle.Render(strings.Repeat(track, width))
		}
		pos := min(width-1, max(0, filled-1))
		return emptyStyle.Render(strings.Repeat(track, pos)) +
			fullStyle.Render(marker) +
			emptyStyle.Render(strings.Repeat(track, width-pos-1))
	}
	return fullStyle.Render(strings.Repeat(full, filled)) + emptyStyle.Render(strings.Repeat(empty, width-filled))
}

// GaugeSample returns plain sample glyphs for a style, e.g. for previews in a
// picker. It is unstyled so the caller can colour it to match its own surface.
func (r Renderer) GaugeSample(style GaugeStyle, width int) string {
	width = max(1, width)
	filled := min(width, max(1, width*3/5))
	if r.Styles.PlainUI {
		return strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	}
	full, empty, marker, track := gaugeGlyphs(style)
	if marker != "" {
		pos := min(width-1, max(0, filled-1))
		return strings.Repeat(track, pos) + marker + strings.Repeat(track, width-pos-1)
	}
	return strings.Repeat(full, filled) + strings.Repeat(empty, width-filled)
}

// SparkSample returns a plain sample sparkline for a style, e.g. for previews
// in a picker. It is unstyled so the caller can colour it to match its surface.
func (r Renderer) SparkSample(style SparklineStyle, width int) string {
	width = max(1, width)
	glyphs := sparkGlyphs(style)
	if r.Styles.PlainUI {
		glyphs = []rune(".:-=+*#@")
	}
	var b strings.Builder
	for i := 0; i < width; i++ {
		t := float64(i) / float64(max(1, width-1))
		value := math.Sin(t * math.Pi) // a crest, so the ramp is visible
		index := int(math.Round(value * float64(len(glyphs)-1)))
		b.WriteRune(glyphs[clampIndex(index, len(glyphs))])
	}
	return b.String()
}

// sparkGlyphs returns the low-to-high glyph ramp for a sparkline style.
func sparkGlyphs(style SparklineStyle) []rune {
	switch normalizeSparklineStyle(style) {
	case SparkDots:
		return []rune("·∘○◉●")
	case SparkBraille:
		return []rune("⡀⡄⡆⡇⣇⣧⣷⣿")
	case SparkBullets:
		// A dot, an open ring, then a filled disc. The ramp used to be
		// "∙•●": the bullet operator and the bullet render at the same size
		// in most terminal fonts, so it had two visible steps, not three,
		// and the top of a run was not obviously bigger than the middle.
		return []rune("·○●")
	case SparkTicks:
		return []rune("ˌˈ│┃")
	case SparkShades:
		return []rune("░▒▓█")
	case SparkHeat:
		return []rune(HeatGlyphs)
	case SparkWeighted:
		return []rune(WeightGlyphs)
	case SparkStroke:
		return []rune(StrokeGlyphs)
	default:
		return []rune("▁▂▃▄▅▆▇█")
	}
}

// gaugeGlyphs returns the fill, track, and optional marker glyphs for a style.
func gaugeGlyphs(style GaugeStyle) (full, empty, marker, track string) {
	switch normalizeGaugeStyle(style) {
	case GaugeBlocks:
		return "▰", "▱", "", ""
	case GaugeCircles:
		return "●", "○", "", ""
	case GaugeFisheye:
		return "◉", "○", "", ""
	case GaugeMarker:
		return "", "", "●", "─"
	case GaugeBars:
		return "▮", "▯", "", ""
	default:
		return "█", "░", "", ""
	}
}

// RenderSparkline renders a trend glyph run of exactly width cells. Cells are
// coloured green through yellow, orange, and red by their position between the
// run's minimum and maximum, so the shape and the colour agree.
func (r Renderer) RenderSparkline(spark Sparkline, bg lipgloss.Color) string {
	width := max(1, spark.Width)
	if ramp, bands, ok := r.bandedRamp(); ok {
		return r.renderBandedSparkline(spark, ramp, bands, width, bg)
	}
	glyphs := sparkGlyphs(r.Styles.Sparkline)
	if r.Styles.PlainUI {
		glyphs = []rune(".:-=+*#@")
	}
	values := resample(spark.Values, width)
	low, high := valueRange(values)
	ws := r.Styles.Workspace
	// Glyph and colour must come from the same number. The glyph used to be
	// picked from the absolute value while the colour was scaled to the run,
	// so an idle GPU drew eight identical smallest marks and painted the
	// highest one red: the reddest cell was also the smallest.
	varying := high-low > sparkFlatRange
	var b strings.Builder
	for _, value := range values {
		level := clamp01(value)          // flat run: size by the absolute value
		color := r.ToneColor(spark.Tone) // flat run: keep the caller's tone
		if varying {
			level = (value - low) / (high - low)
			color = ws.MetricGradient(level)
		}
		index := int(math.Round(level * float64(len(glyphs)-1)))
		b.WriteString(lipgloss.NewStyle().Background(bg).Foreground(color).
			Render(string(glyphs[clampIndex(index, len(glyphs))])))
	}
	return b.String()
}

// sparkFlatRange is the spread below which a run counts as flat. Scaling a run
// to its own min and max turns sampling noise into a full-height, full-colour
// swing, so a run that barely moves is drawn at its actual level instead.
const sparkFlatRange = 0.02

// bandedRamp returns the ramp and thresholds for a banded sparkline style, or
// false for the run-relative styles.
func (r Renderer) bandedRamp() ([]rune, []float64, bool) {
	def, banded := bandedDefaults[r.Styles.Sparkline]
	if !banded {
		return nil, nil, false
	}
	ramp := r.Styles.Banded[r.Styles.Sparkline]
	if len(ramp.Glyphs) == 0 || len(ramp.Bands) != len(ramp.Glyphs) {
		// A zero-value Styles (constructed directly rather than through
		// BuildStyles) still has to render something.
		glyphs, bands := normalizeBanded("", nil, r.Styles.PlainUI, def)
		return glyphs, bands, true
	}
	return ramp.Glyphs, ramp.Bands, true
}

// renderBandedSparkline draws each sample at its own absolute level: the band
// a value falls into picks both the glyph and the colour, so a quiet run stays
// light and green instead of being stretched across the whole ramp the way a
// run-relative sparkline would. One glyph per sample, no connecting marks and
// no padding between them, every glyph a single cell. Because the ramp itself
// climbs in weight, the run still reads where colour is unavailable.
func (r Renderer) renderBandedSparkline(spark Sparkline, glyphs []rune, bands []float64, width int, bg lipgloss.Color) string {
	ws := r.Styles.Workspace
	var b strings.Builder
	for _, value := range resample(spark.Values, width) {
		band := heatBand(clamp01(value), bands)
		// Colour steps with the band so weight and severity always agree, and
		// so the configured thresholds govern both.
		color := ws.MetricGradient(float64(band) / float64(len(bands)-1))
		b.WriteString(lipgloss.NewStyle().Background(bg).Foreground(color).
			Render(string(glyphs[band])))
	}
	return b.String()
}

// heatBand returns the index of the band a value falls in.
func heatBand(value float64, bands []float64) int {
	for i, upper := range bands {
		if value <= upper {
			return i
		}
	}
	return len(bands) - 1
}

// valueRange returns the minimum and maximum of a non-empty slice.
func valueRange(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	low, high := values[0], values[0]
	for _, value := range values[1:] {
		low = math.Min(low, value)
		high = math.Max(high, value)
	}
	return low, high
}

// RenderMetricRow renders one aligned metric line. When TotalWidth is set the
// result is exactly that many cells; otherwise it is as wide as its content.
func (r Renderer) RenderMetricRow(m MetricRow, bg lipgloss.Color) string {
	if m.LabelWidth <= 0 {
		m.LabelWidth = lipgloss.Width(m.Label)
	}
	if m.ValueWidth <= 0 {
		m.ValueWidth = lipgloss.Width(m.Value + m.Unit)
	}
	value := m.Value
	if m.Unit != "" {
		value += m.Unit
	}
	if m.Trend != 0 {
		arrow := "▲"
		if m.Trend < 0 {
			arrow = "▼"
		}
		value += " " + arrow
	}
	label := padRight(ansi.Truncate(m.Label, m.LabelWidth, "…"), m.LabelWidth)
	valueCell := padLeft(ansi.Truncate(value, m.ValueWidth, "…"), m.ValueWidth)

	ws := r.Styles.Workspace
	gap := lipgloss.NewStyle().Background(bg).Render("  ")
	line := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyMutedFg).Render(label) +
		gap +
		lipgloss.NewStyle().Background(bg).Foreground(r.ToneColor(m.Tone)).Bold(true).Render(valueCell)

	if plotWidth := m.PlotWidth(); plotWidth > 0 {
		plot := ""
		if m.Bar {
			plot = r.RenderProgressBar(ProgressBar{Fraction: m.Fraction, Width: plotWidth, Tone: m.Tone}, bg)
		} else if len(m.Spark) > 0 {
			plot = r.RenderSparkline(Sparkline{Values: m.Spark, Width: plotWidth, Tone: m.Tone}, bg)
		}
		if plot != "" {
			line += gap + plot
		}
	}
	if m.TotalWidth > 0 {
		return padStyled(line, m.TotalWidth, bg)
	}
	return line
}

// ToneColor resolves a semantic tone to a colour from the active theme. It is
// exported so an application can colour its own chrome to match a widget's
// tone without naming a colour, which is the whole point of the vocabulary.
func (r Renderer) ToneColor(tone Tone) lipgloss.Color {
	ws := r.Styles.Workspace
	switch tone {
	case ToneGood:
		return ws.MetricGood
	case ToneWarning:
		return ws.MetricWarning
	case ToneDanger:
		return ws.MetricBad
	case ToneMuted:
		return ws.HintFg
	default:
		return ws.BodyFg
	}
}

// --- List items -----------------------------------------------------------

// RenderListItem renders a selectable list row, using an accent rail for the
// selection and a subtle fill for the active panel's selection. Right-aligned
// meta and counters are styled separately so they support the row rather than
// competing with it.
func (r Renderer) RenderListItem(item ListItem, width int) string {
	if width <= 0 {
		return ""
	}
	styles := r.Styles
	ws := styles.Workspace
	bg := ws.Bg

	// A focused panel already draws its own accent rail, so a selected row in
	// it is marked by the fill and weight rather than a second rail. An
	// unfocused panel keeps the rail so context survives.
	bar := " "
	barStyle := lipgloss.NewStyle().Background(bg).Foreground(bg)
	if item.Selected && !item.Active {
		bar = markerGlyph(styles.PlainUI)
		if bar == ">" {
			bar = "|"
		}
		barStyle = barStyle.Foreground(ws.SelectionBar)
	}

	rowBg := bg
	textStyle := lipgloss.NewStyle().Background(bg).Foreground(ws.BodyFg)
	metaColor := ws.BodyMutedFg
	counterColor := ws.BodyFg
	switch {
	case item.Selected && item.Active:
		rowBg = ws.SelectionBg
		textStyle = lipgloss.NewStyle().Background(rowBg).Foreground(ws.SelectionFg).Bold(true)
		metaColor = ws.SelectionFg
		counterColor = ws.SelectionFg
	case item.Selected:
		rowBg = ws.SelectionInactiveBg
		textStyle = lipgloss.NewStyle().Background(rowBg).Foreground(ws.SelectionInactiveFg)
		metaColor = ws.SelectionInactiveFg
		counterColor = ws.SelectionInactiveFg
	case item.Hover:
		rowBg = ws.HoverBg
		textStyle = textStyle.Background(rowBg)
	case item.Muted:
		textStyle = textStyle.Foreground(ws.BodyDimmedFg)
	}
	barStyle = barStyle.Background(rowBg)
	metaStyle := lipgloss.NewStyle().Background(rowBg).Foreground(metaColor)
	counterStyle := lipgloss.NewStyle().Background(rowBg).Foreground(counterColor).Bold(true)

	prefix := strings.Repeat(" ", max(0, item.Indent))
	if item.Icon != "" {
		prefix += item.Icon + " "
	}

	meta := ""
	if item.Meta != "" && styles.Density.ShowsSecondary() {
		meta = item.Meta
	}
	counter := item.Counter
	if !styles.Density.ShowsSecondary() {
		counter = ""
	}

	barCell := barStyle.Render(bar)
	contentWidth := width - lipgloss.Width(barCell)

	// Right side first, so the label yields space to it.
	right := ""
	rightWidth := 0
	if counter != "" {
		right = counterStyle.Render(counter)
		rightWidth = lipgloss.Width(right)
	}
	if meta != "" {
		if right != "" {
			right = metaStyle.Render(meta) + lipgloss.NewStyle().Background(rowBg).Render("  ") + right
			rightWidth = lipgloss.Width(meta) + 2 + rightWidth
		} else {
			right = metaStyle.Render(meta)
			rightWidth = lipgloss.Width(right)
		}
	}
	if rightWidth > contentWidth/2 {
		plain := strings.TrimSpace(ansi.Strip(right))
		plain = ansi.Truncate(plain, max(1, contentWidth/2), "…")
		right = metaStyle.Render(plain)
		rightWidth = lipgloss.Width(right)
	}
	gap := 0
	if rightWidth > 0 {
		gap = 2
	}
	textBudget := contentWidth - rightWidth - gap
	if textBudget < 1 {
		textBudget = contentWidth
		right = ""
		rightWidth = 0
		gap = 0
	}
	left := ansi.Truncate(prefix+item.Text, textBudget, "…")
	fill := max(0, contentWidth-lipgloss.Width(left)-rightWidth-gap)

	return barCell +
		textStyle.Render(left) +
		lipgloss.NewStyle().Background(rowBg).Render(strings.Repeat(" ", fill)) +
		lipgloss.NewStyle().Background(rowBg).Render(strings.Repeat(" ", gap)) +
		right
}

// --- Status regions -------------------------------------------------------

// RenderStatusRegions renders the workspace status strip with three regions:
// an identity cluster on the left, an optional transient mode capsule, and
// right-aligned commands. Commands are dropped first, then the mode capsule,
// then the identity is truncated — so the most important information survives
// narrow terminals.
func (r Renderer) RenderStatusRegions(identity, mode, commands string, width int) string {
	if width <= 0 {
		return ""
	}
	s := r.Styles
	bg := s.Theme.StatusBar
	bar := s.StatusBar.Copy().UnsetPadding()
	identity = strings.TrimSpace(identity)
	mode = strings.TrimSpace(mode)
	commands = strings.TrimSpace(commands)

	identitySeg := func(text string) string {
		return bar.Copy().Bold(true).Render(" " + text + " ")
	}
	modeSeg := ""
	if mode != "" {
		modeSeg = lipgloss.NewStyle().Background(s.Theme.BorderFocus).
			Foreground(readableText(s.Theme.Fg, s.Theme.BorderFocus, 4.5)).Bold(true).
			Render(" " + mode + " ")
	}
	sep := bar.Render(" ")

	// Commands are the lowest priority and are dropped first.
	if commands != "" {
		candidate := identitySeg(identity)
		if modeSeg != "" {
			candidate += sep + modeSeg
		}
		commandSeg := s.StatusHint.Render(" " + commands + " ")
		if lipgloss.Width(candidate)+lipgloss.Width(commandSeg) <= width {
			fill := width - lipgloss.Width(candidate) - lipgloss.Width(commandSeg)
			return candidate + bar.Render(strings.Repeat(" ", fill)) + commandSeg
		}
	}

	// Keep the transient mode capsule when there is room for a short identity;
	// otherwise the identity wins on its own.
	if modeSeg != "" && width >= lipgloss.Width(modeSeg)+13 {
		full := identitySeg(identity) + sep + modeSeg
		if lipgloss.Width(full) <= width {
			return clampView(full, width, 1, bg)
		}
		budget := width - lipgloss.Width(modeSeg) - 3
		truncated := ansi.Truncate(identity, max(1, budget), "…")
		return clampView(identitySeg(truncated)+sep+modeSeg, width, 1, bg)
	}
	if identity == "" {
		if modeSeg != "" {
			return clampView(modeSeg, width, 1, bg)
		}
		return bar.Render(strings.Repeat(" ", width))
	}
	return clampView(identitySeg(identity), width, 1, bg)
}

// RenderWorkspaceStatus renders the workspace status strip with a clear
// hierarchy: a bold primary identity, optional muted secondary metadata, a
// transient mode capsule, and right-aligned key hints. It degrades in priority
// order — hints first, then secondary metadata — and never exceeds width.
func (r Renderer) RenderWorkspaceStatus(primary, secondary, mode string, hints []KeyHint, width int) string {
	if width <= 0 {
		return ""
	}
	s := r.Styles
	bg := s.Theme.StatusBar
	bar := s.StatusBar.Copy().UnsetPadding()
	primary = strings.TrimSpace(primary)
	secondary = strings.TrimSpace(secondary)
	mode = strings.TrimSpace(mode)

	primarySeg := ""
	if primary != "" {
		primarySeg = bar.Copy().Bold(true).Render(" " + primary + " ")
	}
	// A transient mode (arrange/resize) owns the bar, so secondary metadata
	// yields to its instructions.
	secondarySeg := ""
	if secondary != "" && mode == "" {
		secondarySeg = bar.Render("  ·  ") + bar.Render(strings.ToLower(secondary)+" ")
	}
	modeSeg := ""
	if mode != "" {
		modeSeg = lipgloss.NewStyle().Background(s.Theme.BorderFocus).
			Foreground(readableText(s.Theme.Fg, s.Theme.BorderFocus, 4.5)).Bold(true).
			Render(" " + mode + " ")
	}

	compose := func(left string, right string) string {
		if lipgloss.Width(left)+lipgloss.Width(right) > width {
			return ""
		}
		gap := max(0, width-lipgloss.Width(left)-lipgloss.Width(right))
		return left + bar.Render(strings.Repeat(" ", gap)) + right
	}

	hintsFor := func(left string) string {
		budget := width - lipgloss.Width(left) - 2
		if budget <= 6 {
			return ""
		}
		return r.RenderStatusKeyHints(hints, budget)
	}

	left := primarySeg + secondarySeg + modeSeg
	if view := compose(left, hintsFor(left)); view != "" {
		return view
	}
	// Drop secondary metadata, then the hints, keeping identity and mode.
	left = primarySeg + modeSeg
	if view := compose(left, hintsFor(left)); view != "" {
		return view
	}
	return clampView(left, width, 1, bg)
}

// --- helpers --------------------------------------------------------------

func panelFrameColor(styles Styles, focused, dimmed bool, accent lipgloss.Color) lipgloss.Color {
	return FocusChrome{Presentation: FocusPresentation{}}.FrameColor(styles, focused, dimmed, accent)
}

func markerGlyph(plain bool) string {
	if plain {
		return ">"
	}
	return "▎"
}

// resample maps values onto width buckets by averaging.
func resample(values []float64, width int) []float64 {
	out := make([]float64, width)
	if len(values) == 0 {
		return out
	}
	if len(values) <= width {
		for i := range out {
			out[i] = values[i*len(values)/width]
		}
		return out
	}
	for i := 0; i < width; i++ {
		start := i * len(values) / width
		end := (i + 1) * len(values) / width
		if end <= start {
			end = start + 1
		}
		sum := 0.0
		for _, v := range values[start:end] {
			sum += v
		}
		out[i] = sum / float64(end-start)
	}
	return out
}

func padLeft(value string, width int) string {
	if pad := width - lipgloss.Width(value); pad > 0 {
		return strings.Repeat(" ", pad) + value
	}
	return value
}

func clampStyled(value string, width int, bg lipgloss.Color) string {
	return padStyled(ansi.Truncate(value, width, ""), width, bg)
}

// MetricPlotWidth returns the room left for a gauge given the row's fixed
// columns.
func MetricPlotWidth(total, labelWidth, valueWidth int) int {
	return max(0, total-labelWidth-valueWidth-4)
}
