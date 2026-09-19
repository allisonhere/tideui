package tideui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// SoftPanel describes the Tide-family soft modal chrome: a rounded panel with
// its application prefix and title embedded in the top border.
type SoftPanel struct {
	Prefix  string
	Title   string
	Content string
	Width   int
}

// SoftHint is one key/action pair in a soft modal footer.
type SoftHint struct {
	Key   string
	Label string
}

// SoftRow is a single-line selectable row for soft modal lists and palettes.
type SoftRow struct {
	Prefix   string
	Text     string
	Suffix   string
	Selected bool
	Muted    bool
	Accent   bool // foreground uses the modal accent, e.g. for action buttons
}

type softChrome struct {
	baseBg       lipgloss.Color
	accent       lipgloss.Color
	border       lipgloss.Color
	text         lipgloss.Color
	muted        lipgloss.Color
	selectedBg   lipgloss.Color
	selectedText lipgloss.Color
	plain        bool
}

func newSoftChrome(styles Styles) softChrome {
	return newSoftChromeOn(styles, modalSurface(styles.Theme))
}

// newSoftChromeOn resolves the soft palette against a surface the caller
// names, so a row can sit on a pane as convincingly as it sits in a modal.
// Everything else - selection, muting, contrast correction - is derived from
// that surface, which is what keeps a row readable wherever it is drawn.
func newSoftChromeOn(styles Styles, baseBg lipgloss.Color) softChrome {
	accent := styles.Theme.BorderFocus
	if accent == "" {
		accent = styles.Theme.OverlayBorder
	}
	if accent == "" {
		accent = styles.Theme.Border
	}
	border := styles.Theme.OverlayBorder
	if border == "" {
		border = styles.Theme.Border
	}
	if border == "" {
		border = accent
	}
	text := readableText(styles.Theme.Fg, baseBg, 4.5)
	selectedBg := selectionBgForRatio(baseBg, selectedBgMinContrast)
	return softChrome{
		baseBg:       baseBg,
		accent:       accent,
		border:       border,
		text:         text,
		muted:        mutedText(text, baseBg),
		selectedBg:   selectedBg,
		selectedText: readableText(accent, selectedBg, 4.5),
		plain:        styles.PlainUI,
	}
}

// RenderSoftPanel renders a soft modal box. The returned string is the box
// only; assign it to Overlay.Content through SoftPanelOverlay or place it with
// application-owned overlay logic.
func (r Renderer) RenderSoftPanel(panel SoftPanel) string {
	width := panel.Width
	if width <= 0 {
		width = 48
	}
	width = max(1, width)
	chrome := newSoftChrome(r.Styles)
	inner := clampView(panel.Content, width, max(1, strings.Count(panel.Content, "\n")+1), chrome.baseBg)
	if chrome.plain {
		label := " " + panel.Prefix + " - "
		title := ansi.Truncate(panel.Title, max(1, width-lipgloss.Width(label)-4), "")
		borderStyle := lipgloss.NewStyle().Background(chrome.baseBg).Foreground(chrome.border)
		top := borderStyle.Render("+-") +
			lipgloss.NewStyle().Background(chrome.baseBg).Foreground(chrome.muted).Render(label) +
			lipgloss.NewStyle().Background(chrome.baseBg).Foreground(chrome.accent).Bold(true).Render(title) +
			borderStyle.Render(" ")
		top += borderStyle.Render(strings.Repeat("-", max(0, width+2-lipgloss.Width(top)-1)) + "+")
		bottom := borderStyle.Render("+" + strings.Repeat("-", width) + "+")
		lines := []string{top}
		for _, line := range strings.Split(inner, "\n") {
			lines = append(lines, borderStyle.Render("|")+padStyled(line, width, chrome.baseBg)+borderStyle.Render("|"))
		}
		lines = append(lines, bottom)
		return strings.Join(lines, "\n")
	}

	borderStyle := lipgloss.NewStyle().Background(chrome.baseBg).Foreground(chrome.border)
	label := panel.Prefix + " · "
	title := ansi.Truncate(panel.Title, max(1, width-lipgloss.Width(label)-6), "…")
	top := borderStyle.Render("╭─ ") +
		lipgloss.NewStyle().Background(chrome.baseBg).Foreground(chrome.muted).Render(label) +
		lipgloss.NewStyle().Background(chrome.baseBg).Foreground(chrome.accent).Bold(true).Render(title) +
		borderStyle.Render(" ")
	gap := max(0, width+2-lipgloss.Width(top)-1)
	top += borderStyle.Render(strings.Repeat("─", gap) + "╮")

	body := lipgloss.NewStyle().
		Background(chrome.baseBg).
		Border(lipgloss.RoundedBorder(), false, true, true, true).
		BorderForeground(chrome.border).
		BorderBackground(chrome.baseBg).
		Width(width).
		Render(inner)
	return top + "\n" + body
}

