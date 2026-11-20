package emojipicker

import (
	"image"
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
	"github.com/xmhrnx/giode/styles/properties"
)

func TestData(t *testing.T) {
	all := All()
	if len(all) < 1800 {
		t.Fatalf("%d emoji", len(all))
	}
	var tones int
	for _, e := range all {
		if e.Emoji == "" || e.Name == "" || e.Group == "" {
			t.Fatalf("bad entry %+v", e)
		}
		if e.SkinTones != nil {
			tones++
			if len(e.SkinTones) != 5 {
				t.Fatalf("%s: %d tones", e.Name, len(e.SkinTones))
			}
		}
	}
	if tones < 250 {
		t.Errorf("%d emoji with skin tones", tones)
	}
}

func TestSearch(t *testing.T) {
	got := Search("thumbs")
	if len(got) < 2 || got[0].Emoji != "👍" {
		t.Fatalf("thumbs: %v", got)
	}
	if got := Search("rocket"); len(got) == 0 || got[0].Emoji != "🚀" {
		t.Errorf("rocket: %v", got)
	}
	if len(Search("")) != 0 || len(Search("zzzzqqq")) != 0 {
		t.Error("empty or nonsense matched")
	}
}

func TestPickToneAndRecents(t *testing.T) {
	var picked []string
	p := New().OnPick(func(e string) { picked = append(picked, e) }).SkinTone(3)
	wave := -1
	for i, d := range emojis {
		if d.name == "waving hand" {
			wave = i
		}
	}
	p.pick(wave)
	p.pick(0)
	if len(picked) != 2 || picked[0] != "👋🏽" || picked[1] != "😀" {
		t.Fatalf("picked %q", picked)
	}
	if r := p.Recents(); len(r) != 2 || r[0] != "😀" {
		t.Errorf("recents %q", r)
	}
	q := New()
	q.SetRecents(p.Recents())
	if len(q.recents) != 2 {
		t.Error("SetRecents")
	}
}

// TestShot draws the picker to GIODE_SHOT_DIR, for looking at.
func TestShot(t *testing.T) {
	dir := os.Getenv("GIODE_SHOT_DIR")
	if dir == "" {
		t.Skip("set GIODE_SHOT_DIR")
	}
	if runtime.GOOS == "linux" && os.Getenv("EGL_PLATFORM") == "" {
		t.Setenv("EGL_PLATFORM", "surfaceless")
	}
	fonts.UseGofont()
	p := New()
	p.SetRecents([]string{"👍", "🚀", "🎉"})
	w, h := 720, 840
	win, err := headless.NewWindow(w, h)
	if err != nil {
		t.Skip(err)
	}
	defer win.Release()
	var r input.Router
	for i := 0; i < 3; i++ {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Now: time.Unix(int64(i), 0), Source: r.Source(), Constraints: layout.Constraints{Max: image.Pt(w, h)}, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
		paint.Fill(gtx.Ops, properties.CalcColor("theme.background"))
		elements.BeginFrame()
		elements.SetWindowSize(gtx.Constraints.Max)
		elements.SetMetric(gtx.Metric)
		gtx.Constraints.Max.X = 640
		p.Layout(gtx)
		elements.DrainPortals(gtx)
		elements.RunAfterLayout()
		elements.EndFrame()
		r.Frame(&ops)
		if err := win.Frame(&ops); err != nil {
			t.Fatal(err)
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	win.Screenshot(img)
	f, _ := os.Create(dir + "/emojipicker.png")
	png.Encode(f, img)
	f.Close()
}
