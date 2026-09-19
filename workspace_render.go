package tideui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// WorkspaceRenderOptions tunes the optional chrome the workspace draws.
type WorkspaceRenderOptions struct {
	// ShowStatus reserves a bottom status strip.
	ShowStatus bool
	// StatusLeft, StatusSecondary, and StatusRight override the status strip's
	// identity, secondary metadata, and right-hand commands.
	StatusLeft      string
	StatusSecondary string
	StatusRight     string
	// ShowKeyHints renders focused-panel action hints in the panel footer.
	ShowKeyHints bool
	// ShowArrangeCard shows the compact arrange-mode cheat sheet.
	ShowArrangeCard bool
	// StatusHints are appended to the default status-bar key hints for
	// application-level shortcuts (for example a settings panel).
	StatusHints []KeyHint
	// StatusNotice, when set, is shown as the status strip's mode capsule
	// while no transient mode (arrange/resize/zoom) is active. Use it for
	// short-lived application feedback such as "settings applied".
	StatusNotice string
}

// WorkspaceRenderer turns a solved workspace into a themed, bounded string. It
// owns no state beyond the Renderer whose styles and theme it shares.
type WorkspaceRenderer struct {
	Renderer Renderer
	Options  WorkspaceRenderOptions
}

// NewWorkspaceRenderer creates a workspace renderer over an existing Renderer.
func NewWorkspaceRenderer(renderer Renderer) WorkspaceRenderer {
	return WorkspaceRenderer{Renderer: renderer, Options: WorkspaceRenderOptions{
		ShowStatus:   true,
		ShowKeyHints: true,
		// Arrange instructions live in the status strip by default, so no
		// floating card is drawn unless an application opts in.
		ShowArrangeCard: false,
	}}
}

// Render lays out and draws the workspace for a terminal size. It recomputes
// the layout every call, so terminal resizes are reflected immediately and
// never produce an out-of-bounds rectangle.
func (wr WorkspaceRenderer) Render(ws *Workspace, width, height int) string {
	if ws == nil || width <= 0 || height <= 0 {
		return ""
	}
	styles := wr.Renderer.Styles
	statusHeight := 0
	if wr.Options.ShowStatus && height > 2 {
		statusHeight = 1
	}
	contentHeight := max(0, height-statusHeight)

	ws.beginRender()
	solved := ws.Solve(width, contentHeight)
	canvas := blankCanvas(width, contentHeight, styles.Workspace.Bg)

	focus := ws.Focused()
	for _, region := range solved.Regions {
		box := wr.renderRegion(ws, region, focus)
		canvas = placeBoxAt(canvas, box, region.Rect.X, region.Rect.Y, width, contentHeight, styles.Workspace.Bg)
	}

	if peek := ws.Peeked(); peek != "" {
		canvas = wr.renderPeek(ws, canvas, width, contentHeight, peek)
	}
	if wr.Options.ShowArrangeCard && ws.Arranging() {
		canvas = wr.renderArrangeCard(ws, canvas, width, contentHeight)
	}

	view := canvas
	if statusHeight > 0 {
		view = lipgloss.JoinVertical(lipgloss.Left, canvas, wr.renderStrip(ws, width))
	}
	return clampView(view, width, height, styles.Workspace.Bg)
}

