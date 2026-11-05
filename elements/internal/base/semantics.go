package base

import (
	"image"
	"sync"

	"gioui.org/layout"
)

// semantics is the recorder for the frame being laid out; nil (the
// default) disables recording, which then costs a nil check.
var (
	semantics   *[]SemanticNode
	scrollStack []string
	semanticIDs = map[any]string{}
	semanticSeq = map[SemanticKind]int{}
	semanticMu  sync.Mutex
)

// BeginSemantics starts recording the semantic tree for a frame. The
// app calls it when semantics are enabled (the MCP server enables
// them); component authors don't need to.
func BeginSemantics() {
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
