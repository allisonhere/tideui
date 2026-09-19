package tideui

import (
	"sort"
	"strings"
)

// Orientation selects the axis a SplitNode divides along.
type Orientation int

const (
	// SplitHorizontal places children side by side, splitting the width.
	SplitHorizontal Orientation = iota
	// SplitVertical stacks children top to bottom, splitting the height.
	SplitVertical
)

// LayoutNode is one node of a workspace layout tree. Trees describe relative
// structure — splits, tab stacks, and panel leaves — never screen coordinates.
// The solver turns a tree plus terminal dimensions into rectangles.
type LayoutNode interface {
	isLayoutNode()
	cloneNode() LayoutNode
}

// LeafNode is a single panel occupying one region.
type LeafNode struct {
	ID     string
	Weight float64 // zero means "use the panel's grow weight"
}

// TabStackNode is a set of panels sharing one region, shown one at a time.
type TabStackNode struct {
	Panels []string
	Active int // index into Panels; clamped by the solver
	Weight float64
}

// SplitNode divides its region among children along Orientation.
type SplitNode struct {
	Orientation Orientation
	Children    []LayoutNode
	Weight      float64
}

func (*LeafNode) isLayoutNode()     {}
func (*TabStackNode) isLayoutNode() {}
func (*SplitNode) isLayoutNode()    {}

func (n *LeafNode) cloneNode() LayoutNode {
	c := *n
	return &c
}

func (n *TabStackNode) cloneNode() LayoutNode {
	c := *n
	c.Panels = append([]string(nil), n.Panels...)
	return &c
}

func (n *SplitNode) cloneNode() LayoutNode {
	c := *n
	c.Children = make([]LayoutNode, len(n.Children))
	for i, child := range n.Children {
		c.Children[i] = child.cloneNode()
	}
	return &c
}

// Leaf returns a layout node for a single panel.
func Leaf(id string) LayoutNode { return &LeafNode{ID: id} }

// Tabs returns a layout node whose panels share one region as a tab stack.
func Tabs(ids ...string) LayoutNode {
	return &TabStackNode{Panels: append([]string(nil), ids...)}
}

// HStack arranges child nodes in columns, left to right.
func HStack(children ...LayoutNode) LayoutNode {
	return &SplitNode{Orientation: SplitHorizontal, Children: children}
}

// VStack arranges child nodes in rows, top to bottom.
func VStack(children ...LayoutNode) LayoutNode {
	return &SplitNode{Orientation: SplitVertical, Children: children}
}

// Weighted assigns an explicit share weight to a node within its parent split.
// It mutates and returns the same node, so it is safe to nest inside a
// constructor call. A weight of zero (the default) defers to a panel's Grow
// weight for leaves and to 1 for groups.
func Weighted(node LayoutNode, weight float64) LayoutNode {
	switch n := node.(type) {
	case *LeafNode:
		n.Weight = weight
	case *TabStackNode:
		n.Weight = weight
	case *SplitNode:
		n.Weight = weight
	}
	return node
}

// CloneLayout returns a deep copy of a layout tree.
func CloneLayout(root LayoutNode) LayoutNode {
	if root == nil {
		return nil
	}
	return root.cloneNode()
}