func (wr WorkspaceRenderer) renderRegion(ws *Workspace, region SolvedRegion, focus string) string {
	rect := region.Rect
	if rect.Empty() {
		return ""
	}
	asset := region.ActivePanel()
	panel := ws.panels[asset]
	if panel == nil {
		styles := wr.Renderer.Styles
		frame := workspaceFrame{
			width: rect.Width, height: rect.Height,
			styles: styles,
			border: paneFrameBorder(styles.PlainUI, styles.PaneCorners == RoundCorners),
		}
		if region.TabStack {
			frame.tabs, frame.activeTab = ws.tabItems(region)
		}
		return renderWorkspaceFrame(wr.Renderer, frame)
	}
	renderer := wr.panelRenderer(panel)
	styles := renderer.Styles
	focused := asset == focus

	// While arranging, the moving panel keeps the spotlight and the rest of
	// the workspace recedes so the target preview reads clearly.
	dimmed := false
	if ws.Arranging() {
		dimmed = !focused
	} else if !focused && ws.focusPres.DimInactive {
		switch panel.role {
		case RoleTelemetry, RoleOptional:
			dimmed = true
		case RoleInspector:
			dimmed = ws.focusPres.MutedSecondary
		}
	}

	frame := workspaceFrame{
		width:    rect.Width,
		height:   rect.Height,
		focused:  focused,
		dimmed:   dimmed,
		accent:   panel.accent,
		title:    panel.TitleText(),
		subtitle: panel.subtitle,
		capsule:  ws.focusPres.TitleCapsule,
		styles:   styles,
		border:   paneFrameBorder(styles.PlainUI, styles.PaneCorners == RoundCorners),
	}
	if panel.badge != "" {
		badge := NewBadge(panel.badge)
		if panel.badgeTone != ToneNeutral {
			badge = badge.WithTone(panel.badgeTone)
		}
		frame.badge = &badge
	}
	if ws.focusPres.AccentMarker {
		frame.accentMarker = true
	}
	frame.rail = focused && ws.focusPres.FocusRail
	if region.TabStack {
		frame.tabs, frame.activeTab = ws.tabItems(region)
		wr.recordTabHits(ws, region, frame.tabs)
	}
	if ws.Arranging() && focused {
		frame.mode = "moving"
	} else {
		frame.footerHints = wr.footerHints(ws, panel, focused)
	}

	innerWidth := max(0, rect.Width-2)
	innerHeight := max(0, rect.Height-2)
	if ws.collapsed[asset] {
		frame.body = ""
		frame.rail = false
	} else {
		ctxWidth := innerWidth
		if frame.rail && ctxWidth > 0 {
			ctxWidth--
		}
		ctx := PanelContext{
			ID:       asset,
			Title:    panel.TitleText(),
			Width:    ctxWidth,
			Height:   innerHeight,
			Focused:  focused,
			Zoomed:   ws.zoomCandidate() == asset,
			Entered:  ws.PaneHasKeyboard(asset),
			Arrange:  ws.Arranging(),
			Renderer: renderer,
		}
		frame.body = panel.Render(ctx)
	}
	return renderWorkspaceFrame(renderer, frame)
}

// panelRenderer resolves the renderer a panel draws with. A panel theme
// replaces the workspace theme, panel overrides recolor it, a panel gauge
// style replaces the workspace default, and the global density, corner style,
// and shadow settings are always preserved. Panels without panel-scoped
// overrides reuse the workspace renderer directly.
func (wr WorkspaceRenderer) panelRenderer(panel *Panel) Renderer {
	if panel == nil {
		return wr.Renderer
	}
	base := wr.Renderer.Styles
	theme := base.Theme
	themeChanged := false
	if panel.theme != nil {
		theme = *panel.theme
		themeChanged = true
	}
	if panel.overrides != (ThemeOverrides{}) {
		theme = panel.overrides.Apply(theme)
		themeChanged = true
	}
	gauge := base.Gauge
	if panel.gaugeSet {
		gauge = normalizeGaugeStyle(panel.gauge)
	}
	sparkline := base.Sparkline
	if panel.sparkSet {
		sparkline = normalizeSparklineStyle(panel.sparkline)
	}
	if !themeChanged {
		if gauge == base.Gauge && sparkline == base.Sparkline {
			return wr.Renderer
		}
		// Only the metric styles differ; reuse the workspace styles.
		out := wr.Renderer
		out.Styles.Gauge = gauge
		out.Styles.Sparkline = sparkline
		return out
	}
	return Renderer{Styles: BuildStyles(theme, StyleOptions{
		Density:     base.Density,
		PaneCorners: base.PaneCorners,
		Gauge:       gauge,
		Sparkline:   sparkline,
		ClockFont:   base.ClockFont,
		ModalShadow: base.ModalShadow,
	})}
}

// tabItems returns the tabs and active index for a tab-stack region, including
// each panel's badge.
func (ws *Workspace) tabItems(region SolvedRegion) ([]TabItem, int) {
	items := make([]TabItem, 0, len(region.PanelIDs))
	for _, id := range region.PanelIDs {
		if panel := ws.panels[id]; panel != nil {
			items = append(items, TabItem{Title: panel.TitleText(), Badge: panel.badge})
		} else {
			items = append(items, TabItem{Title: titleWords(id)})
		}
	}
	return items, region.ActiveIndex
}

func (wr WorkspaceRenderer) recordTabHits(ws *Workspace, region SolvedRegion, items []TabItem) {
	innerWidth := max(0, region.Rect.Width-2)
	segments, _ := tabSegments(items, innerWidth)
	panelIndex := 0
	for _, id := range region.PanelIDs {
		if panelIndex >= len(segments) {
			break
		}
		if ws.panels[id] != nil {
			segment := segments[panelIndex]
			ws.recordTabHit(Rect{
				X:      region.Rect.X + 1 + segment.start,
				Y:      region.Rect.Y,
				Width:  segment.width,
				Height: 1,
			}, id)
		}
		panelIndex++
	}
}

