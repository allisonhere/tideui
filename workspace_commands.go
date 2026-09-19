package tideui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Command is one runnable entry in the command palette. Workspace and panel
// actions are projected into Commands automatically, so an action declared on
// a panel is searchable without being registered twice.
type Command struct {
	ID       string
	Label    string
	Category string
	Key      string
	Panel    string
	Run      func()
}

// Display renders the command for palette rows and diagnostics.
func (c Command) Display() string {
	if c.Category != "" {
		return c.Category + " · " + c.Label
	}
	return c.Label
}

func (ws *Workspace) buildCommands() []Command {
	var out []Command
	add := func(cmd Command) { out = append(out, cmd) }

	add(Command{ID: "workspace.arrange", Label: "Arrange Panels", Category: "Workspace", Key: "m",
		Run: func() { ws.ToggleArrange() }})
	add(Command{ID: "workspace.picker", Label: "Panels", Category: "Workspace", Key: "w",
		Run: func() { ws.OpenPanelPicker() }})
	add(Command{ID: "workspace.reset", Label: "Reset Layout", Category: "Workspace",
		Run: func() { ws.ResetLayout() }})
	add(Command{ID: "workspace.undo", Label: "Undo Layout", Category: "Workspace",
		Run: func() { ws.Undo() }})
	add(Command{ID: "workspace.redo", Label: "Redo Layout", Category: "Workspace",
		Run: func() { ws.Redo() }})
	add(Command{ID: "workspace.peek.dismiss", Label: "Dismiss Peek", Category: "Workspace",
		Run: func() { ws.Unpeek() }})

	for _, name := range ws.presetOrder {
		preset := name
		add(Command{ID: "workspace.preset." + preset, Label: "Preset: " + preset, Category: "Workspace",
			Run: func() { ws.ApplyPreset(preset) }})
	}

	for _, id := range ws.order {
		panel := ws.panels[id]
		if panel == nil {
			continue
		}
		pid := id
		title := panel.TitleText()
		add(Command{ID: "panel.focus." + pid, Label: "Focus " + title, Category: "Panel",
			Run: func() { ws.Focus(pid) }})
		if panel.CanHide() {
			hidden := ws.isHidden(pid)
			label := "Hide " + title
			if hidden {
				label = "Show " + title
			}
			add(Command{ID: "panel.toggle." + pid, Label: label, Category: "Panel",
				Run: func() { ws.TogglePanel(pid) }})
		}
		if panel.CanZoom() {
			zoomed := ws.zoomCandidate() == pid
			label := "Zoom " + title
			if zoomed {
				label = "Restore " + title
			}
			add(Command{ID: "panel.zoom." + pid, Label: label, Category: "Panel", Key: "shift+space",
				Run: func() {
					if ws.zoomCandidate() == pid {
						ws.Unzoom()
					} else {
						ws.Zoom(pid)
					}
				}})
		}
		add(Command{ID: "panel.peek." + pid, Label: "Peek " + title, Category: "Panel",
			Run: func() { ws.Peek(pid) }})

		for _, action := range panel.actions {
			a := action
			category := a.Category
			if category == "" {
				category = title
			}
			add(Command{
				ID:       "action." + pid + "." + a.ID,
				Label:    a.displayLabel(),
				Category: category,
				Key:      a.Key,
				Panel:    pid,
				Run: func() {
					if a.Handler != nil {
						a.Handler(ws)
					}
				},
			})
		}
	}
	return out
}

// PaletteAction reports how a command palette update ended.
type PaletteAction int

const (
	// PaletteNone indicates navigation or a query edit.
	PaletteNone PaletteAction = iota
	// PaletteRun indicates the selected command ran.
	PaletteRun
	// PaletteClose indicates the palette was dismissed.
	PaletteClose
)

// CommandPalette is a searchable list over Commands. It is framework-provided
// so workspace and panel actions always have a home.
type CommandPalette struct {
	all      []Command
	filtered []Command
	cursor   int
	query    string
	opened   bool
}

// NewCommandPalette creates an empty palette.
func NewCommandPalette() *CommandPalette {
	return &CommandPalette{}
}

// Open shows the palette over a command set.
func (p *CommandPalette) Open(commands []Command) {
	p.all = append([]Command(nil), commands...)
	p.query = ""
	p.cursor = 0
	p.opened = true
	p.refilter()
}

// Opened reports whether the palette is displayed.
func (p CommandPalette) Opened() bool { return p.opened }

// Update handles query editing, navigation, and execution.
func (p *CommandPalette) Update(msg tea.KeyMsg) PaletteAction {
	if !p.opened {
		return PaletteNone
	}
	switch msg.String() {
	case "esc", "ctrl+c":
		p.opened = false
		return PaletteClose
	case "up", "ctrl+k":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "ctrl+j":
		if p.cursor < len(p.filtered)-1 {
			p.cursor++
		}
	case "enter":
		if p.cursor >= 0 && p.cursor < len(p.filtered) {
			command := p.filtered[p.cursor]
			p.opened = false
			if command.Run != nil {
				command.Run()
			}
			return PaletteRun
		}
	case "backspace":
		if p.query != "" {
			runes := []rune(p.query)
			p.query = string(runes[:len(runes)-1])
			p.refilter()
		}
	default:
		if msg.Type == tea.KeyRunes {
			p.query += string(msg.Runes)
			p.refilter()
		}
	}
	return PaletteNone
}

