package tideui

import "time"

// AnimationOptions configures the lightweight workspace animator.
type AnimationOptions struct {
	// Enabled turns interpolation on. Off by default: the workspace never
	// installs timers of its own, so applications opt in.
	Enabled bool
	// ReducedMotion disables transitions while leaving the animator addressable,
	// mirroring the accessibility preference of the same name.
	ReducedMotion bool
	// TickRate is the suggested interval between application ticks.
	TickRate time.Duration
}

// Animator interpolates a small set of named scalars toward targets. It is
// deliberately tiny and timer-free: the application drives it with Tick on
// whatever schedule it already uses, so animation never blocks input and can
// be switched off without touching widgets.
type Animator struct {
	enabled bool
	reduced bool
	rate    time.Duration
	values  map[string]float64
	targets map[string]float64
	step    float64
}

// NewAnimator creates an animator. Zero options disable it.
func NewAnimator(options AnimationOptions) *Animator {
	rate := options.TickRate
	if rate <= 0 {
		rate = 60 * time.Millisecond
	}
	return &Animator{
		enabled: options.Enabled && !options.ReducedMotion,
		reduced: options.ReducedMotion,
		rate:    rate,
		values:  map[string]float64{},
		targets: map[string]float64{},
		step:    0.2,
	}
}

// Enabled reports whether interpolation is on.
func (a *Animator) Enabled() bool { return a != nil && a.enabled }

// ReducedMotion reports whether reduced motion was requested.
func (a *Animator) ReducedMotion() bool { return a != nil && a.reduced }

// TickRate returns the suggested tick interval.
func (a *Animator) TickRate() time.Duration {
	if a == nil {
		return 0
	}
	return a.rate
}

// Set moves a named value toward target over subsequent ticks.
func (a *Animator) Set(key string, target float64) {
	if a == nil {
		return
	}
	if !a.enabled {
		a.values[key] = target
		a.targets[key] = target
		return
	}
	if _, ok := a.values[key]; !ok {
		a.values[key] = 0
	}
	a.targets[key] = target
}

// Value returns the current animated value, defaulting to target when motion
// is disabled.
func (a *Animator) Value(key string) float64 {
	if a == nil {
		return 0
	}
	if !a.enabled {
		return a.targets[key]
	}
	return a.values[key]
}

// Tick advances every animated value one step. It returns true while any value
// is still moving.
func (a *Animator) Tick() bool {
	if a == nil || !a.enabled {
		return false
	}
	active := false
	for key, target := range a.targets {
		current, ok := a.values[key]
		if !ok {
			a.values[key] = target
			continue
		}
		delta := target - current
		if abs(delta) < 0.01 {
			a.values[key] = target
			continue
		}
		a.values[key] = current + delta*a.step
		active = true
	}
	return active
}

// Ease returns an eased 0..1 progress for a value, for callers that want a
// smoother curve than the raw interpolation.
func (a *Animator) Ease(key string) float64 {
	v := clamp01(a.Value(key))
	return v * v * (3 - 2*v)
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
