package palette

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/shortcut"
)

type harness struct {
	r   input.Router
	p   *Palette
	now time.Time
}

func (h *harness) frame() {
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	max := image.Pt(800, 600)
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: max},
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	elements.BeginFrame()
	elements.SetWindowSize(max)
	elements.SetMetric(gtx.Metric)
	h.p.Layout(gtx)
	elements.DrainContextMenus(gtx)
	elements.DrainPortals(gtx)
	elements.DispatchBack(gtx)
	elements.DispatchKeys(gtx)
	elements.RunAfterLayout()
	elements.EndFrame()
	h.r.Frame(&ops)
}

func (h *harness) typeText(s string) {
	for _, r := range s {
		p := len([]rune(h.p.field.Text()))
		h.r.Queue(key.EditEvent{Range: key.Range{Start: p, End: p}, Text: string(r)})
		h.frame()
	}
	h.frame()
}

func (h *harness) key(n key.Name) {
	h.r.Queue(key.Event{Name: n, State: key.Press}, key.Event{Name: n, State: key.Release})
	h.frame()
	h.frame()
}

func TestPalette(t *testing.T) {
	fonts.UseGofont()
	var ran []string
	cmd := func(id string) func() { return func() { ran = append(ran, id) } }
	p := New(
		Command{ID: "new-tab", Title: "New tab", Shortcut: "Mod+T", Run: cmd("new-tab")},
		Command{ID: "close", Title: "Close window", Keywords: []string{"quit"}, Run: cmd("close")},
		Command{ID: "theme", Title: "Toggle dark mode", Group: "View", Run: cmd("theme")},
		Command{ID: "off", Title: "Not now", Disabled: true, Run: cmd("off")},
	)
	h := &harness{p: p, now: time.Unix(0, 0)}
	h.frame()

	// Mod+K, through the shortcuts, opens it.
	sc := p.Shortcuts()
	if len(sc) != 2 || !shortcut.Handle(key.Event{Name: "K", Modifiers: key.ModShortcut, State: key.Press}, sc) || !p.Opened() {
		t.Fatalf("Mod+K: %d shortcuts, open %v", len(sc), p.Opened())
	}
	h.frame()
	h.frame()
	if len(p.list.Items) != 4 {
		t.Fatalf("items %d", len(p.list.Items))
	}
	h.typeText("nwtb")
	if o, ok := p.list.Selected(); !ok || o.Value != "new-tab" || o.Detail == "" {
		t.Fatalf("fuzzy: %+v", p.list.Items)
	}
	h.key(key.NameReturn)
	if len(ran) != 1 || ran[0] != "new-tab" || p.Opened() {
		t.Fatalf("ran %v open %v", ran, p.Opened())
	}
	// Reopened, the recent command leads.
	p.Open()
	h.frame()
	if p.list.Items[0].Value != "new-tab" || p.list.Items[0].Group != "Recent" {
		t.Errorf("recents: %+v", p.list.Items[0])
	}
	h.typeText("quit")
	h.key(key.NameReturn)
	if len(ran) != 2 || ran[1] != "close" {
		t.Errorf("keyword: %v", ran)
	}
	// Escape and a press on the scrim close it.
	p.Open()
	h.frame()
	h.key(key.NameEscape)
	if p.Opened() {
		t.Error("escape")
	}
	p.Open()
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(5, 590), Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(5, 590)})
	h.frame()
	if p.Opened() {
		t.Error("scrim press")
	}
	// A command's own shortcut runs it and counts as recent.
	shortcut.Handle(key.Event{Name: "T", Modifiers: key.ModShortcut, State: key.Press}, p.Shortcuts())
	if len(ran) != 3 || p.Recents()[0] != "new-tab" {
		t.Errorf("shortcut run: %v %v", ran, p.Recents())
	}
}