// SoftPanelOverlay returns an Overlay whose content is already soft-panel
// chrome. The Overlay title/footer are intentionally empty so the legacy modal
// header bar is not rendered around it.
func (r Renderer) SoftPanelOverlay(panel SoftPanel) Overlay {
	return Overlay{Visible: true, Content: r.RenderSoftPanel(panel), Width: panel.Width + 2, Raw: true}
}

// RenderSoftHints renders quiet lowercase key/action hints for soft modals.
func (r Renderer) RenderSoftHints(width int, hints ...SoftHint) string {
	chrome := newSoftChrome(r.Styles)
	keyStyle := lipgloss.NewStyle().Background(chrome.baseBg).Foreground(chrome.text)
	labelStyle := lipgloss.NewStyle().Background(chrome.baseBg).Foreground(chrome.muted)
	gap := lipgloss.NewStyle().Background(chrome.baseBg).Render("   ")
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		if hint.Key == "" && hint.Label == "" {
			continue
		}
		parts = append(parts, keyStyle.Render(r.keyGlyph(hint.Key))+labelStyle.Render(" "+strings.ToLower(hint.Label)))
	}
	line := lipgloss.NewStyle().Background(chrome.baseBg).Render("  ") + strings.Join(parts, gap)
	return padStyled(line, max(1, width), chrome.baseBg)
}

// RenderSoftBody pads and clamps pre-rendered soft modal content line-by-line.
// Use it when content contains styled rows that must not be re-wrapped by
// Lipgloss after ANSI styling has already been applied.
func (r Renderer) RenderSoftBody(width int, content string) string {
	chrome := newSoftChrome(r.Styles)
	width = max(1, width)
	innerWidth := max(1, width-4)
	pad := lipgloss.NewStyle().Background(chrome.baseBg).Render("  ")
	blank := lipgloss.NewStyle().Background(chrome.baseBg).Render(strings.Repeat(" ", width))
	lines := []string{blank}
	if content != "" {
		for _, line := range strings.Split(content, "\n") {
			if !strings.Contains(line, "\x1b[") {
				line = lipgloss.NewStyle().
					Background(chrome.baseBg).
					Foreground(chrome.text).
					Render(ansi.Truncate(line, innerWidth, ""))
			} else {
				line = ContinuousBackground(ansi.Truncate(line, innerWidth, ""), chrome.baseBg)
			}
			lines = append(lines, pad+padStyled(line, innerWidth, chrome.baseBg)+pad)
		}
	}
	lines = append(lines, blank)
	return strings.Join(lines, "\n")
}

// RenderSoftRow renders a soft modal row with a two-cell selected rail, on the
// modal surface.
func (r Renderer) RenderSoftRow(row SoftRow, width int) string {
	return r.RenderSoftRowOn(row, width, modalSurface(r.Styles.Theme))
}

// RenderSoftRowOn renders a soft row on a named background.
//
// A row drawn with RenderSoftRow sits on the modal surface, which is a lifted
// shade of the theme background. That is right inside a modal and wrong inside
// a pane: the rows come out a different colour from the blank space around
// them, and the pane looks banded. A screen that draws soft rows directly into
// a pane passes the pane's own background here.
func (r Renderer) RenderSoftRowOn(row SoftRow, width int, surface lipgloss.Color) string {
	chrome := newSoftChromeOn(r.Styles, surface)
	width = max(1, width)
	bg := chrome.baseBg
	fg := chrome.text
	switch {
	case row.Selected:
		bg = chrome.selectedBg
		fg = chrome.selectedText
	case row.Accent:
		fg = chrome.accent
	case row.Muted:
		fg = chrome.muted
	}
	rail := r.softRailOn(row.Selected, bg, chrome)
	contentW := max(1, width-lipgloss.Width(rail))
	content := alignRow(row.Prefix, row.Text, row.Suffix, contentW)
	content = StyleOver(lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(row.Selected), content)
	return rail + padStyled(content, contentW, bg)
}

// SoftRail renders the two-cell focus marker used in soft modal rows.
func (r Renderer) SoftRail(active bool, bg lipgloss.Color) string {
	return r.softRailOn(active, bg, newSoftChrome(r.Styles))
}

func (r Renderer) softRailOn(active bool, bg lipgloss.Color, chrome softChrome) string {
	glyph := "▌ "
	if chrome.plain {
		glyph = "> "
	}
	if !active {
		return lipgloss.NewStyle().Background(bg).Width(2).Render("")
	}
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(chrome.accent).
		Bold(true).
		Width(2).
		Render(glyph)
}

func padStyled(value string, width int, bg lipgloss.Color) string {
	width = max(0, width)
	value = ContinuousBackground(ansi.Truncate(value, width, ""), bg)
	padding := max(0, width-lipgloss.Width(value))
	if padding == 0 {
		return value
	}
	return value + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", padding))
}