func (wr WorkspaceRenderer) footerHints(ws *Workspace, panel *Panel, focused bool) []KeyHint {
	if !focused || !wr.Options.ShowKeyHints || !ws.focusPres.KeyHints {
		return nil
	}
	hints := make([]KeyHint, 0, len(panel.actions)+1)
	if panel.hint != "" {
		hints = append(hints, KeyHint{Label: strings.ToLower(panel.hint)})
	}
	for _, action := range panel.actions {
		if action.Key == "" {
			continue
		}
		hints = append(hints, KeyHint{Key: action.Key, Label: action.displayLabel()})
	}
	return hints
}

// --- Frames ---------------------------------------------------------------

type workspaceFrame struct {
	width, height int
	title         string
	subtitle      string
	badge         *Badge
	right         string
	tabs          []TabItem
	activeTab     int
	footerHints   []KeyHint
	mode          string
	focused       bool
	dimmed        bool
	accentMarker  bool
	capsule       bool
	rail          bool
	accent        lipgloss.Color
	body          string
	styles        Styles
	border        lipgloss.Border
}

func renderWorkspaceFrame(renderer Renderer, f workspaceFrame) string {
	if f.width <= 0 || f.height <= 0 {
		return ""
	}
	styles := f.styles
	bg := styles.Workspace.Bg
	borderColor := FocusChrome{Presentation: FocusPresentation{}}.
		FrameColor(styles, f.focused, f.dimmed, f.accent)
	borderStyle := lipgloss.NewStyle().Background(bg).Foreground(borderColor)

	if f.width == 1 {
		lines := make([]string, f.height)
		for i := range lines {
			lines[i] = borderStyle.Render(f.border.Left)
		}
		return strings.Join(lines, "\n")
	}

	header := PanelHeader{
		Title: f.title, Subtitle: f.subtitle, Badge: f.badge, Right: f.right,
		Tabs: f.tabs, ActiveTab: f.activeTab, Focused: f.focused, Dimmed: f.dimmed,
		AccentMarker: f.accentMarker, Capsule: f.capsule,
		Accent: f.accent, Density: styles.Density, Border: f.border,
	}
	top := borderStyle.Render(f.border.TopLeft) + renderer.RenderPanelHeader(header, max(0, f.width-2)) + borderStyle.Render(f.border.TopRight)
	if f.height == 1 {
		return top
	}

	innerWidth := f.width - 2
	innerHeight := f.height - 2
	bodyWidth := innerWidth
	railGlyph := markerGlyph(styles.PlainUI)
	railStyle := lipgloss.NewStyle().Background(bg).Foreground(styles.Workspace.FocusRail)
	if f.rail && innerWidth > 0 {
		bodyWidth = innerWidth - 1
	}

	lines := []string{top}
	if innerHeight > 0 {
		body := clampView(f.body, bodyWidth, innerHeight, bg)
		for _, line := range strings.Split(body, "\n") {
			prefix := borderStyle.Render(f.border.Left)
			if f.rail {
				prefix += railStyle.Render(railGlyph)
			}
			lines = append(lines, prefix+padStyled(line, bodyWidth, bg)+borderStyle.Render(f.border.Right))
		}
	}

	footer := PanelFooter{
		Hints: f.footerHints, Mode: f.mode, Focused: f.focused, Dimmed: f.dimmed,
		Accent: f.accent, Density: styles.Density, Border: f.border,
	}
	bottom := borderStyle.Render(f.border.BottomLeft) + renderer.RenderPanelFooter(footer, innerWidth) + borderStyle.Render(f.border.BottomRight)
	lines = append(lines, bottom)
	return strings.Join(lines, "\n")
}

// --- Overlays -------------------------------------------------------------

// DockSide names are used in the preview label.
func (d DockSide) String() string {
	switch d {
	case DockLeft:
		return "left"
	case DockRight:
		return "right"
	case DockAbove:
		return "up"
	case DockBelow:
		return "down"
	case DockRowAbove:
		return "row up"
	case DockRowBelow:
		return "row down"
	case DockCenter:
		return "stack"
	default:
		return "here"
	}
}

