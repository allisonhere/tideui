# tideui

`tideui` is a reusable Bubble Tea/Lipgloss presentation toolkit extracted from
TideMail's themeable terminal interface. It renders application-provided content
inside themed pane shells, status bars, and overlays.

The package is intentionally view-oriented. Applications retain ownership of
their Bubble Tea model, commands, key bindings, persistence, and viewport state.

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

## Theme Pickers

The library exposes `BuiltinThemes`, `ThemeByName`, and instant renderer
reconstruction so an application can preview theme selections while navigating
a picker. It does not own picker state, keys, confirmation, or config
persistence; those remain application responsibilities in v1.

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
