package base

import (
	"fmt"
	"image"
	"sync"
	"sync/atomic"

	"gioui.org/layout"
)

// semantics is the recorder for the frame being laid out; nil (the
// default) disables recording, which then costs a nil check.
var (
	semantics   *[]SemanticNode
	scrollStack []string
	semanticIDs = map[any]string{}
	semanticSeq = map[SemanticKind]int{}
	// containerSeq numbers the frame's untargeted containers.
	containerSeq = map[SemanticKind]int{}
	semanticMu   sync.Mutex
)

// BeginSemantics starts recording the semantic tree for a frame. The
// app calls it when semantics are enabled (the MCP server enables
// them); component authors don't need to.
func BeginSemantics() {
	clear(containerSeq)
	nodes := make([]SemanticNode, 0, 64)
	semantics = &nodes
	scrollStack = scrollStack[:0]
}

// EndSemantics stops recording and returns the frame's nodes.
func EndSemantics() []SemanticNode {
	if semantics == nil {
		return nil
	}
	nodes := *semantics
	semantics = nil
	return nodes
}

// SemanticsEnabled reports whether the current frame records a
// semantic tree. Components can skip computing labels when it doesn't.
func SemanticsEnabled() bool {
	return semantics != nil
}

// RecordSemantic adds a node to the semantic tree of the frame being
// laid out, a no-op unless recording. Call it from a custom
// component's Layout, after laying out, with the component's size:
//
//	elements.RecordSemantic(gtx, elements.SemanticNode{
//		Kind: elements.SemanticButton, Label: b.label, Target: b,
//	}, dims.Size)
//
// Bounds and ID are filled in when left empty.
func RecordSemantic(gtx layout.Context, n SemanticNode, size image.Point) {
	if semantics == nil {
		return
	}
	if n.Bounds.Empty() {
		origin := AbsolutePosition()
		n.Bounds = image.Rectangle{Min: origin, Max: origin.Add(size)}
	}
	if n.ID == "" {
		n.ID = semanticID(n.Kind, n.Target)
	}
	if n.Scroll == "" && len(scrollStack) > 0 {
		n.Scroll = scrollStack[len(scrollStack)-1]
	}
	*semantics = append(*semantics, n)
}

// PushSemanticScroll marks the nodes recorded until pop is called as
// content of the scroll container target, which should then record
// itself as a SemanticScroll node with that Target. A scroll component
// calls it around laying out its content, so automation can scroll the
// container to reach a node:
//
//	pop := elements.PushSemanticScroll(s)
//	dims := s.content().Layout(cgtx)
//	pop()
//	...
//	elements.RecordSemantic(gtx, elements.SemanticNode{Kind: elements.SemanticScroll, Target: s}, viewport)
//
// For automation to scroll it, target should have the methods
// ScrollBy(dx, dy float32) and ScrollOffset() (x, y float32), in
// pixels.
func PushSemanticScroll(target any) (pop func()) {
	if semantics == nil {
		return func() {}
	}
	scrollStack = append(scrollStack, semanticID(SemanticScroll, target))
	n := len(scrollStack)
	return func() {
		if len(scrollStack) >= n {
			scrollStack = scrollStack[:n-1]
		}
	}
}

// recordSemantic records a node without a target.
func recordSemantic(gtx layout.Context, kind SemanticKind, label, value string, size image.Point) {
	if semantics == nil || label == "" {
		return
	}
	RecordSemantic(gtx, SemanticNode{Kind: kind, Label: label, Value: value}, size)
}

// RecordSemanticLabeled records a node without a target, skipping
// empty labels. Element packages use it for their routine text nodes.
func RecordSemanticLabeled(gtx layout.Context, kind SemanticKind, label, value string, size image.Point) {
	recordSemantic(gtx, kind, label, value, size)
}

// semanticsWanted is set while something reads the tree every frame:
// an accessibility bridge for an active screen reader.
var semanticsWanted atomic.Bool

