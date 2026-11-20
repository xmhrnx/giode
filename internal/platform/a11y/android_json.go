package a11y

import (
	"encoding/json"

	"github.com/xmhrnx/giode/elements"
)

// Android's AccessibilityNodeInfo action IDs.
const (
	androidActionFocus          = 1
	androidActionClick          = 16
	androidActionScrollForward  = 4096
	androidActionScrollBackward = 8192
	androidActionExpand         = 262144
	androidActionCollapse       = 524288
	androidActionSetText        = 2097152
	androidActionSetSelection   = 131072
)

// androidNode is a node as giode's patched GioView reads it to fill an
// AccessibilityNodeInfo.
type androidNode struct {
	ID       int32   `json:"id"`
	Parent   int32   `json:"parent"`
	Children []int32 `json:"children,omitempty"`
	L        int     `json:"l"`
	T        int     `json:"t"`
	R        int     `json:"r"`
	B        int     `json:"b"`
	Class    string  `json:"cls,omitempty"`
	Text     string  `json:"text,omitempty"`
	Desc     string  `json:"desc,omitempty"`
	Hint     string  `json:"hint,omitempty"`
	State    string  `json:"state,omitempty"`
	Error    string  `json:"error,omitempty"`
	Pane     string  `json:"pane,omitempty"`

	Checkable bool `json:"checkable,omitempty"`
	Checked   bool `json:"checked,omitempty"`
	Selected  bool `json:"selected,omitempty"`
	Disabled  bool `json:"disabled,omitempty"`
	Focusable bool `json:"focusable,omitempty"`
	Focused   bool `json:"focused,omitempty"`
	Clickable bool `json:"clickable,omitempty"`
	Scroll    bool `json:"scrollable,omitempty"`
	Heading   bool `json:"heading,omitempty"`
	Editable  bool `json:"editable,omitempty"`
	Password  bool `json:"password,omitempty"`
	Important bool `json:"important,omitempty"`
	// Live is 0, 1 polite, 2 assertive.
	Live int `json:"live,omitempty"`

	Range bool    `json:"range,omitempty"`
	Min   float64 `json:"min,omitempty"`
	Max   float64 `json:"max,omitempty"`
	Now   float64 `json:"now,omitempty"`

	// Item is a place in a list: Row from 0 when ItemSet.
	ItemSet bool `json:"item,omitempty"`
	Row     int  `json:"row,omitempty"`

	Actions []int `json:"actions,omitempty"`

	// A text field's selection in UTF-16 units, as Android counts.
	SelStart *int `json:"selStart,omitempty"`
	SelEnd   *int `json:"selEnd,omitempty"`
}

// androidClass is the Android widget class TalkBack describes a kind as.
func androidClass(k elements.SemanticKind) string {
	switch k {
	case elements.SemanticButton, elements.SemanticMenuItem:
		return "android.widget.Button"
	case elements.SemanticCheckbox:
		return "android.widget.CheckBox"
	case elements.SemanticSwitch:
		return "android.widget.Switch"
	case elements.SemanticRadio:
		return "android.widget.RadioButton"
	case elements.SemanticInput:
		return "android.widget.EditText"
	case elements.SemanticSlider:
		return "android.widget.SeekBar"
	case elements.SemanticProgress:
		return "android.widget.ProgressBar"
	case elements.SemanticSelect, elements.SemanticComboBox:
		return "android.widget.Spinner"
	case elements.SemanticTab:
		return "android.app.ActionBar$Tab"
	case elements.SemanticImage:
		return "android.widget.ImageView"
	case elements.SemanticScroll:
		return "android.widget.ScrollView"
	case elements.SemanticList, elements.SemanticTree, elements.SemanticMenu:
		return "android.widget.ListView"
	case elements.SemanticTable:
		return "android.widget.GridView"
	case elements.SemanticTabList:
		return "android.widget.TabWidget"
	case elements.SemanticText, elements.SemanticHeading, elements.SemanticStatus, elements.SemanticAlert, elements.SemanticLink:
		return "android.widget.TextView"
	}
	return "android.view.View"
}

