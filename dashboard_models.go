package tideui

import (
	"strings"
	"time"
)

// Data models for the first-party dashboard widgets. They are deliberately
// plain values: renderers consume them, so a real provider can populate them
// later without changing any rendering code.

// ForecastPoint is one entry in an hourly or daily forecast.
type ForecastPoint struct {
	Label       string // "3PM", "Mon"
	Temperature int
	Condition   string
	RainChance  int
}

// WeatherKind is a coarse condition class used to pick a glyph and colour.
type WeatherKind int

const (
	WeatherUnknown WeatherKind = iota
	WeatherClear
	WeatherPartly
	WeatherCloudy
	WeatherFog
	WeatherRain
	WeatherSnow
	WeatherStorm
)

// WeatherKindFromCondition maps a free-form condition string to a coarse kind.
// It matches common words, so provider text and hand-written demo strings both
// resolve to the same small vocabulary.
func WeatherKindFromCondition(condition string) WeatherKind {
	c := strings.ToLower(condition)
	switch {
	case strings.Contains(c, "thunder") || strings.Contains(c, "storm") || strings.Contains(c, "lightning"):
		return WeatherStorm
	case strings.Contains(c, "snow") || strings.Contains(c, "sleet") || strings.Contains(c, "blizzard") || strings.Contains(c, "flurr"):
		return WeatherSnow
	case strings.Contains(c, "rain") || strings.Contains(c, "drizzle") || strings.Contains(c, "shower"):
		return WeatherRain
	case strings.Contains(c, "fog") || strings.Contains(c, "mist") || strings.Contains(c, "haze"):
		return WeatherFog
	case strings.Contains(c, "partly") || strings.Contains(c, "scattered"):
		return WeatherPartly
	case strings.Contains(c, "cloud") || strings.Contains(c, "overcast"):
		return WeatherCloudy
	case strings.Contains(c, "clear") || strings.Contains(c, "sun") || strings.Contains(c, "fair"):
		return WeatherClear
	default:
		return WeatherUnknown
	}
}

// Glyph returns a two-cell colour weather emoji for the kind. The glyphs are
// astral emoji with U+FE0F so the width table and the terminal agree on two
// cells; astral emoji are also rendered by the colour emoji font without the
// fallback the BMP symbols hit. Plain UI falls back to ASCII.
func (k WeatherKind) Glyph(plain bool) string {
	if plain {
		switch k {
		case WeatherClear:
			return "*"
		case WeatherPartly:
			return "o"
		case WeatherCloudy:
			return "~"
		case WeatherFog:
			return "="
		case WeatherRain:
			return "/"
		case WeatherSnow:
			return "+"
		case WeatherStorm:
			return "!"
		default:
			return "."
		}
	}
	switch k {
	case WeatherClear:
		return "\U0001F31E\uFE0F" // sun with face
	case WeatherPartly:
		return "\U0001F324\uFE0F" // sun behind small cloud
	case WeatherCloudy:
		return "\U0001F325\uFE0F" // sun behind large cloud
	case WeatherFog:
		return "\U0001F32B\uFE0F" // fog
	case WeatherRain:
		return "\U0001F327\uFE0F" // rain
	case WeatherSnow:
		return "\U0001F328\uFE0F" // snow
	case WeatherStorm:
		return "\U0001F329\uFE0F" // cloud with lightning
	default:
		return "·"
	}
}

// WeatherData backs the weather widget.
type WeatherData struct {
	Location     string
	Temperature  int
	Unit         string // "F" or "C"
	Condition    string
	Kind         WeatherKind // optional; derived from Condition when unknown
	FeelsLike    int
	HasFeelsLike bool
	High         int
	Low          int
	RainChance   int
	WindSpeed    int
	WindUnit     string
	Hourly       []ForecastPoint
	Daily        []ForecastPoint
	Updated      time.Time
}

// EffectiveKind returns the explicit Kind when set, otherwise the kind implied
// by the Condition text.
func (w WeatherData) EffectiveKind() WeatherKind {
	if w.Kind != WeatherUnknown {
		return w.Kind
	}
	return WeatherKindFromCondition(w.Condition)
}

// AgendaItem is one calendar event.
type AgendaItem struct {
	Title    string
	Start    time.Time
	End      time.Time
	Location string
	Category string
	Tone     Tone
	Done     bool
	// AllDay marks an event with no time of day. Its Start is midnight local,
	// so rendering it as "00:00" would be misleading.
	AllDay bool
}

// WorldClock is one city in the clock widget.
type WorldClock struct {
	City   string
	Time   time.Time
	Offset string // "+9", "-5"
}