// SetSemanticsWanted makes the app record the semantic tree every
// frame (true) or only when automation asks (false). Accessibility
// bridges set it while a screen reader is on, from any goroutine.
func SetSemanticsWanted(on bool) { semanticsWanted.Store(on) }

// SemanticsWanted reports whether a bridge reads the tree.
func SemanticsWanted() bool { return semanticsWanted.Load() }

// SemanticContainer is a container node being laid out: the nodes recorded
// until End become its children (their Parent is its ID).
type SemanticContainer struct {
	node  SemanticNode
	start int
	on    bool
	pop   func()
}

// BeginSemanticGroup starts a container node - a list, a table, a
// dialog, a region - around laying out its content:
//
//	g := elements.BeginSemanticGroup(elements.SemanticNode{Kind: elements.SemanticList, Label: "Inbox", Target: l})
//	dims := content.Layout(gtx)
//	g.End(gtx, dims.Size)
//
// Nodes recorded in between without a Parent get the group's ID.
func BeginSemanticGroup(n SemanticNode) *SemanticContainer {
	g := &SemanticContainer{node: n}
	if semantics == nil {
		return g
	}
	g.on = true
	if g.node.ID == "" {
		if n.Target == nil {
			// A group needs its ID before its children: one per frame
			// order, as untargeted nodes have.
			containerSeq[n.Kind]++
			g.node.ID = fmt.Sprintf("%s#%d", n.Kind, containerSeq[n.Kind])
		} else {
			g.node.ID = semanticID(n.Kind, n.Target)
		}
	}
	g.start = len(*semantics)
	return g
}

// SetLabel sets the group's label before End: a dialog titled by its
// content.
func (g *SemanticContainer) SetLabel(label string) {
	if g != nil {
		g.node.Label = label
	}
}

// End records the group with its size at the current position and
// makes the nodes recorded since Begin its children.
func (g *SemanticContainer) End(gtx layout.Context, size image.Point) {
	if g == nil || !g.on || semantics == nil {
		return
	}
	nodes := *semantics
	for i := g.start; i < len(nodes); i++ {
		if nodes[i].Parent == "" && nodes[i].ID != g.node.ID {
			nodes[i].Parent = g.node.ID
		}
	}
	RecordSemantic(gtx, g.node, size)
}

// ID is the group's node ID, "" when no tree is recorded.
func (g *SemanticContainer) ID() string {
	if g == nil {
		return ""
	}
	return g.node.ID
}

// accessible is Accessible's element.
type accessible struct {
	el Element
	fn func(n *SemanticNode)
}

// Accessible lays el out and passes each node it records to fn, to
// change its role or states: a text becoming a heading, a box's text a
// status message.
//
//	elements.Accessible(elements.Text("Settings", big), func(n *elements.SemanticNode) {
//		n.Kind, n.Level = elements.SemanticHeading, 1
//	})
func Accessible(el Element, fn func(n *SemanticNode)) Element {
	return accessible{el: el, fn: fn}
}

func (a accessible) Layout(gtx layout.Context) layout.Dimensions {
	if semantics == nil || a.fn == nil {
		return a.el.Layout(gtx)
	}
	start := len(*semantics)
	dims := a.el.Layout(gtx)
	nodes := *semantics
	for i := start; i < len(nodes); i++ {
		a.fn(&nodes[i])
	}
	return dims
}

// accessibleGroup is AccessibleGroup's element.
type accessibleGroup struct {
	n  SemanticNode
	el Element
}

// AccessibleGroup lays el out as a container node n - a region, a
// list, a dialog, a toolbar - whose children are el's nodes.
func AccessibleGroup(n SemanticNode, el Element) Element {
	return accessibleGroup{n: n, el: el}
}

func (g accessibleGroup) Layout(gtx layout.Context) layout.Dimensions {
	c := BeginSemanticGroup(g.n)
	dims := g.el.Layout(gtx)
	c.End(gtx, dims.Size)
	return dims
}
