package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/allisonhere/tideui"
)

type model struct {
	width, height int
	focus         int
	theme         tideui.Theme
	picker        tideui.ThemePicker
	density       tideui.Density
	mode          tideui.LayoutMode
	showOverlay   bool
}

func newModel() model {
	theme := tideui.BuiltinThemes[0]
	return model{
		focus:   1,
		theme:   theme,
		picker:  tideui.NewThemePicker(tideui.ThemePickerOptions{InitialTheme: theme.Name}),
		density: tideui.Compact,
		mode:    tideui.StackedRight,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.picker.Opened() {
			m.picker.Update(msg)
			m.theme = m.picker.PreviewTheme()
			return m, nil
		}
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 3
		case "shift+tab":
			m.focus = (m.focus + 2) % 3
		case "t":
			m.showOverlay = false
			m.picker.Open(m.theme.Name)
		case "d":
			if m.density == tideui.Compact {
				m.density = tideui.Comfortable
			} else {
				m.density = tideui.Compact
			}
		case "l":
			if m.mode == tideui.StackedRight {
				m.mode = tideui.ThreeColumn
			} else {
				m.mode = tideui.StackedRight
			}
		case "o":
			m.showOverlay = !m.showOverlay
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	renderer := tideui.NewRenderer(m.theme, tideui.StyleOptions{Density: m.density})
	styles := renderer.Styles
	items := []string{
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Inbox", Suffix: "8", Selected: m.focus == 0}, 20),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Projects"}, 20),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Archive", Muted: true}, 20),
	}
	tasks := []string{
		renderer.RenderRow(tideui.Row{Prefix: "* ", Text: "Extract UI toolkit", Selected: m.focus == 1}, 28),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Publish module", Muted: true}, 28),
	}
	detail := styles.DetailTitle.Render("Extract UI toolkit") + "\n\n" +
		styles.DetailBody.Render("This content is owned by the application.\nThe library supplies its shell and theme.")

	layoutName := "stacked-right"
	if m.mode == tideui.ThreeColumn {
		layoutName = "three-column"
	}
	var modal *tideui.Overlay
	if m.picker.Opened() {
		pickerModal := m.picker.Modal(renderer, 42, m.height)
		modal = &pickerModal
	} else if m.showOverlay {
		modal = &tideui.Overlay{
			Visible: true,
			Title:   "TIDEUI",
			Content: "This overlay is rendered by the library.\nThe application controls whether it is open.",
			Footer:  "o close",
			Width:   52,
		}
	}
	return renderer.Render(tideui.Layout{
		Width: m.width, Height: m.height, Mode: m.mode,
		Panes: [3]tideui.Pane{
			{Title: "Boards", Hint: "tab focus", Content: strings.Join(items, "\n"), Focused: m.focus == 0},
			{Title: "Tasks", Hint: "t themes", Content: strings.Join(tasks, "\n"), Focused: m.focus == 1},
			{Title: "Detail", Hint: "o modal", Content: detail, Focused: m.focus == 2},
		},
		Status: &tideui.StatusBar{
			Left:  fmt.Sprintf("%s | %s | %s", m.theme.Name, m.density, layoutName),
			Right: "tab focus  t themes  d density  l layout  o overlay  q quit",
		},
		Modal: modal,
	})
}

func main() {
	if _, err := tea.NewProgram(newModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
