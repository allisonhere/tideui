package tideui

// ResponsivePolicy interprets panels' generalized fallback rules and role
// defaults to produce a layout that fits a given width. It is stateless apart
// from the caller-owned threshold memory, which supplies hysteresis so a
// workspace does not flicker between layouts around a breakpoint.
type ResponsivePolicy struct {
	// Enabled turns semantic reflow on. When false the tree is returned as-is.
	Enabled bool
	// Hysteresis is the extra width required to leave a collapsed state once
	// entered, in columns. Zero uses 2.
	Hysteresis int
}

type responsiveAction struct {
	id       string
	mode     FallbackMode
	with     string
	target   string
	below    int
	explicit bool
}

// Apply returns a reflowed copy of root for width, along with the panels it
// hid and the panels it reduced to a collapsed header. The state map records
// the threshold currently active per panel and is updated in place.
func (p ResponsivePolicy) Apply(root LayoutNode, width int, panels map[string]*Panel, order []string, state map[string]int) (LayoutNode, map[string]bool, map[string]bool) {
	hidden := map[string]bool{}
	collapsed := map[string]bool{}
	if root == nil || width <= 0 || len(panels) == 0 {
		return root, hidden, collapsed
	}
	hysteresis := p.Hysteresis
	if hysteresis <= 0 {
		hysteresis = 2
	}
	primary := primaryPanelID(panels, order)

	var actions []responsiveAction
	for _, id := range order {
		panel := panels[id]
		if panel == nil {
			continue
		}
		action, ok := selectFallback(panel, width, primary, state, hysteresis)
		if !ok {
			delete(state, id)
			continue
		}
		state[id] = action.below
		actions = append(actions, action)
	}

	tree := CloneLayout(root)
	// Hides first, so later moves never target a removed panel.
	for _, action := range actions {
		if action.mode == FallbackHide {
			if panel := panels[action.id]; panel != nil && !panel.CanHide() {
				continue
			}
			if next, removed := RemovePanel(tree, action.id); removed {
				tree = next
				hidden[action.id] = true
			}
		}
	}
	for _, action := range actions {
		if hidden[action.id] || !LayoutContainsPanel(tree, action.id) {
			continue
		}
		switch action.mode {
		case FallbackStack:
			if action.with == "" || action.with == action.id {
				continue
			}
			if !LayoutContainsPanel(tree, action.with) {
				continue
			}
			tree = MovePanel(tree, action.id, action.with, DockCenter)
		case FallbackMoveBelow:
			if action.target == "" || action.target == action.id {
				continue
			}
			if !LayoutContainsPanel(tree, action.target) {
				continue
			}
			tree = MovePanel(tree, action.id, action.target, DockBelow)
		case FallbackMoveRight:
			if action.target == "" || action.target == action.id {
				continue
			}
			if !LayoutContainsPanel(tree, action.target) {
				continue
			}
			tree = MovePanel(tree, action.id, action.target, DockRight)
		case FallbackCollapse:
			if panel := panels[action.id]; panel != nil && !panel.CanHide() {
				continue
			}
			collapsed[action.id] = true
		}
	}
	return NormalizeLayout(tree), hidden, collapsed
}

// selectFallback chooses the most severe applicable rule for a panel. Explicit
// rules win over role defaults; among applicable rules the narrowest threshold
// (the most severe) wins. Hysteresis keeps the previously chosen threshold
// active until the workspace widens past it.
func selectFallback(panel *Panel, width int, primary string, state map[string]int, hysteresis int) (responsiveAction, bool) {
	active := state[panel.id]
	match := func(below int) bool {
		if width < below {
			return true
		}
		return active == below && width < below+hysteresis
	}

	best := responsiveAction{}
	found := false
	for _, fb := range panel.fallbacks {
		if fb.Below <= 0 || !match(fb.Below) {
			continue
		}
		if !found || fb.Below < best.below {
			best = responsiveAction{id: panel.id, mode: fb.Mode, with: fb.With, target: fb.Target, below: fb.Below, explicit: true}
			found = true
		}
	}
	if found {
		return best, true
	}

	if def, ok := defaultFallback(panel, primary); ok && match(def.below) {
		return def, true
	}
	return responsiveAction{}, false
}

// defaultFallback derives a gentle rule from a panel's role for applications
// that do not declare their own ladders.
func defaultFallback(panel *Panel, primary string) (responsiveAction, bool) {
	action := responsiveAction{id: panel.id, explicit: false}
	switch panel.role {
	case RoleOptional:
		action.mode, action.below = FallbackHide, 80
	case RoleTelemetry:
		action.mode, action.below = FallbackHide, 70
	case RoleNavigation:
		action.mode, action.below = FallbackCollapse, 60
	case RoleSecondary:
		action.mode, action.below = FallbackCollapse, 45
	case RoleInspector:
		if primary == "" || primary == panel.id {
			return responsiveAction{}, false
		}
		action.mode, action.target, action.below = FallbackMoveBelow, primary, 100
	default:
		return responsiveAction{}, false
	}
	return action, true
}

func primaryPanelID(panels map[string]*Panel, order []string) string {
	for _, id := range order {
		if panel := panels[id]; panel != nil && panel.role == RolePrimary {
			return id
		}
	}
	return ""
}

// ResponsiveWidth returns the width at which a panel's most severe explicit
// rule triggers, or 0 when it has none. Used by tests and diagnostics.
func ResponsiveWidth(panel *Panel) int {
	best := 0
	for _, fb := range panel.fallbacks {
		if best == 0 || fb.Below < best {
			best = fb.Below
		}
	}
	return best
}
