package tideui

import "github.com/charmbracelet/lipgloss"

const (
	// ThemeNameVT52 identifies the ASCII-style amber terminal theme.
	ThemeNameVT52 = "vt52"
	// ThemeNameVT100 identifies the green phosphor terminal theme.
	ThemeNameVT100 = "vt100"
)

// Theme contains the semantic colors used by the shell and its components.
type Theme struct {
	Name          string         // identifier used by ThemeByName and ThemePicker
	Bg            lipgloss.Color // pane background
	Fg            lipgloss.Color // primary text
	Border        lipgloss.Color // unfocused pane border and inactive tab header background
	BorderFocus   lipgloss.Color // focused pane border, active tab header background, and accent highlight
	Selected      lipgloss.Color // selected list item background (falls back to BorderFocus)
	Unread        lipgloss.Color // badge / unread-count foreground
	Dimmed        lipgloss.Color // muted text; the renderer raises contrast if the raw value fails WCAG 3:1
	StatusBar     lipgloss.Color // status bar background
	StatusFg      lipgloss.Color // status bar text
	Error         lipgloss.Color // error text in the status bar
	Overlay       lipgloss.Color // modal background; defaults to a lightness-adjusted Bg when empty
	OverlayBorder lipgloss.Color // modal border; falls back to Border when empty

	// Family and Variant group related palettes in pickers, e.g. "Catppuccin"
	// and "Mocha". Both may be empty for a theme that stands alone.
	Family  string
	Variant string

	// Optional semantic roles. An empty role falls back to a color the theme
	// already has (see the *Color accessors), so a Theme literal that sets only
	// the fields above keeps working.
	Success lipgloss.Color // confirmations and "ok" states
	Warning lipgloss.Color // cautions
	Link    lipgloss.Color // hyperlinks
	Quote   lipgloss.Color // quoted text
	Match   lipgloss.Color // search matches
	Tag     lipgloss.Color // tags and pills
	Cursor  lipgloss.Color // text cursor and focus-line marker
}

// ThemeOverrides replaces presentation colors without requiring a new theme.
type ThemeOverrides struct {
	Background lipgloss.Color
	Foreground lipgloss.Color
	Accent     lipgloss.Color
}

// Apply returns base with each non-empty override applied.
func (o ThemeOverrides) Apply(base Theme) Theme {
	out := base
	if o.Background != "" {
		out.Bg = o.Background
	}
	if o.Foreground != "" {
		out.Fg = o.Foreground
		out.StatusFg = o.Foreground
	}
	if o.Accent != "" {
		out.BorderFocus = o.Accent
		out.Selected = o.Accent
		out.OverlayBorder = o.Accent
	}
	return out
}

// Label is the human-readable name pickers show: the family and variant, or the
// Name when the theme has no family.
func (t Theme) Label() string {
	switch {
	case t.Family == "":
		return t.Name
	case t.Variant == "" || t.Variant == t.Family:
		return t.Family
	}
	return t.Family + " " + t.Variant
}

// IsLight reports whether the theme has a light background.
func (t Theme) IsLight() bool { return !isDark(t.Bg) }

func firstColor(c, fallback lipgloss.Color) lipgloss.Color {
	if c != "" {
		return c
	}
	return fallback
}

// textRole resolves an optional role color to text that is readable on the
// theme background: the palette's own color, lightness-adjusted by Legible when
// it falls under minimum contrast.
func (t Theme) textRole(c, fallback lipgloss.Color, minimum float64) lipgloss.Color {
	return Legible(firstColor(c, fallback), t.Bg, minimum)
}

// SuccessColor is Success (Unread when unset), readable on Bg at 4.5:1.
func (t Theme) SuccessColor() lipgloss.Color { return t.textRole(t.Success, t.Unread, 4.5) }

// WarningColor is Warning (Error when unset), readable on Bg at 4.5:1.
func (t Theme) WarningColor() lipgloss.Color { return t.textRole(t.Warning, t.Error, 4.5) }

// ErrorColor is Error readable on Bg at 4.5:1.
func (t Theme) ErrorColor() lipgloss.Color { return t.textRole(t.Error, t.Fg, 4.5) }

// LinkColor is Link (BorderFocus when unset), readable on Bg at 4.5:1.
func (t Theme) LinkColor() lipgloss.Color { return t.textRole(t.Link, t.BorderFocus, 4.5) }

// QuoteColor is Quote (Dimmed when unset), readable on Bg at 4.5:1.
func (t Theme) QuoteColor() lipgloss.Color { return t.textRole(t.Quote, t.Dimmed, 4.5) }

// TagColor is Tag (BorderFocus when unset), readable on Bg at 4.5:1.
func (t Theme) TagColor() lipgloss.Color { return t.textRole(t.Tag, t.BorderFocus, 4.5) }

// UnreadColor is Unread readable on Bg at 4.5:1.
func (t Theme) UnreadColor() lipgloss.Color { return t.textRole(t.Unread, t.Fg, 4.5) }

// MatchColor is Match (BorderFocus when unset), visible on Bg at 3:1.
func (t Theme) MatchColor() lipgloss.Color { return t.textRole(t.Match, t.BorderFocus, 3) }

// CursorColor is Cursor (Fg when unset), visible on Bg at 3:1.
func (t Theme) CursorColor() lipgloss.Color { return t.textRole(t.Cursor, t.Fg, 3) }

// UsesASCII reports whether this theme uses plain terminal presentation.
func (t Theme) UsesASCII() bool { return t.Name == ThemeNameVT52 }

// ThemeByName returns a built-in theme by name, falling back to Catppuccin Mocha.
func ThemeByName(name string) (Theme, bool) {
	for _, theme := range BuiltinThemes {
		if theme.Name == name {
			return theme, true
		}
	}
	return BuiltinThemes[0], false
}
