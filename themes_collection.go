package tideui

import "github.com/charmbracelet/lipgloss"

// The built-in palettes. Each family below is taken from its upstream project
// (source and license in the comment above its group, and in docs/themes.md);
// the colors were copied from those projects' palette files rather than
// sampled from screenshots. Monokai Pro is deliberately absent: its palette is
// proprietary and not licensed for redistribution.
//
// A palette names the colors a theme needs. collectionTheme maps them onto the
// Theme fields: the accent drives focus, selection and the modal border, green
// is the unread/success color, and the remaining roles fill the optional
// semantic fields.
type palette struct {
	bg, fg, border, accent, surface, muted string
	red, green, yellow                     string
	link, quote, tag, match, cursor        string
}

func collectionTheme(name, family, variant string, p palette) Theme {
	c := func(hex string) lipgloss.Color { return lipgloss.Color("#" + hex) }
	return Theme{
		Name: name, Family: family, Variant: variant,
		Bg: c(p.bg), Fg: c(p.fg), Border: c(p.border), BorderFocus: c(p.accent),
		Selected: c(p.accent), Unread: c(p.green), Dimmed: c(p.muted),
		StatusBar: c(p.surface), StatusFg: Legible(c(p.fg), c(p.surface), 4.5), Error: c(p.red),
		Overlay: c(p.surface), OverlayBorder: c(p.accent),
		Success: c(p.green), Warning: c(p.yellow), Link: c(p.link),
		Quote: c(p.quote), Match: c(p.match), Tag: c(p.tag), Cursor: c(p.cursor),
	}
}

// Catppuccin — https://github.com/catppuccin/palette (MIT).
var (
	// CatppuccinMocha is the Catppuccin Mocha palette.
	CatppuccinMocha = collectionTheme("catppuccin-mocha", "Catppuccin", "Mocha", palette{bg: "1e1e2e", fg: "cdd6f4", border: "6c7086", accent: "89b4fa", surface: "313244", muted: "7f849c", red: "f38ba8", green: "a6e3a1", yellow: "f9e2af", link: "74c7ec", quote: "94e2d5", tag: "cba6f7", match: "fab387", cursor: "f5e0dc"})
	// CatppuccinMacchiato is the Catppuccin Macchiato palette.
	CatppuccinMacchiato = collectionTheme("catppuccin-macchiato", "Catppuccin", "Macchiato", palette{bg: "24273a", fg: "cad3f5", border: "6e738d", accent: "8aadf4", surface: "363a4f", muted: "8087a2", red: "ed8796", green: "a6da95", yellow: "eed49f", link: "7dc4e4", quote: "8bd5ca", tag: "c6a0f6", match: "f5a97f", cursor: "f4dbd6"})
	// CatppuccinFrappe is the Catppuccin Frappé palette.
	CatppuccinFrappe = collectionTheme("catppuccin-frappe", "Catppuccin", "Frappé", palette{bg: "303446", fg: "c6d0f5", border: "737994", accent: "8caaee", surface: "414559", muted: "838ba7", red: "e78284", green: "a6d189", yellow: "e5c890", link: "85c1dc", quote: "81c8be", tag: "ca9ee6", match: "ef9f76", cursor: "f2d5cf"})
	// CatppuccinLatte is the Catppuccin Latte palette.
	CatppuccinLatte = collectionTheme("catppuccin-latte", "Catppuccin", "Latte", palette{bg: "eff1f5", fg: "4c4f69", border: "9ca0b0", accent: "1e66f5", surface: "ccd0da", muted: "8c8fa1", red: "d20f39", green: "40a02b", yellow: "df8e1d", link: "209fb5", quote: "179299", tag: "8839ef", match: "fe640b", cursor: "dc8a78"})
)

