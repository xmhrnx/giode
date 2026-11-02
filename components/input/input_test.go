package input

import (
	"image"
	"image/color"
	"os"
	"runtime"
	"strings"
	"testing"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

type harness struct {
	r   input.Router
	in  *Input
	max image.Point
}

func (h *harness) frame() layout.Dimensions {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Constraints: layout.Constraints{Max: h.max}}
	dims := h.in.Layout(gtx)
	h.r.Frame(&ops)
	return dims
}

func (h *harness) tap(x, y float32, src pointer.Source) {
	pos := f32.Pt(x, y)
	h.r.Queue(
		pointer.Event{Kind: pointer.Press, Source: src, Position: pos, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: src, Position: pos},
	)
	h.frame()
	h.frame()
}

func TestInput_TapAnywhereFocuses(t *testing.T) {
	for _, tc := range []struct {
		name string
		x, y float32
	}{
		// Inside the padding, left of the text.
		{"padding", 4, 20},
		// Far right of an empty field, past the old 80dp text area.
		{"blank", 380, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &harness{in: New("Name", styles.Styles{Height: 40}), max: image.Pt(400, 400)}
			h.frame()
			h.tap(tc.x, tc.y, pointer.Touch)
			if !h.in.Focused() {
				t.Errorf("tap at (%v, %v) didn't focus the field", tc.x, tc.y)
			}
		})
	}
}

func TestInput_EditorFillsWidth(t *testing.T) {
	in := New("Name")
	in.SetText("ab")
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(300, 400)}}
	if got := (&editorEl{input: in}).Layout(gtx).Size.X; got != 300 {
		t.Errorf("editor width = %d, want 300 (the whole field)", got)
	}

	// Without a width to fill (a horizontal scroll), it hugs the text.
	gtx.Constraints.Max.X = 1e6
	if got := (&editorEl{input: in}).Layout(gtx).Size.X; got >= 300 {
		t.Errorf("unbounded editor width = %d, want the text's width", got)
	}
}

func TestInput_SecondaryClickDoesNotFocus(t *testing.T) {
	h := &harness{in: New("Name", styles.Styles{Height: 40}), max: image.Pt(400, 400)}
	h.frame()
	pos := f32.Pt(4, 20)
	h.r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: pos, Buttons: pointer.ButtonSecondary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos},
	)
	h.frame()
	h.frame()
	if h.in.Focused() {
		t.Error("secondary click in the padding focused the field")
	}
}

func TestInput_KeyboardHint(t *testing.T) {
	in := New("URL").Keyboard(KeyboardURL)
	if in.editor.InputHint != key.HintURL {
		t.Errorf("hint = %v, want URL", in.editor.InputHint)
	}
	in.Password(true)
	if in.editor.InputHint != key.HintPassword {
		t.Errorf("password hint = %v, want Password", in.editor.InputHint)
	}
	in.Password(false)
	if in.editor.InputHint != key.HintURL {
		t.Errorf("hint after Password(false) = %v, want URL", in.editor.InputHint)
	}
}

func TestInput_EnterUnlessShift(t *testing.T) {
	fonts.UseGofont()
	defer elements.SetTouchInput(elements.TouchInput())
	elements.SetTouchInput(false)
	var sent []string
	in := New("Message").Lines(1, 4).SubmitOn(SubmitEnterUnlessShift).Submit(func(s string) { sent = append(sent, s) })
	h := &harness{in: in, max: image.Pt(400, 400)}
	in.Focus()
	h.frame()
	h.frame()
	in.InsertText("hi")
	h.r.Queue(key.Event{Name: key.NameReturn, Modifiers: key.ModShift, State: key.Press})
	h.frame()
	if in.Text() != "hi\n" || len(sent) != 0 {
		t.Fatalf("after Shift+Enter: text %q, sent %v; want a newline and nothing sent", in.Text(), sent)
	}
	h.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	h.frame()
	if len(sent) != 1 || sent[0] != "hi\n" {
		t.Errorf("after Enter: sent %v, want [\"hi\\n\"]", sent)
	}
}

func TestInput_LinesGrowThenCap(t *testing.T) {
	fonts.UseGofont()
	in := New("", styles.Styles{FontSize: 14}).Lines(1, 3)
	h := &harness{in: in, max: image.Pt(400, 1000)}
	one := h.frame().Size.Y
	in.SetText("a\nb")
	two := h.frame().Size.Y
	in.SetText("a\nb\nc\nd\ne\nf")
	capped := h.frame().Size.Y
	if !(one < two && two < capped) {
		t.Fatalf("heights %d, %d, %d: want growth from 1 to 3 lines", one, two, capped)
	}
	in.SetText("a\nb\nc\nd\ne\nf\ng\nh")
	if more := h.frame().Size.Y; more != capped {
		t.Errorf("height %d past max lines, want it capped at %d", more, capped)
	}
}

func TestInput_FocusBlurAndSelectAll(t *testing.T) {
	var events []string
	in := New("x").OnFocus(func() { events = append(events, "focus") }).OnBlur(func() { events = append(events, "blur") })
	h := &harness{in: in, max: image.Pt(400, 400)}
	h.frame()
	in.Focus()
	h.frame()
	h.frame()
	h.r.Source().Execute(key.FocusCmd{})
	h.frame()
	if strings.Join(events, ",") != "focus,blur" {
		t.Errorf("events %v, want focus then blur", events)
	}
	in.SetText("hello")
	in.SelectAll()
	if s, e := in.editor.Selection(); min(s, e) != 0 || max(s, e) != 5 {
		t.Errorf("selection %d..%d, want 0..5", s, e)
	}
}

func TestInput_PlaceholderStaysOneLine(t *testing.T) {
	fonts.UseGofont()
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	win, err := headless.NewWindow(200, 300)
	if err != nil {
		t.Skipf("no offscreen renderer: %v", err)
	}
	defer win.Release()
	long := strings.Repeat("a long placeholder that would wrap ", 8)
	in := New("x", styles.Styles{Color: "#000000"}).Placeholder(long)
	var ops op.Ops
	paint.Fill(&ops, color.NRGBA{255, 255, 255, 255})
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(200, 300)}}
	h := in.Layout(gtx).Size.Y
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 200, 300))
	if err := win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	for y := h; y < 300; y++ {
		for x := 0; x < 200; x++ {
			if c := img.RGBAAt(x, y); c.R < 200 {
				t.Fatalf("placeholder painted at (%d, %d), below the %d px field", x, y, h)
			}
		}
	}
}
