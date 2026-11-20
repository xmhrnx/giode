package combobox

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

	cinput "github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
)

type harness struct {
	t     *testing.T
	r     input.Router
	view  elements.Element
	field *cinput.Input
	now   time.Time
	max   image.Point
}

func newHarness(t *testing.T, view elements.Element, field *cinput.Input) *harness {
	fonts.UseGofont()
	h := &harness{t: t, view: view, field: field, now: time.Unix(0, 0), max: image.Pt(500, 400)}
	h.frame()
	field.Focus()
	h.frame()
	h.frame()
	return h
}

func (h *harness) frame() {
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: h.max},
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	elements.BeginFrame()
	elements.SetWindowSize(h.max)
	elements.SetMetric(gtx.Metric)
	h.view.Layout(gtx)
	elements.DrainContextMenus(gtx)
	elements.DrainPortals(gtx)
	elements.DispatchBack(gtx)
	elements.DispatchKeys(gtx)
	elements.RunAfterLayout()
	elements.EndFrame()
	h.r.Frame(&ops)
}

// wait runs frames for d of the harness clock.
func (h *harness) wait(d time.Duration) {
	for end := h.now.Add(d); h.now.Before(end); {
		h.frame()
		time.Sleep(16 * time.Millisecond)
	}
}

func (h *harness) typeText(s string) {
	for _, r := range s {
		p := len([]rune(h.field.Text()))
		h.r.Queue(key.EditEvent{Range: key.Range{Start: p, End: p}, Text: string(r)})
		h.frame()
	}
	h.frame()
}

func (h *harness) key(name key.Name) {
	h.r.Queue(key.Event{Name: name, State: key.Press}, key.Event{Name: name, State: key.Release})
	h.frame()
	h.frame()
}

func (h *harness) click(p f32.Point) {
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: p, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p},
	)
	h.frame()
	h.frame()
}