// One Dark — https://github.com/Binaryify/OneDark-Pro (MIT).
var (
	// OneDark is the One Dark Pro palette.
	OneDark = collectionTheme("one-dark", "One Dark", "Pro", palette{bg: "282c34", fg: "abb2bf", border: "3e4452", accent: "61afef", surface: "21252b", muted: "5c6370", red: "e05561", green: "8cc265", yellow: "d19a66", link: "42b3c2", quote: "4aa5f0", tag: "c162de", match: "d19a66", cursor: "528bff"})
	// OneDarkDarker is the One Dark Pro Darker palette.
	OneDarkDarker = collectionTheme("one-dark-darker", "One Dark", "Pro Darker", palette{bg: "23272e", fg: "abb2bf", border: "3e4452", accent: "61afef", surface: "1e2227", muted: "5c6370", red: "e05561", green: "8cc265", yellow: "d19a66", link: "42b3c2", quote: "4aa5f0", tag: "c162de", match: "d19a66", cursor: "528bff"})
)

// Tokyo Night — https://github.com/folke/tokyonight.nvim (Apache-2.0).
var (
	// TokyoNight is the Tokyo Night Night palette.
	TokyoNight = collectionTheme("tokyo-night", "Tokyo Night", "Night", palette{bg: "1a1b26", fg: "c0caf5", border: "414868", accent: "7aa2f7", surface: "16161e", muted: "565f89", red: "f7768e", green: "9ece6a", yellow: "e0af68", link: "7dcfff", quote: "73daca", tag: "bb9af7", match: "ff9e64", cursor: "c0caf5"})
	// TokyoNightStorm is the Tokyo Night Storm palette.
	TokyoNightStorm = collectionTheme("tokyo-night-storm", "Tokyo Night", "Storm", palette{bg: "24283b", fg: "c0caf5", border: "414868", accent: "7aa2f7", surface: "1f2335", muted: "565f89", red: "f7768e", green: "9ece6a", yellow: "e0af68", link: "7dcfff", quote: "73daca", tag: "bb9af7", match: "ff9e64", cursor: "c0caf5"})
	// TokyoNightMoon is the Tokyo Night Moon palette.
	TokyoNightMoon = collectionTheme("tokyo-night-moon", "Tokyo Night", "Moon", palette{bg: "222436", fg: "c8d3f5", border: "444a73", accent: "82aaff", surface: "1e2030", muted: "636da6", red: "ff757f", green: "c3e88d", yellow: "ffc777", link: "86e1fc", quote: "4fd6be", tag: "c099ff", match: "ff966c", cursor: "c8d3f5"})
	// TokyoNightDay is the Tokyo Night Day palette.
	TokyoNightDay = collectionTheme("tokyo-night-day", "Tokyo Night", "Day", palette{bg: "e1e2e7", fg: "3760bf", border: "a1a6c5", accent: "2e7de9", surface: "d0d5e3", muted: "848cb5", red: "f52a65", green: "587539", yellow: "8c6c3e", link: "007197", quote: "118c74", tag: "9854f1", match: "b15c00", cursor: "3760bf"})
)

// Dracula — https://github.com/dracula/dracula-theme (MIT).
var (
	// Dracula is the Dracula Dracula palette.
	Dracula = collectionTheme("dracula", "Dracula", "Dracula", palette{bg: "282a36", fg: "f8f8f2", border: "6272a4", accent: "bd93f9", surface: "44475a", muted: "6272a4", red: "ff5555", green: "50fa7b", yellow: "f1fa8c", link: "8be9fd", quote: "6272a4", tag: "ff79c6", match: "ffb86c", cursor: "f8f8f2"})
	// DraculaAlucard is the Dracula Alucard palette.
	DraculaAlucard = collectionTheme("dracula-alucard", "Dracula", "Alucard", palette{bg: "fffbeb", fg: "1f1f1f", border: "cfcfde", accent: "644ac9", surface: "cfcfde", muted: "6c664b", red: "cb3a2a", green: "14710a", yellow: "846e15", link: "036a96", quote: "6c664b", tag: "a3144d", match: "a34d14", cursor: "1f1f1f"})
)

