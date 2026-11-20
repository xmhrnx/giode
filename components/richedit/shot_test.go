package richedit

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/richdoc"
	"github.com/xmhrnx/giode/styles/properties"
)

// TestShot writes PNGs of editors to GIODE_SHOT_DIR, for looking at.
func TestShot(t *testing.T) {
	dir := os.Getenv("GIODE_SHOT_DIR")
	if dir == "" {
		t.Skip("set GIODE_SHOT_DIR")
	}
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	fonts.UseGofont()
	shot := func(name string, w, h int, e elements.Element) {
		win, err := headless.NewWindow(w, h)
		if err != nil {
			t.Skipf("no offscreen renderer: %v", err)
		}
		defer win.Release()
		var r input.Router
		var img *image.RGBA
		for i := 0; i < 3; i++ {
			var ops op.Ops
			gtx := layout.Context{Ops: &ops, Now: time.Unix(int64(i), 0), Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(w, h)}, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
			paint.Fill(gtx.Ops, properties.CalcColor("theme.background"))
			frame(gtx, e)
			r.Frame(&ops)
			if err := win.Frame(&ops); err != nil {
				t.Fatal(err)
			}
			img = image.NewRGBA(image.Rectangle{Max: image.Pt(w, h)})
			if err := win.Screenshot(img); err != nil {
				t.Fatal(err)
			}
		}
		f, _ := os.Create(dir + "/" + name + ".png")
		png.Encode(f, img)
		f.Close()
	}
	e := New("Write something…")
	e.SetMarkdown("# Launch plan\n\nShip the **editor** with *inline* marks, `code`, ~~old~~ and <u>underlined</u> text, a [link](https://example.com), <span style=\"color: #c62828\">red</span> and <mark>highlighted</mark> words. Ask [@Ada Lovelace](mention:1) about [#4521](ref:4521).\n\n" +
		"- First point\n   - Nested point\n- Second point\n\n1. One\n2. Two\n   1. Two-a\n\n- [x] Write the model\n- [ ] Draw the caret\n\n> Quoted words of wisdom.\n\n```go\nfunc main() {\n\tprintln(\"hi\")\n}\n```\n\n| Name | Role |\n| --- | --- |\n| Ada | Engineer |\n\n---\n\nThe end.")
	e.Focus()
	e.Select(richdoc.Selection{Anchor: richdoc.Pos{Block: 1, Offset: 5}, Head: richdoc.Pos{Block: 1, Offset: 30}})
	shot("richedit", 720, 1500, e)
	tb := New("")
	tb.SetText("toolbar")
	tb.OnLinkEdit(func(string, func(string)) {})
	tb.Select(richdoc.Selection{Head: richdoc.Pos{Offset: 7}})
	tb.ToggleMark(richdoc.Bold)
	shot("richedit-toolbar", 720, 90, tbWrap{tb})
	empty := New("Message #general")
	shot("richedit-empty", 600, 120, empty)
	_ = color.NRGBA{}
}

type tbWrap struct{ e *RichEditor }

func (w tbWrap) Layout(gtx layout.Context) layout.Dimensions {
	d := w.e.Toolbar().Layout(gtx)
	return d
}
