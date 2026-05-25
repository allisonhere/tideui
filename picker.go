package tideui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ThemePickerOptions configures a reusable theme selection modal.
type ThemePickerOptions struct {
	// Themes supplies selectable palettes. Empty uses BuiltinThemes.
	Themes []Theme
	// InitialTheme is the initially confirmed theme name.
	InitialTheme string
	// Title labels the modal. Empty uses "THEME".
	Title string
}

// ThemePickerAction reports terminal picker actions after an input update.
type ThemePickerAction int

const (
	// ThemePickerNone indicates navigation or ignored input.
	ThemePickerNone ThemePickerAction = iota
	// ThemePickerConfirm indicates that the previewed theme was confirmed.
	ThemePickerConfirm
	// ThemePickerCancel indicates that the preview was reverted.
	ThemePickerCancel
)

// ThemePicker manages theme navigation, preview, confirmation, and modal output.
//
// Applications remain responsible for persistence and terminal background
// control sequences after inspecting the picker result and active theme.
type ThemePicker struct {
	themes    []Theme
	title     string
	cursor    int
	confirmed int
	opened    bool
}

// NewThemePicker creates a picker with an initially confirmed selection.
func NewThemePicker(options ThemePickerOptions) ThemePicker {
	themes := options.Themes
	if len(themes) == 0 {
		themes = BuiltinThemes
	}
	themes = append([]Theme(nil), themes...)
	title := options.Title
	if title == "" {
		title = "THEME"
	}
	selected := themeIndex(themes, options.InitialTheme)
	return ThemePicker{
		themes:    themes,
		title:     title,
		cursor:    selected,
		confirmed: selected,
	}
}

// Open displays the picker and begins a preview session from confirmedName.
// An unknown name selects the first configured theme.
func (p *ThemePicker) Open(confirmedName string) {
	p.ensureInitialized()
	selected := themeIndex(p.themes, confirmedName)
	p.cursor = selected
	p.confirmed = selected
	p.opened = true
}

// Opened reports whether the picker should currently be displayed.
func (p ThemePicker) Opened() bool { return p.opened }

// PreviewTheme returns the currently highlighted theme.
func (p ThemePicker) PreviewTheme() Theme {
	p.ensureReadable()
	return p.themes[p.cursor]
}

// ConfirmedTheme returns the most recently confirmed theme.
func (p ThemePicker) ConfirmedTheme() Theme {
	p.ensureReadable()
	return p.themes[p.confirmed]
}

// Update applies TideMail-compatible picker navigation and completion keys.
// After navigation, applications can rebuild their renderer from PreviewTheme.
func (p *ThemePicker) Update(msg tea.KeyMsg) ThemePickerAction {
	p.ensureInitialized()
	if !p.opened {
		return ThemePickerNone
	}

	switch msg.String() {
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case "j", "down":
		if p.cursor < len(p.themes)-1 {
			p.cursor++
		}
	case "enter":
		p.confirmed = p.cursor
		p.opened = false
		return ThemePickerConfirm
	case "esc":
		p.cursor = p.confirmed
		p.opened = false
		return ThemePickerCancel
	}
	return ThemePickerNone
}

// Modal renders the picker as an Overlay for assignment to Layout.Modal.
// Height limits the number of rendered themes while keeping the cursor visible.
func (p ThemePicker) Modal(renderer Renderer, width, height int) Overlay {
	p.ensureReadable()
	if width <= 0 {
		width = 40
	}

	titleHeight := lipgloss.Height(renderer.Styles.OverlayTitle.Render(p.title))
	footer := "enter confirm  esc revert"
	footerHeight := lipgloss.Height(renderer.Styles.OverlayHint.Render(footer))
	rowsAvailable := height - renderer.Styles.Overlay.GetVerticalFrameSize() - titleHeight - footerHeight
	rowsAvailable = max(1, rowsAvailable)

	first, last := visibleRange(len(p.themes), p.cursor, rowsAvailable)
	innerWidth := max(1, width-renderer.Styles.Overlay.GetHorizontalFrameSize())
	rows := make([]string, 0, last-first)
	for index := first; index < last; index++ {
		prefix := "  "
		style := renderer.Styles.OverlayBody
		if index == p.cursor {
			prefix = "▶ "
			if renderer.Styles.PlainUI {
				prefix = "> "
			}
			accent := renderer.Styles.Theme.BorderFocus
			if accent == "" {
				accent = renderer.Styles.Theme.OverlayBorder
			}
			style = style.Copy().Background(accent).
				Foreground(readableText(renderer.Styles.Theme.Fg, accent, 4.5)).Bold(true)
		}
		row := alignRow(prefix, p.themes[index].Name, "", innerWidth)
		rows = append(rows, style.Width(innerWidth).Render(row))
	}

	return Overlay{
		Visible: p.opened,
		Title:   p.title,
		Content: strings.Join(rows, "\n"),
		Footer:  footer,
		Width:   width,
	}
}

func (p *ThemePicker) ensureInitialized() {
	if len(p.themes) != 0 {
		return
	}
	*p = NewThemePicker(ThemePickerOptions{})
}

func (p *ThemePicker) ensureReadable() {
	p.ensureInitialized()
	p.cursor = max(0, min(p.cursor, len(p.themes)-1))
	p.confirmed = max(0, min(p.confirmed, len(p.themes)-1))
}

func themeIndex(themes []Theme, name string) int {
	for index, theme := range themes {
		if theme.Name == name {
			return index
		}
	}
	return 0
}

func visibleRange(total, cursor, limit int) (int, int) {
	if limit >= total {
		return 0, total
	}
	start := cursor - limit/2
	start = max(0, min(start, total-limit))
	return start, start + limit
}