// GitHub — https://github.com/primer/github-vscode-theme (MIT).
var (
	// GithubDark is the GitHub Dark palette.
	GithubDark = collectionTheme("github-dark", "GitHub", "Dark", palette{bg: "0d1117", fg: "c9d1d9", border: "30363d", accent: "58a6ff", surface: "161b22", muted: "8b949e", red: "f85149", green: "3fb950", yellow: "d29922", link: "58a6ff", quote: "8b949e", tag: "a371f7", match: "d29922", cursor: "c9d1d9"})
	// GithubDarkDimmed is the GitHub Dark Dimmed palette.
	GithubDarkDimmed = collectionTheme("github-dark-dimmed", "GitHub", "Dark Dimmed", palette{bg: "22272e", fg: "adbac7", border: "444c56", accent: "539bf5", surface: "2d333b", muted: "768390", red: "e5534b", green: "57ab5a", yellow: "c69026", link: "539bf5", quote: "768390", tag: "986ee2", match: "c69026", cursor: "adbac7"})
	// GithubLight is the GitHub Light palette.
	GithubLight = collectionTheme("github-light", "GitHub", "Light", palette{bg: "ffffff", fg: "24292f", border: "d0d7de", accent: "0969da", surface: "f6f8fa", muted: "57606a", red: "cf222e", green: "1a7f37", yellow: "9a6700", link: "0969da", quote: "57606a", tag: "8250df", match: "9a6700", cursor: "24292f"})
)

// Gruvbox — https://github.com/morhetz/gruvbox (MIT).
var (
	// GruvboxDark is the Gruvbox Dark palette.
	GruvboxDark = collectionTheme("gruvbox-dark", "Gruvbox", "Dark", palette{bg: "282828", fg: "ebdbb2", border: "665c54", accent: "83a598", surface: "3c3836", muted: "928374", red: "fb4934", green: "b8bb26", yellow: "fabd2f", link: "83a598", quote: "8ec07c", tag: "d3869b", match: "fe8019", cursor: "ebdbb2"})
	// GruvboxLight is the Gruvbox Light palette.
	GruvboxLight = collectionTheme("gruvbox-light", "Gruvbox", "Light", palette{bg: "fbf1c7", fg: "3c3836", border: "bdae93", accent: "076678", surface: "ebdbb2", muted: "7c6f64", red: "9d0006", green: "79740e", yellow: "b57614", link: "076678", quote: "427b58", tag: "8f3f71", match: "af3a03", cursor: "3c3836"})
)

// Nord — https://github.com/nordtheme/nord (MIT).
var (
	// Nord is the Nord Nord palette.
	Nord = collectionTheme("nord", "Nord", "Nord", palette{bg: "2e3440", fg: "d8dee9", border: "4c566a", accent: "88c0d0", surface: "3b4252", muted: "4c566a", red: "bf616a", green: "a3be8c", yellow: "ebcb8b", link: "81a1c1", quote: "8fbcbb", tag: "b48ead", match: "d08770", cursor: "d8dee9"})
)

// Rosé Pine — https://github.com/rose-pine/neovim (MIT).
var (
	// RosePine is the Rosé Pine Main palette.
	RosePine = collectionTheme("rose-pine", "Rosé Pine", "Main", palette{bg: "191724", fg: "e0def4", border: "403d52", accent: "c4a7e7", surface: "1f1d2e", muted: "6e6a86", red: "eb6f92", green: "9ccfd8", yellow: "f6c177", link: "9ccfd8", quote: "908caa", tag: "ebbcba", match: "f6c177", cursor: "e0def4"})
	// RosePineMoon is the Rosé Pine Moon palette.
	RosePineMoon = collectionTheme("rose-pine-moon", "Rosé Pine", "Moon", palette{bg: "232136", fg: "e0def4", border: "44415a", accent: "c4a7e7", surface: "2a273f", muted: "6e6a86", red: "eb6f92", green: "9ccfd8", yellow: "f6c177", link: "9ccfd8", quote: "908caa", tag: "ea9a97", match: "f6c177", cursor: "e0def4"})
	// RosePineDawn is the Rosé Pine Dawn palette.
	RosePineDawn = collectionTheme("rose-pine-dawn", "Rosé Pine", "Dawn", palette{bg: "faf4ed", fg: "464261", border: "cecacd", accent: "907aa9", surface: "fffaf3", muted: "9893a5", red: "b4637a", green: "56949f", yellow: "ea9d34", link: "56949f", quote: "797593", tag: "d7827e", match: "ea9d34", cursor: "464261"})
)

