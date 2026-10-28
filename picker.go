package tideui

import (
	"strings"
	"unicode/utf8"

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
	// HideSwatch reports themes that should be listed without a palette
	// swatch, for a placeholder whose colors are only known at runtime.
	HideSwatch func(Theme) bool
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
//
// Typing "/" starts a filter: each word must match a theme's family, variant,
// name, or "light"/"dark". While a filter is applied, navigation steps through
// the matching themes only.
type ThemePicker struct {
	themes     []Theme
	title      string
	cursor     int
	confirmed  int
	opened     bool
	filter     string
	filtering  bool
	hideSwatch func(Theme) bool
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
		themes:     themes,
		title:      title,
		cursor:     selected,
		confirmed:  selected,
		hideSwatch: options.HideSwatch,
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
	p.filter, p.filtering = "", false
}

// SetTitle changes the modal title. Empty restores the default "THEME". Use it
// to label what the picker is editing, e.g. a specific panel's theme.
func (p *ThemePicker) SetTitle(title string) {
	p.ensureInitialized()
	if title == "" {
		title = "THEME"
	}
	p.title = title
}

// Filter returns the search text, and Filtering reports whether the picker is
// currently taking keystrokes for it.
func (p ThemePicker) Filter() string { return p.filter }

// Filtering reports whether keystrokes are being typed into the search.
func (p ThemePicker) Filtering() bool { return p.filtering }

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

// Update applies the picker's standard navigation and completion keys.
// After navigation, applications can rebuild their renderer from PreviewTheme.
func (p *ThemePicker) Update(msg tea.KeyMsg) ThemePickerAction {
	p.ensureInitialized()
	if !p.opened {
		return ThemePickerNone
	}
	if p.filtering {
		p.updateFilter(msg)
		return ThemePickerNone
	}

	switch msg.String() {
	case "/":
		p.filtering = true
	case "k", "up":
		p.step(-1)
	case "j", "down":
		p.step(1)
	case "enter":
		// A filter can hide the highlighted theme; confirming it would pick
		// something the person cannot see.
		if entryPos(p.entries(), p.cursor) < 0 {
			return ThemePickerNone
		}
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

// updateFilter handles a keystroke while the search line has focus.
func (p *ThemePicker) updateFilter(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyEsc:
		if p.filter != "" {
			p.filter = ""
			p.keepCursorVisible()
		} else {
			p.filtering = false
		}
	case tea.KeyEnter:
		p.filtering = false
	case tea.KeyBackspace:
		if _, size := utf8.DecodeLastRuneInString(p.filter); size > 0 {
			p.filter = p.filter[:len(p.filter)-size]
			p.keepCursorVisible()
		}
	case tea.KeyUp:
		p.step(-1)
	case tea.KeyDown:
		p.step(1)
	case tea.KeyRunes, tea.KeySpace:
		if !msg.Alt {
			p.filter += string(msg.Runes)
			p.keepCursorVisible()
		}
	}
}

// entries is the grouped, filtered list the picker shows.
func (p ThemePicker) entries() []pickerEntry { return pickerEntries(p.themes, p.filter) }

// step moves the cursor delta themes through the ones the filter leaves
// visible, stopping at the ends.
func (p *ThemePicker) step(delta int) {
	var visible []int
	for _, entry := range p.entries() {
		if entry.header == "" {
			visible = append(visible, entry.idx)
		}
	}
	if len(visible) == 0 {
		return
	}
	pos := -1
	for i, idx := range visible {
		if idx == p.cursor {
			pos = i
		}
	}
	next := 0
	if pos != -1 {
		next = max(0, min(len(visible)-1, pos+delta))
	}
	p.cursor = visible[next]
}

// keepCursorVisible moves the cursor onto the first visible theme when a filter
// change hides the highlighted one.
func (p *ThemePicker) keepCursorVisible() {
	entries := p.entries()
	if entryPos(entries, p.cursor) >= 0 {
		return
	}
	for _, entry := range entries {
		if entry.header == "" {
			p.cursor = entry.idx
			return
		}
	}
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

	searching := p.filtering || p.filter != ""
	if searching {
		rowsAvailable = max(1, rowsAvailable-1)
	}
	visible := p.visibleThemes()
	first, last := VisibleRange(len(visible), max(0, indexOf(visible, p.cursor)), rowsAvailable)
	innerWidth := max(1, width-renderer.Styles.Overlay.GetHorizontalFrameSize())
	rows := make([]string, 0, last-first+1)
	if searching {
		line := "/ " + p.filter
		if p.filtering {
			line += "▏"
		}
		rows = append(rows, renderer.Styles.OverlayBody.Width(innerWidth).Render(alignRow("", line, "", innerWidth)))
	}
	for _, index := range visible[first:last] {
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
		row := alignRow(prefix, p.themes[index].Label(), "", innerWidth)
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

// SoftModal renders the picker as a soft-panel Overlay for assignment to
// Layout.Modal. Prefix labels the app family member in the top border; empty
// uses "tide".
//
// Themes are grouped under their family, with a search line above, a palette
// swatch beside each variant and a sample of the highlighted theme below.
// Height is the whole modal; the list scrolls to keep the cursor in view, and
// the sample is left out when there is no room for it.
func (p ThemePicker) SoftModal(renderer Renderer, width, height int, prefix string) Overlay {
	p.ensureReadable()
	if width <= 0 {
		width = 40
	}
	if prefix == "" {
		prefix = "tide"
	}
	innerWidth := max(1, width-4)
	chrome := newSoftChrome(renderer.Styles)
	plain := renderer.Styles.PlainUI
	line := func(s string) string { return padStyled(s, innerWidth, chrome.baseBg) }

	// Search line.
	searchText, searchMuted := "to search", true
	switch {
	case p.filtering:
		searchText, searchMuted = p.filter+"▏", false
	case p.filter != "":
		searchText = p.filter
	}
	search := renderer.RenderSoftRow(SoftRow{Prefix: "/ ", Text: searchText, Muted: searchMuted}, innerWidth)

	// The panel borders and the body's blank rows take four lines. Below the
	// search line and the hints sits the list, then, when it fits, a blank line
	// and the three-line sample.
	room := max(1, height-4)
	sampleLines := 0
	if room-2-4 >= 5 {
		sampleLines = 4
	}
	// The list takes what is left, but never more than the unfiltered list
	// needs: a short list does not pad out the modal, and a long one keeps its
	// height while a filter narrows it.
	listHeight := max(1, min(room-2-sampleLines, len(pickerEntries(p.themes, ""))))

	entries := p.entries()
	rows := make([]string, 0, listHeight)
	if len(entries) == 0 {
		rows = append(rows, renderer.RenderSoftRow(SoftRow{Text: "no matching themes", Muted: true}, innerWidth))
	}
	first, last := VisibleRange(len(entries), max(0, entryPos(entries, p.cursor)), listHeight)
	for pos := first; pos < last && len(entries) > 0; pos++ {
		entry := entries[pos]
		if entry.header != "" {
			rows = append(rows, renderer.RenderSoftRow(SoftRow{Prefix: " ", Text: entry.header, Muted: true}, innerWidth))
			continue
		}
		theme := p.themes[entry.idx]
		label, indent := theme.Variant, "  "
		if label == "" || entry.solo || theme.Family == "" {
			label, indent = theme.Label(), " "
		}
		if entry.idx == p.confirmed {
			label += " ●"
			if plain {
				label = strings.TrimSuffix(label, " ●") + " *"
			}
		}
		suffix := ""
		if p.hideSwatch == nil || !p.hideSwatch(theme) {
			suffix = ThemeSwatch(theme, plain)
		}
		rows = append(rows, renderer.RenderSoftRow(SoftRow{
			Prefix:   indent,
			Text:     label,
			Suffix:   suffix,
			Selected: entry.idx == p.cursor,
		}, innerWidth))
	}
	for len(rows) < listHeight {
		rows = append(rows, line(""))
	}

	hints := renderer.RenderSoftHints(innerWidth,
		SoftHint{Key: "/", Label: "search"},
		SoftHint{Key: "enter", Label: "confirm"},
		SoftHint{Key: "esc", Label: "revert"},
	)
	if p.filtering {
		hints = renderer.RenderSoftHints(innerWidth,
			SoftHint{Key: "type", Label: "filter"},
			SoftHint{Key: "enter", Label: "done"},
			SoftHint{Key: "esc", Label: "clear"},
		)
	}

	lines := append([]string{search}, rows...)
	if sampleLines > 0 {
		lines = append(lines, line(""))
		lines = append(lines, renderPickerSample(p.themes[p.cursor], plain, chrome.baseBg, innerWidth)...)
	}
	lines = append(lines, hints)
	return renderer.SoftPanelOverlay(SoftPanel{
		Prefix:  prefix,
		Title:   strings.ToLower(p.title),
		Content: renderer.RenderSoftBody(width, strings.Join(lines, "\n")),
		Width:   width,
	})
}

// visibleThemes lists the indices of the themes the filter leaves visible.
func (p ThemePicker) visibleThemes() []int {
	var out []int
	for _, entry := range p.entries() {
		if entry.header == "" {
			out = append(out, entry.idx)
		}
	}
	return out
}

func indexOf(values []int, want int) int {
	for i, v := range values {
		if v == want {
			return i
		}
	}
	return -1
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

// VisibleRange returns the half-open window of rows to draw so that cursor
// stays visible: centred on the cursor, clamped to the ends, and the whole
// range when it fits. It is exported because every scrolling list wants the
// same answer, and the alternative is each one growing its own copy that
// drifts - which is exactly what had already happened.
func VisibleRange(total, cursor, limit int) (int, int) {
	if limit >= total {
		return 0, total
	}
	start := cursor - limit/2
	start = max(0, min(start, total-limit))
	return start, start + limit
}
