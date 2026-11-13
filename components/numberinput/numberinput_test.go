package numberinput

import (
	"image"
	"math"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/fonts"
)

type harness struct {
	r   input.Router
	n   *NumberInput
	now time.Time
}

func (h *harness) frame() layout.Dimensions {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: h.r.Source(), Now: h.now, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{Max: image.Pt(400, 400)}}
	dims := h.n.Layout(gtx)
	h.r.Frame(&ops)
	return dims
}

func (h *harness) press(x float32) {
	h.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: f32.Pt(x, 10), Buttons: pointer.ButtonPrimary})
	h.frame()
}

func (h *harness) release(x float32) {
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(x, 10)})
	h.frame()
}

func TestNumberInput(t *testing.T) {
	fonts.UseGofont()
	var changes []float64
	n := New(10).Range(8, 12).OnChange(func(v float64) { changes = append(changes, v) })
	h := &harness{n: n, now: time.Unix(0, 0)}
	dims := h.frame()
	w := float32(dims.Size.X)
	if dims.Size.X != 140 {
		t.Errorf("width %d, want the default 140dp", dims.Size.X)
	}

	// + steps once per click, and stops at the maximum.
	for range 4 {
		h.press(w - 5)
		h.release(w - 5)
	}
	if n.Value() != 12 || n.Field().Text() != "12" {
		t.Errorf("after clicking + four times: %v %q, want 12, the maximum", n.Value(), n.Field().Text())
	}
	if len(changes) != 2 {
		t.Errorf("changes %v, want 11 and 12 only", changes)
	}

	// Holding − repeats after a delay.
	h.press(5)
	if n.Value() != 11 {
		t.Fatalf("pressing −: %v, want 11", n.Value())
	}
	h.now = h.now.Add(200 * time.Millisecond)
	h.frame()
	if n.Value() != 11 {
		t.Errorf("held 200ms: %v, want still 11", n.Value())
	}
	h.now = h.now.Add(250 * time.Millisecond)
	h.frame()
	h.now = h.now.Add(70 * time.Millisecond)
	h.frame()
	if n.Value() != 9 {
		t.Errorf("held past the delay: %v, want 9", n.Value())
	}
	h.release(5)

	// Typing applies numbers in range at once, and commits the rest on
	// Enter: back in range, or back to the value.
	n.Field().Focus()
	h.frame()
	h.frame()
	for _, c := range []struct {
		typed string
		live  float64
		after float64
		text  string
	}{
		{"11,5", 11.5, 12, "12"},
		{"3", 12, 8, "8"},
		{"abc", 8, 8, "8"},
	} {
		n.Decimals(0)
		n.Field().SetText("")
		h.r.Queue(key.EditEvent{Text: c.typed})
		h.frame()
		if c.typed == "11,5" {
			// Rounded to no decimals once applied.
			if n.Value() != 12 {
				t.Errorf("typed %q: %v, want 12", c.typed, n.Value())
			}
		} else if n.Value() != c.live {
			t.Errorf("typed %q: value %v, want %v while typing", c.typed, n.Value(), c.live)
		}
		h.r.Queue(key.Event{Name: key.NameReturn, State: key.Press}, key.Event{Name: key.NameReturn, State: key.Release})
		h.frame()
		h.frame()
		if n.Value() != c.after || n.Field().Text() != c.text {
			t.Errorf("typed %q, Enter: %v %q, want %v %q", c.typed, n.Value(), n.Field().Text(), c.after, c.text)
		}
	}

	// Up and Down step, by ten with Shift.
	n.Range(math.Inf(-1), math.Inf(1)).Step(0.5).Decimals(1)
	n.SetValue(1)
	for _, k := range []struct {
		name key.Name
		mods key.Modifiers
		want float64
	}{
		{key.NameUpArrow, 0, 1.5},
		{key.NameUpArrow, key.ModShift, 6.5},
		{key.NameDownArrow, 0, 6},
	} {
		h.r.Queue(key.Event{Name: k.name, Modifiers: k.mods, State: key.Press}, key.Event{Name: k.name, Modifiers: k.mods, State: key.Release})
		h.frame()
		if n.Value() != k.want {
			t.Errorf("%v%v: %v, want %v", k.mods, k.name, n.Value(), k.want)
		}
	}
	if n.Field().Text() != "6.0" {
		t.Errorf("text %q, want 6.0 with one decimal", n.Field().Text())
	}
}
