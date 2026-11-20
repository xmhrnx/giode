package combobox

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
)

func TestMatch(t *testing.T) {
	sc1, pos, ok := Match("nt", "New tab")
	if !ok || len(pos) != 2 || pos[0] != 0 || pos[1] != 4 {
		t.Fatalf("New tab: %v %v", pos, ok)
	}
	sc2, _, _ := Match("nt", "Paint")
	if sc1 <= sc2 {
		t.Errorf("word starts should score higher: %d vs %d", sc1, sc2)
	}
	if _, _, ok := Match("xyz", "New tab"); ok {
		t.Error("xyz matched")
	}
	got := Filter([]Option{{Value: "Paint"}, {Value: "New tab"}, {Value: "Close", Keywords: []string{"quit"}}}, "nt")
	if len(got) != 2 || got[0].Value != "New tab" {
		t.Errorf("filter %v", got)
	}
	if got := Filter([]Option{{Value: "Close", Keywords: []string{"quit"}}}, "qui"); len(got) != 1 {
		t.Error("keywords not searched")
	}
}

func TestComboboxTypeAndPick(t *testing.T) {
	c := New("Fruit").Options(
		Option{Value: "apple", Group: "Pome"}, Option{Value: "pear", Group: "Pome"},
		Option{Value: "apricot", Group: "Stone"}, Option{Value: "cherry", Group: "Stone", Disabled: true},
	)
	var picked []string
	c.OnSelect(func(o Option) { picked = append(picked, o.Value) })
	h := newHarness(t, c, c.Input())
	if !c.Opened() || len(c.list.Items) != 4 {
		t.Fatalf("focus should open all options: %v %d", c.Opened(), len(c.list.Items))
	}
	h.typeText("ap")
	if len(c.list.Items) != 2 {
		t.Fatalf("ap: %v", c.list.Items)
	}
	h.key(key.NameDownArrow)
	h.key(key.NameReturn)
	if len(picked) != 1 || picked[0] != "apricot" || c.Text() != "apricot" || c.Opened() {
		t.Fatalf("picked %v, text %q, open %v", picked, c.Text(), c.Opened())
	}
	if v, ok := c.Value(); !ok || v.Value != "apricot" {
		t.Error("value")
	}
	// Down reopens with every option; the disabled one is skipped.
	h.key(key.NameDownArrow)
	if !c.Opened() || len(c.list.Items) != 4 {
		t.Fatalf("reopen: %v %d", c.Opened(), len(c.list.Items))
	}
	c.list.Index = 2
	c.list.Move(1)
	if c.list.Index != 0 {
		t.Errorf("disabled cherry not skipped: %d", c.list.Index)
	}
	h.key(key.NameEscape)
	if c.Opened() {
		t.Error("escape didn't close")
	}
	// A click outside closes; a click on an option picks it.
	h.key(key.NameDownArrow)
	h.click(f32.Pt(480, 380))
	if c.Opened() {
		t.Error("outside press didn't close")
	}
	h.key(key.NameDownArrow)
	// The first option, under the group heading: find its row.
	for y := float32(40); y < 120 && len(picked) < 2; y += 4 {
		h.click(f32.Pt(40, y))
		if !c.Opened() && len(picked) < 2 {
			h.key(key.NameDownArrow)
		}
	}
	if len(picked) != 2 || picked[1] != "apple" {
		t.Errorf("click picked %v", picked)
	}
}

func TestComboboxAsync(t *testing.T) {
	var calls atomic.Int32
	c := New("City").Debounce(100 * time.Millisecond).MinChars(2).Source(func(ctx context.Context, q string) ([]Option, error) {
		calls.Add(1)
		if q == "zz" {
			return nil, errors.New("server down")
		}
		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []Option{{Value: q + "-ville"}, {Value: q + "-burg"}}, nil
	})
	h := newHarness(t, c, c.Input())
	h.typeText("p")
	h.wait(200 * time.Millisecond)
	if calls.Load() != 0 || len(c.list.Items) != 0 {
		t.Fatalf("asked below MinChars: %d", calls.Load())
	}
	h.typeText("ar")
	if !c.list.Loading {
		t.Error("not loading during the debounce")
	}
	h.wait(300 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("calls %d: the debounce should ask once", calls.Load())
	}
	if c.list.Loading || len(c.list.Items) != 2 || c.list.Items[0].Value != "par-ville" {
		t.Fatalf("items %v loading %v", c.list.Items, c.list.Loading)
	}
	if len(c.list.Items[0].matched) != 3 {
		t.Error("server results not highlighted")
	}
	c.Input().SetText("zz")
	h.wait(300 * time.Millisecond)
	if c.list.Err == nil {
		t.Error("error not shown")
	}
}
