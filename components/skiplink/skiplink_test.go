package skiplink_test

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/components/button"
	"github.com/xmhrnx/giode/components/skiplink"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/ui"
)

func TestSkipLinkJumpsToLandmark(t *testing.T) {
	var r input.Router
	nav1, nav2 := button.New("Home"), button.New("Settings")
	first := button.New("Compose")
	main := skiplink.NewLandmark("Main content")
	skip := skiplink.New("Skip to main content", main)
	now := time.Unix(0, 0)
	var sems []elements.SemanticNode
	frame := func() {
		now = now.Add(16 * time.Millisecond)
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Now: now, Source: r.Source(), Constraints: layout.Exact(image.Pt(600, 400))}
		elements.BeginSemantics()
		ui.Frame(gtx, elements.Stack(styles.Styles{},
			skip, nav1, nav2, main.Wrap(elements.Stack(styles.Styles{}, first))))
		sems = elements.EndSemantics()
		r.Frame(&ops)
	}
	frame()
	// The first Tab focuses the first control: the skip link.
	r.MoveFocus(key.FocusForward)
	frame()
	frame()
	if !skip.Focused() {
		t.Fatal("first Tab didn't focus the skip link")
	}
	var link, region bool
	for _, n := range sems {
		if n.Kind == elements.SemanticLink && n.Label == "Skip to main content" && n.Bounds.Dx() > 1 {
			link = true
		}
		if n.Kind == elements.SemanticRegion && n.Label == "Main content" {
			region = true
		}
	}
	if !link || !region {
		t.Errorf("semantics: link shown %v, region %v", link, region)
	}
	r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	frame()
	frame()
	if !first.Focused() {
		t.Fatal("Enter didn't move focus to the landmark's first control")
	}
	if skip.Focused() {
		t.Error("skip link still focused")
	}
}
