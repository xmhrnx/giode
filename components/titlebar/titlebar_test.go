package titlebar

import (
	"image"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

type fakeWindow struct {
	minimized, closed int
	maximized         bool
	focused           bool
}

func (w *fakeWindow) Minimize()       { w.minimized++ }
func (w *fakeWindow) ToggleMaximize() { w.maximized = !w.maximized }
func (w *fakeWindow) Close()          { w.closed++ }
func (w *fakeWindow) Maximized() bool { return w.maximized }
func (w *fakeWindow) Focused() bool   { return w.focused }

func gtxFor(w, h int, scale float32) layout.Context {
	return layout.Context{
		Ops:         new(op.Ops),
		Metric:      unit.Metric{PxPerDp: scale, PxPerSp: scale},
		Constraints: layout.Constraints{Max: image.Pt(w, h)},
	}
}

func TestPress(t *testing.T) {
	w := &fakeWindow{}
	bar := New(w, "App")
	bar.Press(ButtonMinimize)
	bar.Press(ButtonMaximize)
	bar.Press(ButtonClose)
	if w.minimized != 1 || !w.maximized || w.closed != 1 {
		t.Errorf("after pressing every button: %+v", w)
	}
	bar.Press(ButtonMaximize)
	if w.maximized {
		t.Error("second maximize press didn't restore")
	}
}

func TestOnCloseVeto(t *testing.T) {
	w := &fakeWindow{}
	allow := false
	bar := New(w, "App").OnClose(func() bool { return allow })
	bar.Press(ButtonClose)
	if w.closed != 0 {
		t.Fatal("OnClose returning false didn't keep the window open")
	}
	allow = true
	bar.Press(ButtonClose)
	if w.closed != 1 {
		t.Fatal("OnClose returning true didn't close")
	}
}

func TestLayoutSizeAndButtons(t *testing.T) {
	w := &fakeWindow{focused: true}
	elements.BeginSemantics()
	dims := New(w, "My app").Layout(gtxFor(800, 600, 2))
	nodes := elements.EndSemantics()
	if dims.Size != image.Pt(800, 72) {
		t.Errorf("size = %v, want full width x 36dp at 2x", dims.Size)
	}
	labels := map[string]image.Rectangle{}
	for _, n := range nodes {
		if n.Kind == elements.SemanticButton {
			labels[n.Label] = n.Bounds
		}
	}
	for _, l := range []string{"Minimize", "Maximize", "Close"} {
		if _, ok := labels[l]; !ok {
			t.Errorf("no %s button in %v", l, labels)
		}
	}
	if c := labels["Close"]; c.Max.X != 800 || c.Dx() != 92 {
		t.Errorf("close button at %v, want flush right, 46dp wide", c)
	}
}

func TestMaximizedShowsRestore(t *testing.T) {
	w := &fakeWindow{maximized: true}
	elements.BeginSemantics()
	New(w, "App").Layout(gtxFor(600, 400, 1))
	found := false
	for _, n := range elements.EndSemantics() {
		if n.Label == "Restore" {
			found = true
		}
	}
	if !found {
		t.Error("maximized window: no Restore button")
	}
}

func TestHideAndMacVariant(t *testing.T) {
	w := &fakeWindow{}
	elements.BeginSemantics()
	dims := New(w, "App", styles.Styles{Height: 40}).Variant(VariantMac).Hide(ButtonMaximize).Layout(gtxFor(600, 400, 1))
	nodes := elements.EndSemantics()
	if dims.Size.Y != 40 {
		t.Errorf("height = %d, want the 40dp override", dims.Size.Y)
	}
	var buttons []string
	for _, n := range nodes {
		if n.Kind == elements.SemanticButton {
			buttons = append(buttons, n.Label)
			if n.Bounds.Min.X > 100 {
				t.Errorf("mac button %s at %v, want on the left", n.Label, n.Bounds)
			}
		}
	}
	if len(buttons) != 2 {
		t.Errorf("buttons = %v, want close and minimize", buttons)
	}
}

func TestActorClicks(t *testing.T) {
	w := &fakeWindow{}
	bar := New(w, "App")
	buttonActor{bar, ButtonMinimize}.Click()
	if w.minimized != 1 {
		t.Error("actor didn't press the button")
	}
}

func TestDragAreaCoversBarButNotButtons(t *testing.T) {
	var r input.Router
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(800, 600)}, Source: r.Source()}
	menuBtn := elements.Box(styles.Styles{Width: 60, Height: 20})
	New(&fakeWindow{}, "App").Leading(menuBtn).Layout(gtx)
	r.Frame(&ops)

	isMove := func(x, y float32) bool {
		a, ok := r.ActionAt(f32.Pt(x, y))
		return ok && a == system.ActionMove
	}
	if !isMove(400, 18) {
		t.Error("empty bar space isn't a drag handle")
	}
	if isMove(800-20, 18) {
		t.Error("close button is draggable, so clicks would move the window")
	}
	if isMove(400, 100) {
		t.Error("area below the bar is draggable")
	}
}