func (wr WorkspaceRenderer) renderArrangeCard(ws *Workspace, canvas string, width, height int) string {
	rows := [][2]string{
		{"h j k l", "move"},
		{"enter", "dock"},
		{"t", "tab stack"},
		{"esc", "cancel"},
	}
	cardWidth := 26
	var body []string
	for _, row := range rows {
		body = append(body, wr.Renderer.RenderKeyHints([]KeyHint{{Key: row[0], Label: row[1]}}, cardWidth-4))
	}
	frame := workspaceFrame{
		width: cardWidth, height: len(rows) + 2,
		title: "arrange", focused: true,
		body:   strings.Join(body, "\n"),
		styles: wr.Renderer.Styles,
		border: paneFrameBorder(wr.Renderer.Styles.PlainUI, wr.Renderer.Styles.PaneCorners == RoundCorners),
	}
	box := renderWorkspaceFrame(wr.Renderer, frame)
	x := max(0, width-cardWidth-2)
	y := max(0, height-frame.height-2)
	return placeBoxAt(canvas, box, x, y, width, height, wr.Renderer.Styles.Workspace.Bg)
}

func (wr WorkspaceRenderer) renderPeek(ws *Workspace, canvas string, width, height int, id string) string {
	panel := ws.panels[id]
	if panel == nil {
		return canvas
	}
	panelWidth := min(max(20, width*3/5), max(1, width-4))
	panelHeight := min(max(5, height*3/5), max(1, height-2))
	if panelWidth <= 2 || panelHeight <= 2 {
		return canvas
	}
	renderer := wr.panelRenderer(panel)
	ctx := PanelContext{
		ID: id, Title: panel.TitleText(),
		Width: max(0, panelWidth-2), Height: max(0, panelHeight-3),
		Focused: true, Peeked: true, Renderer: renderer,
	}
	badge := NewBadge("peek").WithTone(ToneAccent)
	frame := workspaceFrame{
		width: panelWidth, height: panelHeight,
		title: panel.TitleText(), badge: &badge,
		body: panel.Render(ctx), focused: true, rail: true,
		styles: renderer.Styles,
		border: paneFrameBorder(renderer.Styles.PlainUI, renderer.Styles.PaneCorners == RoundCorners),
	}
	frame.accent = panel.accent
	frame.footerHints = []KeyHint{{Key: "esc", Label: "dismiss"}}
	box := renderWorkspaceFrame(renderer, frame)
	x := max(0, (width-panelWidth)/2)
	y := max(0, (height-panelHeight)/2)
	return placeBoxAt(canvas, box, x, y, width, height, wr.Renderer.Styles.Workspace.Bg)
}

func (wr WorkspaceRenderer) renderStrip(ws *Workspace, width int) string {
	primary := wr.Options.StatusLeft
	if primary == "" {
		parts := nonEmpty([]string{
			wr.Renderer.Styles.Theme.Name,
			ws.ActivePreset(),
		})
		if panel, ok := ws.panels[ws.Focused()]; ok {
			parts = append(parts, panel.TitleText())
		}
		primary = strings.Join(parts, "  ·  ")
	}

	mode := ""
	switch {
	case ws.Arranging():
		mode = "ARRANGE"
	case ws.Zoomed() != "":
		mode = "ZOOM"
	}
	if mode == "" {
		mode = ws.ResizeStatus()
	}
	if mode == "" {
		mode = wr.Options.StatusNotice
	}

	// A caller-provided right segment takes over the legacy two-region path.
	if wr.Options.StatusRight != "" {
		return wr.Renderer.RenderStatusRegions(primary, mode, wr.Options.StatusRight, width)
	}
	hints := wr.statusHints(ws)
	if !ws.Arranging() && len(wr.Options.StatusHints) > 0 {
		// Application hints lead, so a shortcut like "settings" stays
		// discoverable when the strip is narrow.
		hints = append(append([]KeyHint{}, wr.Options.StatusHints...), hints...)
	}
	return wr.Renderer.RenderWorkspaceStatus(primary, wr.Options.StatusSecondary, mode, hints, width)
}

// statusHints returns the transient instructions that match the active mode,
// falling back to the global shortcuts.
func (wr WorkspaceRenderer) statusHints(ws *Workspace) []KeyHint {
	switch {
	case ws.Arranging():
		return []KeyHint{
			Hint("h/j/k/l", "move"), Hint("t", "stack"), Hint("esc", "done"),
		}
	default:
		zoomKey := "⇧space"
		if wr.Renderer.Styles.PlainUI {
			zoomKey = "shift+space"
		}
		return []KeyHint{
			Hint("⇧arrows", "resize"), Hint("tab", "focus"), Hint("m", "arrange"),
			Hint(zoomKey, "zoom"), Hint("w", "panels"), Hint("ctrl+p", "commands"),
		}
	}
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

// blankCanvas builds a width x height background-filled surface.
func blankCanvas(width, height int, bg lipgloss.Color) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	line := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width))
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// padRight pads a plain string with spaces to width cells.
func padRight(value string, width int) string {
	if pad := width - lipgloss.Width(value); pad > 0 {
		return value + strings.Repeat(" ", pad)
	}
	return value
}
