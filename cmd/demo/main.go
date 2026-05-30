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
	scrollers     [3]tideui.PaneScroller
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
			switch m.mode {
			case tideui.StackedRight:
				m.mode = tideui.ThreeColumn
			case tideui.ThreeColumn:
				m.mode = tideui.SidebarOnly
			case tideui.SidebarOnly:
				m.mode = tideui.Tabbed
			case tideui.Tabbed:
				m.mode = tideui.Floating
			default:
				m.mode = tideui.StackedRight
			}
		case "o":
			m.showOverlay = !m.showOverlay
		case "j", "down":
			m.scrollers[m.focus].ScrollDown(1)
		case "k", "up":
			m.scrollers[m.focus].ScrollUp(1)
		case "g":
			m.scrollers[m.focus].ScrollToTop()
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	renderer := tideui.NewRenderer(m.theme, tideui.StyleOptions{Density: m.density})
	boardWidth := 20
	items := []string{
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Inbox", Suffix: "8", Selected: m.focus == 0}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Projects"}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Archive", Muted: true}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Drafts", Suffix: "2"}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Sent"}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Spam", Muted: true}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Trash", Muted: true}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Work"}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Personal"}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Reading list"}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Bookmarks", Suffix: "5"}, boardWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Starred"}, boardWidth),
	}
	taskWidth := 28
	tasks := []string{
		renderer.RenderRow(tideui.Row{Prefix: "* ", Text: "Extract UI toolkit", Selected: m.focus == 1}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "* ", Text: "Write tests"}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "* ", Text: "Add scroll support"}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "* ", Text: "Theme picker"}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Publish module", Muted: true}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Write README", Muted: true}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Add examples", Muted: true}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Tag release", Muted: true}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Announce", Muted: true}, taskWidth),
		renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Update docs", Muted: true}, taskWidth),
	}
	messages := []tideui.Block{
		{Prefix: "● ", Header: "alice", Meta: "09:41", Body: "The UI toolkit is looking great — themes and layout modes are working perfectly."},
		{Prefix: "● ", Header: "bob", Meta: "09:43", Body: "Agreed. Just added multi-line block support so message threads like this one render cleanly.", Selected: m.focus == 2},
		{Prefix: "○ ", Header: "alice", Meta: "09:45", Body: "Does it handle scrolling? Long threads should scroll per-pane.", Muted: true},
		{Prefix: "○ ", Header: "bob", Meta: "09:46", Body: "Yes — use j/k to scroll any pane. The offset is owned by the app, not the library.", Muted: true},
		{Prefix: "○ ", Header: "alice", Meta: "09:48", Body: "Nice. And density?", Muted: true},
		{Prefix: "○ ", Header: "bob", Meta: "09:49", Body: "Press d to toggle compact vs comfortable spacing. Both modes work for blocks.", Muted: true},
		{Prefix: "○ ", Header: "alice", Meta: "09:51", Body: "What about the floating and tabbed layouts?", Muted: true},
		{Prefix: "○ ", Header: "bob", Meta: "09:52", Body: "Press l to cycle through all five. Floating panels sit over pane 0. Tabbed shows one pane at a time.", Muted: true},
	}
	detail := strings.Join(func() []string {
		out := make([]string, len(messages))
		for i, b := range messages {
			out[i] = renderer.RenderBlock(b, 44)
		}
		return out
	}(), "\n")

	layoutNames := map[tideui.LayoutMode]string{
		tideui.StackedRight: "stacked-right",
		tideui.ThreeColumn:  "three-column",
		tideui.SidebarOnly:  "sidebar-only",
		tideui.Tabbed:       "tabbed",
		tideui.Floating:     "floating",
	}
	layoutName := layoutNames[m.mode]
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
			{Title: "Boards", Hint: "tab focus", Content: strings.Join(items, "\n"), Focused: m.focus == 0, ScrollOffset: m.scrollers[0].Offset()},
			{Title: "Tasks", Hint: "t themes", Content: strings.Join(tasks, "\n"), Focused: m.focus == 1, ScrollOffset: m.scrollers[1].Offset()},
			{Title: "Detail", Hint: "o modal", Content: detail, Focused: m.focus == 2, ScrollOffset: m.scrollers[2].Offset()},
		},
		Status: &tideui.StatusBar{
			Left:  fmt.Sprintf("%s  %s  %s", m.theme.Name, m.density, layoutName),
			Right: "tab  j/k scroll  g top  t  d  l  o  q",
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