// Kanagawa — https://github.com/rebelot/kanagawa.nvim (MIT).
var (
	// KanagawaWave is the Kanagawa Wave palette.
	KanagawaWave = collectionTheme("kanagawa-wave", "Kanagawa", "Wave", palette{bg: "1f1f28", fg: "dcd7ba", border: "54546d", accent: "7e9cd8", surface: "2a2a37", muted: "727169", red: "e46876", green: "98bb6c", yellow: "e6c384", link: "7fb4ca", quote: "7aa89f", tag: "957fb8", match: "ffa066", cursor: "c8c093"})
	// KanagawaDragon is the Kanagawa Dragon palette.
	KanagawaDragon = collectionTheme("kanagawa-dragon", "Kanagawa", "Dragon", palette{bg: "181616", fg: "c5c9c5", border: "625e5a", accent: "8ba4b0", surface: "282727", muted: "737c73", red: "c4746e", green: "87a987", yellow: "c4b28a", link: "949fb5", quote: "8ea4a2", tag: "a292a3", match: "b6927b", cursor: "c8c093"})
	// KanagawaLotus is the Kanagawa Lotus palette.
	KanagawaLotus = collectionTheme("kanagawa-lotus", "Kanagawa", "Lotus", palette{bg: "f2ecbc", fg: "545464", border: "8a8980", accent: "4d699b", surface: "e7dba0", muted: "716e61", red: "c84053", green: "6f894e", yellow: "77713f", link: "4e8ca2", quote: "597b75", tag: "766b90", match: "cc6d00", cursor: "43436c"})
)

// Solarized — https://github.com/altercation/solarized (MIT).
var (
	// SolarizedDark is the Solarized Dark palette.
	SolarizedDark = collectionTheme("solarized-dark", "Solarized", "Dark", palette{bg: "002b36", fg: "93a1a1", border: "586e75", accent: "268bd2", surface: "073642", muted: "586e75", red: "dc322f", green: "859900", yellow: "b58900", link: "268bd2", quote: "2aa198", tag: "6c71c4", match: "cb4b16", cursor: "93a1a1"})
	// SolarizedLight is the Solarized Light palette.
	SolarizedLight = collectionTheme("solarized-light", "Solarized", "Light", palette{bg: "fdf6e3", fg: "586e75", border: "93a1a1", accent: "268bd2", surface: "eee8d5", muted: "93a1a1", red: "dc322f", green: "859900", yellow: "b58900", link: "268bd2", quote: "2aa198", tag: "6c71c4", match: "cb4b16", cursor: "586e75"})
)

// Everforest — https://github.com/sainnhe/everforest (MIT).
var (
	// EverforestDark is the Everforest Dark palette.
	EverforestDark = collectionTheme("everforest-dark", "Everforest", "Dark", palette{bg: "2d353b", fg: "d3c6aa", border: "7a8478", accent: "7fbbb3", surface: "343f44", muted: "859289", red: "e67e80", green: "a7c080", yellow: "dbbc7f", link: "7fbbb3", quote: "83c092", tag: "d699b6", match: "e69875", cursor: "d3c6aa"})
	// EverforestLight is the Everforest Light palette.
	EverforestLight = collectionTheme("everforest-light", "Everforest", "Light", palette{bg: "fdf6e3", fg: "5c6a72", border: "a6b0a0", accent: "3a94c5", surface: "f4f0d9", muted: "939f91", red: "f85552", green: "8da101", yellow: "dfa000", link: "3a94c5", quote: "35a77c", tag: "df69ba", match: "f57d26", cursor: "5c6a72"})
)

