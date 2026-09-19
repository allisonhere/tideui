package tideui

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// LayoutFormatVersion is the persisted layout schema version. It is bumped when
// the serialized shape changes so older or newer files can be migrated or
// rejected cleanly.
const LayoutFormatVersion = 1

// LayoutStore is the pluggable backing store for workspace persistence. The
// library never touches the filesystem on its own; applications supply a store
// when they want durable layouts.
type LayoutStore interface {
	Load(key string) ([]byte, error)
	Save(key string, data []byte) error
}

// MemoryStore is an in-memory LayoutStore, suitable for tests and ephemeral
// sessions.
type MemoryStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: map[string][]byte{}}
}

// Load returns the stored bytes for a key.
func (m *MemoryStore) Load(key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("tideui: no layout stored for %q", key)
	}
	return append([]byte(nil), data...), nil
}

// Save stores bytes for a key.
func (m *MemoryStore) Save(key string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = append([]byte(nil), data...)
	return nil
}

type persistedLayout struct {
	Version int            `json:"version"`
	Preset  string         `json:"preset,omitempty"`
	Root    *persistedNode `json:"root,omitempty"`
	Hidden  []string       `json:"hidden,omitempty"`
}

type persistedNode struct {
	Type        string          `json:"type"`
	Orientation string          `json:"orientation,omitempty"`
	ID          string          `json:"id,omitempty"`
	Panels      []string        `json:"panels,omitempty"`
	Active      int             `json:"active,omitempty"`
	Weight      float64         `json:"weight,omitempty"`
	Children    []persistedNode `json:"children,omitempty"`
}

// Persist writes the current layout to the configured store. Zoom, peek, and
// other transient modes are deliberately omitted.
func (ws *Workspace) Persist() error {
	if ws.store == nil || ws.persistenceID == "" {
		return ErrNoStore
	}
	payload := ws.persistedState()
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return ws.store.Save(ws.persistenceID, data)
}

// Restore loads a persisted layout. Invalid or empty layouts leave the current
// state untouched and return an error. Unknown panels (for example after an app
// upgrade) are dropped, and the remaining layout is used when it is still
// valid.
func (ws *Workspace) Restore() error {
	if ws.store == nil || ws.persistenceID == "" {
		return ErrNoStore
	}
	ws.restoreTried = true
	data, err := ws.store.Load(ws.persistenceID)
	if err != nil {
		return err
	}
	var payload persistedLayout
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("tideui: corrupt layout: %w", err)
	}
	if payload.Version <= 0 || payload.Version > LayoutFormatVersion {
		return fmt.Errorf("tideui: unsupported layout version %d", payload.Version)
	}
	root := ws.decodeNode(payload.Root)
	if root == nil {
		return fmt.Errorf("tideui: layout has no usable panels")
	}
	ws.root = SanitizeLayout(root)
	ws.explicitRoot = true
	ws.hidden = map[string]bool{}
	for _, id := range payload.Hidden {
		if _, ok := ws.panels[id]; !ok {
			continue
		}
		if panel := ws.panels[id]; panel != nil && !panel.CanHide() {
			continue
		}
		ws.hidden[id] = true
	}
	for _, panel := range ws.panels {
		panel.hidden = ws.hidden[panel.id]
	}
	if payload.Preset != "" {
		if _, ok := ws.presets[payload.Preset]; ok {
			ws.activePreset = payload.Preset
		}
	}
	ws.ensureFocus()
	ws.history.Reset(ws.snapshot())
	return nil
}

// PersistedJSON returns the serialized layout without writing it, for callers
// that manage their own storage.
func (ws *Workspace) PersistedJSON() ([]byte, error) {
	return json.Marshal(ws.persistedState())
}

// RestoreJSON replaces the layout from serialized bytes.
func (ws *Workspace) RestoreJSON(data []byte) error {
	var payload persistedLayout
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if payload.Version <= 0 || payload.Version > LayoutFormatVersion {
		return fmt.Errorf("tideui: unsupported layout version %d", payload.Version)
	}
	root := ws.decodeNode(payload.Root)
	if root == nil {
		return fmt.Errorf("tideui: layout has no usable panels")
	}
	ws.root = SanitizeLayout(root)
	ws.explicitRoot = true
	for _, panel := range ws.panels {
		panel.hidden = false
	}
	ws.hidden = map[string]bool{}
	for _, id := range payload.Hidden {
		panel, ok := ws.panels[id]
		if !ok || !panel.CanHide() {
			continue
		}
		ws.hidden[id] = true
		panel.hidden = true
	}
	ws.ensureFocus()
	ws.commit()
	return nil
}

func (ws *Workspace) persistedState() persistedLayout {
	var hidden []string
	for _, id := range ws.order {
		if ws.isHidden(id) {
			hidden = append(hidden, id)
		}
	}
	sort.Strings(hidden)
	return persistedLayout{
		Version: LayoutFormatVersion,
		Preset:  ws.activePreset,
		Root:    encodeNode(ws.ensureRoot()),
		Hidden:  hidden,
	}
}

func encodeNode(node LayoutNode) *persistedNode {
	switch n := node.(type) {
	case *LeafNode:
		return &persistedNode{Type: "leaf", ID: n.ID, Weight: n.Weight}
	case *TabStackNode:
		return &persistedNode{Type: "tabs", Panels: append([]string(nil), n.Panels...), Active: n.Active, Weight: n.Weight}
	case *SplitNode:
		out := &persistedNode{Type: "split", Weight: n.Weight}
		if n.Orientation == SplitHorizontal {
			out.Orientation = "horizontal"
		} else {
			out.Orientation = "vertical"
		}
		for _, child := range n.Children {
			if encoded := encodeNode(child); encoded != nil {
				out.Children = append(out.Children, *encoded)
			}
		}
		return out
	}
	return nil
}

// decodeNode rebuilds a tree, dropping unknown panels, duplicates, and empty
// nodes. It returns nil when nothing usable remains.
func (ws *Workspace) decodeNode(node *persistedNode) LayoutNode {
	if node == nil {
		return nil
	}
	switch node.Type {
	case "leaf":
		if !ws.knownPanel(node.ID) {
			return nil
		}
		return Weighted(Leaf(node.ID), node.Weight)
	case "tabs":
		var panels []string
		seen := map[string]bool{}
		for _, id := range node.Panels {
			if !ws.knownPanel(id) || seen[id] {
				continue
			}
			seen[id] = true
			panels = append(panels, id)
		}
		if len(panels) == 0 {
			return nil
		}
		if len(panels) == 1 {
			return Leaf(panels[0])
		}
		return Weighted(&TabStackNode{Panels: panels, Active: node.Active}, node.Weight)
	case "split":
		var children []LayoutNode
		seen := map[string]bool{}
		for i := range node.Children {
			child := ws.decodeNode(&node.Children[i])
			if child == nil {
				continue
			}
			// Drop duplicate panel leaves that a malformed file might contain.
			duplicate := false
			for _, id := range LayoutPanelIDs(child) {
				if seen[id] {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			for _, id := range LayoutPanelIDs(child) {
				seen[id] = true
			}
			children = append(children, child)
		}
		if len(children) == 0 {
			return nil
		}
		orientation := SplitHorizontal
		if node.Orientation == "vertical" {
			orientation = SplitVertical
		}
		return Weighted(&SplitNode{Orientation: orientation, Children: children}, node.Weight)
	}
	return nil
}

func (ws *Workspace) knownPanel(id string) bool {
	if id == "" {
		return false
	}
	_, ok := ws.panels[id]
	return ok
}
