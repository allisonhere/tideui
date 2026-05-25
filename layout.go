package tideui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// LayoutMode selects how the three panes are arranged in the rendered shell.
type LayoutMode int

const (
	// StackedRight places pane 0 on the left and stacks panes 1 and 2 on the right.
	StackedRight LayoutMode = iota
	// ThreeColumn places all three panes side by side.
	ThreeColumn
)

// Pane supplies a header and rendered body for one region of a Layout.
type Pane struct {
	Title   string
	Hint    string
	Content string
	Focused bool
	Accent  lipgloss.Color
}

// StatusBar supplies optional left- and right-aligned footer text.
type StatusBar struct {
	Left  string
	Right string
}

// Overlay describes a centered modal displayed over the rendered layout.
type Overlay struct {
	Visible bool
	Title   string
	Content string
	Footer  string
	Width   int
}

// Layout supplies terminal dimensions, three panes, and optional shell chrome.
type Layout struct {
	Width  int
	Height int
	Mode   LayoutMode
	Panes  [3]Pane
	Status *StatusBar
	Modal  *Overlay

	// SidebarRatio controls pane 0 in StackedRight mode. Zero uses 0.28.
	SidebarRatio float64
	// UpperRightRatio controls pane 1 in StackedRight mode. Zero uses 0.40.
	UpperRightRatio float64
	// ColumnRatios controls widths in ThreeColumn mode. Zero values use equal columns.
	ColumnRatios [3]float64
}

// Renderer renders Layout and Row values using one resolved set of styles.
type Renderer struct {
	Styles Styles
}

// NewRenderer creates a renderer for a theme and style options.
func NewRenderer(theme Theme, options StyleOptions) Renderer {
	return Renderer{Styles: BuildStyles(theme, options)}
}

// Render produces a terminal-sized themed view for layout.
func (r Renderer) Render(layout Layout) string {
	if layout.Width <= 0 || layout.Height <= 0 {
		return ""
	}
	statusHeight := 0
	if layout.Status != nil {
		statusHeight = 1
	}
	mainHeight := max(1, layout.Height-statusHeight)

	var main string
	switch layout.Mode {
	case ThreeColumn:
		main = r.renderThreeColumn(layout.Panes, layout.Width, mainHeight, layout.ColumnRatios)
	default:
		main = r.renderStackedRight(layout.Panes, layout.Width, mainHeight, layout.SidebarRatio, layout.UpperRightRatio)
	}
	view := main
	if layout.Status != nil {
		view = lipgloss.JoinVertical(lipgloss.Left, main, r.renderStatus(*layout.Status, layout.Width))
	}
	if layout.Modal != nil && layout.Modal.Visible {
		view = overlayOnBase(view, r.renderOverlay(*layout.Modal, layout.Width), layout.Width, layout.Height, r.Styles.Theme.Bg)
	}
	return clampView(view, layout.Width, layout.Height, r.Styles.Theme.Bg)
}

// Row is a generic themed list row with optional left and right content.
type Row struct {
	Prefix   string
	Text     string
	Suffix   string
	Selected bool
	Muted    bool
}

// RenderRow formats one Row to the requested content width.
func (r Renderer) RenderRow(row Row, width int) string {
	style := r.Styles.Item
	if row.Muted {
		style = r.Styles.ItemMuted
	}
	if row.Selected {
		style = r.Styles.ItemSelected
	}
	return style.Width(width).Render(alignRow(row.Prefix, row.Text, row.Suffix, width))
}

func (r Renderer) renderStackedRight(panes [3]Pane, width, height int, sidebarRatio, upperRatio float64) string {
	if sidebarRatio <= 0 || sidebarRatio >= 1 {
		sidebarRatio = 0.28
	}
	if upperRatio <= 0 || upperRatio >= 1 {
		upperRatio = 0.40
	}
	sidebarWidth := ratioSize(width, sidebarRatio)
	rightWidth := width - sidebarWidth
	upperHeight := ratioSize(height, upperRatio)
	lowerHeight := height - upperHeight

	left := r.renderPane(panes[0], sidebarWidth, height, true, false)
	right := lipgloss.JoinVertical(lipgloss.Left,
		r.renderPane(panes[1], rightWidth, upperHeight, false, true),
		r.renderPane(panes[2], rightWidth, lowerHeight, false, false),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (r Renderer) renderThreeColumn(panes [3]Pane, width, height int, ratios [3]float64) string {
	widths := columnSizes(width, ratios)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		r.renderPane(panes[0], widths[0], height, true, false),
		r.renderPane(panes[1], widths[1], height, true, false),
		r.renderPane(panes[2], widths[2], height, false, false),
	)
}

func (r Renderer) renderPane(pane Pane, width, height int, rightBorder, bottomBorder bool) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	// Keep separators inside their assigned region on constrained terminals.
	rightBorder = rightBorder && width > 1
	bottomBorder = bottomBorder && height > 1
	innerWidth := width
	innerHeight := height
	if rightBorder {
		innerWidth--
	}
	if bottomBorder {
		innerHeight--
	}
	innerWidth = max(1, innerWidth)
	innerHeight = max(1, innerHeight)

	contentHeight := max(0, innerHeight-1)
	header := r.renderHeader(pane, innerWidth)
	bodyContent := r.Styles.DetailBody.Width(innerWidth).Render(pane.Content)
	body := clampView(bodyContent, innerWidth, contentHeight, r.Styles.Theme.Bg)
	content := header
	if contentHeight > 0 {
		content += "\n" + body
	}

	borderColor := r.Styles.Theme.Border
	if pane.Focused {
		borderColor = r.Styles.Theme.BorderFocus
		if pane.Accent != "" {
			borderColor = pane.Accent
		}
	}
	return r.Styles.Pane.Copy().
		Border(paneBorder(r.Styles.PlainUI), false, rightBorder, bottomBorder, false).
		BorderForeground(borderColor).
		Width(innerWidth).
		Height(innerHeight).
		Render(content)
}