// Nightfox — https://github.com/EdenEast/nightfox.nvim (MIT).
var (
	// Nightfox is the Nightfox Nightfox palette.
	Nightfox = collectionTheme("nightfox", "Nightfox", "Nightfox", palette{bg: "192330", fg: "cdcecf", border: "39506d", accent: "719cd6", surface: "131a24", muted: "738091", red: "c94f6d", green: "81b29a", yellow: "dbc074", link: "63cdcf", quote: "81b29a", tag: "9d79d6", match: "f4a261", cursor: "cdcecf"})
	// Dayfox is the Nightfox Dayfox palette.
	Dayfox = collectionTheme("dayfox", "Nightfox", "Dayfox", palette{bg: "f6f2ee", fg: "3d2b5a", border: "aab0ad", accent: "2848a9", surface: "e4dcd4", muted: "837a72", red: "a5222f", green: "396847", yellow: "ac5402", link: "287980", quote: "396847", tag: "6e33ce", match: "955f61", cursor: "3d2b5a"})
	// Dawnfox is the Nightfox Dawnfox palette.
	Dawnfox = collectionTheme("dawnfox", "Nightfox", "Dawnfox", palette{bg: "faf4ed", fg: "575279", border: "bdbfc9", accent: "286983", surface: "ebe5df", muted: "9893a5", red: "b4637a", green: "618774", yellow: "ea9d34", link: "56949f", quote: "618774", tag: "907aa9", match: "d7827e", cursor: "575279"})
	// Duskfox is the Nightfox Duskfox palette.
	Duskfox = collectionTheme("duskfox", "Nightfox", "Duskfox", palette{bg: "232136", fg: "e0def4", border: "4b4673", accent: "569fba", surface: "191726", muted: "817c9c", red: "eb6f92", green: "a3be8c", yellow: "f6c177", link: "9ccfd8", quote: "a3be8c", tag: "c4a7e7", match: "ea9a97", cursor: "e0def4"})
	// Nordfox is the Nightfox Nordfox palette.
	Nordfox = collectionTheme("nordfox", "Nightfox", "Nordfox", palette{bg: "2e3440", fg: "cdcecf", border: "5a657d", accent: "81a1c1", surface: "232831", muted: "60728a", red: "bf616a", green: "a3be8c", yellow: "ebcb8b", link: "88c0d0", quote: "a3be8c", tag: "b48ead", match: "c9826b", cursor: "cdcecf"})
	// Terafox is the Nightfox Terafox palette.
	Terafox = collectionTheme("terafox", "Nightfox", "Terafox", palette{bg: "152528", fg: "e6eaea", border: "2d4f56", accent: "5a93aa", surface: "0f1c1e", muted: "6d7f8b", red: "e85c51", green: "7aa4a1", yellow: "fda47f", link: "a1cdd8", quote: "7aa4a1", tag: "ad5c7c", match: "ff8349", cursor: "e6eaea"})
	// Carbonfox is the Nightfox Carbonfox palette.
	Carbonfox = collectionTheme("carbonfox", "Nightfox", "Carbonfox", palette{bg: "161616", fg: "f2f4f8", border: "525253", accent: "78a9ff", surface: "2a2a2a", muted: "6e6f70", red: "ee5396", green: "25be6a", yellow: "08bdba", link: "33b1ff", quote: "3ddbd9", tag: "be95ff", match: "ff7eb6", cursor: "f2f4f8"})
)

