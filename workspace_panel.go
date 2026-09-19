package tideui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PanelRole is a panel's semantic purpose. Roles drive the default responsive
// behaviour and the order in which panels survive shrinking, so an application
// rarely has to spell out a breakpoint by hand.
type PanelRole int

const (
	// RolePrimary is the content the workspace exists to show. It survives the
	// longest when space runs out.
	RolePrimary PanelRole = iota
	// RoleNavigation moves between things (lists, files, repositories).
	RoleNavigation
	// RoleSecondary supports the primary content (details, previews, forms).
	RoleSecondary
	// RoleInspector shows properties of the current selection.
	RoleInspector
	// RoleTelemetry is ambient information such as logs or metrics.
	RoleTelemetry
	// RoleOptional is nice to have and disappears first.
	RoleOptional
)

func rolePriority(role PanelRole) int {
	switch role {
	case RolePrimary:
		return 100
	case RoleNavigation:
		return 80
	case RoleSecondary:
		return 70
	case RoleInspector:
		return 60
	case RoleTelemetry:
		return 40
	case RoleOptional:
		return 20
	default:
		return 50
	}
}

// String returns a stable name for debugging and command categories.
func (r PanelRole) String() string {
	switch r {
	case RolePrimary:
		return "primary"
	case RoleNavigation:
		return "navigation"
	case RoleSecondary:
		return "secondary"
	case RoleInspector:
		return "inspector"
	case RoleTelemetry:
		return "telemetry"
	case RoleOptional:
		return "optional"
	default:
		return "panel"
	}
}

// PanelContext is supplied to a panel's view so it can adapt its content to
// the space and interaction state the workspace allocated it. Renderer is
// resolved for this panel, so views that render through it inherit any
// panel-scoped theme automatically.
type PanelContext struct {
	ID      string
	Title   string
	Width   int
	Height  int
	Focused bool
	Zoomed  bool
	// Entered says this panel has the keyboard: space entered it, or it owns
	// the screen. A panel that draws its own cursor draws it when Entered is
	// set - Focused alone means the workspace is pointed here, which is a
	// different question from who reads the keys.
	Entered  bool
	Peeked   bool
	Arrange  bool
	Renderer Renderer
}

// PanelView renders a panel's body for the given context.
type PanelView func(PanelContext) string

// Text returns a fixed-content view, for panels whose body does not change.
func Text(content string) PanelView {
	return func(PanelContext) string { return content }
}

// ActionHandler runs a panel or workspace action. Running an action never
// mutates layout directly unless it calls back into the workspace; this keeps
// application data changes out of layout undo history.
type ActionHandler func(*Workspace)

// PanelAction is one contextual action advertised by a panel.
type PanelAction struct {
	// ID is the stable action identifier ("stage", "open").
	ID string
	// Label is human-readable, used by key hints and the command palette.
	Label string
	// Key is the hint key shown in the panel footer ("s", "ctrl+r").
	Key string
	// Category groups the action in the command palette.
	Category string
	// Handler runs the action.
	Handler ActionHandler
}

// Action builds a panel action. The id doubles as the label when none is set.
func Action(id, key string, handler ActionHandler) PanelAction {
	return PanelAction{ID: id, Key: key, Handler: handler}
}

// Labeled sets a human-readable label on an action.
func (a PanelAction) Labeled(label string) PanelAction {
	a.Label = label
	return a
}

// In sets an action's command-palette category.
func (a PanelAction) In(category string) PanelAction {
	a.Category = category
	return a
}

func (a PanelAction) displayLabel() string {
	if a.Label != "" {
		return a.Label
	}
	return titleWords(a.ID)
}

// FallbackMode is the generalized responsive behaviour a panel adopts when
// the workspace falls below a width threshold.
type FallbackMode int

const (
	// FallbackHide removes the panel entirely.
	FallbackHide FallbackMode = iota
	// FallbackCollapse keeps a compact header-only strip.
	FallbackCollapse
	// FallbackStack merges the panel into another panel's tab stack.
	FallbackStack
	// FallbackMoveBelow relocates the panel beneath a target panel.
	FallbackMoveBelow
	// FallbackMoveRight relocates the panel to the right of a target panel.
	FallbackMoveRight
)