// androidJSON describes node k (0 the root) for the patched GioView.
func (t *Tree) androidJSON(k int32) string {
	n, parent, children, ok := t.node(k)
	if !ok {
		return ""
	}
	out := androidNode{ID: k, Parent: parent, Children: children}
	if k != 0 {
		t.mu.Lock()
		focusable := t.focusable(n, k)
		t.mu.Unlock()
		fillAndroidNode(&out, n, focusable)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func fillAndroidNode(out *androidNode, n elements.SemanticNode, focusable bool) {
	r := n.Bounds
	out.L, out.T, out.R, out.B = r.Min.X, r.Min.Y, r.Max.X, r.Max.Y
	out.Class = androidClass(n.Kind)
	out.Disabled = n.Disabled
	out.Focused = n.Focused
	out.Selected = n.Selected
	out.Important = focusable
	out.Focusable = focusable && actionable(n)
	switch n.Kind {
	case elements.SemanticInput:
		out.Editable = true
		out.Text = n.Value
		out.Hint = n.Label
		if n.Required {
			out.Hint += ", required"
		}
	default:
		out.Text = n.Label
		if out.Text == "" {
			out.Text = n.Value
		}
	}
	out.Desc = n.Description
	if n.Invalid {
		out.Error = n.Description
		if out.Error == "" {
			out.Error = "Invalid"
		}
	}
	if n.Checked != elements.NotCheckable {
		out.Checkable = true
		out.Checked = n.Checked == elements.Checked
		if n.Checked == elements.Mixed {
			out.State = "partially checked"
		}
	}
	switch n.Expanded {
	case elements.Expanded:
		out.State = "expanded"
	case elements.Collapsed:
		out.State = "collapsed"
	}
	if n.Busy {
		out.State = "busy"
	}
	if n.Kind == elements.SemanticHeading {
		out.Heading = true
	}
	if n.Kind == elements.SemanticDialog {
		out.Pane = n.Label
	}
	switch n.Live {
	case elements.LivePolite:
		out.Live = 1
	case elements.LiveAssertive:
		out.Live = 2
	}
	if n.Kind == elements.SemanticAlert && out.Live == 0 {
		out.Live = 2
	}
	if n.Kind == elements.SemanticSlider || (n.Kind == elements.SemanticProgress && !n.Busy) {
		out.Range, out.Min, out.Max, out.Now = true, n.Min, n.Max, n.Now
	}
	if n.Pos > 0 {
		out.ItemSet, out.Row = true, n.Pos-1
	}
	activate, scroll, setText, focus := a11yActions(n)
	if activate {
		out.Clickable = true
		out.Actions = append(out.Actions, androidActionClick)
	}
	if n.Expanded == elements.Collapsed {
		out.Actions = append(out.Actions, androidActionExpand)
	}
	if n.Expanded == elements.Expanded {
		out.Actions = append(out.Actions, androidActionCollapse)
	}
	if scroll {
		out.Scroll = true
		out.Actions = append(out.Actions, androidActionScrollForward, androidActionScrollBackward)
	}
	if setText && n.Kind == elements.SemanticInput {
		out.Actions = append(out.Actions, androidActionSetText)
	}
	if focus {
		out.Actions = append(out.Actions, androidActionFocus)
	}
	if start, end, ok := a11ySelection(n); ok {
		s, e := utf16Offset(n.Value, start), utf16Offset(n.Value, end)
		if n.Caret < n.Anchor {
			s, e = e, s
		}
		out.SelStart, out.SelEnd = &s, &e
		if _, ok := n.Target.(a11ySelectable); ok {
			out.Actions = append(out.Actions, androidActionSetSelection)
		}
	}
}

// androidSelection returns the focused text field and its selection in
// UTF-16 units (start the anchor, end the caret) and its length, for
// TalkBack to read what the caret moves over.
func (t *Tree) androidSelection() (id int32, start, end, count int, ok bool) {
	k := t.focusedNode()
	n, _, _, found := t.node(k)
	if !found || !n.HasCaret {
		return 0, 0, 0, 0, false
	}
	return k, utf16Offset(n.Value, n.Anchor), utf16Offset(n.Value, n.Caret), utf16Offset(n.Value, len([]rune(n.Value))), true
}

// androidAction maps an Android action ID to a11yAction.
func androidAction(id int) a11yAction {
	switch id {
	case androidActionClick:
		return a11yActivate
	case androidActionScrollForward:
		return a11yScrollForward
	case androidActionScrollBackward:
		return a11yScrollBackward
	case androidActionSetText:
		return a11ySetText
	case androidActionExpand:
		return a11yExpand
	case androidActionCollapse:
		return a11yCollapse
	case androidActionFocus:
		return a11yFocus
	case androidActionSetSelection:
		return a11ySetSelection
	}
	return 0
}