// Ayu — https://github.com/ayu-theme/vscode-ayu (MIT).
var (
	// AyuDark is the Ayu Dark palette.
	AyuDark = collectionTheme("ayu-dark", "Ayu", "Dark", palette{bg: "10141c", fg: "bfbdb6", border: "5a6378", accent: "e6b450", surface: "0d1017", muted: "5a6378", red: "f06b73", green: "70bf56", yellow: "e6b450", link: "4fbfff", quote: "93e2c8", tag: "d0a1ff", match: "fdb04c", cursor: "bfbdb6"})
	// AyuMirage is the Ayu Mirage palette.
	AyuMirage = collectionTheme("ayu-mirage", "Ayu", "Mirage", palette{bg: "242936", fg: "cccac2", border: "707a8c", accent: "ffcc66", surface: "1f2430", muted: "707a8c", red: "f28273", green: "87d96c", yellow: "ffcc66", link: "6acdff", quote: "93e2c8", tag: "ddbbff", match: "fcca60", cursor: "cccac2"})
	// AyuLight is the Ayu Light palette.
	AyuLight = collectionTheme("ayu-light", "Ayu", "Light", palette{bg: "fcfcfc", fg: "5c6166", border: "828e9f", accent: "f29718", surface: "ebeef0", muted: "828e9f", red: "f06b6c", green: "6cbf43", yellow: "e7a100", link: "21a1e2", quote: "4abc96", tag: "a176cb", match: "f29718", cursor: "5c6166"})
)

// Palenight — https://github.com/whizkydee/vscode-palenight-theme (MIT).
var (
	// Palenight is the Palenight Palenight palette.
	Palenight = collectionTheme("palenight", "Palenight", "Palenight", palette{bg: "292d3e", fg: "bfc7d5", border: "676e95", accent: "82aaff", surface: "282c3d", muted: "676e95", red: "ff5572", green: "a9c77d", yellow: "ffcb6b", link: "89ddff", quote: "89ddff", tag: "c792ea", match: "f78c6c", cursor: "bfc7d5"})
)

// Night Owl — https://github.com/sdras/night-owl-vscode-theme (MIT).
var (
	// NightOwl is the Night Owl Night Owl palette.
	NightOwl = collectionTheme("night-owl", "Night Owl", "Night Owl", palette{bg: "011627", fg: "d6deeb", border: "5f7e97", accent: "82aaff", surface: "0b2942", muted: "637777", red: "ef5350", green: "22da6e", yellow: "ecc48d", link: "82aaff", quote: "21c7a8", tag: "c792ea", match: "f78c6c", cursor: "80a4c2"})
	// NightOwlLight is the Night Owl Light palette.
	NightOwlLight = collectionTheme("night-owl-light", "Night Owl", "Light", palette{bg: "fbfbfb", fg: "403f53", border: "93a1a1", accent: "288ed7", surface: "f0f0f0", muted: "989fb1", red: "de3d3b", green: "08916a", yellow: "e0af02", link: "288ed7", quote: "2aa298", tag: "d6438a", match: "daaa01", cursor: "403f53"})
)

// SynthWave '84 — https://github.com/robb0wen/synthwave-vscode (MIT).
var (
	// Synthwave84 is the SynthWave '84 SynthWave '84 palette.
	Synthwave84 = collectionTheme("synthwave-84", "SynthWave '84", "SynthWave '84", palette{bg: "262335", fg: "ffffff", border: "495495", accent: "ff7edb", surface: "241b2f", muted: "848bbd", red: "fe4450", green: "72f1b8", yellow: "fede5d", link: "36f9f6", quote: "03edf9", tag: "b893ce", match: "f97e72", cursor: "ffffff"})
)

// Shades of Purple — https://github.com/ahmadawais/shades-of-purple-vscode (MIT).
var (
	// ShadesOfPurple is the Shades of Purple Shades of Purple palette.
	ShadesOfPurple = collectionTheme("shades-of-purple", "Shades of Purple", "Shades of Purple", palette{bg: "2d2b55", fg: "ffffff", border: "1e1e3f", accent: "b362ff", surface: "222244", muted: "a599e9", red: "ec3a37", green: "3ad900", yellow: "fad000", link: "b362ff", quote: "80fcff", tag: "ff2c70", match: "ff7300", cursor: "fad000"})
)

