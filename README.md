# tideui

`tideui` is a reusable Bubble Tea/Lipgloss presentation toolkit extracted from
TideMail's themeable terminal interface. It renders application-provided content
inside themed pane shells, status bars, and overlays.

The package is intentionally view-oriented. Applications retain ownership of
their Bubble Tea model, commands, key bindings, persistence, and viewport state.

## Install

```bash
go get github.com/allisonhere/tideui
```

## Features

- Nineteen built-in palettes with optional background, foreground, and accent overrides.
- `StackedRight` layout matching TideMail and a general `ThreeColumn` layout.
- Compact and comfortable density modes plus VT52 ASCII presentation.
- Themed pane headers, rows, status bars, overlays, and terminal background sequences.
- Output constrained to the requested terminal dimensions, including very small windows.

## Usage

```go
import "github.com/allisonhere/tideui"

theme, _ := tideui.ThemeByName("catppuccin-mocha")
renderer := tideui.NewRenderer(theme, tideui.StyleOptions{Density: tideui.Compact})

view := renderer.Render(tideui.Layout{
    Width: 80, Height: 24, Mode: tideui.StackedRight,
    Panes: [3]tideui.Pane{
        {Title: "Projects", Content: "inbox\narchive", Focused: true},
        {Title: "Tasks", Content: "ship tideui"},
        {Title: "Detail", Content: "Application-owned content."},
    },
    Status: &tideui.StatusBar{Left: "ready", Right: "? help"},
})
```

## Bubble Tea Integration

Store terminal dimensions from `tea.WindowSizeMsg`, keep your application state
in your own model, and construct the renderer from the currently selected theme:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if size, ok := msg.(tea.WindowSizeMsg); ok {
        m.width, m.height = size.Width, size.Height
    }
    return m, nil
}

func (m model) View() string {
    renderer := tideui.NewRenderer(m.theme, tideui.StyleOptions{Density: m.density})
    return renderer.Render(tideui.Layout{
        Width: m.width, Height: m.height, Mode: tideui.ThreeColumn,
        Panes: m.panes(),
    })
}
```

## Layouts

`StackedRight` renders pane `0` as a left sidebar, pane `1` above pane `2` on
the right, and defaults to TideMail's `28%` sidebar and `40%` upper-right
height. Configure it with `SidebarRatio` and `UpperRightRatio`.

`ThreeColumn` renders the three panes from left to right. Use `ColumnRatios` to
allocate relative widths:

```go
layout.Mode = tideui.ThreeColumn
layout.ColumnRatios = [3]float64{2, 3, 5}
```

## Rows And Content

Pane content is application-provided text. Use `RenderRow` for themed list rows
and the exported `Styles` for custom detail content:

```go
rows := []string{
    renderer.RenderRow(tideui.Row{Prefix: "* ", Text: "Selected", Suffix: "3", Selected: true}, 26),
    renderer.RenderRow(tideui.Row{Prefix: "  ", Text: "Archived", Muted: true}, 26),
}
detail := renderer.Styles.DetailTitle.Render("Selected") + "\n\n" +
    renderer.Styles.DetailBody.Render("Application-owned detail content.")
```

Focused panes use the theme accent. Set `Pane.Accent` only when an individual
pane should intentionally use another accent color.

## Themes

Choose from `BuiltinThemes`, resolve a saved name with `ThemeByName`, or adjust
a built-in theme through `ThemeOverrides`:

```go
renderer := tideui.NewRenderer(tideui.VT100, tideui.StyleOptions{
    Density: tideui.Compact,
    Overrides: tideui.ThemeOverrides{
        Background: "#080b08",
        Foreground: "#8cff8c",
        Accent:     "#33ff33",
    },
})
```

Built-in theme names:

`catppuccin-mocha`, `catppuccin-latte`, `catppuccin-frappe`,
`catppuccin-macchiato`, `nord`, `dracula`, `gruvbox-dark`, `gruvbox-light`,
`tokyo-night`, `tokyo-night-day`, `rose-pine`, `rose-pine-moon`,
`rose-pine-dawn`, `one-dark`, `magenta-geode`, `coral-sunset`,
`lavender-fields-forever`, `vt100`, and `vt52`.

## Theme Pickers

The library exposes `BuiltinThemes`, `ThemeByName`, and instant renderer
reconstruction so an application can preview theme selections while navigating
a picker. It does not own picker state, keys, confirmation, or config
persistence; those remain application responsibilities in v1.

## Status Bars And Overlays

Provide a `StatusBar` and optional `Overlay` in the layout; `Width` on an
overlay is the full modal width including its border:

```go
layout.Status = &tideui.StatusBar{Left: "ready", Right: "? help"}
layout.Modal = &tideui.Overlay{
    Visible: showHelp,
    Title:   "HELP",
    Content: "j/k move\nenter select",
    Footer:  "esc close",
    Width:   36,
}
```

## Terminal Background

`TerminalBackgroundSequences` returns OSC sequences for terminals that support
changing their default background color. It does not write to stdout or the
terminal; the application decides whether and where to emit the strings.

## API Boundaries

In v1, `tideui` renders presentation primitives. The consuming application owns:

- Bubble Tea `Update` behavior and commands.
- Keyboard navigation and focus state.
- Viewport scrolling and content formatting.
- Theme picker state and persisted configuration.
- Terminal control sequence output.

## Requirements

The module currently targets Go 1.26 or newer and uses Bubble Tea and Lipgloss.

## Development

```bash
go test ./...
go vet ./...
```

Run the demo with `go run ./cmd/demo`. Use `tab` to move focus, `l` to change
layout, `t` to cycle themes, `d` to switch density, `o` to toggle the overlay,
and `q` to quit.
