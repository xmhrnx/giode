package giode

import (
	"io"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/xmhrnx/giode/components/kit"
)

// maxClipboardRead bounds how much clipboard text ReadClipboard reads.
const maxClipboardRead = 16 << 20

// clipboardState queues clipboard work requested from any goroutine
// until the next frame, where Gio executes clipboard commands.
type clipboardState struct {
	write    *string
	readers  []func(string)
	inFlight []func(string)
}

// SetClipboard copies text to the system clipboard. It is safe to call
// from any goroutine; the copy happens on the next frame, which it
// requests. Inside a Layout, kit.WriteClipboard copies immediately.
func (a *App) SetClipboard(text string) {
	a.uiMu.Lock()
	a.clip.write = &text
	a.uiMu.Unlock()
	a.invalidate()
}

// ReadClipboard reads the system clipboard's text and calls fn with it
// on the UI goroutine (with "" when the clipboard holds no text). It is
// safe to call from any goroutine and doesn't block; the read starts
// on the next frame, which it requests.
func (a *App) ReadClipboard(fn func(text string)) {
	a.uiMu.Lock()
	a.clip.readers = append(a.clip.readers, fn)
	a.uiMu.Unlock()
	a.invalidate()
}

// layoutClipboard applies queued clipboard work and delivers read
// results; the frame loop calls it every frame.
func (a *App) layoutClipboard(gtx layout.Context) {
	a.uiMu.Lock()
	write := a.clip.write
	a.clip.write = nil
	readers := a.clip.readers
	a.clip.readers = nil
	a.uiMu.Unlock()

	if write != nil {
		kit.WriteClipboard(gtx, *write)
	}
	if len(readers) > 0 {
		if len(a.clip.inFlight) == 0 {
			gtx.Execute(clipboard.ReadCmd{Tag: &a.clip})
		}
		a.clip.inFlight = append(a.clip.inFlight, readers...)
	}
	if len(a.clip.inFlight) == 0 {
		return
	}
	// The read result is delivered to this tag.
	ps := pointer.PassOp{}.Push(gtx.Ops)
	cl := clip.Rect{}.Push(gtx.Ops)
	event.Op(gtx.Ops, &a.clip)
	cl.Pop()
	ps.Pop()
	for {
		ev, ok := gtx.Event(transfer.TargetFilter{Target: &a.clip, Type: "application/text"})
		if !ok {
			break
		}
		de, ok := ev.(transfer.DataEvent)
		if !ok {
			continue
		}
		text := ""
		if r := de.Open(); r != nil {
			data, _ := io.ReadAll(io.LimitReader(r, maxClipboardRead))
			r.Close()
			text = string(data)
		}
		waiting := a.clip.inFlight
		a.clip.inFlight = nil
		for _, fn := range waiting {
			fn(text)
		}
	}
	if len(a.clip.inFlight) > 0 {
		kit.RequestFrame(gtx)
	}
}