// Oxocarbon — https://github.com/nyoom-engineering/oxocarbon.nvim (MIT).
var (
	// OxocarbonDark is the Oxocarbon Dark palette.
	OxocarbonDark = collectionTheme("oxocarbon-dark", "Oxocarbon", "Dark", palette{bg: "161616", fg: "f2f2f2", border: "525252", accent: "78a9ff", surface: "262626", muted: "878d96", red: "ee5396", green: "42be65", yellow: "ff7eb6", link: "33b1ff", quote: "08bdba", tag: "be95ff", match: "3ddbd9", cursor: "f2f2f2"})
	// OxocarbonLight is the Oxocarbon Light palette.
	OxocarbonLight = collectionTheme("oxocarbon-light", "Oxocarbon", "Light", palette{bg: "ffffff", fg: "525252", border: "90a4ae", accent: "0f62fe", surface: "f2f2f2", muted: "90a4ae", red: "ee5396", green: "42be65", yellow: "ff6f00", link: "0f62fe", quote: "08bdba", tag: "673ab7", match: "ffab91", cursor: "525252"})
)

// Cyberdream — https://github.com/scottmckendry/cyberdream.nvim (MIT).
var (
	// Cyberdream is the Cyberdream Dark palette.
	Cyberdream = collectionTheme("cyberdream", "Cyberdream", "Dark", palette{bg: "16181a", fg: "ffffff", border: "3c4048", accent: "5ea1ff", surface: "1e2124", muted: "7b8496", red: "ff6e5e", green: "5eff6c", yellow: "f1ff5e", link: "5ef1ff", quote: "7b8496", tag: "ff5ef1", match: "ffbd5e", cursor: "ffffff"})
	// CyberdreamLight is the Cyberdream Light palette.
	CyberdreamLight = collectionTheme("cyberdream-light", "Cyberdream", "Light", palette{bg: "ffffff", fg: "16181a", border: "acacac", accent: "0057d1", surface: "eaeaea", muted: "7b8496", red: "d11500", green: "008b0c", yellow: "997b00", link: "008c99", quote: "7b8496", tag: "d100bf", match: "d17c00", cursor: "16181a"})
)