func (p *CommandPalette) refilter() {
	query := strings.ToLower(strings.TrimSpace(p.query))
	if query == "" {
		p.filtered = append([]Command(nil), p.all...)
	} else {
		p.filtered = p.filtered[:0]
		for _, command := range p.all {
			haystack := strings.ToLower(command.Label + " " + command.Category + " " + command.Key + " " + command.ID)
			if matchesAllTerms(haystack, query) {
				p.filtered = append(p.filtered, command)
			}
		}
	}
	p.cursor = clampIndex(p.cursor, max(1, len(p.filtered)))
}

func matchesAllTerms(haystack, query string) bool {
	for _, term := range strings.Fields(query) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

// Render draws the palette as a soft-panel overlay.
func (p CommandPalette) Render(r Renderer, width, height int) Overlay {
	if !p.opened {
		return Overlay{}
	}
	panelWidth := min(72, max(30, width-8))
	rowsAvailable := max(1, height-6)
	first, last := VisibleRange(len(p.filtered), p.cursor, rowsAvailable)
	innerWidth := max(1, panelWidth-4)
	rows := []string{r.Styles.OverlayHint.Width(innerWidth).Render("> " + p.query)}
	if len(p.filtered) == 0 {
		rows = append(rows, r.RenderSoftRow(SoftRow{Text: "No matching commands", Muted: true}, innerWidth))
	}
	for index := first; index < last; index++ {
		command := p.filtered[index]
		rows = append(rows, r.RenderSoftRow(SoftRow{
			Text:     command.Display(),
			Suffix:   r.keyGlyph(command.Key),
			Selected: index == p.cursor,
		}, innerWidth))
	}
	rows = append(rows, "", r.RenderSoftHints(innerWidth,
		SoftHint{Key: "enter", Label: "run"},
		SoftHint{Key: "esc", Label: "close"},
	))
	return r.SoftPanelOverlay(SoftPanel{
		Prefix:  "tide",
		Title:   "commands",
		Content: r.RenderSoftBody(panelWidth, strings.Join(rows, "\n")),
		Width:   panelWidth,
	})
}

// PanelPickerAction reports how a panel picker update ended.
type PanelPickerAction int

const (
	// PanelPickerNone indicates navigation.
	PanelPickerNone PanelPickerAction = iota
	// PanelPickerToggle indicates a panel's visibility changed.
	PanelPickerToggle
	// PanelPickerClose indicates the picker was dismissed.
	PanelPickerClose
)

// PanelPicker lists every registered panel with its visibility, allowing
// toggling without disturbing the rest of the layout.
type PanelPicker struct {
	ws     *Workspace
	order  []string
	cursor int
	opened bool
}

// NewPanelPicker creates an empty picker.
func NewPanelPicker() *PanelPicker { return &PanelPicker{} }

// Open shows the picker for a workspace.
func (p *PanelPicker) Open(panels map[string]*Panel, order []string, ws *Workspace) {
	p.ws = ws
	p.order = append([]string(nil), order...)
	p.cursor = clampIndex(p.cursor, max(1, len(p.order)))
	p.opened = true
	// Start on the focused panel when possible.
	if ws != nil {
		for i, id := range p.order {
			if id == ws.Focused() {
				p.cursor = i
				break
			}
		}
	}
}

// Opened reports whether the picker is displayed.
func (p PanelPicker) Opened() bool { return p.opened }

// Update handles navigation and toggling.
func (p *PanelPicker) Update(msg tea.KeyMsg) PanelPickerAction {
	if !p.opened {
		return PanelPickerNone
	}
	switch msg.String() {
	case "esc", "q", "ctrl+c":
		p.opened = false
		return PanelPickerClose
	case "j", "down":
		if p.cursor < len(p.order)-1 {
			p.cursor++
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case " ", "enter", "t":
		if p.cursor >= 0 && p.cursor < len(p.order) && p.ws != nil {
			p.ws.TogglePanel(p.order[p.cursor])
			return PanelPickerToggle
		}
	}
	return PanelPickerNone
}

// Render draws the picker as a soft-panel overlay.
func (p PanelPicker) Render(r Renderer, width, height int) Overlay {
	if !p.opened || p.ws == nil {
		return Overlay{}
	}
	panelWidth := min(64, max(30, width-8))
	rowsAvailable := max(1, height-6)
	first, last := VisibleRange(len(p.order), p.cursor, rowsAvailable)
	innerWidth := max(1, panelWidth-4)
	var rows []string
	for index := first; index < last; index++ {
		id := p.order[index]
		panel := p.ws.panels[id]
		if panel == nil {
			continue
		}
		mark := "[ ]"
		if !p.ws.isHidden(id) {
			mark = "[x]"
		}
		label := panel.TitleText()
		if id == p.ws.Focused() {
			label += "  ·"
		}
		rows = append(rows, r.RenderSoftRow(SoftRow{
			Prefix:   mark + " ",
			Text:     label,
			Suffix:   panel.role.String(),
			Selected: index == p.cursor,
		}, innerWidth))
	}
	rows = append(rows, "", r.RenderSoftHints(innerWidth,
		SoftHint{Key: "space", Label: "toggle"},
		SoftHint{Key: "esc", Label: "close"},
	))
	return r.SoftPanelOverlay(SoftPanel{
		Prefix:  "tide",
		Title:   "panels",
		Content: r.RenderSoftBody(panelWidth, strings.Join(rows, "\n")),
		Width:   panelWidth,
	})
}
