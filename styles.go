package tideui

import "github.com/charmbracelet/lipgloss"

// Density controls vertical spacing in rows and overlays.
type Density string

const (
	// Compact removes optional spacer rows and modal padding.
	Compact Density = "compact"
	// Comfortable adds breathing room between rows and in modals.
	Comfortable Density = "comfortable"
)

// StyleOptions controls density and optional theme color replacements.
type StyleOptions struct {
	Density   Density
	Overrides ThemeOverrides
}

// Styles exposes resolved Lipgloss styles for composing application content.
type Styles struct {
	Theme   Theme
	PlainUI bool
	Density Density

	Pane               lipgloss.Style
	PaneHeaderActive   lipgloss.Style
	PaneHeaderInactive lipgloss.Style
	Item               lipgloss.Style
	ItemMuted          lipgloss.Style
	ItemSelected       lipgloss.Style
	Badge              lipgloss.Style
	DetailTitle        lipgloss.Style
	DetailMeta         lipgloss.Style
	DetailBody         lipgloss.Style
	DetailFocusLine    lipgloss.Style
	SearchMatch        lipgloss.Style

	StatusBar       lipgloss.Style
	StatusError     lipgloss.Style
	StatusHint      lipgloss.Style
	StatusBarJoiner lipgloss.Style
	StatusNotice    lipgloss.Style

	Overlay      lipgloss.Style
	OverlayTitle lipgloss.Style
	OverlayBody  lipgloss.Style
	OverlayHint  lipgloss.Style
	InputFocused lipgloss.Style
	InputIdle    lipgloss.Style
	InputLabel   lipgloss.Style
}

func normalizeDensity(d Density) Density {
	if d == Comfortable {
		return Comfortable
	}
	return Compact
}

// ListItemLineStride returns the terminal-line height expected per rendered row.
func (s Styles) ListItemLineStride() int {
	if s.Density == Comfortable {
		return 2
	}
	return 1
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

// BuildStyles resolves a theme and options into reusable Lipgloss styles.
func BuildStyles(base Theme, options StyleOptions) Styles {
	t := options.Overrides.Apply(base)
	density := normalizeDensity(options.Density)
	plain := t.UsesASCII()
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

	selectedBG := adjustLightness(t.Bg, 0.12)
	if !isDark(t.Bg) {
		selectedBG = adjustLightness(t.Bg, -0.12)
	}
	focusBG := focusLineBg(t)
	modalFG := readableText(t.Fg, modalBG, 4.5)
	modalMuted := mutedText(modalFG, modalBG)

	return Styles{
		Theme: t, PlainUI: plain, Density: density,
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
			Border(paneBorder(plain)).BorderForeground(modalAccent).Padding(0, 1),
		InputIdle: lipgloss.NewStyle().Background(modalBG).Foreground(modalFG).
			Border(paneBorder(plain)).BorderForeground(modalBorder).Padding(0, 1),
		InputLabel: lipgloss.NewStyle().Foreground(modalMuted),
	}
}