// Retro and original palettes.
var (
	// MagentaGeode is an original dark mineral palette: deep plum with polished
	// magenta, fuchsia and violet highlights.
	MagentaGeode = Theme{
		Name: "magenta-geode", Family: "Magenta Geode", Bg: "#47003c", Fg: "#f3b0dc", Border: "#aa4d84", BorderFocus: "#c83fa9",
		Selected: "#c83fa9", Unread: "#f3b0dc", Dimmed: "#77176e", StatusBar: "#77176e",
		StatusFg: "#f3b0dc", Error: "#ff7062", Overlay: "#77176e", OverlayBorder: "#c83fa9",
		Success: "#6ee7b7", Warning: "#f9c74f", Link: "#c9a0ff", Quote: "#d98cc2",
		Match: "#ffb86b", Tag: "#e07bff", Cursor: "#f3b0dc",
	}
	// CoralSunset is an original dark coastal-sunset palette: coral, salmon,
	// peach and rose-gold over a muted dusk purple.
	CoralSunset = Theme{
		Name: "coral-sunset", Family: "Coral Sunset", Bg: "#444154", Fg: "#fec9c1", Border: "#fc8b79", BorderFocus: "#ff7062",
		Selected: "#ff7062", Unread: "#fec9c1", Dimmed: "#7a637f", StatusBar: "#7a637f",
		StatusFg: Legible("#fec9c1", "#7a637f", 4.5), Error: "#ff7062", Overlay: "#7a637f", OverlayBorder: "#ff7062",
		Success: "#8fd9a8", Warning: "#ffd08a", Link: "#8fd3ff", Quote: "#e8a8a0",
		Match: "#ffd08a", Tag: "#ff9eb5", Cursor: "#fec9c1",
	}
	// LavenderFieldsForever is an original gentle lilac palette with heather
	// purple and soft violet accents over readable neutral text.
	LavenderFieldsForever = Theme{
		Name: "lavender-fields-forever", Family: "Lavender Fields Forever", Bg: "#382d72", Fg: "#e5ccf4", Border: "#b7c2c6",
		BorderFocus: "#a080e1", Selected: "#a080e1", Unread: "#e5ccf4", Dimmed: "#5c509c",
		StatusBar: "#5c509c", StatusFg: "#e5ccf4", Error: "#ff7062", Overlay: "#5c509c",
		OverlayBorder: "#a080e1",
		Success:       "#9be8c0", Warning: "#f5d98a", Link: "#c5b3ff", Quote: "#b7a8e6",
		Match: "#f5d98a", Tag: "#d9a8f5", Cursor: "#e5ccf4",
	}
	// VT100 is a classic green phosphor palette on black, with dim and bright
	// green levels. It is an aesthetic preset, not a CRT effect.
	VT100 = Theme{
		Name: ThemeNameVT100, Family: "VT100", Bg: "#000000", Fg: "#33ff33", Border: "#145214", BorderFocus: "#00ff00",
		Selected: "#00ff00", Unread: "#66ff66", Dimmed: "#3dcc3d", StatusBar: "#001a00",
		StatusFg: "#33ff33", Error: "#ff6b6b", Overlay: "#001200", OverlayBorder: "#00ff00",
		Success: "#66ff66", Warning: "#d6ff5c", Link: "#66ff66", Quote: "#2fb82f",
		Match: "#00ff00", Tag: "#99ff99", Cursor: "#33ff33",
	}
	// VT52 is an amber-monitor-inspired palette: warm golden text on charcoal
	// black with muted bronze accents and ASCII presentation. It is an aesthetic
	// preset, not a claim about the colors of any historical VT52 display.
	VT52 = Theme{
		Name: ThemeNameVT52, Family: "VT52", Bg: "#14110d", Fg: "#ffcc66", Border: "#6b4e14", BorderFocus: "#ffb020",
		Selected: "#ffb020", Unread: "#ffe6a8", Dimmed: "#a67c2e", StatusBar: "#1f1810",
		StatusFg: "#ffcc66", Error: "#ff6666", Overlay: "#1a140d", OverlayBorder: "#ffb020",
		Success: "#ffe6a8", Warning: "#ffb020", Link: "#ffe6a8", Quote: "#a67c2e",
		Match: "#ffb020", Tag: "#cc9a3c", Cursor: "#ffcc66",
	}
)

// BuiltinThemes contains the palettes supplied by the library, grouped by
// family. The first entry is the default for unknown names.
var BuiltinThemes = []Theme{
	CatppuccinMocha,
	CatppuccinMacchiato,
	CatppuccinFrappe,
	CatppuccinLatte,
	OneDark,
	OneDarkDarker,
	TokyoNight,
	TokyoNightStorm,
	TokyoNightMoon,
	TokyoNightDay,
	Dracula,
	DraculaAlucard,
	GithubDark,
	GithubDarkDimmed,
	GithubLight,
	GruvboxDark,
	GruvboxLight,
	Nord,
	RosePine,
	RosePineMoon,
	RosePineDawn,
	KanagawaWave,
	KanagawaDragon,
	KanagawaLotus,
	SolarizedDark,
	SolarizedLight,
	EverforestDark,
	EverforestLight,
	Nightfox,
	Dayfox,
	Dawnfox,
	Duskfox,
	Nordfox,
	Terafox,
	Carbonfox,
	AyuDark,
	AyuMirage,
	AyuLight,
	Palenight,
	NightOwl,
	NightOwlLight,
	Synthwave84,
	ShadesOfPurple,
	OxocarbonDark,
	OxocarbonLight,
	Cyberdream,
	CyberdreamLight,
	MagentaGeode, CoralSunset, LavenderFieldsForever, VT100, VT52,
}