// Fallback is one responsive rule: "below this width, do this". Rules are
// evaluated from the narrowest threshold outward, so a panel can express a
// graceful degradation ladder rather than a single special case.
type Fallback struct {
	Below  int
	Mode   FallbackMode
	With   string // FallbackStack: the panel to merge into
	Target string // FallbackMoveBelow / FallbackMoveRight: the anchor
}

// Panel describes one registered workspace region. Panels are created through
// Workspace.Panel and configured with the fluent methods below.
type Panel struct {
	id          string
	title       string
	description string
	role        PanelRole
	view        PanelView

	minWidth    int
	minHeight   int
	prefWidth   int
	prefHeight  int
	grow        float64
	shrink      float64
	priority    int
	prioritySet bool

	focusable bool
	hideable  bool
	zoomable  bool

	subtitle  string
	badge     string
	badgeTone Tone
	hint      string
	accent    lipgloss.Color

	// Panel-scoped theming. When theme is set it replaces the workspace
	// theme for this panel; overrides then recolor it partially. Density,
	// corner style, gutters, and global chrome stay workspace-wide.
	theme     *Theme
	overrides ThemeOverrides

	// Panel-scoped gauge and sparkline styles. When set, the panel's progress
	// bars or sparklines use them instead of the workspace default.
	gauge     GaugeStyle
	gaugeSet  bool
	sparkline SparklineStyle
	sparkSet  bool

	actions   []PanelAction
	fallbacks []Fallback
	hidden    bool
	// startHidden remembers that the panel was declared hidden, so a preset
	// that does not mention it cannot reveal a panel meant to stay off until
	// it is enabled. hidden is the current state; this is the default.
	startHidden bool
}

func newPanel(id string, view PanelView) *Panel {
	return &Panel{
		id:        id,
		role:      RoleSecondary,
		view:      view,
		minWidth:  10,
		minHeight: 3,
		grow:      1,
		shrink:    1,
		focusable: true,
		hideable:  true,
		zoomable:  true,
	}
}

// --- Read-only accessors -------------------------------------------------

// ID returns the panel's stable identifier.
func (p *Panel) ID() string { return p.id }

// TitleText returns the panel's display title.
func (p *Panel) TitleText() string {
	if p.title != "" {
		return p.title
	}
	return titleWords(p.id)
}

// Desc returns the panel's optional description.
func (p *Panel) Desc() string { return p.description }

// SemanticRole returns the panel's role.
func (p *Panel) SemanticRole() PanelRole { return p.role }

// BadgeText returns the panel's optional status badge.
func (p *Panel) BadgeText() string { return p.badge }

// HintText returns the panel's optional passive hint text.
func (p *Panel) HintText() string { return p.hint }

// MinWidthValue returns the resolved minimum width.
func (p *Panel) MinWidthValue() int { return p.minWidth }

// MinHeightValue returns the resolved minimum height.
func (p *Panel) MinHeightValue() int { return p.minHeight }

// GrowValue returns the grow weight.
func (p *Panel) GrowValue() float64 { return p.grow }

// PriorityValue returns the resolved priority, using the role default when no
// explicit value was set.
func (p *Panel) PriorityValue() int {
	if p.prioritySet {
		return p.priority
	}
	return rolePriority(p.role)
}

// Fallbacks returns the panel's responsive fallback rules.
func (p *Panel) Fallbacks() []Fallback { return p.fallbacks }

// ActionList returns the panel's contextual actions.
func (p *Panel) ActionList() []PanelAction { return p.actions }

// StartsHidden reports whether the panel was declared hidden, which is its
// default until a preset or an explicit Show changes it.
func (p *Panel) StartsHidden() bool { return p.startHidden }

// CanFocus reports whether focus traversal may select this panel.
func (p *Panel) CanFocus() bool { return p.focusable }

// CanHide reports whether the panel may be hidden.
func (p *Panel) CanHide() bool { return p.hideable }

// CanZoom reports whether the panel may fill the workspace.
func (p *Panel) CanZoom() bool { return p.zoomable }

