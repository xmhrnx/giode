package elements

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

func TestFlexGrow_UnboundedColumnKeepsNaturalHeight(t *testing.T) {
	var logs []string
	SetDebugLogger(func(format string, v ...any) { logs = append(logs, fmt.Sprintf(format, v...)) })
	defer SetDebugLogger(nil)

	grow := Box(styles.Styles{FlexGrow: 1}, Text("Engine card"))
	col := Stack(styles.Styles{Padding: properties.UniformInset(8)}, grow, Box(styles.Styles{Height: 20}))
	var ops op.Ops
	// What a Scroll hands its content: unlimited height.
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(300, unbounded)}}
	dims := col.Layout(gtx)
	if dims.Size.Y > 200 {
		t.Errorf("column height %d: the FlexGrow child grew without limit", dims.Size.Y)
	}
	col.Layout(gtx)
	if len(logs) != 1 || !strings.Contains(logs[0], `Box containing "Engine card"`) {
		t.Errorf("logs = %q, want one warning naming the box", logs)
	}

	// With a height to fill, it still grows.
	gtx.Constraints = layout.Exact(image.Pt(300, 400))
	if h := Stack(styles.Styles{Height: -1}, Box(styles.Styles{FlexGrow: 1}, Text("x")), Box(styles.Styles{Height: 20})).Layout(gtx).Size.Y; h != 400 {
		t.Errorf("bounded column height %d, want 400", h)
	}
}

func TestFillHeight_InsidePaddedUnboundedBox(t *testing.T) {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(300, unbounded)}}
	box := Box(styles.Styles{Padding: properties.UniformInset(8)}, Box(styles.Styles{Height: -1}, Text("x")))
	if h := box.Layout(gtx).Size.Y; h > 200 {
		t.Errorf("Height -1 inside a padded unbounded box filled %d px", h)
	}
}