// LayoutPanelIDs returns every panel referenced by a tree, in traversal order,
// including duplicates exactly once each.
func LayoutPanelIDs(root LayoutNode) []string {
	seen := map[string]bool{}
	var out []string
	walkLayout(root, func(node LayoutNode) {
		for _, id := range nodePanelIDs(node) {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	})
	return out
}

// LayoutContainsPanel reports whether the tree references id.
func LayoutContainsPanel(root LayoutNode, id string) bool {
	found := false
	walkLayout(root, func(node LayoutNode) {
		for _, panel := range nodePanelIDs(node) {
			if panel == id {
				found = true
			}
		}
	})
	return found
}

// nodePanelIDs returns the panel ids a single node names directly.
func nodePanelIDs(node LayoutNode) []string {
	switch n := node.(type) {
	case *LeafNode:
		return []string{n.ID}
	case *TabStackNode:
		return n.Panels
	}
	return nil
}

// walkLayout visits every node depth-first.
func walkLayout(root LayoutNode, fn func(LayoutNode)) {
	if root == nil {
		return
	}
	fn(root)
	if split, ok := root.(*SplitNode); ok {
		for _, child := range split.Children {
			walkLayout(child, fn)
		}
	}
}

// mutateLayout returns a cloned tree with fn applied to every node. fn must
// not mutate shared state; it returns the replacement node.
func mapLayout(root LayoutNode, fn func(LayoutNode) LayoutNode) LayoutNode {
	if root == nil {
		return nil
	}
	clone := root.cloneNode()
	switch n := clone.(type) {
	case *SplitNode:
		for i, child := range n.Children {
			n.Children[i] = mapLayout(child, fn)
		}
	}
	return fn(clone)
}

// NormalizeLayout collapses single-child splits and empty tab stacks, and
// drops splits with no children. The result is structurally minimal, so a
// sequence of moves and hides never leaves redundant scaffolding behind.
func NormalizeLayout(root LayoutNode) LayoutNode {
	if root == nil {
		return nil
	}
	switch n := root.(type) {
	case *LeafNode:
		return n
	case *TabStackNode:
		if len(n.Panels) == 0 {
			return nil
		}
		if len(n.Panels) == 1 {
			return &LeafNode{ID: n.Panels[0], Weight: n.Weight}
		}
		n.Panels = dedupeStrings(n.Panels)
		if len(n.Panels) == 1 {
			return &LeafNode{ID: n.Panels[0], Weight: n.Weight}
		}
		n.Active = clampIndex(n.Active, len(n.Panels))
		return n
	case *SplitNode:
		children := make([]LayoutNode, 0, len(n.Children))
		for _, child := range n.Children {
			if child == nil {
				continue
			}
			normalized := NormalizeLayout(child)
			if normalized == nil {
				continue
			}
			// Flatten a child split of the same orientation into the parent.
			if inner, ok := normalized.(*SplitNode); ok && inner.Orientation == n.Orientation && inner.Weight == 0 {
				for _, grandchild := range inner.Children {
					children = append(children, grandchild)
				}
				continue
			}
			children = append(children, normalized)
		}
		if len(children) == 0 {
			return nil
		}
		if len(children) == 1 {
			if leaf, ok := children[0].(*LeafNode); ok {
				leaf.Weight = n.Weight
			}
			return children[0]
		}
		n.Children = children
		return n
	}
	return root
}

// SanitizeLayout returns a normalized copy of root in which each panel id
// appears at most once (the first occurrence wins) and empty nodes are
// dropped. It defends against hand-authored or persisted layouts that name the
// same panel in two places, which would otherwise solve to overlapping
// rectangles.
func SanitizeLayout(root LayoutNode) LayoutNode {
	if root == nil {
		return nil
	}
	seen := map[string]bool{}
	var walk func(LayoutNode) LayoutNode
	walk = func(node LayoutNode) LayoutNode {
		switch n := node.(type) {
		case *LeafNode:
			if n.ID == "" || seen[n.ID] {
				return nil
			}
			seen[n.ID] = true
			return n
		case *TabStackNode:
			var panels []string
			for _, id := range n.Panels {
				if id == "" || seen[id] {
					continue
				}
				seen[id] = true
				panels = append(panels, id)
			}
			if len(panels) == 0 {
				return nil
			}
			if len(panels) == 1 {
				return &LeafNode{ID: panels[0], Weight: n.Weight}
			}
			n.Panels = panels
			n.Active = clampIndex(n.Active, len(panels))
			return n
		case *SplitNode:
			children := make([]LayoutNode, 0, len(n.Children))
			for _, child := range n.Children {
				if next := walk(child); next != nil {
					children = append(children, next)
				}
			}
			if len(children) == 0 {
				return nil
			}
			n.Children = children
			return n
		}
		return nil
	}
	return NormalizeLayout(walk(root))
}

// RemovePanel returns a cloned tree with id removed. Splits left empty or
// with a single child collapse, so removing a panel closes the gap it left.
// The boolean reports whether the panel was present.
func RemovePanel(root LayoutNode, id string) (LayoutNode, bool) {
	isEmpty := func(n LayoutNode) bool { return n == nil }
	var remove func(LayoutNode) (LayoutNode, bool)
	remove = func(node LayoutNode) (LayoutNode, bool) {
		if isEmpty(node) {
			return nil, false
		}
		clone := node.cloneNode()
		switch n := clone.(type) {
		case *LeafNode:
			if n.ID == id {
				return nil, true
			}
			return n, false
		case *TabStackNode:
			for i, p := range n.Panels {
				if p == id {
					n.Panels = append(n.Panels[:i:i], n.Panels[i+1:]...)
					if len(n.Panels) == 0 {
						return nil, true
					}
					n.Active = clampIndex(n.Active, len(n.Panels))
					return n, true
				}
			}
			return n, false
		case *SplitNode:
			removed := false
			children := make([]LayoutNode, 0, len(n.Children))
			for _, child := range n.Children {
				next, ok := remove(child)
				if ok {
					removed = true
				}
				if next != nil {
					children = append(children, next)
				}
			}
			n.Children = children
			return n, removed
		}
		return clone, false
	}
	next, removed := remove(root)
	if !removed {
		return root, false
	}
	return NormalizeLayout(next), true
}

// SwapPanels exchanges the positions of two panels anywhere in the tree. If
// both live in the same tab stack their tab order is swapped instead.
func SwapPanels(root LayoutNode, a, b string) LayoutNode {
	return mapLayout(root, func(node LayoutNode) LayoutNode {
		switch n := node.(type) {
		case *LeafNode:
			if n.ID == a {
				n.ID = b
			} else if n.ID == b {
				n.ID = a
			}
		case *TabStackNode:
			for i, p := range n.Panels {
				if p == a {
					n.Panels[i] = b
				} else if p == b {
					n.Panels[i] = a
				}
			}
		}
		return node
	})
}

// DockSide describes where a moved panel lands relative to a target panel.
type DockSide int

const (
	// DockCenter merges the moving panel into the target's tab stack.
	DockCenter DockSide = iota
	// DockLeft places the moving panel to the left of the target.
	DockLeft
	// DockRight places the moving panel to the right of the target.
	DockRight
	// DockAbove places the moving panel above the target, stacking them.
	DockAbove
	// DockBelow places the moving panel below the target, stacking them.
	DockBelow
	// DockRowAbove joins the target's row as another column, placed before it.
	DockRowAbove
	// DockRowBelow joins the target's row as another column, placed after it.
	DockRowBelow
)

// MovedDock returns the side a panel docks to when the cursor moves in a
// direction. Left/right dock beside the target; up/down join the target's row
// as another column, so a move between rows shares that row instead of
// stacking a new one.
func MovedDock(d Direction) DockSide {
	switch d {
	case DirLeft:
		return DockLeft
	case DirRight:
		return DockRight
	case DirUp:
		return DockRowAbove
	case DirDown:
		return DockRowBelow
	}
	return DockCenter
}

// MovePanel returns a cloned tree with moving repositioned relative to target
// on side. The panel is first detached (closing its old gap), then docked. A
// missing moving panel or target leaves the tree unchanged. Moving a panel
// onto itself is a no-op.
func MovePanel(root LayoutNode, moving string, target string, side DockSide) LayoutNode {
	if moving == target || moving == "" || target == "" {
		return root
	}
	if !LayoutContainsPanel(root, moving) || !LayoutContainsPanel(root, target) {
		return root
	}
	detached, _ := RemovePanel(root, moving)
	if detached == nil {
		return root
	}
	if side == DockCenter {
		return mergeIntoStack(detached, moving, target)
	}
	return insertRelative(detached, moving, target, side)
}

// mergeIntoStack folds moving into the tab stack that contains target, or
// wraps target's leaf in a new stack.
func mergeIntoStack(root LayoutNode, moving, target string) LayoutNode {
	return mapLayout(root, func(node LayoutNode) LayoutNode {
		switch n := node.(type) {
		case *LeafNode:
			if n.ID == target {
				return &TabStackNode{Panels: []string{n.ID, moving}}
			}
		case *TabStackNode:
			for _, p := range n.Panels {
				if p == target {
					n.Panels = append(n.Panels, moving)
					return n
				}
			}
		}
		return node
	})
}

// insertRelative places moving next to target, creating a split when the
// target's parent does not already run along the needed axis.
func insertRelative(root LayoutNode, moving, target string, side DockSide) LayoutNode {
	// Directional docks stack along the dock axis (left/right share a row,
	// above/below stack a column); row docks always share the target's row as
	// an extra column, wrapping a lone pane into a row when needed.
	horizontal := side == DockLeft || side == DockRight || side == DockRowAbove || side == DockRowBelow
	before := side == DockLeft || side == DockAbove || side == DockRowAbove
	orientation := orientationFor(horizontal)

	var insert func(LayoutNode) (LayoutNode, bool)
	insert = func(node LayoutNode) (LayoutNode, bool) {
		split, ok := node.(*SplitNode)
		if !ok {
			return node, false
		}
		for i, child := range split.Children {
			if isPanelNode(child, target) {
				if split.Orientation == orientation {
					// The target already lives in a container on the wanted
					// axis: insert as a sibling next to it.
					children := make([]LayoutNode, 0, len(split.Children)+1)
					children = append(children, split.Children[:i]...)
					if before {
						children = append(children, Leaf(moving), split.Children[i])
					} else {
						children = append(children, split.Children[i], Leaf(moving))
					}
					children = append(children, split.Children[i+1:]...)
					split.Children = children
					return split, true
				}
				// The target's container runs the other way: wrap the target
				// in place so the move lands in the target's slot rather than
				// skipping the row.
				setChildWeight(child, 1)
				wrapped := &SplitNode{Orientation: orientation}
				if before {
					wrapped.Children = []LayoutNode{Leaf(moving), child}
				} else {
					wrapped.Children = []LayoutNode{child, Leaf(moving)}
				}
				split.Children[i] = wrapped
				return split, true
			}
			if replaced, ok := insert(child); ok {
				split.Children[i] = replaced
				return split, true
			}
		}
		return split, false
	}
	if next, ok := insert(root); ok {
		return NormalizeLayout(next)
	}
	// The target is the root itself (no containing split): wrap the two.
	if before {
		return NormalizeLayout(&SplitNode{Orientation: orientation, Children: []LayoutNode{Leaf(moving), root}})
	}
	return NormalizeLayout(&SplitNode{Orientation: orientation, Children: []LayoutNode{root, Leaf(moving)}})
}

func orientationFor(horizontal bool) Orientation {
	if horizontal {
		return SplitHorizontal
	}
	return SplitVertical
}

func isPanelNode(node LayoutNode, id string) bool {
	switch n := node.(type) {
	case *LeafNode:
		return n.ID == id
	case *TabStackNode:
		for _, p := range n.Panels {
			if p == id {
				return true
			}
		}
	}
	return false
}

// SetActiveTab points the tab stack containing id at that panel.
func SetActiveTab(root LayoutNode, id string) LayoutNode {
	return mapLayout(root, func(node LayoutNode) LayoutNode {
		if stack, ok := node.(*TabStackNode); ok {
			for i, p := range stack.Panels {
				if p == id {
					stack.Active = i
				}
			}
		}
		return node
	})
}

// Rect is a solved region in terminal cells.
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Empty reports whether the rectangle has no drawable area.
func (r Rect) Empty() bool { return r.Width <= 0 || r.Height <= 0 }

// Contains reports whether a cell falls inside the rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// SolvedRegion is one solved layout region. Leaf regions carry a single panel
// id; tab stacks carry several, with ActiveIndex naming the visible one.
type SolvedRegion struct {
	Rect        Rect
	PanelIDs    []string
	ActiveIndex int
	TabStack    bool
}

// ActivePanel returns the panel currently visible in the region.
func (r SolvedRegion) ActivePanel() string {
	if len(r.PanelIDs) == 0 {
		return ""
	}
	return r.PanelIDs[clampIndex(r.ActiveIndex, len(r.PanelIDs))]
}

// SolvedLayout is the rectangle result of solving a layout tree.
type SolvedLayout struct {
	Width    int
	Height   int
	Regions  []SolvedRegion
	Rects    map[string]Rect
	Overflow bool // true when minimum sizes could not all be honored
}

// RegionFor returns the region containing a panel, if any.
func (s SolvedLayout) RegionFor(id string) (SolvedRegion, bool) {
	for _, region := range s.Regions {
		for _, p := range region.PanelIDs {
			if p == id {
				return region, true
			}
		}
	}
	return SolvedRegion{}, false
}

// RegionAt returns the region containing a cell, if any.
func (s SolvedLayout) RegionAt(x, y int) (SolvedRegion, bool) {
	for _, region := range s.Regions {
		if region.Rect.Contains(x, y) {
			return region, true
		}
	}
	return SolvedRegion{}, false
}

// LayoutSolver turns a layout tree into rectangles. It is deliberately free of
// any terminal or widget state so layout can be tested on its own.
type LayoutSolver struct {
	// HGap is the blank cells left between columns of a horizontal split.
	HGap int
	// VGap is the blank cells left between rows of a vertical split.
	VGap int
	// MinWidth and MinHeight report a panel's minimum drawable size.
	MinWidth  func(id string) int
	MinHeight func(id string) int
	// Weight reports a node's share weight within its parent split.
	Weight func(node LayoutNode) float64
}

// Solve lays root out inside width x height.
func (s LayoutSolver) Solve(root LayoutNode, width, height int) SolvedLayout {
	out := SolvedLayout{Width: width, Height: height, Rects: map[string]Rect{}}
	if root == nil || width <= 0 || height <= 0 {
		return out
	}
	s.solve(root, Rect{Width: width, Height: height}, &out)
	return out
}

func (s LayoutSolver) solve(node LayoutNode, area Rect, out *SolvedLayout) {
	if node == nil || area.Empty() {
		return
	}
	switch n := node.(type) {
	case *LeafNode:
		if n.ID == "" {
			return
		}
		out.Regions = append(out.Regions, SolvedRegion{Rect: area, PanelIDs: []string{n.ID}})
		out.Rects[n.ID] = area
	case *TabStackNode:
		panels := dedupeStrings(n.Panels)
		if len(panels) == 0 {
			return
		}
		active := clampIndex(n.Active, len(panels))
		out.Regions = append(out.Regions, SolvedRegion{Rect: area, PanelIDs: panels, ActiveIndex: active, TabStack: true})
		for _, p := range panels {
			out.Rects[p] = area
		}
	case *SplitNode:
		if len(n.Children) == 0 {
			return
		}
		horizontal := n.Orientation == SplitHorizontal
		available := area.Width
		if !horizontal {
			available = area.Height
		}
		gap := s.HGap
		if !horizontal {
			gap = s.VGap
		}
		if gap < 0 {
			gap = 0
		}
		gaps := gap * (len(n.Children) - 1)
		content := max(0, available-gaps)
		mins := make([]int, len(n.Children))
		weights := make([]float64, len(n.Children))
		for i, child := range n.Children {
			mins[i] = s.minAlong(child, horizontal)
			weights[i] = s.nodeWeight(child)
		}
		sizes, overflow := distribute(content, mins, weights)
		out.Overflow = out.Overflow || overflow
		pos := area.X
		if !horizontal {
			pos = area.Y
		}
		for i, child := range n.Children {
			size := sizes[i]
			if size <= 0 {
				continue
			}
			childArea := area
			if horizontal {
				childArea.X = pos
				childArea.Width = size
			} else {
				childArea.Y = pos
				childArea.Height = size
			}
			s.solve(child, childArea, out)
			pos += size + gap
		}
	}
}

// minAlong returns the minimum extent a node needs along an axis.
func (s LayoutSolver) minAlong(node LayoutNode, horizontal bool) int {
	switch n := node.(type) {
	case *LeafNode:
		return s.panelMin(n.ID, horizontal)
	case *TabStackNode:
		panels := dedupeStrings(n.Panels)
		if len(panels) == 0 {
			return 1
		}
		return s.panelMin(panels[clampIndex(n.Active, len(panels))], horizontal)
	case *SplitNode:
		if len(n.Children) == 0 {
			return 1
		}
		if (n.Orientation == SplitHorizontal) == horizontal {
			gap := s.HGap
			if !horizontal {
				gap = s.VGap
			}
			total := max(0, gap) * (len(n.Children) - 1)
			for _, child := range n.Children {
				total += s.minAlong(child, horizontal)
			}
			return max(1, total)
		}
		best := 1
		for _, child := range n.Children {
			best = max(best, s.minAlong(child, horizontal))
		}
		return best
	}
	return 1
}

func (s LayoutSolver) panelMin(id string, horizontal bool) int {
	if horizontal {
		if s.MinWidth != nil {
			return max(1, s.MinWidth(id))
		}
		return 1
	}
	if s.MinHeight != nil {
		return max(1, s.MinHeight(id))
	}
	return 1
}

func (s LayoutSolver) nodeWeight(node LayoutNode) float64 {
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
	if s.Weight != nil {
		if w := s.Weight(node); w > 0 {
			return w
		}
	}
	return 1
}

// distribute divides total cells among children, honoring minimums while
// sharing the whole extent by weight. When the minimums do not fit, sizes
// shrink proportionally and overflow is reported. The result always sums to
// total (given total >= 0) and is deterministic.
func distribute(total int, mins []int, weights []float64) ([]int, bool) {
	n := len(mins)
	sizes := make([]int, n)
	if n == 0 || total <= 0 {
		return sizes, total > 0
	}
	if total <= n {
		for i := 0; i < total && i < n; i++ {
			sizes[i] = 1
		}
		return sizes, true
	}
	floors := make([]int, n)
	sumMin := 0
	for i, m := range mins {
		floors[i] = max(1, m)
		sumMin += floors[i]
	}
	if total < sumMin {
		return scaleToFit(total, floors), true
	}
	sumWeight := 0.0
	for _, w := range weights {
		if w > 0 {
			sumWeight += w
		}
	}
	if sumWeight <= 0 {
		sumWeight = float64(n)
	}
	ideal := make([]float64, n)
	for i := range sizes {
		w := weights[i]
		if w <= 0 {
			w = 1
		}
		ideal[i] = float64(total) * w / sumWeight
		if floor := int(ideal[i]); floor > floors[i] {
			sizes[i] = floor
		} else {
			sizes[i] = floors[i]
		}
	}
	assigned := 0
	for _, size := range sizes {
		assigned += size
	}
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	if assigned < total {
		sort.SliceStable(order, func(a, b int) bool {
			fa := ideal[order[a]] - float64(sizes[order[a]])
			fb := ideal[order[b]] - float64(sizes[order[b]])
			return fa > fb
		})
		for i := 0; assigned < total; i++ {
			sizes[order[i%n]]++
			assigned++
		}
	} else if assigned > total {
		sort.SliceStable(order, func(a, b int) bool {
			da := sizes[order[a]] - floors[order[a]]
			db := sizes[order[b]] - floors[order[b]]
			if da != db {
				return da > db
			}
			return sizes[order[a]] > sizes[order[b]]
		})
		for _, idx := range order {
			if assigned == total {
				break
			}
			take := min(assigned-total, sizes[idx]-floors[idx])
			if take > 0 {
				sizes[idx] -= take
				assigned -= take
			}
		}
	}
	return sizes, false
}

// scaleToFit divides total proportionally to want, each at least 1.
func scaleToFit(total int, want []int) []int {
	n := len(want)
	sizes := make([]int, n)
	if n == 0 {
		return sizes
	}
	if total < n {
		for i := 0; i < total; i++ {
			sizes[i] = 1
		}
		return sizes
	}
	sum := 0
	for _, w := range want {
		sum += max(1, w)
	}
	assigned := 0
	frac := make([]float64, n)
	for i := range sizes {
		share := float64(total) * float64(max(1, want[i])) / float64(sum)
		base := int(share)
		if base < 1 {
			base = 1
		}
		sizes[i] = base
		assigned += base
		frac[i] = share - float64(int(share))
	}
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return frac[order[a]] > frac[order[b]] })
	for assigned > total {
		progress := false
		for i := n - 1; i >= 0 && assigned > total; i-- {
			if sizes[i] > 1 {
				sizes[i]--
				assigned--
				progress = true
			}
		}
		if !progress {
			break
		}
	}
	for i := 0; assigned < total; i++ {
		sizes[order[i%n]]++
		assigned++
	}
	return sizes
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func clampIndex(index, length int) int {
	if length <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

// LayoutString renders a tree as an indented debug view, used in tests.
func LayoutString(root LayoutNode) string {
	var b strings.Builder
	var walk func(node LayoutNode, depth int)
	walk = func(node LayoutNode, depth int) {
		indent := strings.Repeat("  ", depth)
		switch n := node.(type) {
		case *LeafNode:
			b.WriteString(indent + "leaf " + n.ID + "\n")
		case *TabStackNode:
			b.WriteString(indent + "tabs [" + strings.Join(n.Panels, ", ") + "]\n")
		case *SplitNode:
			orientation := "H"
			if n.Orientation == SplitVertical {
				orientation = "V"
			}
			b.WriteString(indent + orientation + "{\n")
			for _, child := range n.Children {
				walk(child, depth+1)
			}
			b.WriteString(indent + "}\n")
		}
	}
	walk(root, 0)
	return b.String()
}