// PanelTheme returns the panel's theme, if it has one. The second result is
// false when the panel follows the workspace theme.
func (p *Panel) PanelTheme() (Theme, bool) {
	if p.theme == nil {
		return Theme{}, false
	}
	return *p.theme, true
}

// PanelOverrides returns the panel's own color overrides.
func (p *Panel) PanelOverrides() ThemeOverrides { return p.overrides }

// HasPanelTheme reports whether the panel has any panel-scoped theming.
func (p *Panel) HasPanelTheme() bool { return p.theme != nil || p.overrides != (ThemeOverrides{}) }

// PanelGauge returns the panel's gauge override, if it has one. The second
// result is false when the panel follows the workspace gauge style.
func (p *Panel) PanelGauge() (GaugeStyle, bool) {
	if !p.gaugeSet {
		return "", false
	}
	return p.gauge, true
}

// HasPanelGauge reports whether the panel overrides the workspace gauge style.
func (p *Panel) HasPanelGauge() bool { return p.gaugeSet }

// PanelSparkline returns the panel's sparkline override, if it has one. The
// second result is false when the panel follows the workspace sparkline style.
func (p *Panel) PanelSparkline() (SparklineStyle, bool) {
	if !p.sparkSet {
		return "", false
	}
	return p.sparkline, true
}

// HasPanelSparkline reports whether the panel overrides the workspace style.
func (p *Panel) HasPanelSparkline() bool { return p.sparkSet }

// Render asks the panel to render itself for ctx.
func (p *Panel) Render(ctx PanelContext) string {
	if p.view == nil {
		return ""
	}
	return p.view(ctx)
}

// --- Fluent configuration ------------------------------------------------

// Title sets the display title.
func (p *Panel) Title(title string) *Panel { p.title = title; return p }

// Description sets a longer description used by pickers and the palette.
func (p *Panel) Description(description string) *Panel { p.description = description; return p }

// Role sets the panel's semantic role and, unless overridden, its priority.
func (p *Panel) Role(role PanelRole) *Panel { p.role = role; return p }

// MinWidth sets the minimum drawable width.
func (p *Panel) MinWidth(width int) *Panel { p.minWidth = max(1, width); return p }

// MinHeight sets the minimum drawable height.
func (p *Panel) MinHeight(height int) *Panel { p.minHeight = max(1, height); return p }

// PreferredWidth sets the ideal width used when space is plentiful.
func (p *Panel) PreferredWidth(width int) *Panel { p.prefWidth = max(0, width); return p }

// PreferredHeight sets the ideal height used when space is plentiful.
func (p *Panel) PreferredHeight(height int) *Panel { p.prefHeight = max(0, height); return p }

// Grow sets the share of spare space the panel claims.
func (p *Panel) Grow(weight float64) *Panel {
	if weight > 0 {
		p.grow = weight
	}
	return p
}

// Shrink sets how readily the panel gives up space; lower values hold their
// size longer.
func (p *Panel) Shrink(weight float64) *Panel {
	if weight > 0 {
		p.shrink = weight
	}
	return p
}

// Priority sets an explicit survival priority, overriding the role default.
func (p *Panel) Priority(priority int) *Panel {
	p.priority = priority
	p.prioritySet = true
	return p
}

// Focusable controls whether focus traversal may land on the panel.
func (p *Panel) Focusable(focusable bool) *Panel { p.focusable = focusable; return p }

// Hideable controls whether the panel may be hidden or collapsed.
func (p *Panel) Hideable(hideable bool) *Panel { p.hideable = hideable; return p }

// Zoomable controls whether the panel may fill the workspace.
func (p *Panel) Zoomable(zoomable bool) *Panel { p.zoomable = zoomable; return p }

// Subtitle sets a secondary label rendered after the title (hidden at Dense
// density).
func (p *Panel) Subtitle(subtitle string) *Panel { p.subtitle = subtitle; return p }

// Badge sets a short status badge rendered in the panel frame.
func (p *Panel) Badge(badge string) *Panel { p.badge = badge; return p }

// BadgeTone tints the panel badge with a semantic tone.
func (p *Panel) BadgeTone(tone Tone) *Panel { p.badgeTone = tone; return p }

// Hint sets passive hint text shown at the end of the title bar.
func (p *Panel) Hint(hint string) *Panel { p.hint = hint; return p }

