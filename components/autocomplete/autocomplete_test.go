package autocomplete

import (
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
)

func TestAutoComplete_FiltersAsTyped(t *testing.T) {
	elements.SetWindowSize(image.Pt(400, 400))
	defer elements.SetWindowSize(image.Point{})

	a := New("Fruit", []string{"apple", "apricot", "banana", "cherry"})

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(200, 40))}

	frame := func() {
		ops.Reset()
		a.Layout(gtx)
		elements.DrainPortals(gtx)
		router.Frame(&ops)
	}

	frame() // register the field

	// A keystroke reaching the underlying editor is what a real user
	// typing would produce; a is otherwise a black box, so this pokes
	// the field directly the way the router would after processing a
	// key.Event.
	a.field.SetText("ap")
	frame()
	frame()
	if got := a.filtered; len(got) != 2 {
		t.Fatalf("filtered for %q = %v, want 2 matches (apple, apricot)", "ap", got)
	}

	a.field.SetText("apr")
	frame()
	frame()
	if got := a.filtered; len(got) != 1 || got[0] != "apricot" {
		t.Errorf("filtered for %q = %v, want [apricot]", "apr", got)
	}
}

func TestAutoComplete_SetTextSuppressesPanel(t *testing.T) {
	elements.SetWindowSize(image.Pt(400, 400))
	defer elements.SetWindowSize(image.Point{})

	a := New("Fruit", []string{"apple", "apricot"})

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(200, 40))}

	frame := func() {
		ops.Reset()
		a.Layout(gtx)
		elements.DrainPortals(gtx)
		router.Frame(&ops)
	}

	frame()
	// SetText is meant for programmatic resets (an external value, a
	// selection already made) rather than the user typing, so it
	// should not reopen the panel even though "apple" matches itself.
	a.SetText("apple")
	frame()
	frame()
	if got := a.filtered; len(got) != 0 {
		t.Errorf("filtered after SetText = %v, want none (suppressed)", got)
	}

	// A further real edit un-suppresses it again.
	a.field.SetText("app")
	frame()
	frame()
	if got := a.filtered; len(got) != 1 || got[0] != "apple" {
		t.Errorf("filtered after a real edit = %v, want [apple]", got)
	}
}

func TestAutoComplete_SelectSetsTextAndSuppresses(t *testing.T) {
	elements.SetWindowSize(image.Pt(400, 400))
	defer elements.SetWindowSize(image.Point{})

	var selected string
	a := New("Fruit", []string{"apple", "apricot", "banana"}).OnSelect(func(v string) { selected = v })

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(200, 40))}

	frame := func() {
		ops.Reset()
		a.Layout(gtx)
		elements.DrainPortals(gtx)
		router.Frame(&ops)
	}

	frame()
	a.field.SetText("ap")
	frame()
	frame()
	if len(a.filtered) != 2 {
		t.Fatalf("filtered = %v, want 2 matches (apple, apricot)", a.filtered)
	}

	// Exercise the click-to-select path the same way a real pointer
	// click on the first suggestion row would.
	if len(a.itemClicks) == 0 {
		t.Fatal("no suggestion rows to select from")
	}
	a.itemClicks[0].Click()
	frame()

	if selected != "apple" {
		t.Errorf("OnSelect fired with %q, want %q", selected, "apple")
	}
	if a.Text() != "apple" {
		t.Errorf("Text() = %q after selection, want %q", a.Text(), "apple")
	}

	frame()
	if len(a.filtered) != 0 {
		t.Errorf("filtered after selection = %v, want none (suppressed until the next real edit)", a.filtered)
	}
}

func TestAutoComplete_EmptyTextShowsNoSuggestions(t *testing.T) {
	elements.SetWindowSize(image.Pt(400, 400))
	defer elements.SetWindowSize(image.Point{})

	a := New("Fruit", []string{"apple", "apricot"})

	var ops op.Ops
	var router input.Router
	gtx := layout.Context{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(image.Pt(200, 40))}

	ops.Reset()
	a.Layout(gtx)
	elements.DrainPortals(gtx)
	router.Frame(&ops)

	if len(a.filtered) != 0 {
		t.Errorf("filtered with empty text = %v, want none", a.filtered)
	}
}
