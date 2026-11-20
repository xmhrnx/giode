package giode

import (
	"encoding/json"
	"image"
	"slices"
	"testing"

	"github.com/xmhrnx/giode/elements"
)

type macStepper struct{ v float32 }

func (s *macStepper) Value() float32   { return s.v }
func (s *macStepper) Change(v float32) { s.v = v }

func TestMacNodes(t *testing.T) {
	tree := newA11yTree()
	tree.update([]elements.SemanticNode{
		{ID: "sw", Kind: elements.SemanticSwitch, Label: "Wi-Fi", Checked: elements.Checked, Bounds: image.Rect(0, 0, 100, 30)},
		{ID: "vol", Kind: elements.SemanticSlider, Label: "Volume", Min: 0, Max: 1, Now: 0.4, Target: &macStepper{}, Bounds: image.Rect(0, 40, 200, 60)},
		{ID: "h", Kind: elements.SemanticHeading, Label: "Network", Level: 2, Bounds: image.Rect(0, 70, 200, 90)},
		{ID: "main", Kind: elements.SemanticRegion, Label: "Main content", Bounds: image.Rect(0, 100, 300, 400)},
	})
	get := func(id string) macNode {
		var n macNode
		if err := json.Unmarshal([]byte(tree.macJSON(tree.num[id])), &n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	sw := get("sw")
	if sw.Role != "AXCheckBox" || sw.Subrole != "AXSwitch" || sw.Value != 1.0 || sw.Label != "Wi-Fi" || !sw.Element {
		t.Errorf("switch = %+v", sw)
	}
	vol := get("vol")
	if vol.Role != "AXSlider" || !vol.Range || vol.Max != 1 || vol.Value != 0.4 || !slices.Contains(vol.Actions, "increment") {
		t.Errorf("slider = %+v", vol)
	}
	if h := get("h"); h.Role != "AXHeading" || h.Value != 2.0 || h.Label != "Network" {
		t.Errorf("heading = %+v", h)
	}
	if m := get("main"); m.Subrole != "AXLandmarkRegion" || m.Label != "Main content" {
		t.Errorf("region = %+v", m)
	}
	var root macNode
	json.Unmarshal([]byte(tree.macJSON(0)), &root)
	if len(root.Children) != 4 {
		t.Errorf("root children = %v", root.Children)
	}
	if macAction("increment") != a11yIncrement || macAction("nope") != 0 {
		t.Error("macAction")
	}
}
