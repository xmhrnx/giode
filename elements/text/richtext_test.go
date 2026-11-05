package text

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/elements/box"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

type textHarness struct {
	r    input.Router
	view func() base.Element
	now  time.Time
	dims layout.Dimensions
}

func (h *textHarness) frame() {
	h.now = h.now.Add(20 * time.Millisecond)
	base.SetWindowSize(image.Pt(600, 400))
	base.BeginFrame()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Constraints: layout.Constraints{Max: image.Pt(600, 400)}}
	h.dims = h.view().Layout(gtx)
	base.DrainPortals(gtx)
	base.DispatchBack(gtx)
	base.RunAfterLayout()
	h.r.Frame(&ops)
}

func (h *textHarness) press(x, y float32, src pointer.Source, btn pointer.Buttons, hold time.Duration) {
	pos := f32.Pt(x, y)
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: src, Position: pos, Buttons: btn})
	h.frame()
	for d := time.Duration(0); d < hold; d += 100 * time.Millisecond {
		h.now = h.now.Add(100 * time.Millisecond)
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: src, Position: pos})
	h.frame()
	h.frame()
}

func TestRichText_LinkTaps(t *testing.T) {
	fonts.UseGofont()
	var opened []string
	h := &textHarness{now: time.Unix(0, 0), view: func() base.Element {
		return RichText(styles.Styles{FontSize: 16}, Link("docs", "https://example.com")).
			OnLink(func(url string) { opened = append(opened, url) })
	}}
	h.frame()
	if h.dims.Size.X == 0 {
		t.Fatal("rich text has no size")
	}
	h.press(5, 5, pointer.Mouse, pointer.ButtonPrimary, 0)
	if len(opened) != 1 || opened[0] != "https://example.com" {
		t.Errorf("opened %v, want the link", opened)
	}
}

func TestRichText_DefaultOpener(t *testing.T) {
	fonts.UseGofont()
	var opened string
	SetLinkOpener(func(url string) { opened = url })
	defer SetLinkOpener(nil)
	h := &textHarness{now: time.Unix(0, 0), view: func() base.Element {
		return RichText(styles.Styles{}, Link("site", "https://gioui.org"))
	}}
	h.frame()
	h.press(5, 5, pointer.Touch, pointer.ButtonPrimary, 0)
	if opened != "https://gioui.org" {
		t.Errorf("opener got %q", opened)
	}
}

func TestSelectable_RightClickCopy(t *testing.T) {
	fonts.UseGofont()
	defer base.SetTouchInput(base.TouchInput())
	base.SetTouchInput(false)
	h := &textHarness{now: time.Unix(0, 0), view: func() base.Element {
		return box.Box(styles.Styles{Padding: styles.Styles{}.Padding}, Text("hello world", styles.Styles{FontSize: 16}).Selectable("msg-1"))
	}}
	h.frame()
	h.press(10, 8, pointer.Mouse, pointer.ButtonSecondary, 0)
	s := base.Keyed("giode.selectable", "msg-1", newSelectState)
	if !s.all {
		t.Fatal("right-click didn't select the text")
	}
	// The Copy action sits below the text (no room above it).
	clickCopy(t, h)
	if _, data, ok := h.r.WriteClipboard(); !ok || string(data) != "hello world" {
		t.Errorf("clipboard = %q (%v), want the text", data, ok)
	}
	if s.all {
		t.Error("selection still active after Copy")
	}
}

// clickCopy finds the Copy action's semantic node and clicks it.
func clickCopy(t *testing.T, h *textHarness) {
	t.Helper()
	base.BeginSemantics()
	h.frame()
	nodes := base.EndSemantics()
	for _, n := range nodes {
		if n.Kind == base.SemanticButton && n.Label == "Copy" {
			c := n.Bounds.Min.Add(n.Bounds.Size().Div(2))
			h.press(float32(c.X), float32(c.Y), pointer.Mouse, pointer.ButtonPrimary, 0)
			return
		}
	}
	t.Fatalf("no Copy button among %v", nodes)
}

func TestSelectable_TouchLongPress(t *testing.T) {
	fonts.UseGofont()
	defer base.SetTouchInput(base.TouchInput())
	base.SetTouchInput(true)
	h := &textHarness{now: time.Unix(0, 0), view: func() base.Element {
		return RichText(styles.Styles{FontSize: 16}, SpanText("bold ", styles.Styles{FontWeight: "bold"}), SpanText("plain")).Selectable("msg-2")
	}}
	h.frame()
	s := base.Keyed("giode.selectable", "msg-2", newSelectState)
	h.press(10, 8, pointer.Touch, pointer.ButtonPrimary, 100*time.Millisecond)
	if s.all {
		t.Fatal("a short tap selected the text")
	}
	h.press(10, 8, pointer.Touch, pointer.ButtonPrimary, 700*time.Millisecond)
	if !s.all {
		t.Fatal("a long-press didn't select the text")
	}
	clickCopy(t, h)
	if _, data, ok := h.r.WriteClipboard(); !ok || string(data) != "bold plain" {
		t.Errorf("clipboard = %q, want the plain text", data)
	}
}
