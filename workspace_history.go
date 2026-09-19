package tideui

// workspaceSnapshot is one reversible layout state. It intentionally excludes
// transient modes (zoom, peek, arrange) and all application data, so undo and
// redo only ever walk back through structural layout changes.
type workspaceSnapshot struct {
	root         LayoutNode
	hidden       map[string]bool
	focus        string
	activePreset string
}

func (s workspaceSnapshot) clone() workspaceSnapshot {
	out := workspaceSnapshot{
		root:         CloneLayout(s.root),
		hidden:       cloneStringSet(s.hidden),
		focus:        s.focus,
		activePreset: s.activePreset,
	}
	return out
}

// LayoutHistory is a bounded undo/redo stack over layout snapshots.
type LayoutHistory struct {
	entries []workspaceSnapshot
	index   int
	limit   int
}

// NewLayoutHistory creates an empty history bounded to limit entries (at
// least 1). Zero uses 64.
func NewLayoutHistory(limit int) *LayoutHistory {
	if limit <= 0 {
		limit = 64
	}
	return &LayoutHistory{limit: limit, index: -1}
}

// Reset clears the history and records state as the initial entry.
func (h *LayoutHistory) Reset(state workspaceSnapshot) {
	h.entries = []workspaceSnapshot{state.clone()}
	h.index = 0
}

// Push records a new state, discarding any redo entries after the cursor.
// Pushing a state identical to the current one is ignored.
func (h *LayoutHistory) Push(state workspaceSnapshot) {
	if h.index >= 0 && sameSnapshot(h.entries[h.index], state) {
		return
	}
	h.entries = append(h.entries[:h.index+1], state.clone())
	h.index = len(h.entries) - 1
	if len(h.entries) > h.limit {
		drop := len(h.entries) - h.limit
		h.entries = append([]workspaceSnapshot(nil), h.entries[drop:]...)
		h.index -= drop
	}
}

// CanUndo reports whether an earlier state exists.
func (h *LayoutHistory) CanUndo() bool { return h.index > 0 }

// CanRedo reports whether a later state exists.
func (h *LayoutHistory) CanRedo() bool { return h.index >= 0 && h.index < len(h.entries)-1 }

// Undo moves the cursor back and returns the restored state.
func (h *LayoutHistory) Undo() (workspaceSnapshot, bool) {
	if !h.CanUndo() {
		return workspaceSnapshot{}, false
	}
	h.index--
	return h.entries[h.index].clone(), true
}

// Redo moves the cursor forward and returns the restored state.
func (h *LayoutHistory) Redo() (workspaceSnapshot, bool) {
	if !h.CanRedo() {
		return workspaceSnapshot{}, false
	}
	h.index++
	return h.entries[h.index].clone(), true
}

// Len returns the number of recorded states.
func (h *LayoutHistory) Len() int { return len(h.entries) }

func sameSnapshot(a, b workspaceSnapshot) bool {
	if a.focus != b.focus || a.activePreset != b.activePreset {
		return false
	}
	if len(a.hidden) != len(b.hidden) {
		return false
	}
	for id, hidden := range a.hidden {
		if b.hidden[id] != hidden {
			return false
		}
	}
	return LayoutString(a.root) == LayoutString(b.root)
}

func cloneStringSet(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
