package tideui

import (
	"errors"
	"fmt"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Workspace owns a set of registered panels, a layout tree, and all the
// interaction state around them: focus, visibility, zoom, peek, arrange mode,
// pane resizing, presets, and undo/redo. It never talks to the terminal and
// never renders; applications drive it and hand the resolved view to a
// WorkspaceRenderer.
type Workspace struct {
	panels       map[string]*Panel
	order        []string
	root         LayoutNode
	defaultRoot  LayoutNode
	explicitRoot bool
	restoreTried bool

	focus FocusManager

	hidden    map[string]bool
	rspHidden map[string]bool
	collapsed map[string]bool

	zoomed string
	peeked string

	// entered is the pane holding the keyboard, if any. It is not the focus
	// (which panel the workspace is pointed at) and not the zoom (how much room
	// a panel gets): space enters a pane so its own keys work, esc leaves it,
	// and moving focus or tabbing away leaves it behind.
	entered string

	arrange        bool
	arrangeCursor  string
	resizeNotice   string
	resizeNoticeAt time.Time

	drag    *mouseDrag
	tabHits []tabHit

	width, height int
	solved        SolvedLayout
	solvedTree    LayoutNode

	hGap     int
	vGap     int
	adaptive bool
	policy   ResponsivePolicy
	rspState map[string]int

	history      *LayoutHistory
	presets      map[string]Preset
	presetOrder  []string
	activePreset string

	store         LayoutStore
	persistenceID string

	focusPres FocusPresentation
	anim      *Animator
	picker    *PanelPicker
	palette   *CommandPalette
}

// WorkspaceOption configures a Workspace at construction time.
type WorkspaceOption func(*Workspace)

// WithPersistence enables layout persistence under an application-scoped key.
// The default store is in-memory; pair with WithStore for durable storage.
func WithPersistence(id string) WorkspaceOption {
	return func(ws *Workspace) { ws.persistenceID = id }
}

// WithStore installs a layout store for persistence.
func WithStore(store LayoutStore) WorkspaceOption {
	return func(ws *Workspace) { ws.store = store }
}

// WithAdaptiveLayout enables semantic responsive reflow.
func WithAdaptiveLayout() WorkspaceOption {
	return func(ws *Workspace) {
		ws.adaptive = true
		ws.policy.Enabled = true
	}
}

// WithGap sets the blank cells left between regions on both axes.
func WithGap(gap int) WorkspaceOption {
	return func(ws *Workspace) {
		ws.hGap = max(0, gap)
		ws.vGap = max(0, gap)
	}
}

// WithGaps sets the horizontal (column) and vertical (row) gaps separately.
// A vertical gap of zero stacks panels flush, so vertically adjacent borders
// touch instead of leaving a blank row between them.
func WithGaps(horizontal, vertical int) WorkspaceOption {
	return func(ws *Workspace) {
		ws.hGap = max(0, horizontal)
		ws.vGap = max(0, vertical)
	}
}

// WithFocusPresentation overrides how focus is signalled.
func WithFocusPresentation(p FocusPresentation) WorkspaceOption {
	return func(ws *Workspace) { ws.focusPres = p }
}

// WithAnimation overrides animation behaviour.
func WithAnimation(options AnimationOptions) WorkspaceOption {
	return func(ws *Workspace) { ws.anim = NewAnimator(options) }
}

// WithHistoryLimit bounds the layout undo stack.
func WithHistoryLimit(limit int) WorkspaceOption {
	return func(ws *Workspace) { ws.history = NewLayoutHistory(limit) }
}

// WithPresets installs application-provided workspace presets.
func WithPresets(presets ...Preset) WorkspaceOption {
	return func(ws *Workspace) {
		for _, preset := range presets {
			ws.addPreset(preset)
		}
	}
}

// NewWorkspace creates an empty workspace. Register panels with Panel, then
// declare a layout with Layout; without an explicit layout an adaptive default
// is derived from panel roles.
func NewWorkspace(options ...WorkspaceOption) *Workspace {
	ws := &Workspace{
		panels:    map[string]*Panel{},
		hidden:    map[string]bool{},
		rspHidden: map[string]bool{},
		collapsed: map[string]bool{},
		rspState:  map[string]int{},
		hGap:      1,
		vGap:      1,
		policy:    ResponsivePolicy{Enabled: true, Hysteresis: 2},
		history:   NewLayoutHistory(64),
		presets:   map[string]Preset{},
		focusPres: DefaultFocusPresentation(),
		anim:      NewAnimator(AnimationOptions{}),
		picker:    NewPanelPicker(),
		palette:   NewCommandPalette(),
	}
	for _, option := range options {
		if option != nil {
			option(ws)
		}
	}
	if ws.persistenceID != "" && ws.store == nil {
		ws.store = NewMemoryStore()
	}
	ws.history.Reset(ws.snapshot())
	return ws
}

// --- Panel registration ---------------------------------------------------

// Panel registers a panel, or returns the existing one (updating its view) if
// the id is already registered. The returned panel is configured fluently.
func (ws *Workspace) Panel(id string, view PanelView) *Panel {
	if existing, ok := ws.panels[id]; ok {
		if view != nil {
			existing.view = view
		}
		return existing
	}
	panel := newPanel(id, view)
	ws.panels[id] = panel
	ws.order = append(ws.order, id)
	if panel.hidden {
		ws.hidden[id] = true
	}
	return panel
}

// Panels returns registered panels in registration order.
func (ws *Workspace) Panels() []*Panel {
	out := make([]*Panel, 0, len(ws.order))
	for _, id := range ws.order {
		if panel := ws.panels[id]; panel != nil {
			out = append(out, panel)
		}
	}
	return out
}

// Lookup returns a registered panel by id.
func (ws *Workspace) Lookup(id string) (*Panel, bool) {
	panel, ok := ws.panels[id]
	return panel, ok
}

// PanelIDs returns registered panel ids in registration order.
func (ws *Workspace) PanelIDs() []string {
	return append([]string(nil), ws.order...)
}

// --- Layout ---------------------------------------------------------------

// Layout replaces the layout tree. A missing or empty tree falls back to the
// role-derived default.
func (ws *Workspace) Layout(root LayoutNode) *Workspace {
	if root == nil || len(LayoutPanelIDs(root)) == 0 {
		ws.root = nil
		ws.defaultRoot = nil
		ws.explicitRoot = false
	} else {
		normalized := SanitizeLayout(root)
		if normalized == nil {
			ws.root = nil
			ws.defaultRoot = nil
			ws.explicitRoot = false
		} else {
			ws.root = normalized
			ws.defaultRoot = CloneLayout(normalized)
			ws.explicitRoot = true
		}
	}
	ws.ensureFocus()
	ws.commit()
	return ws
}

// RootLayout returns the current explicit tree, or nil when the default is in
// use.
func (ws *Workspace) RootLayout() LayoutNode {
	return ws.root
}

// Solve recomputes rectangles for a terminal size and caches the result for
// hit-testing. It is safe to call on every render and never panics, even at
// extremely small sizes.
func (ws *Workspace) Solve(width, height int) SolvedLayout {
	ws.width, ws.height = max(0, width), max(0, height)
	tree, respHidden, collapsed := ws.effectiveTree()
	ws.solvedTree = tree
	ws.rspHidden = respHidden
	ws.collapsed = collapsed
	if width <= 0 || height <= 0 {
		ws.solved = SolvedLayout{Width: max(0, width), Height: max(0, height), Rects: map[string]Rect{}}
		return ws.solved
	}
	if id := ws.zoomCandidate(); id != "" {
		rect := Rect{Width: width, Height: height}
		ws.solved = SolvedLayout{
			Width:   width,
			Height:  height,
			Rects:   map[string]Rect{id: rect},
			Regions: []SolvedRegion{{Rect: rect, PanelIDs: []string{id}}},
		}
		return ws.solved
	}
	solver := LayoutSolver{
		HGap:      ws.hGap,
		VGap:      ws.vGap,
		MinWidth:  ws.minWidthFor,
		MinHeight: ws.minHeightFor,
		Weight:    ws.weightFor,
	}
	ws.solved = solver.Solve(tree, width, height)
	ws.ensureFocusInSolved()
	return ws.solved
}

// Solved returns the most recent solved layout.
func (ws *Workspace) Solved() SolvedLayout { return ws.solved }

// SolvedTree returns the tree used for the most recent solve, after hidden
// panels were removed and responsive rules applied.
func (ws *Workspace) SolvedTree() LayoutNode { return ws.solvedTree }

func (ws *Workspace) effectiveTree() (LayoutNode, map[string]bool, map[string]bool) {
	root := ws.ensureRoot()
	tree := CloneLayout(root)
	for _, id := range ws.order {
		if ws.isHidden(id) {
			tree, _ = RemovePanel(tree, id)
		}
	}
	respHidden := map[string]bool{}
	collapsed := map[string]bool{}
	if ws.adaptive && ws.width > 0 {
		tree, respHidden, collapsed = ws.policy.Apply(tree, ws.width, ws.panels, ws.order, ws.rspState)
	}
	return tree, respHidden, collapsed
}

// ensureRoot returns the current tree, attempting a lazy restore of any
// persisted layout on first use, then falling back to the declared or
// role-derived default.
func (ws *Workspace) ensureRoot() LayoutNode {
	ws.maybeRestore()
	if ws.root != nil {
		return ws.root
	}
	if ws.defaultRoot != nil {
		ws.root = CloneLayout(ws.defaultRoot)
		return ws.root
	}
	ws.root = ws.defaultLayout()
	return ws.root
}

// maybeRestore loads a persisted layout once, after panels are registered.
// A missing or invalid stored layout leaves the declared default in place.
func (ws *Workspace) maybeRestore() {
	if ws.restoreTried || ws.store == nil || ws.persistenceID == "" {
		return
	}
	ws.restoreTried = true
	if err := ws.Restore(); err == nil {
		ws.history.Reset(ws.snapshot())
	}
}

// defaultLayout derives a sensible starting arrangement from panel roles:
// navigation on the left, primary in the middle, everything else on the right.
func (ws *Workspace) defaultLayout() LayoutNode {
	visible := make([]string, 0, len(ws.order))
	for _, id := range ws.order {
		if !ws.isHidden(id) {
			visible = append(visible, id)
		}
	}
	if len(visible) == 0 {
		return nil
	}
	var nav, primary, rest []string
	for _, id := range visible {
		switch ws.panels[id].role {
		case RoleNavigation:
			nav = append(nav, id)
		case RolePrimary:
			primary = append(primary, id)
		default:
			rest = append(rest, id)
		}
	}
	// Without an explicit primary, the highest-priority panel takes the
	// central, largest slot; priority then orders the remaining groups.
	if len(primary) == 0 {
		if center := ws.highestPriority(visible); center != "" {
			primary = []string{center}
			nav = removeString(nav, center)
			rest = removeString(rest, center)
		}
	}
	sortByPriority(ws, nav)
	sortByPriority(ws, primary)
	sortByPriority(ws, rest)
	var groups []LayoutNode
	if len(nav) > 0 {
		groups = append(groups, leaves(nav))
	}
	groups = append(groups, Weighted(leaves(primary), 2))
	if len(rest) > 0 {
		groups = append(groups, leaves(rest))
	}
	if len(groups) == 1 {
		return groups[0]
	}
	return HStack(groups...)
}

// highestPriority returns the id with the greatest resolved priority.
func (ws *Workspace) highestPriority(ids []string) string {
	best := ""
	bestPriority := 0
	for _, id := range ids {
		if priority := ws.panels[id].PriorityValue(); best == "" || priority > bestPriority {
			best = id
			bestPriority = priority
		}
	}
	return best
}

func removeString(list []string, value string) []string {
	out := list[:0]
	for _, v := range list {
		if v != value {
			out = append(out, v)
		}
	}
	return out
}

// sortByPriority orders ids by descending priority, preserving ties.
func sortByPriority(ws *Workspace, ids []string) {
	sort.SliceStable(ids, func(i, j int) bool {
		return ws.panels[ids[i]].PriorityValue() > ws.panels[ids[j]].PriorityValue()
	})
}

// leaves builds a vertical stack of single-panel leaves, or the single leaf.
func leaves(ids []string) LayoutNode {
	nodes := make([]LayoutNode, len(ids))
	for i, id := range ids {
		nodes[i] = Leaf(id)
	}
	if len(nodes) == 1 {
		return nodes[0]
	}
	return VStack(nodes...)
}

func (ws *Workspace) minWidthFor(id string) int {
	panel := ws.panels[id]
	if panel == nil {
		return 1
	}
	if ws.collapsed[id] {
		return 6
	}
	return max(1, panel.minWidth)
}

func (ws *Workspace) minHeightFor(id string) int {
	panel := ws.panels[id]
	if panel == nil {
		return 1
	}
	if ws.collapsed[id] {
		return 1
	}
	return max(1, panel.minHeight)
}

func (ws *Workspace) weightFor(node LayoutNode) float64 {
	if leaf, ok := node.(*LeafNode); ok {
		if panel := ws.panels[leaf.ID]; panel != nil && panel.grow > 0 {
			return panel.grow
		}
	}
	return 1
}

// --- Visibility -----------------------------------------------------------

// isHidden reports manual hidden state, including a panel configured hidden
// through its fluent Hide method.
func (ws *Workspace) isHidden(id string) bool {
	if ws.hidden[id] {
		return true
	}
	if panel := ws.panels[id]; panel != nil {
		return panel.hidden
	}
	return false
}

// Hidden reports whether a panel is manually hidden.
func (ws *Workspace) Hidden(id string) bool { return ws.isHidden(id) }

// IsVisible reports whether a panel is currently laid out, taking responsive
// hiding into account. Call after Solve.
func (ws *Workspace) IsVisible(id string) bool {
	return LayoutContainsPanel(ws.solvedTree, id)
}

// VisiblePanels returns the panels present in the last solved tree.
func (ws *Workspace) VisiblePanels() []string {
	return LayoutPanelIDs(ws.solvedTree)
}

// Hide hides a hideable panel and records it in history.
func (ws *Workspace) Hide(id string) bool {
	panel, ok := ws.panels[id]
	if !ok || !panel.CanHide() || ws.isHidden(id) {
		return false
	}
	ws.hidden[id] = true
	panel.hidden = true
	if ws.zoomed == id {
		ws.zoomed = ""
	}
	ws.ensureFocus()
	ws.commit()
	return true
}

// Show reveals a hidden panel and records it in history.
func (ws *Workspace) Show(id string) bool {
	panel, ok := ws.panels[id]
	if !ok || !ws.isHidden(id) {
		return false
	}
	delete(ws.hidden, id)
	panel.hidden = false
	if ws.root != nil && !LayoutContainsPanel(ws.root, id) {
		ws.root = insertPanelIntoTallStack(ws.root, id)
	}
	// Revealing a panel moves focus to it. You turned it on to use it, and
	// without this a panel that takes typing (a calculator) would receive
	// nothing while whatever was focused before kept the keyboard.
	if panel.focusable {
		ws.focus.Set(id)
	}
	ws.commit()
	return true
}

// RemovePanel unregisters a panel entirely: it leaves the panel list, the
// layout tree and the view state. Unlike Hide, it cannot be shown again, so it
// is for a panel that no longer exists - an uninstalled plugin.
func (ws *Workspace) RemovePanel(id string) bool {
	if _, ok := ws.panels[id]; !ok {
		return false
	}
	delete(ws.panels, id)
	delete(ws.hidden, id)
	for i, existing := range ws.order {
		if existing == id {
			ws.order = append(ws.order[:i], ws.order[i+1:]...)
			break
		}
	}
	if ws.root != nil {
		ws.root, _ = RemovePanel(ws.root, id)
	}
	if ws.solvedTree != nil {
		ws.solvedTree, _ = RemovePanel(ws.solvedTree, id)
	}
	if ws.zoomed == id {
		ws.zoomed = ""
	}
	if ws.peeked == id {
		ws.peeked = ""
	}
	ws.ensureFocus()
	ws.commit()
	return true
}

// TogglePanel flips a panel's manual visibility.
func (ws *Workspace) TogglePanel(id string) bool {
	if ws.isHidden(id) {
		return ws.Show(id)
	}
	return ws.Hide(id)
}

// insertPanelIntoTallStack appends a panel to the root split when one exists
// so a revealed panel always has a home without disturbing the rest of the
// tree; otherwise it wraps the tree in a new horizontal split.
func insertPanelIntoTallStack(root LayoutNode, id string) LayoutNode {
	if root == nil {
		return Leaf(id)
	}
	if split, ok := root.(*SplitNode); ok {
		split.Children = append(split.Children, Leaf(id))
		return NormalizeLayout(split)
	}
	return NormalizeLayout(HStack(root, Leaf(id)))
}

// --- Focus ----------------------------------------------------------------

// Focused returns the focused panel id, or "".
func (ws *Workspace) Focused() string { return ws.focus.Current() }

// Focus sets focus if the panel exists, is focusable, and is visible.
func (ws *Workspace) Focus(id string) bool {
	if !ws.canFocus(id) {
		return false
	}
	ws.setFocus(id)
	return true
}

// setFocus moves the focus and drops the entered pane: the keyboard belongs to
// where you are, so pointing the workspace somewhere else hands the keys back.
// Every way of moving focus comes through here, so tabbing away cannot leave a
// pane reading keys the reader meant for the workspace.
func (ws *Workspace) setFocus(id string) {
	if ws.entered != "" && ws.entered != id {
		ws.entered = ""
	}
	ws.focus.Set(id)
}

// CanFocus reports whether a panel may currently receive focus.
func (ws *Workspace) CanFocus(id string) bool { return ws.canFocus(id) }

func (ws *Workspace) canFocus(id string) bool {
	panel, ok := ws.panels[id]
	if !ok || !panel.focusable || ws.isHidden(id) || ws.rspHidden[id] {
		return false
	}
	if ws.solvedTree != nil && !LayoutContainsPanel(ws.solvedTree, id) {
		return false
	}
	return true
}

// FocusNext moves focus to the next visible focusable panel.
func (ws *Workspace) FocusNext() bool {
	order := ws.focusOrder()
	if len(order) == 0 {
		return false
	}
	ws.setFocus(ws.focus.Next(order))
	return true
}

// FocusPrev moves focus to the previous visible focusable panel.
func (ws *Workspace) FocusPrev() bool {
	order := ws.focusOrder()
	if len(order) == 0 {
		return false
	}
	ws.setFocus(ws.focus.Prev(order))
	return true
}

// FocusDirection moves focus to the nearest panel in a direction.
func (ws *Workspace) FocusDirection(dir Direction) bool {
	next := ws.focus.Directional(ws.solved, dir, ws.canFocus)
	if next == "" {
		return false
	}
	ws.setFocus(next)
	return true
}

func (ws *Workspace) focusOrder() []string {
	base := LayoutPanelIDs(ws.solvedTree)
	if len(base) == 0 {
		base = ws.order
	}
	out := make([]string, 0, len(base))
	for _, id := range base {
		if ws.canFocus(id) {
			out = append(out, id)
		}
	}
	return out
}

func (ws *Workspace) ensureFocus() {
	if ws.focus.Current() != "" && ws.canFocusLoose(ws.focus.Current()) {
		return
	}
	for _, id := range ws.order {
		if ws.canFocusLoose(id) {
			ws.focus.Set(id)
			return
		}
	}
	ws.focus.Set("")
}

// canFocusLoose ignores the solved tree, used before the first solve.
func (ws *Workspace) canFocusLoose(id string) bool {
	panel, ok := ws.panels[id]
	if !ok || !panel.focusable || ws.isHidden(id) {
		return false
	}
	if ws.root != nil && !LayoutContainsPanel(ws.root, id) {
		return false
	}
	return true
}

func (ws *Workspace) ensureFocusInSolved() {
	if ws.canFocus(ws.focus.Current()) {
		return
	}
	order := ws.focusOrder()
	if len(order) > 0 {
		ws.focus.Set(order[0])
		return
	}
	ws.focus.Set("")
}

// --- Zoom -----------------------------------------------------------------

func (ws *Workspace) zoomCandidate() string {
	if ws.zoomed == "" {
		return ""
	}
	panel, ok := ws.panels[ws.zoomed]
	if !ok || !panel.zoomable || ws.isHidden(ws.zoomed) {
		return ""
	}
	if !LayoutContainsPanel(ws.solvedTree, ws.zoomed) {
		return ""
	}
	return ws.zoomed
}

// Zoomed returns the temporarily maximized panel id, or "".
func (ws *Workspace) Zoomed() string { return ws.zoomCandidate() }

// Zoom maximizes a panel without altering the saved layout.
func (ws *Workspace) Zoom(id string) bool {
	panel, ok := ws.panels[id]
	if !ok || !panel.zoomable || ws.isHidden(id) {
		return false
	}
	ws.zoomed = id
	ws.setFocus(id)
	return true
}

// Unzoom restores the pre-zoom layout.
func (ws *Workspace) Unzoom() bool {
	if ws.zoomed == "" {
		return false
	}
	ws.zoomed = ""
	return true
}

// EnterPane gives the focused pane the keyboard: its own keys work - a list's
// cursor, a form's fields - while the tiled layout stays on screen, so a panel
// can be walked without zooming it. Any pane can be entered; one with nothing of
// its own to walk simply takes the keys it has, which keeps the gesture the same
// everywhere. LeavePane hands them back, and so does moving focus.
//
// Entering is not zooming. Zoom decides how much room a panel gets; entering
// decides who reads the keys, which is why a zoomed panel has the keyboard too.
func (ws *Workspace) EnterPane() bool {
	id := ws.focus.Current()
	if id == "" || ws.entered == id {
		return false
	}
	ws.entered = id
	return true
}

// LeavePane returns the keyboard to the workspace.
func (ws *Workspace) LeavePane() bool {
	if ws.entered == "" {
		return false
	}
	ws.entered = ""
	return true
}

// EnteredPane is the pane holding the keyboard, or "" when the workspace has it.
func (ws *Workspace) EnteredPane() string { return ws.entered }

// PaneHasKeyboard reports whether a pane reads the keys itself, because it was
// entered or because it owns the screen. One question with one answer: a panel
// that draws a cursor asks this rather than inferring it from the zoom.
func (ws *Workspace) PaneHasKeyboard(id string) bool {
	return id != "" && (ws.entered == id || ws.zoomCandidate() == id)
}

// ToggleZoom maximizes the focused panel, or restores the layout.
func (ws *Workspace) ToggleZoom() bool {
	if ws.zoomed != "" {
		return ws.Unzoom()
	}
	if ws.focus.Current() == "" {
		return false
	}
	return ws.Zoom(ws.focus.Current())
}

// --- Peek -----------------------------------------------------------------

// Peek temporarily shows a panel without changing the saved layout.
func (ws *Workspace) Peek(id string) bool {
	if _, ok := ws.panels[id]; !ok {
		return false
	}
	ws.peeked = id
	return true
}

// Unpeek dismisses the peeked panel.
func (ws *Workspace) Unpeek() bool {
	if ws.peeked == "" {
		return false
	}
	ws.peeked = ""
	return true
}

// Peeked returns the peeked panel id, or "".
func (ws *Workspace) Peeked() string { return ws.peeked }

// --- Arrange --------------------------------------------------------------

// Arranging reports whether arrange mode is active.
func (ws *Workspace) Arranging() bool { return ws.arrange }

// EnterArrange starts live panel rearrangement on the focused panel. Direction
// keys move the panel between neighbouring regions immediately, so the real
// layout is its own preview: the panel is shown where it will land, gaps close
// behind it, and every move is recorded in history. t folds the panel into a
// neighbouring tab stack; Escape leaves the mode.
func (ws *Workspace) EnterArrange() bool {
	if ws.focus.Current() == "" {
		return false
	}
	ws.arrange = true
	ws.arrangeCursor = ws.focus.Current()
	return true
}

// ExitArrange leaves arrange mode.
func (ws *Workspace) ExitArrange() {
	ws.arrange = false
	ws.arrangeCursor = ""
}

// ArrangeCursor returns the neighbour the last move docked against.
func (ws *Workspace) ArrangeCursor() string { return ws.arrangeCursor }

// ToggleArrange enters or leaves arrange mode.
func (ws *Workspace) ToggleArrange() bool {
	if ws.arrange {
		ws.ExitArrange()
		return true
	}
	return ws.EnterArrange()
}

// ArrangeMove moves the focused panel one region in a direction. The layout
// updates immediately and the move is recorded, so the moving panel is visible
// in its new position rather than behind a preview overlay.
func (ws *Workspace) ArrangeMove(dir Direction) bool {
	if !ws.arrange {
		return false
	}
	moving := ws.focus.Current()
	if moving == "" {
		return false
	}
	target := ws.focus.Directional(ws.solved, dir, func(id string) bool {
		return id != moving && LayoutContainsPanel(ws.solvedTree, id)
	})
	if target == "" || target == moving {
		return false
	}
	next := MovePanel(ws.ensureRoot(), moving, target, MovedDock(dir))
	if next == nil || !LayoutContainsPanel(next, moving) {
		return false
	}
	ws.root = next
	ws.explicitRoot = true
	ws.arrangeCursor = target
	ws.commit()
	ws.focus.Set(moving)
	if ws.width > 0 && ws.height > 0 {
		ws.Solve(ws.width, ws.height)
	}
	return true
}

// ArrangeDrop finishes arrange mode. Moves are applied as they happen, so
// dropping only leaves the mode.
func (ws *Workspace) ArrangeDrop() bool {
	if !ws.arrange {
		return false
	}
	ws.ExitArrange()
	return true
}

// ArrangeMerge folds the focused panel into the cursor's tab stack.
func (ws *Workspace) ArrangeMerge() bool {
	moving := ws.focus.Current()
	target := ws.arrangeCursor
	if target == "" || target == moving {
		target = ws.neighborAny()
	}
	if moving == "" || target == "" || target == moving {
		return false
	}
	if !LayoutContainsPanel(ws.solvedTree, target) {
		return false
	}
	next := MovePanel(ws.ensureRoot(), moving, target, DockCenter)
	if next == nil {
		return false
	}
	ws.root = next
	ws.explicitRoot = true
	ws.commit()
	ws.focus.Set(moving)
	ws.arrangeCursor = moving
	if ws.width > 0 && ws.height > 0 {
		ws.Solve(ws.width, ws.height)
	}
	return true
}

// --- Resize ---------------------------------------------------------------

// resizeStepPercent is the share of a split's extent one resize key adjusts,
// matching Tide's 5% pane-resize step.
const resizeStepPercent = 5

// resizeNoticeLifetime bounds how long the size percentage lingers.
const resizeNoticeLifetime = 3 * time.Second

// ResizeEdge moves the focused pane's shared edge in a direction, Tide-style.
// The edge on the arrow's side moves that way: when a neighbour sits there the
// pane grows into it, and when it does not the opposite edge moves inward and
// the pane shrinks. The step is a share of the split's extent, clamped so
// neither side crosses its minimum size.
func (ws *Workspace) ResizeEdge(dir Direction) bool {
	horizontal := dir.Horizontal()
	return ws.resizeFocused(horizontal, func(index, count int) (int, bool) {
		forward, backward := index+1, index-1
		if forward >= count {
			forward = -1
		}
		if backward < 0 {
			backward = -1
		}
		if dir.Forward() {
			if forward >= 0 {
				return forward, true
			}
			return backward, false
		}
		if backward >= 0 {
			return backward, true
		}
		return forward, false
	}, 0, true)
}

// ResizeEdgePixels moves the focused pane's shared edge by a pixel amount in a
// direction; a negative amount moves it the other way. It is used by mouse
// separator dragging, which batches history until the drag ends.
func (ws *Workspace) ResizeEdgePixels(dir Direction, pixels int, record bool) bool {
	if pixels == 0 {
		return false
	}
	grow := pixels > 0
	move := pixels
	if move < 0 {
		move = -move
	}
	return ws.resizeFocused(dir.Horizontal(), func(index, count int) (int, bool) {
		if index+1 < count {
			return index + 1, grow
		}
		return -1, false
	}, move, record)
}

// ResizeWidth grows (grow=true) or shrinks the focused pane's width, trading
// with the nearest neighbour along the width axis.
func (ws *Workspace) ResizeWidth(grow bool) bool {
	return ws.resizeFocused(true, forwardNeighbour(grow), 0, true)
}

// ResizeHeight grows (grow=true) or shrinks the focused pane's height.
func (ws *Workspace) ResizeHeight(grow bool) bool {
	return ws.resizeFocused(false, forwardNeighbour(grow), 0, true)
}

// ResizeGrow increases the focused pane along whichever axis it can resize.
// The delta argument is retained for API compatibility; the step is a share of
// the split extent.
func (ws *Workspace) ResizeGrow(delta float64) bool {
	return ws.ResizeWidth(true) || ws.ResizeHeight(true)
}

// ResizeShrink decreases the focused pane along whichever axis it can resize.
func (ws *Workspace) ResizeShrink(delta float64) bool {
	return ws.ResizeWidth(false) || ws.ResizeHeight(false)
}

// forwardNeighbour picks the next sibling, falling back to the previous one.
func forwardNeighbour(grow bool) func(index, count int) (int, bool) {
	return func(index, count int) (int, bool) {
		if index+1 < count {
			return index + 1, grow
		}
		if index-1 >= 0 {
			return index - 1, grow
		}
		return -1, false
	}
}

// resizeFocused finds the split that governs the focused pane on an axis and
// trades pixels with the chosen sibling. move is a pixel amount, or zero to use
// the default step; record controls whether the change joins layout history.
func (ws *Workspace) resizeFocused(horizontal bool, pick func(index, count int) (int, bool), move int, record bool) bool {
	focus := ws.focus.Current()
	if focus == "" || ws.width <= 0 {
		return false
	}
	clone := ws.ensureRoot().cloneNode()
	split, index, ok := findResizeSplit(clone, focus, horizontal)
	if !ok {
		return false
	}
	neighbor, grow := pick(index, len(split.Children))
	if neighbor < 0 || neighbor == index {
		return false
	}
	if !ws.applyResize(split, index, neighbor, horizontal, grow, move) {
		return false
	}
	ws.root = NormalizeLayout(clone)
	ws.explicitRoot = true
	if record {
		ws.commit()
	}
	if ws.width > 0 && ws.height > 0 {
		ws.Solve(ws.width, ws.height)
	}
	ws.setResizeNotice(split, index, horizontal)
	return true
}

// applyResize moves the shared edge between two children of split, scaling
// their weights so the solver lands on the new pixel sizes.
func (ws *Workspace) applyResize(split *SplitNode, index, neighbor int, horizontal, grow bool, move int) bool {
	if index < 0 || neighbor < 0 || index >= len(split.Children) || neighbor >= len(split.Children) {
		return false
	}
	extent, ok := ws.splitExtent(split, horizontal)
	if !ok || extent <= 0 {
		return false
	}
	focusRect, focusOK := solvedNodeBounds(split.Children[index], ws.solved.Rects)
	neighborRect, neighborOK := solvedNodeBounds(split.Children[neighbor], ws.solved.Rects)
	if !focusOK || !neighborOK {
		return false
	}
	focusPixels := rectExtent(focusRect, horizontal)
	neighborPixels := rectExtent(neighborRect, horizontal)
	if move <= 0 {
		move = max(1, extent*resizeStepPercent/100)
	}

	if grow {
		move = min(move, max(0, neighborPixels-ws.minExtent(split.Children[neighbor], horizontal)))
	} else {
		move = min(move, max(0, focusPixels-ws.minExtent(split.Children[index], horizontal)))
	}
	if move <= 0 {
		return false
	}
	newFocus, newNeighbor := focusPixels, neighborPixels
	if grow {
		newFocus += move
		newNeighbor -= move
	} else {
		newFocus -= move
		newNeighbor += move
	}
	setChildWeight(split.Children[index], scaleWeight(childWeight(split.Children[index]), focusPixels, newFocus))
	setChildWeight(split.Children[neighbor], scaleWeight(childWeight(split.Children[neighbor]), neighborPixels, newNeighbor))
	return true
}

// findResizeSplit returns the deepest split of the requested orientation that
// contains the focused pane, and the pane's child index within it.
func findResizeSplit(node LayoutNode, focus string, horizontal bool) (*SplitNode, int, bool) {
	split, ok := node.(*SplitNode)
	if !ok {
		return nil, 0, false
	}
	index := -1
	for i, child := range split.Children {
		if LayoutContainsPanel(child, focus) {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, 0, false
	}
	if deeper, childIndex, ok := findResizeSplit(split.Children[index], focus, horizontal); ok {
		return deeper, childIndex, true
	}
	if split.Orientation != orientationFor(horizontal) || len(split.Children) < 2 {
		return nil, 0, false
	}
	return split, index, true
}

// splitExtent returns a split's pixel size along an axis from the solved
// rectangles.
func (ws *Workspace) splitExtent(split *SplitNode, horizontal bool) (int, bool) {
	bounds, ok := solvedNodeBounds(split, ws.solved.Rects)
	if !ok {
		return 0, false
	}
	return rectExtent(bounds, horizontal), true
}

// minExtent returns the smallest pixel size a subtree needs along an axis:
// panel minimums summed along the split axis and maxed across perpendicular
// nested groups, plus the configured gaps. It mirrors the solver's minAlong.
func (ws *Workspace) minExtent(node LayoutNode, horizontal bool) int {
	switch n := node.(type) {
	case *LeafNode:
		if horizontal {
			return ws.minWidthFor(n.ID)
		}
		return ws.minHeightFor(n.ID)
	case *TabStackNode:
		best := 1
		for _, id := range n.Panels {
			best = max(best, ws.minExtent(Leaf(id), horizontal))
		}
		return best
	case *SplitNode:
		if (n.Orientation == SplitHorizontal) == horizontal {
			gap := ws.hGap
			if !horizontal {
				gap = ws.vGap
			}
			total := max(0, gap) * (len(n.Children) - 1)
			for _, child := range n.Children {
				total += ws.minExtent(child, horizontal)
			}
			return total
		}
		best := 1
		for _, child := range n.Children {
			best = max(best, ws.minExtent(child, horizontal))
		}
		return best
	}
	return 1
}

// setResizeNotice records the focused pane's new percentage for status
// feedback.
func (ws *Workspace) setResizeNotice(split *SplitNode, index int, horizontal bool) {
	extent, ok := ws.splitExtent(split, horizontal)
	if !ok || extent <= 0 {
		return
	}
	rect, ok := solvedNodeBounds(split.Children[index], ws.solved.Rects)
	if !ok {
		return
	}
	axis := "width"
	if !horizontal {
		axis = "height"
	}
	ws.resizeNotice = fmt.Sprintf("%s %d%%", axis, rectExtent(rect, horizontal)*100/extent)
	ws.resizeNoticeAt = time.Now()
}

// ResizeStatus returns the transient size feedback ("width 37%") while it is
// fresh, or "".
func (ws *Workspace) ResizeStatus() string {
	if ws.resizeNotice == "" || time.Since(ws.resizeNoticeAt) > resizeNoticeLifetime {
		return ""
	}
	return ws.resizeNotice
}

func rectExtent(r Rect, horizontal bool) int {
	if horizontal {
		return r.Width
	}
	return r.Height
}

func scaleWeight(weight float64, oldPixels, newPixels int) float64 {
	if oldPixels <= 0 || newPixels <= 0 {
		return weight
	}
	scaled := weight * float64(newPixels) / float64(oldPixels)
	if scaled <= 0 {
		return 0.01
	}
	return scaled
}

// solvedNodeBounds returns the bounding rectangle of a subtree from the solved
// leaf rectangles.
func solvedNodeBounds(node LayoutNode, rects map[string]Rect) (Rect, bool) {
	switch n := node.(type) {
	case *LeafNode:
		rect, ok := rects[n.ID]
		return rect, ok
	case *TabStackNode:
		var out Rect
		found := false
		for _, id := range n.Panels {
			if rect, ok := rects[id]; ok {
				out, found = unionRects(out, rect, found)
			}
		}
		return out, found
	case *SplitNode:
		var out Rect
		found := false
		for _, child := range n.Children {
			if rect, ok := solvedNodeBounds(child, rects); ok {
				out, found = unionRects(out, rect, found)
			}
		}
		return out, found
	}
	return Rect{}, false
}

func unionRects(a, b Rect, haveA bool) (Rect, bool) {
	if !haveA {
		return b, true
	}
	x := min(a.X, b.X)
	y := min(a.Y, b.Y)
	right := max(a.X+a.Width, b.X+b.Width)
	bottom := max(a.Y+a.Height, b.Y+b.Height)
	return Rect{X: x, Y: y, Width: right - x, Height: bottom - y}, true
}

func childWeight(node LayoutNode) float64 {
	switch n := node.(type) {
	case *LeafNode:
		if n.Weight > 0 {
			return n.Weight
		}
	case *TabStackNode:
		if n.Weight > 0 {
			return n.Weight
		}
	case *SplitNode:
		if n.Weight > 0 {
			return n.Weight
		}
	}
	return 1
}

func setChildWeight(node LayoutNode, weight float64) {
	switch n := node.(type) {
	case *LeafNode:
		n.Weight = weight
	case *TabStackNode:
		n.Weight = weight
	case *SplitNode:
		n.Weight = weight
	}
}

// --- History & reset ------------------------------------------------------

func (ws *Workspace) snapshot() workspaceSnapshot {
	hidden := cloneStringSet(ws.hidden)
	for id, panel := range ws.panels {
		if panel.hidden {
			hidden[id] = true
		}
	}
	return workspaceSnapshot{
		root:         CloneLayout(ws.root),
		hidden:       hidden,
		focus:        ws.focus.Current(),
		activePreset: ws.activePreset,
	}
}

func (ws *Workspace) commit() {
	ws.history.Push(ws.snapshot())
}

func (ws *Workspace) restoreState(state workspaceSnapshot) {
	ws.root = CloneLayout(state.root)
	ws.explicitRoot = state.root != nil
	ws.hidden = cloneStringSet(state.hidden)
	for id, panel := range ws.panels {
		panel.hidden = state.hidden[id]
	}
	ws.activePreset = state.activePreset
	ws.zoomed = ""
	ws.peeked = ""
	ws.arrange = false
	ws.focus.Set(state.focus)
	ws.ensureFocus()
}

// Undo restores the previous layout state.
func (ws *Workspace) Undo() bool {
	state, ok := ws.history.Undo()
	if !ok {
		return false
	}
	ws.restoreState(state)
	return true
}

// Redo restores the next layout state.
func (ws *Workspace) Redo() bool {
	state, ok := ws.history.Redo()
	if !ok {
		return false
	}
	ws.restoreState(state)
	return true
}

// CanUndo / CanRedo report history availability.
func (ws *Workspace) CanUndo() bool { return ws.history.CanUndo() }
func (ws *Workspace) CanRedo() bool { return ws.history.CanRedo() }

// ResetLayout returns to the declared default arrangement, or the
// role-derived default when none was declared.
func (ws *Workspace) ResetLayout() {
	ws.hidden = map[string]bool{}
	for _, panel := range ws.panels {
		panel.hidden = false
	}
	if ws.defaultRoot != nil {
		ws.root = CloneLayout(ws.defaultRoot)
	} else {
		ws.root = ws.defaultLayout()
	}
	ws.explicitRoot = ws.root != nil
	ws.zoomed = ""
	ws.peeked = ""
	ws.arrange = false
	ws.activePreset = ""
	ws.ensureFocus()
	ws.commit()
}

// --- Presets --------------------------------------------------------------

// Preset is a named layout configuration applications can ship or users save.
type Preset struct {
	Name   string
	Root   LayoutNode
	Hidden []string
}

func (ws *Workspace) addPreset(preset Preset) {
	if preset.Name == "" {
		return
	}
	if _, exists := ws.presets[preset.Name]; !exists {
		ws.presetOrder = append(ws.presetOrder, preset.Name)
	}
	ws.presets[preset.Name] = preset
}

// AddPreset registers a preset.
func (ws *Workspace) AddPreset(name string, root LayoutNode, hidden ...string) {
	ws.addPreset(Preset{Name: name, Root: CloneLayout(root), Hidden: hidden})
}

// SavePreset captures the current layout as a user preset.
func (ws *Workspace) SavePreset(name string) {
	var hidden []string
	for _, id := range ws.order {
		if ws.isHidden(id) {
			hidden = append(hidden, id)
		}
	}
	ws.addPreset(Preset{Name: name, Root: CloneLayout(ws.ensureRoot()), Hidden: hidden})
}

// PresetNames returns preset names in registration order.
func (ws *Workspace) PresetNames() []string {
	return append([]string(nil), ws.presetOrder...)
}

// Presets returns registered presets.
func (ws *Workspace) Presets() []Preset {
	out := make([]Preset, 0, len(ws.presetOrder))
	for _, name := range ws.presetOrder {
		out = append(out, ws.presets[name])
	}
	return out
}

// ApplyPreset switches to a named preset.
func (ws *Workspace) ApplyPreset(name string) bool {
	preset, ok := ws.presets[name]
	if !ok {
		return false
	}
	if preset.Root == nil {
		return false
	}
	ws.root = NormalizeLayout(CloneLayout(preset.Root))
	ws.explicitRoot = true
	ws.hidden = map[string]bool{}
	for _, id := range ws.order {
		panel := ws.panels[id]
		// A panel declared hidden stays off unless this preset names it, so a
		// plugin does not appear in every preset just by being installed.
		hidden := panel != nil && panel.StartsHidden()
		for _, hiddenID := range preset.Hidden {
			if hiddenID == id {
				hidden = true
				break
			}
		}
		ws.hidden[id] = hidden
		if panel != nil {
			panel.hidden = hidden
		}
	}
	ws.activePreset = name
	ws.zoomed = ""
	ws.peeked = ""
	ws.ensureFocus()
	ws.commit()
	return true
}

// ActivePreset returns the most recently applied preset name, or "".
func (ws *Workspace) ActivePreset() string { return ws.activePreset }

// --- Interaction state ----------------------------------------------------

// FocusPresentation returns the active focus presentation settings.
func (ws *Workspace) FocusPresentation() FocusPresentation { return ws.focusPres }

// SetFocusPresentation replaces the focus presentation.
func (ws *Workspace) SetFocusPresentation(p FocusPresentation) { ws.focusPres = p }

// Animation exposes the workspace animator.
func (ws *Workspace) Animation() *Animator { return ws.anim }

// Width and Height return the last solved terminal size.
func (ws *Workspace) Width() int  { return ws.width }
func (ws *Workspace) Height() int { return ws.height }

// Reflow recomputes the cached responsive choice for a new width, applying
// hysteresis so layouts do not oscillate around a breakpoint.
func (ws *Workspace) Reflow(width, height int) {
	ws.Solve(width, height)
	ws.anim.Tick()
}

// Actions returns every workspace-level and panel-level command.
func (ws *Workspace) Commands() []Command {
	return ws.buildCommands()
}

// PanelPicker returns the framework panel picker.
func (ws *Workspace) PanelPicker() *PanelPicker { return ws.picker }

// CommandPalette returns the framework command palette.
func (ws *Workspace) CommandPalette() *CommandPalette { return ws.palette }

// OpenPanelPicker shows the panel picker.
func (ws *Workspace) OpenPanelPicker() {
	ws.picker.Open(ws.panels, ws.order, ws)
}

// OpenCommandPalette shows the command palette fed by Commands.
func (ws *Workspace) OpenCommandPalette() {
	ws.palette.Open(ws.Commands())
}

// Overlay returns the modal overlay for whichever picker is open, or nil.
func (ws *Workspace) Overlay(r Renderer) *Overlay {
	if ws.picker.Opened() {
		overlay := ws.picker.Render(r, ws.width, ws.height)
		return &overlay
	}
	if ws.palette.Opened() {
		overlay := ws.palette.Render(r, ws.width, ws.height)
		return &overlay
	}
	return nil
}

// HandleKey routes a key to whichever workspace interaction is active. It
// reports whether the key was consumed. Applications handle their own
// domain keys first and fall back to HandleKey.
func (ws *Workspace) HandleKey(msg tea.KeyMsg) bool {
	if ws.picker.Opened() {
		if action := ws.picker.Update(msg); action != PanelPickerNone {
			return true
		}
		return true
	}
	if ws.palette.Opened() {
		if action := ws.palette.Update(msg); action != PaletteNone {
			return true
		}
		return true
	}
	key := msg.String()
	if ws.arrange {
		switch key {
		case "esc":
			ws.ExitArrange()
			return true
		case "enter":
			ws.ArrangeDrop()
			return true
		case "t":
			ws.ArrangeMerge()
			return true
		case "h", "left":
			ws.ArrangeMove(DirLeft)
			return true
		case "j", "down":
			ws.ArrangeMove(DirDown)
			return true
		case "k", "up":
			ws.ArrangeMove(DirUp)
			return true
		case "l", "right":
			ws.ArrangeMove(DirRight)
			return true
		}
		return true
	}
	switch key {
	case "tab":
		return ws.FocusNext()
	case "shift+tab":
		return ws.FocusPrev()
	case "m":
		return ws.ToggleArrange()
	case "w":
		ws.OpenPanelPicker()
		return true
	case "ctrl+p":
		ws.OpenCommandPalette()
		return true
	case "shift+space", "ctrl+space":
		return ws.ToggleZoom()
	case " ":
		// space enters the focused pane, and leaves it. A pane that binds
		// space itself keeps it: the action dispatch at the end of this
		// function is asked first, so the workspace claiming a key can never
		// break a panel that was already using it.
		if ws.RunFocusedAction(key) {
			return true
		}
		if ws.EnterPane() {
			return true
		}
		return ws.LeavePane()
	// Shift+arrows resize the focused pane directly, Tide-style. ctrl+arrows
	// are a silent alias for terminals that swallow shifted arrows.
	case "shift+left", "ctrl+left":
		return ws.ResizeEdge(DirLeft)
	case "shift+right", "ctrl+right":
		return ws.ResizeEdge(DirRight)
	case "shift+up", "ctrl+up":
		return ws.ResizeEdge(DirUp)
	case "shift+down", "ctrl+down":
		return ws.ResizeEdge(DirDown)
	case "esc":
		// Esc peels one layer at a time: the pane's keyboard first, then a
		// peek, then a zoom. Leaving the pane before restoring the layout is
		// what lets a zoomed, entered pane be walked and then unzoomed with two
		// presses rather than one that does both.
		if ws.LeavePane() {
			return true
		}
		if ws.peeked != "" {
			ws.Unpeek()
			return true
		}
		if ws.zoomCandidate() != "" {
			ws.Unzoom()
			return true
		}
	}
	// Finally, dispatch a key to the focused panel's contextual actions.
	return ws.RunFocusedAction(key)
}

// RunFocusedAction runs the focused panel's action bound to key, if any. It
// returns whether an action fired. Applications that handle their own keys
// first can still call this explicitly.
func (ws *Workspace) RunFocusedAction(key string) bool {
	if key == " " {
		key = "space"
	}
	panel := ws.panels[ws.focus.Current()]
	if panel == nil {
		return false
	}
	for _, action := range panel.actions {
		actionKey := action.Key
		if actionKey == " " {
			actionKey = "space"
		}
		if actionKey == key && action.Handler != nil {
			action.Handler(ws)
			return true
		}
	}
	return false
}

func (ws *Workspace) neighborAny() string {
	moving := ws.focus.Current()
	if moving == "" {
		return ""
	}
	for _, dir := range []Direction{DirRight, DirDown, DirLeft, DirUp} {
		if target := ws.focus.Directional(ws.solved, dir, func(id string) bool {
			return id != moving && LayoutContainsPanel(ws.solvedTree, id)
		}); target != "" {
			return target
		}
	}
	return ""
}

// sortedIDs is used by tests and diagnostics to iterate deterministically.
func (ws *Workspace) sortedIDs() []string {
	out := append([]string(nil), ws.order...)
	sort.Strings(out)
	return out
}

// ErrNoStore is returned by Persist when persistence has no backing store.
var ErrNoStore = errors.New("tideui: workspace persistence is not configured")