func (r Renderer) renderHeader(pane Pane, width int) string {
	prefix := "  "
	if pane.Focused {
		prefix = "> "
	}
	return r.paneHeaderStyle(pane).Width(width).Render(alignRow(prefix, pane.Title, pane.Hint, width))
}

func (r Renderer) paneHeaderStyle(pane Pane) lipgloss.Style {
	if !pane.Focused {
		return r.Styles.PaneHeaderInactive
	}
	if pane.Accent == "" {
		return r.Styles.PaneHeaderActive
	}
	return r.Styles.PaneHeaderActive.Copy().Background(pane.Accent).
		Foreground(readableText(r.Styles.Theme.Fg, pane.Accent, 4.5))
}

func (r Renderer) renderStatus(status StatusBar, width int) string {
	innerWidth := max(0, width-2)
	left := strings.ReplaceAll(status.Left, "\n", " ")
	right := strings.ReplaceAll(status.Right, "\n", " ")
	right = ansi.Truncate(right, innerWidth, "")
	leftWidth := innerWidth - lipgloss.Width(right)
	if right != "" && left != "" {
		leftWidth -= 2
	}
	left = ansi.Truncate(left, max(0, leftWidth), "")
	gap := max(0, innerWidth-lipgloss.Width(left)-lipgloss.Width(right))
	line := " " + left + strings.Repeat(" ", gap) + right + " "
	return r.Styles.StatusBar.Copy().UnsetPadding().Render(line)
}

func (r Renderer) renderOverlay(overlay Overlay, windowWidth int) string {
	width := overlay.Width
	if width <= 0 {
		width = min(64, max(16, windowWidth-6))
	}
	width = min(width, max(1, windowWidth-2))
	innerWidth := max(1, width-r.Styles.Overlay.GetHorizontalFrameSize())
	parts := []string{}
	if overlay.Title != "" {
		parts = append(parts, r.Styles.OverlayTitle.Width(innerWidth).Render(overlay.Title))
	}
	if overlay.Content != "" {
		parts = append(parts, r.Styles.OverlayBody.Width(innerWidth).Render(overlay.Content))
	}
	if overlay.Footer != "" {
		parts = append(parts, r.Styles.OverlayHint.Width(innerWidth).Render(overlay.Footer))
	}
	outerWidth := max(1, width-r.Styles.Overlay.GetHorizontalBorderSize())
	return r.Styles.Overlay.Width(outerWidth).Render(strings.Join(parts, "\n"))
}

func alignRow(prefix, text, suffix string, width int) string {
	if width <= 0 {
		return ""
	}
	prefixWidth, suffixWidth := lipgloss.Width(prefix), lipgloss.Width(suffix)
	gap := 0
	if suffix != "" {
		gap = 1
	}
	textWidth := max(0, width-prefixWidth-suffixWidth-gap)
	text = ansi.Truncate(text, textWidth, "")
	line := prefix + text + strings.Repeat(" ", max(0, textWidth-lipgloss.Width(text)))
	if suffix != "" {
		line += " " + suffix
	}
	return line + strings.Repeat(" ", max(0, width-lipgloss.Width(line)))
}

func ratioSize(total int, ratio float64) int {
	if total <= 1 {
		return total
	}
	return max(1, min(total-1, int(float64(total)*ratio)))
}

func columnSizes(width int, ratios [3]float64) [3]int {
	if width <= 2 {
		out := [3]int{}
		for i := 0; i < width; i++ {
			out[i] = 1
		}
		return out
	}
	for i := range ratios {
		if ratios[i] <= 0 {
			ratios[i] = 1
		}
	}
	total := ratios[0] + ratios[1] + ratios[2]
	available := width - 3
	firstExtra := int(float64(available) * ratios[0] / total)
	secondExtra := int(float64(available) * ratios[1] / total)
	return [3]int{1 + firstExtra, 1 + secondExtra, 1 + available - firstExtra - secondExtra}
}

func clampView(view string, width, height int, background lipgloss.Color) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	fill := lipgloss.NewStyle().Background(background)
	lines := strings.Split(view, "\n")
	if view == "" {
		lines = nil
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		line = ansi.Truncate(line, width, "")
		padding := width - lipgloss.Width(line)
		if padding > 0 {
			line += fill.Render(strings.Repeat(" ", padding))
		}
		lines[i] = line
	}
	for len(lines) < height {
		lines = append(lines, fill.Render(strings.Repeat(" ", width)))
	}
	return strings.Join(lines, "\n")
}

func overlayOnBase(base, box string, width, height int, background lipgloss.Color) string {
	base = clampView(base, width, height, background)
	boxLines := strings.Split(box, "\n")
	boxWidth := 0
	for _, line := range boxLines {
		boxWidth = max(boxWidth, lipgloss.Width(line))
	}
	x := max(0, (width-boxWidth)/2)
	y := max(0, (height-len(boxLines))/2)
	lines := strings.Split(base, "\n")
	for i, line := range boxLines {
		target := y + i
		if target >= height {
			break
		}
		left := ansi.Cut(lines[target], 0, x)
		right := ansi.Cut(lines[target], min(width, x+boxWidth), width)
		lines[target] = left + line + right
	}
	return strings.Join(lines, "\n")
}
