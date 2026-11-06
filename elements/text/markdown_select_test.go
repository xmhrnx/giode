package text_test

import (
	"image"
	"strings"
	"testing"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/markdown"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/elements/text"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

const selectDoc = "First paragraph here.\n\nSecond paragraph.\n\n" +
	"| a | b |\n|---|---|\n| one | two |\n\n" +
	"```go\nx := 1\n```\n"

// MarkdownOptions.Selectable makes the document selectable by dragging,
// across its blocks.
func TestMarkdownSelectable(t *testing.T) {
	fonts.UseGofont()
	h := &listHarness{root: markdown.New(selectDoc, markdown.Options{Selectable: "msg", Styles: styles.Styles{FontSize: 20}})}
	for i := 0; i < 2; i++ {
		h.frame()
	}
	h.mouse(pointer.Press, 2, 10)
	h.mouse(pointer.Move, 60, 45)
	h.mouse(pointer.Release, 60, 45)
	got := text.SelectedText([2]any{"giode.markdown", "msg"})
	if !strings.HasPrefix(got, "First paragraph here.\n") || !strings.HasPrefix("Second paragraph.", got[len("First paragraph here.\n"):]) {
		t.Errorf("selected %q, want the first paragraph and the start of the second", got)
	}
}

// Every table cell is part of a selection once, and is one node of the
// semantic tree.
func TestMarkdownTableRegistersOnce(t *testing.T) {
	fonts.UseGofont()
	for _, own := range []bool{true, false} {
		var root elements.Element
		if own {
			root = markdown.New(selectDoc, markdown.Options{Selectable: "doc"})
		} else {
			root = elements.SelectionArea("sel", markdown.New(selectDoc))
		}
		h := &listHarness{root: root}
		h.frame()
		h.frame()
		// Select all: a click in the area, then Ctrl+A.
		h.mouse(pointer.Press, 5, 5)
		h.mouse(pointer.Release, 5, 5)
		h.r.Queue(key.Event{Name: "A", Modifiers: key.ModShortcut, State: key.Press})
		h.frame()
		var got string
		if own {
			got = text.SelectedText([2]any{"giode.markdown", "doc"})
		} else {
			got = text.SelectedText("sel")
		}
		want := "First paragraph here.\nSecond paragraph.\na\nb\none\ntwo\nx := 1"
		if got != want {
			t.Errorf("own area %v: selected %q, want %q", own, got, want)
		}

		elements.BeginFrame()
		elements.BeginSemantics()
		var ops op.Ops
		root.Layout(layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(600, 300)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}})
		nodes := elements.EndSemantics()
		elements.EndFrame()
		cells := map[string]int{}
		for _, n := range nodes {
			switch n.Label {
			case "a", "b", "one", "two":
				cells[n.Label]++
			}
		}
		for _, c := range []string{"a", "b", "one", "two"} {
			if cells[c] != 1 {
				t.Errorf("own area %v: cell %q is %d semantic nodes, want 1", own, c, cells[c])
			}
		}
	}
}