func TestLongTitleLeavesRoomForMenus(t *testing.T) {
	fonts.UseGofont()
	long := strings.Repeat("A very long document title ", 20)
	for _, v := range []Variant{VariantWindows, VariantMac} {
		bar := New(&fakeWindow{focused: true}, long).Variant(v).
			Leading(elements.Text("FileMenu")).Trailing(elements.Text("Trailer"))
		elements.BeginSemantics()
		bar.Layout(gtxFor(600, 100, 1))
		nodes := elements.EndSemantics()
		bounds := map[string]image.Rectangle{}
		for _, n := range nodes {
			bounds[n.Label] = n.Bounds
		}
		limit := 600
		if v == VariantWindows {
			limit -= 3 * winButtonWidth
		}
		menu, trail := bounds["FileMenu"], bounds["Trailer"]
		if menu.Empty() || trail.Empty() || menu.Max.X > limit || trail.Max.X > limit || trail.Min.X < menu.Max.X {
			t.Errorf("%s: menu %v, trailing %v; both must fit before %d", v, menu, trail, limit)
		}
	}
}

func TestRTLButtonsOnTheLeft(t *testing.T) {
	fonts.UseGofont()
	bar := New(&fakeWindow{focused: true}, "عنوان").Leading(elements.Text("File"))
	gtx := gtxFor(600, 100, 1)
	gtx.Locale = system.Locale{Direction: system.RTL}
	elements.BeginSemantics()
	bar.Layout(gtx)
	b := map[string]image.Rectangle{}
	for _, n := range elements.EndSemantics() {
		b[n.Label] = n.Bounds
	}
	bw := 3 * winButtonWidth
	if c := b["Close"]; c.Min.X != 0 {
		t.Errorf("close at %v, want the left edge", c)
	}
	if m := b["Minimize"]; m.Max.X != bw {
		t.Errorf("minimize at %v, want ending at %d", m, bw)
	}
	if f := b["File"]; f.Min.X < bw || f.Max.X > 600 {
		t.Errorf("menu at %v, want right of the buttons", f)
	}
}

type trayWindow struct {
	closed, requested int
}

func (w *trayWindow) Minimize()       {}
func (w *trayWindow) ToggleMaximize() {}
func (w *trayWindow) Close()          { w.closed++ }
func (w *trayWindow) Maximized() bool { return false }
func (w *trayWindow) Focused() bool   { return true }
func (w *trayWindow) RequestClose()   { w.requested++ }

func TestCloseButtonClosesLikeTheSystem(t *testing.T) {
	w := &trayWindow{}
	New(w, "x").Press(ButtonClose)
	if w.requested != 1 || w.closed != 0 {
		t.Errorf("close button: RequestClose %d, Close %d; want the system close (RequestClose)", w.requested, w.closed)
	}
	New(w, "x").OnClose(func() bool { return false }).Press(ButtonClose)
	if w.requested != 1 {
		t.Error("closed although OnClose returned false")
	}
}
