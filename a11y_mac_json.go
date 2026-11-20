package giode

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/xmhrnx/giode/elements"
)

// The macOS bridge's view of a node: what its NSAccessibilityElement
// answers, as JSON for the Objective-C side (a11y_macos.m). Kept out of
// the cgo file so it's tested everywhere.

type macNode struct {
	ID       int32   `json:"id"`
	Parent   int32   `json:"parent"`
	Children []int32 `json:"children,omitempty"`
	// The frame in the view's pixels, from its top-left corner.
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`

	Role     string `json:"role,omitempty"`
	Subrole  string `json:"subrole,omitempty"`
	RoleDesc string `json:"roledesc,omitempty"`
	Label    string `json:"label,omitempty"`
	Value    any    `json:"value,omitempty"`
	Help     string `json:"help,omitempty"`
	// Element is whether VoiceOver stops on it; text inside a control
	// isn't, it's read as the control's.
	Element  bool    `json:"element"`
	Disabled bool    `json:"disabled,omitempty"`
	Focused  bool    `json:"focused,omitempty"`
	Selected bool    `json:"selected,omitempty"`
	Expanded int     `json:"expanded"` // -1 not expandable, 0 collapsed, 1 expanded
	Required bool    `json:"required,omitempty"`
	Invalid  bool    `json:"invalid,omitempty"`
	Range    bool    `json:"range,omitempty"`
	Min      float64 `json:"min,omitempty"`
	Max      float64 `json:"max,omitempty"`
	Index    int     `json:"index"` // place in its list from 0, -1 none
	Count    int     `json:"count,omitempty"`
	Level    int     `json:"level,omitempty"`
	Modal    bool    `json:"modal,omitempty"`
	// A text field's selection, [SelStart, SelEnd) in UTF-16 units as
	// NSRange counts them, when HasCaret.
	HasCaret bool `json:"hascaret,omitempty"`
	SelStart int  `json:"selstart"`
	SelEnd   int  `json:"selend"`
	Length   int  `json:"length"`

	// Actions it takes: "press", "increment", "decrement", "setvalue",
	// "focus".
	Actions []string `json:"actions,omitempty"`
}

// macRole is the AX role and subrole a kind is.
func macRole(n elements.SemanticNode) (role, subrole, desc string) {
	switch n.Kind {
	case elements.SemanticButton:
		return "AXButton", "", ""
	case elements.SemanticLink:
		return "AXLink", "", ""
	case elements.SemanticCheckbox:
		return "AXCheckBox", "", ""
	case elements.SemanticSwitch:
		return "AXCheckBox", "AXSwitch", ""
	case elements.SemanticRadio:
		return "AXRadioButton", "", ""
	case elements.SemanticInput:
		return "AXTextField", "", ""
	case elements.SemanticSlider:
		return "AXSlider", "", ""
	case elements.SemanticProgress:
		if n.Busy {
			return "AXBusyIndicator", "", ""
		}
		return "AXProgressIndicator", "", ""
	case elements.SemanticSelect:
		return "AXPopUpButton", "", ""
	case elements.SemanticComboBox:
		return "AXComboBox", "", ""
	case elements.SemanticTab:
		return "AXRadioButton", "AXTabButton", "tab"
	case elements.SemanticTabList:
		return "AXTabGroup", "", ""
	case elements.SemanticTabPanel:
		return "AXGroup", "", "tab panel"
	case elements.SemanticImage:
		return "AXImage", "", ""
	case elements.SemanticScroll:
		return "AXScrollArea", "", ""
	case elements.SemanticList:
		return "AXList", "", ""
	case elements.SemanticListItem:
		return "AXGroup", "", "list item"
	case elements.SemanticTable:
		return "AXTable", "", ""
	case elements.SemanticRow:
		return "AXRow", "", ""
	case elements.SemanticCell:
		return "AXCell", "", ""
	case elements.SemanticColumnHeader:
		return "AXButton", "AXSortButton", "column header"
	case elements.SemanticTree:
		return "AXOutline", "", ""
	case elements.SemanticTreeItem:
		return "AXRow", "AXOutlineRow", ""
	case elements.SemanticMenu:
		return "AXMenu", "", ""
	case elements.SemanticMenuBar:
		return "AXMenuBar", "", ""
	case elements.SemanticMenuItem:
		return "AXMenuItem", "", ""
	case elements.SemanticToolbar:
		return "AXToolbar", "", ""
	case elements.SemanticHeading:
		return "AXHeading", "", "heading"
	case elements.SemanticDialog:
		return "AXGroup", "AXApplicationDialog", "dialog"
	case elements.SemanticAlert:
		return "AXGroup", "AXApplicationAlert", "alert"
	case elements.SemanticStatus:
		return "AXGroup", "AXApplicationStatus", "status"
	case elements.SemanticRegion:
		return "AXGroup", "AXLandmarkRegion", "region"
	case elements.SemanticGroup:
		return "AXGroup", "", ""
	}
	return "AXStaticText", "", ""
}

// macJSON describes node k (0 the root, the view) for the macOS bridge.
func (t *a11yTree) macJSON(k int32) string {
	n, parent, children, ok := t.node(k)
	if !ok {
		return ""
	}
	out := macNode{ID: k, Parent: parent, Children: children, Expanded: -1, Index: -1}
	if k != 0 {
		out.Element = t.isFocusable(n, k) || !textual(n)
		fillMacNode(&out, n)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func fillMacNode(out *macNode, n elements.SemanticNode) {
	r := n.Bounds
	out.X, out.Y, out.W, out.H = r.Min.X, r.Min.Y, r.Dx(), r.Dy()
	out.Role, out.Subrole, out.RoleDesc = macRole(n)
	out.Help = n.Description
	out.Disabled, out.Focused, out.Selected = n.Disabled, n.Focused, n.Selected
	out.Required, out.Invalid = n.Required, n.Invalid
	out.Modal = n.Kind == elements.SemanticDialog
	switch n.Expanded {
	case elements.Expanded:
		out.Expanded = 1
	case elements.Collapsed:
		out.Expanded = 0
	}
	if n.Pos > 0 {
		out.Index = n.Pos - 1
	}
	out.Count = n.Count
	if n.Kind != elements.SemanticHeading {
		out.Level = n.Level
	}
	switch {
	case n.Checked != elements.NotCheckable:
		out.Label = n.Label
		v := 0
		switch n.Checked {
		case elements.Checked:
			v = 1
		case elements.Mixed:
			v = 2
		}
		out.Value = v
	case n.Kind == elements.SemanticSlider || (n.Kind == elements.SemanticProgress && !n.Busy):
		out.Label = n.Label
		out.Range, out.Min, out.Max = true, n.Min, n.Max
		out.Value = n.Now
	case n.Kind == elements.SemanticHeading:
		out.Label = a11yText(n)
		out.Value = max(n.Level, 1)
	case n.Kind == elements.SemanticInput, n.Kind == elements.SemanticComboBox, n.Kind == elements.SemanticSelect:
		out.Label = n.Label
		out.Value = n.Value
	case n.Kind == elements.SemanticRadio || n.Kind == elements.SemanticTab:
		out.Label = a11yText(n)
		out.Value = 0
		if n.Selected {
			out.Value = 1
		}
	case textual(n) && n.Label == "":
		out.Value = n.Value
	case textual(n):
		out.Value = n.Label
	default:
		out.Label = a11yText(n)
	}
	if start, end, ok := a11ySelection(n); ok {
		out.HasCaret = true
		out.SelStart, out.SelEnd = utf16Offset(n.Value, start), utf16Offset(n.Value, end)
		out.Length = utf16Offset(n.Value, len([]rune(n.Value)))
	}
	activate, _, setText, focus := a11yActions(n)
	if activate {
		out.Actions = append(out.Actions, "press")
	}
	if n.Kind == elements.SemanticSlider && !n.Disabled {
		if _, ok := n.Target.(a11yStepper); ok {
			out.Actions = append(out.Actions, "increment", "decrement")
		}
	}
	if setText {
		out.Actions = append(out.Actions, "setvalue")
	}
	if focus {
		out.Actions = append(out.Actions, "focus")
	}
	if _, ok := n.Target.(a11ySelectable); ok && n.HasCaret {
		out.Actions = append(out.Actions, "select")
	}
}

// utf16Offset returns the UTF-16 length of s's first runes runes.
func utf16Offset(s string, runes int) int {
	n := 0
	for i, r := range []rune(s) {
		if i >= runes {
			break
		}
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// runeOffset returns how many runes of s the first u UTF-16 units span.
func runeOffset(s string, u int) int {
	n, i := 0, 0
	for _, r := range s {
		if n >= u {
			break
		}
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
		i++
	}
	return i
}

// macAction maps a bridge action name to a11yAction.
func macAction(name string) a11yAction {
	switch name {
	case "press":
		return a11yActivate
	case "increment":
		return a11yIncrement
	case "decrement":
		return a11yDecrement
	case "setvalue":
		return a11ySetText
	case "focus":
		return a11yFocus
	case "select":
		return a11ySetSelection
	case "expand":
		return a11yExpand
	case "collapse":
		return a11yCollapse
	}
	return 0
}

// utf16RangeToRunes turns "start,end" in UTF-16 units of node k's value
// into runes.
func utf16RangeToRunes(t *a11yTree, k int32, arg string) string {
	n, _, _, ok := t.node(k)
	a, b, found := strings.Cut(arg, ",")
	start, err1 := strconv.Atoi(a)
	end, err2 := strconv.Atoi(b)
	if !ok || !found || err1 != nil || err2 != nil {
		return arg
	}
	return strconv.Itoa(runeOffset(n.Value, start)) + "," + strconv.Itoa(runeOffset(n.Value, end))
}
