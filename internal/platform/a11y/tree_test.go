package a11y

import (
	"encoding/json"
	"image"
	"testing"

	"github.com/xmhrnx/giode/elements"
)

type a11yClicker struct{ n int }

func (c *a11yClicker) Click() { c.n++ }

func TestA11yTree(t *testing.T) {
	btn := &a11yClicker{}
	nodes := []elements.SemanticNode{
		{ID: "text@0", Kind: elements.SemanticText, Label: "Save", Bounds: image.Rect(12, 12, 60, 28)},
		{ID: "button-1", Kind: elements.SemanticButton, Label: "", Bounds: image.Rect(0, 0, 100, 40), Target: btn},
		{ID: "heading@2", Kind: elements.SemanticHeading, Label: "Settings", Level: 1, Bounds: image.Rect(0, 50, 200, 80), Parent: "dialog#1"},
		{ID: "checkbox-1", Kind: elements.SemanticCheckbox, Label: "Remember", Checked: elements.Checked, Bounds: image.Rect(0, 90, 200, 120), Parent: "dialog#1"},
		{ID: "dialog#1", Kind: elements.SemanticDialog, Label: "Preferences", Bounds: image.Rect(0, 45, 220, 130)},
	}
	tr := NewTree()
	tr.Update(nodes)
	num := func(id string) int32 { return tr.num[id] }

	// The text inside the button is the button's; the dialog holds the
	// heading and the checkbox.
	if tr.parent[num("text@0")] != num("button-1") {
		t.Errorf("text's parent %d, want the button %d", tr.parent[num("text@0")], num("button-1"))
	}
	if tr.parent[num("checkbox-1")] != num("dialog#1") || tr.parent[num("dialog#1")] != 0 {
		t.Error("dialog nesting wrong")
	}
	// A touch on the button's text explores the button.
	if got := tr.at(image.Pt(20, 20)); got != num("button-1") {
		t.Errorf("at the button: %d, want %d", got, num("button-1"))
	}
	if got := tr.at(image.Pt(300, 300)); got != 0 {
		t.Errorf("at nothing: %d", got)
	}

	var cb androidNode
	json.Unmarshal([]byte(tr.androidJSON(num("checkbox-1"))), &cb)
	if cb.Class != "android.widget.CheckBox" || !cb.Checkable || !cb.Checked || cb.Text != "Remember" || cb.Parent != num("dialog#1") {
		t.Errorf("checkbox %+v", cb)
	}
	var h androidNode
	json.Unmarshal([]byte(tr.androidJSON(num("heading@2"))), &h)
	if !h.Heading || h.Text != "Settings" {
		t.Errorf("heading %+v", h)
	}
	var d androidNode
	json.Unmarshal([]byte(tr.androidJSON(num("dialog#1"))), &d)
	if d.Pane != "Preferences" || len(d.Children) != 2 {
		t.Errorf("dialog %+v", d)
	}
	var root androidNode
	json.Unmarshal([]byte(tr.androidJSON(0)), &root)
	if len(root.Children) != 2 {
		t.Errorf("root children %v, want the button and the dialog", root.Children)
	}

	// Numbers stay; changes are reported; a new alert is announced.
	ch, _ := tr.takeChanges()
	if len(ch) == 0 {
		t.Error("no changes on the first frame")
	}
	before := num("checkbox-1")
	nodes[3].Checked = elements.Unchecked
	nodes = append(nodes, elements.SemanticNode{ID: "alert@5", Kind: elements.SemanticAlert, Label: "Couldn't save", Bounds: image.Rect(0, 140, 200, 160)})
	tr.Update(nodes)
	if num("checkbox-1") != before {
		t.Error("the checkbox's number changed")
	}
	ch, ann := tr.takeChanges()
	found := false
	for _, k := range ch {
		if k == before {
			found = true
		}
	}
	if !found || len(ann) != 1 || ann[0].text != "Couldn't save" || !ann[0].assertive {
		t.Errorf("changes %v, announcements %+v", ch, ann)
	}
	tr.Update(nodes)
	if ch, ann := tr.takeChanges(); len(ch) != 0 || len(ann) != 0 {
		t.Errorf("an unchanged frame reported %v %v", ch, ann)
	}
}
