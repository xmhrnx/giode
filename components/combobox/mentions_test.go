package combobox

import (
	"context"
	"testing"
	"time"

	"gioui.org/io/key"

	"github.com/xmhrnx/giode/components/input"
)

func TestMentions(t *testing.T) {
	field := input.New("Message")
	var picked []string
	m := NewMentions(field).
		Trigger('@', Option{Value: "u1", Label: "ada"}, Option{Value: "u2", Label: "alan"}, Option{Value: "u3", Label: "grace"}).
		TriggerSource('#', func(ctx context.Context, q string) ([]Option, error) {
			return []Option{{Value: "c1", Label: "general"}, {Value: "c2", Label: "go"}}, nil
		}).
		OnPick(func(r rune, o Option) { picked = append(picked, string(r)+o.Value) })
	h := newHarness(t, m, field)
	typ := func(s string) {
		h.typeText(s)
		n := len([]rune(field.Text()))
		field.SetSelection(n, n) // the platform moves the caret as it types
		h.frame()
		h.frame()
	}
	typ("hi @a")
	if !m.Opened() || len(m.list.Items) != 3 || m.list.Items[2].Label != "grace" {
		t.Fatalf("@a: open %v items %v", m.Opened(), m.list.Items)
	}
	if r := field.CaretRect(); r.Min.X <= 0 || r.Dy() <= 0 {
		t.Errorf("caret rect %v", r)
	}
	h.key(key.NameDownArrow)
	h.key(key.NameTab)
	if field.Text() != "hi @alan " || m.Opened() || len(picked) != 1 || picked[0] != "@u2" {
		t.Fatalf("text %q open %v picked %v", field.Text(), m.Opened(), picked)
	}
	// Not after a letter: an email address isn't a mention.
	typ("mail me@x")
	if m.Opened() {
		t.Error("opened inside a word")
	}
	// Escape dismisses until the next trigger.
	typ(" @g")
	if !m.Opened() {
		t.Fatal("@g didn't open")
	}
	h.key(key.NameEscape)
	typ("r")
	if m.Opened() {
		t.Error("reopened after Escape")
	}
	// An asynchronous source.
	typ(" #g")
	h.wait(300 * time.Millisecond)
	if !m.Opened() || len(m.list.Items) != 2 {
		t.Fatalf("#g: %v %v", m.Opened(), m.list.Items)
	}
	h.key(key.NameReturn)
	if want := "hi @alan mail me@x @gr #general "; field.Text() != want {
		t.Errorf("text %q, want %q", field.Text(), want)
	}
}
