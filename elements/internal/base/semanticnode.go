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

	// Description is longer help read after the label (a hint, an error).
	Description string `json:"description,omitempty"`
	// Level is a heading's level, 1 to 6, or a tree item's depth from 1.
	Level int `json:"level,omitempty"`
	// Checked is a checkbox's, switch's or radio button's state.
	Checked CheckState `json:"checked,omitempty"`
	// Selected marks a selected tab, list item, row or option.
	Selected bool `json:"selected,omitempty"`
	// Expanded is an expandable control's state: a tree item, an
	// accordion section, a dropdown, a menu button.
	Expanded ExpandState `json:"expanded,omitempty"`
	// Required, Invalid and Busy mark a field that must be filled, one
	// whose value is wrong, and a part of the UI still loading.
	Required bool `json:"required,omitempty"`
	Invalid  bool `json:"invalid,omitempty"`
	Busy     bool `json:"busy,omitempty"`
	// Focused marks the node with keyboard focus.
	Focused bool `json:"focused,omitempty"`
	// Live makes changes to the node's label or value announced:
	// LivePolite after what's being read, LiveAssertive at once (an
	// error). Toasts, new chat messages and validation errors use it.
	Live LiveMode `json:"live,omitempty"`
	// Min, Max and Now are a slider's or progress bar's range and value.
	Min, Max, Now float64 `json:"-"`
	// Parent is the ID of the group, list, table, dialog or other
	// container the node is in, "" at the top.
	Parent string `json:"parent,omitempty"`
	// LabelledBy, DescribedBy and Controls relate the node to others by
	// ID: the nodes naming it, describing it, and those it controls (a
	// tab its panel, a combobox its list).
	LabelledBy  []string `json:"labelledBy,omitempty"`
	DescribedBy []string `json:"describedBy,omitempty"`
	Controls    []string `json:"controls,omitempty"`
	// Pos and Count are an item's place in its list or set, from 1, when
	// the list shows only some items (virtualized).
	Pos, Count int `json:"-"`
	// HasCaret marks a text field showing its caret: Caret is where it
	// is and Anchor the selection's other end, in runes from the start
	// of Value; Anchor == Caret is no selection.
	HasCaret      bool `json:"-"`
	Caret, Anchor int  `json:"-"`
}

// CheckState is a checkable control's state.
type CheckState uint8

const (
	// NotCheckable is a node that isn't checkable.
	NotCheckable CheckState = iota
	Unchecked
	Checked
	// Mixed is a checkbox for a partly selected group.
	Mixed
)

// ExpandState is an expandable control's state.
type ExpandState uint8

const (
	// NotExpandable is a node that doesn't expand.
	NotExpandable ExpandState = iota
	Collapsed
	Expanded
)

// LiveMode says how changes to a node are announced.
type LiveMode string

const (
	LiveOff       LiveMode = ""
	LivePolite    LiveMode = "polite"
	LiveAssertive LiveMode = "assertive"
)

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

	SemanticHeading  SemanticKind = "heading"
	SemanticSwitch   SemanticKind = "switch"
	SemanticRadio    SemanticKind = "radio"
	SemanticProgress SemanticKind = "progress"
	SemanticGroup    SemanticKind = "group"
	SemanticList     SemanticKind = "list"
	SemanticListItem SemanticKind = "listitem"
	SemanticTable    SemanticKind = "table"
	SemanticRow      SemanticKind = "row"
	SemanticCell     SemanticKind = "cell"
	// SemanticColumnHeader is a table's column header cell.
	SemanticColumnHeader SemanticKind = "columnheader"
	SemanticDialog       SemanticKind = "dialog"
	// SemanticAlert is an urgent message: a dialog asking to confirm,
	// an error banner. Its text is announced when it appears.
	SemanticAlert    SemanticKind = "alert"
	SemanticStatus   SemanticKind = "status"
	SemanticMenu     SemanticKind = "menu"
	SemanticMenuBar  SemanticKind = "menubar"
	SemanticTabList  SemanticKind = "tablist"
	SemanticTabPanel SemanticKind = "tabpanel"
	SemanticTree     SemanticKind = "tree"
	SemanticTreeItem SemanticKind = "treeitem"
	SemanticComboBox SemanticKind = "combobox"
	SemanticToolbar  SemanticKind = "toolbar"
	// SemanticRegion is a landmark: a large part of the window a skip
	// link or a screen reader's landmark list goes to.
	SemanticRegion SemanticKind = "region"
)
