package box

import (
	"gioui.org/io/event"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Layout lays out the content over the hit area.
func (p *Pressable) Layout(gtx layout.Context) layout.Dimensions {
	p.update(gtx)
	if p.st.Display == properties.DisplayNone {
		return layout.Dimensions{}
	}
	state := p.State()
	var content base.Element = Box(styles.Styles{})
	if p.content != nil {
		if c := p.content(state); c != nil {
			content = c
		}
	}
	if p.st != (styles.Styles{}) {
		content = Box(p.st, content)
	}

	base.TrackPosition(&p.origin)
	macro := op.Record(gtx.Ops)
	dims := kit.Unselectable(content.Layout)(gtx)
	call := macro.Stop()

	// Hit area first, content on top: interactive children win.
	if !p.disabled {
		cl := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
		semantic.Button.Add(gtx.Ops)
		if p.label != "" {
			semantic.LabelOp(p.label).Add(gtx.Ops)
		}
		event.Op(gtx.Ops, p)
		if p.hovered {
			cursor := p.st.Cursor
			if cursor == "" {
				cursor = p.cursor
			}
			if cursor == "" {
				cursor = properties.CursorPointer
			}
			cursor.CursorFor().Add(gtx.Ops)
		}
		kit.BlockDragHere(gtx)
		cl.Pop()
	} else {
		cl := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
		semantic.EnabledOp(false).Add(gtx.Ops)
		cl.Pop()
	}
	call.Add(gtx.Ops)

	// Enclosing caches deliver input through the pass; without one
	// (or in an opaque subtree), stay live while interacting.
	if p.passFn == nil {
		p.passFn = p.pass
	}
	if !kit.HandleInput(p.passFn) && (p.hovered || p.pressed || p.focused) {
		kit.MarkLive()
	}
	if p.onClick != nil || p.onDouble != nil || p.onSecondary != nil || p.onLongPress != nil {
		// Only something that reacts to clicks is a button to
		// automation; a Pressable used just for hover state isn't.
		base.RecordSemantic(gtx, base.SemanticNode{Kind: base.SemanticButton, Label: p.label, Disabled: p.disabled, Target: p}, dims.Size)
	}
	return dims
}