// Accent sets a panel-specific focus accent, overriding the theme's.
func (p *Panel) Accent(accent lipgloss.Color) *Panel { p.accent = accent; return p }

// Theme gives the panel its own full theme. Density, corners, gutters, and the
// rest of the workspace chrome remain workspace-wide. Pass the theme by value;
// it is copied.
func (p *Panel) Theme(theme Theme) *Panel {
	copied := theme
	p.theme = &copied
	return p
}

// Overrides recolor the panel without selecting a full theme. Overrides apply
// on top of the panel theme when one is set, otherwise on top of the workspace
// theme.
func (p *Panel) Overrides(overrides ThemeOverrides) *Panel {
	p.overrides = overrides
	return p
}

// ClearTheme returns the panel to the workspace theme.
func (p *Panel) ClearTheme() *Panel {
	p.theme = nil
	p.overrides = ThemeOverrides{}
	return p
}

// Gauge overrides the workspace gauge style for this panel's progress bars.
func (p *Panel) Gauge(style GaugeStyle) *Panel {
	p.gauge = normalizeGaugeStyle(style)
	p.gaugeSet = true
	return p
}

// ClearGauge makes the panel follow the workspace gauge style again.
func (p *Panel) ClearGauge() *Panel {
	p.gauge = ""
	p.gaugeSet = false
	return p
}

// Sparkline overrides the workspace sparkline glyph ramp for this panel.
func (p *Panel) Sparkline(style SparklineStyle) *Panel {
	p.sparkline = normalizeSparklineStyle(style)
	p.sparkSet = true
	return p
}

// ClearSparkline makes the panel follow the workspace sparkline style again.
func (p *Panel) ClearSparkline() *Panel {
	p.sparkline = ""
	p.sparkSet = false
	return p
}

// Content sets the panel's view.
func (p *Panel) Content(view PanelView) *Panel { p.view = view; return p }

// Body sets a fixed string body.
func (p *Panel) Body(content string) *Panel { p.view = Text(content); return p }

// Visible marks the panel visible at construction time. Panels are visible by
// default; this exists for fluent symmetry with Hide.
func (p *Panel) Visible() *Panel { p.hidden, p.startHidden = false, false; return p }

// Hide starts the panel hidden.
func (p *Panel) Hide() *Panel { p.hidden, p.startHidden = true, true; return p }

// Actions registers contextual actions. They automatically flow into the
// command palette, so they only need to be declared once.
func (p *Panel) Actions(actions ...PanelAction) *Panel {
	p.actions = append(p.actions, actions...)
	return p
}

// Responsive appends generalized responsive fallback rules.
func (p *Panel) Responsive(fallbacks ...Fallback) *Panel {
	p.fallbacks = append(p.fallbacks, fallbacks...)
	return p
}

// HideBelow hides the panel when the workspace is narrower than width.
func (p *Panel) HideBelow(width int) *Panel {
	return p.Responsive(Fallback{Below: width, Mode: FallbackHide})
}

// CollapseBelow reduces the panel to a compact header below width.
func (p *Panel) CollapseBelow(width int) *Panel {
	return p.Responsive(Fallback{Below: width, Mode: FallbackCollapse})
}

// StackBelow merges the panel into with's tab stack below width.
func (p *Panel) StackBelow(width int, with string) *Panel {
	return p.Responsive(Fallback{Below: width, Mode: FallbackStack, With: with})
}

// MoveBelow relocates the panel beneath target below width.
func (p *Panel) MoveBelow(width int, target string) *Panel {
	return p.Responsive(Fallback{Below: width, Mode: FallbackMoveBelow, Target: target})
}

// MoveRightOf relocates the panel to the right of target below width.
func (p *Panel) MoveRightOf(width int, target string) *Panel {
	return p.Responsive(Fallback{Below: width, Mode: FallbackMoveRight, Target: target})
}

// titleWords converts an identifier such as "recent_files" to "Recent Files".
func titleWords(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	words := strings.Fields(value)
	for i, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(strings.ToLower(word))
		upper := []rune(strings.ToUpper(string(runes[0])))
		runes[0] = upper[0]
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
