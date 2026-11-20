package giode

import (
	"image"
	"strings"
	"sync"

	"github.com/xmhrnx/giode/elements"
)

// The accessibility tree platform bridges read (TalkBack on Android; the
// desktop and iOS bridges build on it too): the last frame's semantic
// nodes, with numeric IDs that stay the same while a node does, parents
// and children, and what changed since the last look.

// a11yTree is safe to read from the platform's accessibility thread
// while the UI goroutine updates it.
type a11yTree struct {
	mu    sync.Mutex
	nodes []elements.SemanticNode
	// num maps node IDs to the numbers platforms see, from 1; next is the
	// next to give.
	num  map[string]int32
	next int32
	// index is the position in nodes of each number; parent is each
	// number's parent (0 for the root); children each number's
	// children, root's under 0.
	index    map[int32]int
	parent   map[int32]int32
	children map[int32][]int32
	// sums are each number's content last frame, to report changes.
	sums map[int32]string
	// changed and announce are what changed since the bridge last took
	// them: node numbers, and text to announce.
	changed  []int32
	announce []a11yAnnouncement
	// notify, when set, runs after each update on the UI goroutine: a
	// platform bridge reporting the changes.
	notify func()
}

type a11yAnnouncement struct {
	text      string
	assertive bool
}

func newA11yTree() *a11yTree {
	return &a11yTree{num: map[string]int32{}, index: map[int32]int{}, parent: map[int32]int32{}, children: map[int32][]int32{}, sums: map[int32]string{}}
}

// actionable reports whether a node does something when activated, so
// the text inside it belongs to it.
func actionable(n elements.SemanticNode) bool {
	switch n.Kind {
	case elements.SemanticButton, elements.SemanticCheckbox, elements.SemanticSwitch, elements.SemanticRadio,
		elements.SemanticTab, elements.SemanticMenuItem, elements.SemanticLink, elements.SemanticSelect,
		elements.SemanticComboBox, elements.SemanticTreeItem, elements.SemanticListItem, elements.SemanticInput:
		return true
	}
	return false
}

// textual reports whether a node is text read as part of what holds it.
func textual(n elements.SemanticNode) bool {
	return n.Kind == elements.SemanticText || n.Kind == elements.SemanticImage
}

// update replaces the tree with a frame's nodes.
func (t *a11yTree) update(nodes []elements.SemanticNode) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes = append(t.nodes[:0], nodes...)
	num := make(map[string]int32, len(nodes))
	clear(t.index)
	for i, n := range t.nodes {
		k, ok := t.num[n.ID]
		if !ok {
			t.next++
			k = t.next
		}
		num[n.ID] = k
		t.index[k] = i
	}
	t.num = num
	clear(t.parent)
	clear(t.children)
	for i, n := range t.nodes {
		k := num[n.ID]
		p := int32(0)
		switch {
		case n.Parent != "":
			p = num[n.Parent]
		case textual(n):
			// Text inside a button, a checkbox's label: the smallest
			// control holding it.
			best := -1
			for j, c := range t.nodes {
				if j == i || !actionable(c) || !n.Bounds.In(c.Bounds) {
					continue
				}
				if best < 0 || area(c.Bounds) < area(t.nodes[best].Bounds) {
					best = j
				}
			}
			if best >= 0 {
				p = num[t.nodes[best].ID]
			} else if n.Scroll != "" {
				p = num[n.Scroll]
			}
		case n.Scroll != "" && n.Scroll != n.ID:
			p = num[n.Scroll]
		}
		if p == k {
			p = 0
		}
		t.parent[k] = p
		t.children[p] = append(t.children[p], k)
	}
	// What changed, and what to announce.
	seen := make(map[int32]bool, len(nodes))
	structure := false
	for _, n := range t.nodes {
		k := num[n.ID]
		seen[k] = true
		sum := a11ySum(n)
		old, had := t.sums[k]
		if had && old == sum {
			continue
		}
		t.sums[k] = sum
		if !had {
			structure = true
		}
		t.changed = append(t.changed, k)
		if n.Live != elements.LiveOff || (!had && n.Kind == elements.SemanticAlert) {
			if text := a11yText(n); text != "" {
				t.announce = append(t.announce, a11yAnnouncement{text: text, assertive: n.Live == elements.LiveAssertive || n.Kind == elements.SemanticAlert})
			}
		}
	}
	for k := range t.sums {
		if !seen[k] {
			delete(t.sums, k)
			structure = true
		}
	}
	if structure {
		t.changed = append(t.changed, 0)
	}
	if len(t.changed) > 256 {
		// A whole new screen: one change at the root says it all.
		t.changed = append(t.changed[:0], 0)
	}
}

