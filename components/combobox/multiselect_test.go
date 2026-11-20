package combobox

import (
	"strings"
	"testing"

	"gioui.org/io/key"
)

func TestMultiSelectPickAndRemove(t *testing.T) {
	m := NewMulti("Fruits").Strings("apple", "apricot", "banana", "cherry")
	var changes int
	m.OnChange(func([]Option) { changes++ })
	h := newHarness(t, m, m.Input())
	if !m.Opened() || len(m.list.Items) != 4 {
		t.Fatalf("focus: %v %d", m.Opened(), len(m.list.Items))
	}
	h.typeText("ap")
	h.key(key.NameReturn) // apple
	h.typeText("ap")
	if len(m.list.Items) != 1 || m.list.Items[0].Value != "apricot" {
		t.Fatalf("picked options still listed: %v", m.list.Items)
	}
	h.key(key.NameReturn)
	if got := strings.Join(m.ValueStrings(), ","); got != "apple,apricot" || m.Input().Text() != "" || changes != 2 {
		t.Fatalf("values %q text %q changes %d", got, m.Input().Text(), changes)
	}
	// Backspace in the empty field removes the last chip.
	h.key(key.NameDeleteBackward)
	if got := strings.Join(m.ValueStrings(), ","); got != "apple" {
		t.Fatalf("after backspace %q", got)
	}
	// With text, Backspace deletes a letter instead.
	h.typeText("ban")
	m.Input().SetSelection(3, 3) // the platform moves the caret as it types
	h.key(key.NameDeleteBackward)
	if m.Input().Text() != "ba" || len(m.Values()) != 1 {
		t.Fatalf("backspace in text: %q %v", m.Input().Text(), m.ValueStrings())
	}
	// Typing an unknown value isn't a tag without Creatable.
	m.Input().SetText("")
	h.frame()
	h.typeText("kiwi")
	h.key(key.NameReturn)
	if len(m.Values()) != 1 {
		t.Errorf("created %v without Creatable", m.ValueStrings())
	}
}

func TestTagInputPasteAndMax(t *testing.T) {
	m := NewMulti("Tags").Creatable(true).Max(4).Strings("Go")
	h := newHarness(t, m, m.Input())
	// A paste of comma-separated values makes tags; the last part,
	// without a trailing comma, stays to be finished.
	m.Input().SetText("go, rust ,zig,  ,ocaml")
	h.frame()
	h.frame()
	if got := strings.Join(m.ValueStrings(), ","); got != "Go,rust,zig" || m.Input().Text() != "ocaml" {
		t.Fatalf("values %q text %q", got, m.Input().Text())
	}
	h.key(key.NameReturn)
	if got := strings.Join(m.ValueStrings(), ","); got != "Go,rust,zig,ocaml" {
		t.Fatalf("enter: %q", got)
	}
	// Full: no more, and the list stays closed.
	h.typeText("lua,")
	if len(m.Values()) != 4 || m.Opened() {
		t.Errorf("past Max: %v open %v", m.ValueStrings(), m.Opened())
	}
	// Duplicates are ignored.
	m.SetValues(Option{Value: "a"}, Option{Value: "a"}, Option{Value: "b"})
	if len(m.Values()) != 2 {
		t.Errorf("duplicates kept: %v", m.ValueStrings())
	}
}
