package giode

import (
	"testing"

	"gioui.org/f32"
)

func TestFileDragState(t *testing.T) {
	a := &App{}
	if _, ok := a.FileDrag(); ok {
		t.Fatal("drag before any event")
	}
	a.drag.active, a.drag.pos = true, f32.Pt(10, 20)
	d, ok := a.FileDrag()
	if !ok || d.Position != f32.Pt(10, 20) || d.Kind != FileDragOver {
		t.Errorf("FileDrag = %+v, %v", d, ok)
	}
	for k, s := range map[FileDragKind]string{FileDragEnter: "enter", FileDragOver: "over", FileDragLeave: "leave", FileDragDrop: "drop", 0: "none"} {
		if k.String() != s {
			t.Errorf("%d.String() = %q, want %q", k, k.String(), s)
		}
	}
}