// a11ySum is what a platform shows of a node, to tell when it changed.
func a11ySum(n elements.SemanticNode) string {
	var b strings.Builder
	b.WriteString(string(n.Kind))
	b.WriteByte(0)
	b.WriteString(n.Label)
	b.WriteByte(0)
	b.WriteString(n.Value)
	b.WriteByte(0)
	b.WriteString(n.Description)
	r := n.Bounds
	for _, v := range []int{r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, int(n.Checked), int(n.Expanded), n.Level} {
		b.WriteByte(0)
		b.WriteString(itoa(v))
	}
	for _, f := range []bool{n.Selected, n.Disabled, n.Focused, n.Invalid, n.Busy} {
		if f {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}

// a11yText is what's read for a node: its label, and its value.
func a11yText(n elements.SemanticNode) string {
	switch {
	case n.Label != "" && n.Value != "" && n.Value != n.Label && n.Kind == elements.SemanticInput:
		return n.Label + ", " + n.Value
	case n.Label != "":
		return n.Label
	}
	return n.Value
}

// takeChanges returns and forgets what changed since the last call.
func (t *a11yTree) takeChanges() ([]int32, []a11yAnnouncement) {
	t.mu.Lock()
	defer t.mu.Unlock()
	c, a := t.changed, t.announce
	t.changed, t.announce = nil, nil
	return c, a
}

// node returns node k, and its parent and children.
func (t *a11yTree) node(k int32) (n elements.SemanticNode, parent int32, children []int32, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if k == 0 {
		return elements.SemanticNode{}, 0, append([]int32(nil), t.children[0]...), true
	}
	i, ok := t.index[k]
	if !ok {
		return elements.SemanticNode{}, 0, nil, false
	}
	return t.nodes[i], t.parent[k], append([]int32(nil), t.children[k]...), true
}

// focusedNode returns the number of the node with keyboard focus, 0
// for none.
func (t *a11yTree) focusedNode() int32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, n := range t.nodes {
		if n.Focused {
			return t.num[n.ID]
		}
	}
	return 0
}

// isFocusable is focusable, taking the lock.
func (t *a11yTree) isFocusable(n elements.SemanticNode, k int32) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.focusable(n, k)
}

// at returns the node a touch at p (window pixels) explores: the
// smallest one there that can be focused, preferring controls; 0 when
// none.
func (t *a11yTree) at(p image.Point) int32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	best, bestScore := int32(0), 0
	for _, n := range t.nodes {
		if !p.In(n.Bounds) {
			continue
		}
		k := t.num[n.ID]
		if !t.focusable(n, k) {
			continue
		}
		// Controls beat text; then the smallest wins.
		score := 1 << 30
		if actionable(n) {
			score = 1 << 29
		}
		score -= min(area(n.Bounds), 1<<28)
		if score > bestScore {
			best, bestScore = k, score
		}
	}
	return best
}

// focusable reports whether a screen reader stops on node k: controls,
// and text that isn't part of a control.
func (t *a11yTree) focusable(n elements.SemanticNode, k int32) bool {
	switch n.Kind {
	case elements.SemanticScroll, elements.SemanticGroup, elements.SemanticList, elements.SemanticTable,
		elements.SemanticTree, elements.SemanticMenu, elements.SemanticMenuBar, elements.SemanticTabList,
		elements.SemanticToolbar, elements.SemanticRegion, elements.SemanticDialog, elements.SemanticRow:
		return false
	}
	if textual(n) {
		if p, ok := t.index[t.parent[k]]; ok && actionable(t.nodes[p]) {
			return false
		}
		return n.Label != ""
	}
	return true
}

func area(r image.Rectangle) int { return r.Dx() * r.Dy() }

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// a11yAction is something a screen reader asks a node to do.
type a11yAction int

const (
	a11yActivate a11yAction = iota + 1
	a11yScrollForward
	a11yScrollBackward
	a11ySetText
	a11yExpand
	a11yCollapse
	a11yFocus
	a11yIncrement
	a11yDecrement
)

// a11yStepper is a control a screen reader can step, like a slider.
type a11yStepper interface {
	Value() float32
	Change(v float32)
}

// perform does action on node k, on the UI goroutine. It reports
// whether the node takes the action.
func (t *a11yTree) perform(a *App, k int32, action a11yAction, arg string) bool {
	n, _, _, ok := t.node(k)
	if !ok || n.Disabled {
		return false
	}
	switch action {
	case a11yActivate, a11yExpand, a11yCollapse:
		if c, ok := n.Target.(interface{ Click() }); ok {
			a.Post(c.Click)
			return true
		}
		// No component to call: a tap at its middle, as a finger would.
		return a11yTap(a, n.Bounds)
	case a11yScrollForward, a11yScrollBackward:
		s, ok := n.Target.(interface{ ScrollBy(dx, dy float32) })
		if !ok {
			return false
		}
		d := float32(n.Bounds.Dy()) * 0.8
		if action == a11yScrollBackward {
			d = -d
		}
		a.Post(func() { s.ScrollBy(0, d) })
		return true
	case a11ySetText:
		s, ok := n.Target.(interface{ SetText(string) })
		if !ok {
			return false
		}
		a.Post(func() { s.SetText(arg) })
		return true
	case a11yIncrement, a11yDecrement:
		st, ok := n.Target.(a11yStepper)
		if !ok {
			return false
		}
		step := float32(n.Max-n.Min) / 20
		if step <= 0 {
			step = 0.05
		}
		if action == a11yDecrement {
			step = -step
		}
		a.Post(func() { st.Change(st.Value() + step) })
		return true
	case a11yFocus:
		f, ok := n.Target.(interface{ Focus() })
		if !ok {
			return false
		}
		a.Post(f.Focus)
		return true
	}
	return false
}

// canDo reports which actions node n takes.
func a11yActions(n elements.SemanticNode) (activate, scroll, setText, focus bool) {
	if n.Disabled {
		return
	}
	_, click := n.Target.(interface{ Click() })
	activate = click || (actionable(n) && n.Kind != elements.SemanticInput)
	_, scroll = n.Target.(interface{ ScrollBy(dx, dy float32) })
	_, setText = n.Target.(interface{ SetText(string) })
	_, focus = n.Target.(interface{ Focus() })
	return
}

func boolInt(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
