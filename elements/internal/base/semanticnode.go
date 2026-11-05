package base

import "image"

// SemanticNode describes one meaningful piece of the UI - a text run,
// a button, a field - as laid out in the last frame. The semantic tree
// is what automation (the built-in MCP server, tests) sees of the UI.
type SemanticNode struct {
	// ID identifies the node. Nodes with a Target keep their ID for
	// the target's lifetime; others are numbered in layout order.
	ID string `json:"id"`
	// Kind classifies the node.
	Kind SemanticKind `json:"kind"`
	// Label is the visible text or accessible name.
	Label string `json:"label,omitempty"`
	// Value is the current value of a control (field text, checked
	// state, slider position).
	Value string `json:"value,omitempty"`
	// Bounds is the node's window-relative area, in pixels, including
	// for children of rows, stacks and scrolled content.
	Bounds image.Rectangle `json:"bounds"`
	// Disabled marks a control that doesn't accept input.
	Disabled bool `json:"disabled,omitempty"`
	// Scroll is the ID of the innermost scroll container the node is
	// laid out in, or "" outside any. A node in a scroll container may
	// lie outside the container's Bounds, scrolled out of view.
	Scroll string `json:"scroll,omitempty"`
	// Target is the stateful component behind the node, if any, so
	// automation can act on it (e.g. call Click). It is not
	// serialized.
	Target any `json:"-"`
}

// SemanticKind classifies a node of the semantic tree.
type SemanticKind string

const (
	SemanticText     SemanticKind = "text"
	SemanticButton   SemanticKind = "button"
	SemanticInput    SemanticKind = "input"
	SemanticCheckbox SemanticKind = "checkbox"
	SemanticSlider   SemanticKind = "slider"
	SemanticSelect   SemanticKind = "select"
	SemanticTab      SemanticKind = "tab"
	SemanticMenuItem SemanticKind = "menuitem"
	SemanticLink     SemanticKind = "link"
	SemanticImage    SemanticKind = "image"
	SemanticScroll   SemanticKind = "scroll"
	SemanticCustom   SemanticKind = "custom"
)