// ClockData backs the clock widget. Hour24 selects 24-hour ("15:04") instead
// of 12-hour ("3:04 PM") formatting.
type ClockData struct {
	Local    time.Time
	Location string
	Hour24   bool
	Zones    []WorldClock
}

// SystemMetrics backs the system-health widget.
type SystemMetrics struct {
	CPUPercent    float64
	CPUSpark      []float64
	Cores         []float64
	MemoryPercent float64
	MemoryUsed    string
	MemoryTotal   string
	TemperatureC  int
	Load          [3]float64
	Uptime        time.Duration
	Processes     int
}

// GPUMetrics backs the GPU widget. Percentages are 0..100 and BusySpark is
// 0..1, matching SystemMetrics.
type GPUMetrics struct {
	Name         string // short driver name, e.g. "amdgpu"
	BusyPercent  float64
	BusySpark    []float64
	MemoryUsed   string
	MemoryTotal  string
	MemoryLabel  string // "VRAM", or "GPU mem" on an integrated GPU
	MemoryFrac   float64
	TemperatureC int
	PowerWatts   float64
	ClockMHz     int
	// Integrated marks a GPU whose memory is carved out of system RAM. Its
	// memory is near-full by construction, so the widget shows the figure
	// without a gauge rather than painting a permanently alarming bar.
	Integrated bool
}

// UpdatePackage is one pending package update.
type UpdatePackage struct {
	Name string
	From string
	To   string
}

// UpdateStatus backs the updates widget.
type UpdateStatus struct {
	Omarchy        string // installed version
	OmarchyPending string // version waiting, empty when current
	Repo           []UpdatePackage
	AUR            []UpdatePackage
	Checked        time.Time
	// Unavailable explains why a count is missing rather than zero, e.g.
	// "checkupdates not installed".
	Unavailable string
}

// Pending reports how many package updates are waiting.
func (u UpdateStatus) Pending() int { return len(u.Repo) + len(u.AUR) }

// NetworkMetrics backs the network widget.
type NetworkMetrics struct {
	Interface string
	Download  float64
	Upload    float64
	Unit      string
	DownSpark []float64
	UpSpark   []float64
	LAN       string
	WAN       string
}

// StorageMount is one mounted filesystem.
type StorageMount struct {
	Path        string
	UsedPercent float64
	Used        string
	Total       string
	Tone        Tone
}

// ServiceStatus is one container or generic service. Status carries the
// shared dashboard vocabulary; State/Tone are optional display overrides for
// providers that want a custom word or colour.
type ServiceStatus struct {
	Name   string
	Status StatusKind
	Age    string // short age, e.g. "3d", "2m", "--"
	Detail string
	Uptime string
	State  string
	Tone   Tone
}

// resolved returns the effective status kind, word, and tone.
func (s ServiceStatus) resolved() (StatusKind, string, Tone) {
	kind := s.Status
	label := s.State
	if label == "" {
		label = kind.Label()
	}
	tone := s.Tone
	if tone == ToneNeutral {
		tone = kind.Tone()
	}
	return kind, label, tone
}

// Headline is one news or RSS item.
type Headline struct {
	Title  string
	Source string
	Age    string
	// Link is the story's URL, so a panel can copy it or open it.
	Link string
	// Selected marks the story under the panel's cursor, so the renderer can
	// highlight it.
	Selected bool
	Unread   bool
	Tone     Tone
}

// Task is one task-list item.
type Task struct {
	Title string
	Done  bool
	Due   string
	Tags  []string
	Tone  Tone
}

// Note is one note.
type Note struct {
	Title  string
	Body   string
	Pinned bool
}

// RepoActivity is one repository's activity summary.
type RepoActivity struct {
	Name    string
	Branch  string
	Summary string
	Commits int // commits made today
	Changes int // uncommitted files
	// Ahead and Behind count commits relative to the tracking branch.
	// Unpushed work is the thing a dashboard is best placed to catch, so it
	// is tracked separately rather than folded into Summary.
	Ahead      int
	Behind     int
	Stashes    int
	NoUpstream bool   // no tracking branch configured, which is not "clean"
	State      string // "rebase" or "merge" when one is half-finished
	Tone       Tone
}

// Clean reports whether a repository has nothing outstanding at all.
func (r RepoActivity) Clean() bool {
	// No branch means this is not a working copy at all — a missing path or a
	// clone URL. Such an entry has nothing outstanding only because nothing
	// could be read, so it must not be ticked as good.
	if r.Branch == "" {
		return false
	}
	return r.Changes == 0 && r.Ahead == 0 && r.Stashes == 0 && r.State == "" && !r.NoUpstream
}

// MarketQuote is one watchlist entry.
type MarketQuote struct {
	Symbol    string
	Price     float64
	High      float64
	Low       float64
	ChangePct float64
	Currency  string
}
