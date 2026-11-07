// Package perf holds frame benchmarks of typical screens: run them with
//
//	go test ./internal/perf -bench . -benchmem
package perf

import (
	"fmt"
	"image"
	"runtime"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/internal/textrender"
	"github.com/xmhrnx/giode/ui"
)

// host lays a view out frame after frame the way App does, with a real
// input router.
type host struct {
	r    input.Router
	ops  op.Ops
	now  time.Time
	size image.Point
	view func() ui.Element
}

func newHost(w, h int, view func() ui.Element) *host {
	ui.UseGofont()
	return &host{size: image.Pt(w, h), now: time.Unix(0, 0), view: view}
}

func (h *host) frame() {
	h.ops.Reset()
	h.now = h.now.Add(16 * time.Millisecond)
	gtx := layout.Context{
		Ops: &h.ops, Now: h.now, Source: h.r.Source(),
		Metric:      unit.Metric{PxPerDp: 2, PxPerSp: 2},
		Constraints: layout.Exact(h.size),
	}
	ui.Frame(gtx, h.view())
	h.r.Frame(&h.ops)
}

func (h *host) run(b *testing.B, between func(i int)) {
	for i := 0; i < 5; i++ {
		h.frame()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if between != nil {
			between(i)
		}
		h.frame()
	}
}

const lorem = "Frames are laid out from the view function every time something changes; " +
	"most of a screen stays the same from one frame to the next."

// controlsView is 200 rows of a hoverable row, a label, a button, a
// checkbox and a switch.
func controlsView() func() ui.Element {
	n := 200
	btns := make([]ui.Element, n)
	cbs := make([]ui.Element, n)
	sws := make([]ui.Element, n)
	for i := range n {
		btns[i] = ui.Button("Open")
		cbs[i] = ui.Input.Checkbox("Done")
		sws[i] = ui.Input.Switch("On")
	}
	scroll := ui.Scroll(func() ui.Element {
		rows := make([]ui.Element, n)
		for i := range n {
			rows[i] = ui.Row(ui.Styles{Gap: 8, Padding: ui.UniformInset(4), Align: ui.AlignCenter},
				ui.Box(ui.Styles{FlexGrow: 1}, ui.Text(fmt.Sprintf("Item %d", i))),
				btns[i], cbs[i], sws[i])
		}
		return ui.Stack(ui.Styles{Width: -1}, rows...)
	})
	return func() ui.Element { return scroll }
}

func BenchmarkControls(b *testing.B) {
	newHost(1200, 900, controlsView()).run(b, nil)
}

// BenchmarkControlsHover moves the pointer over the rows every frame.
func BenchmarkControlsHover(b *testing.B) {
	h := newHost(1200, 900, controlsView())
	h.run(b, func(i int) {
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(300, float32(50+i%800))})
	})
}

// textView is a page of 60 paragraphs: headings, text and rich text.
func textView() func() ui.Element {
	scroll := ui.Scroll(func() ui.Element {
		var kids []ui.Element
		for i := range 20 {
			kids = append(kids,
				ui.H3(fmt.Sprintf("Section %d", i)),
				ui.Text(strings.Repeat(lorem+" ", 3)),
				ui.RichText(ui.Styles{}, ui.Span("Rich "), ui.Span("bold", ui.Styles{FontWeight: ui.FontWeightBold}), ui.Span(" and "+lorem)),
			)
		}
		return ui.Stack(ui.Styles{Width: -1, Gap: 8, Padding: ui.UniformInset(16)}, kids...)
	})
	return func() ui.Element { return scroll }
}

func BenchmarkText(b *testing.B) {
	newHost(1200, 900, textView()).run(b, nil)
}

// BenchmarkTextScroll scrolls the text page a little every frame.
func BenchmarkTextScroll(b *testing.B) {
	h := newHost(1200, 900, textView())
	h.run(b, func(i int) {
		dy := float32(20)
		if i%100 >= 50 {
			dy = -20
		}
		h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(300, 300), Scroll: f32.Pt(0, dy)})
	})
}

const reply = "Here's how it works:\n\n" +
	"1. **Parse** the source once and cache it.\n2. Lay out the `visible` rows.\n\n" +
	"```go\nfor _, r := range rows {\n\tr.Layout(gtx)\n}\n```\n\n" +
	"| Step | Cost |\n|---|---|\n| parse | once |\n| layout | per frame |\n\n" + lorem

type msg struct {
	id   int
	text string
}

// chatView is a bottom-anchored list of 1000 messages rendered as
// Markdown, optionally in a SelectionArea.
func chatView(msgs *[]msg, selectable bool) func() ui.Element {
	list := ui.LazyList(
		func() int { return len(*msgs) },
		func(i int) any { return (*msgs)[i].id },
		func(i int) ui.Element {
			m := (*msgs)[i]
			return ui.Stack(ui.Styles{Padding: ui.UniformInset(8), Gap: 4, Width: -1},
				ui.Text(fmt.Sprintf("User %d", m.id%7), ui.Styles{FontWeight: ui.FontWeightBold}),
				ui.Markdown(m.text),
			)
		},
	).Anchor(ui.LazyAnchorBottom).Version(func(i int) any { return len((*msgs)[i].text) })
	var root ui.Element = list
	if selectable {
		root = ui.SelectionArea("chat", list)
	}
	return func() ui.Element { return root }
}

func chatMsgs() *[]msg {
	// Replies and short messages, so rows differ in height.
	msgs := make([]msg, 1000)
	for i := range msgs {
		switch i % 3 {
		case 0:
			msgs[i] = msg{id: i, text: reply}
		case 1:
			msgs[i] = msg{id: i, text: "Thanks, that **works**."}
		default:
			msgs[i] = msg{id: i, text: lorem}
		}
	}
	return &msgs
}

func BenchmarkChat(b *testing.B) {
	newHost(1000, 900, chatView(chatMsgs(), false)).run(b, nil)
}

func BenchmarkChatSelectionArea(b *testing.B) {
	newHost(1000, 900, chatView(chatMsgs(), true)).run(b, nil)
}

// BenchmarkChatScroll scrolls the chat by a wheel step every frame.
func BenchmarkChatScroll(b *testing.B) {
	h := newHost(1000, 900, chatView(chatMsgs(), false))
	h.run(b, func(i int) {
		dy := float32(-60)
		if i%200 >= 100 {
			dy = 60
		}
		h.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(300, 300), Scroll: f32.Pt(0, dy)})
	})
}

// BenchmarkChatStreaming grows the last message by a word every frame.
func BenchmarkChatStreaming(b *testing.B) {
	msgs := chatMsgs()
	h := newHost(1000, 900, chatView(msgs, false))
	words := strings.Fields(lorem)
	(*msgs)[len(*msgs)-1].text = ""
	h.run(b, func(i int) {
		m := &(*msgs)[len(*msgs)-1]
		if len(m.text) > 4000 {
			m.text = ""
		}
		m.text += words[i%len(words)] + " "
	})
}

// BenchmarkTable lays out a 40 row, 4 column table.
func BenchmarkTable(b *testing.B) {
	view := func() ui.Element {
		rows := []ui.Element{ui.TableHeader(ui.Styles{}, ui.Text("Name"), ui.Text("Size"), ui.Text("Kind"), ui.Text("Modified"))}
		for i := range 40 {
			rows = append(rows, ui.TableRow(ui.Styles{}, ui.Text(fmt.Sprintf("file-%d.go", i)), ui.Text("12 KB"), ui.Text("Go source"), ui.Text("yesterday")))
		}
		return ui.Table(ui.Styles{}, rows...)
	}
	newHost(1000, 1400, view).run(b, nil)
}

// TestRetainedMemory reports the heap kept after a long run of chat
// frames with messages arriving, to catch caches that grow without
// bound.
func TestRetainedMemory(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	msgs := chatMsgs()
	next := len(*msgs)
	h := newHost(1000, 900, chatView(msgs, true))
	heap := func() uint64 {
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	for i := 0; i < 50; i++ {
		h.frame()
	}
	start := heap()
	for round := 0; round < 4; round++ {
		for i := 0; i < 500; i++ {
			// A new message with new text every frame; the oldest goes.
			*msgs = append((*msgs)[1:], msg{id: next, text: fmt.Sprintf("Message %d: %s", next, reply)})
			next++
			h.frame()
		}
		runs, runBytes, plain, rich := textrender.CacheStats()
		t.Logf("after %d more frames: heap %.1f MB (start %.1f MB); line images %d (%.1f MB), layouts %d plain, %d rich",
			(round+1)*500, float64(heap())/1e6, float64(start)/1e6, runs, float64(runBytes)/1e6, plain, rich)
	}
}

// TestRetainedMemoryDistinctText reports the heap kept after a chat of
// long, distinct messages - each new paragraph Gio's shaper would
// otherwise keep, up to a thousand of them.
func TestRetainedMemoryDistinctText(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	msgs := make([]msg, 50)
	for i := range msgs {
		msgs[i] = msg{id: i, text: fmt.Sprintf("Message %d. %s", i, strings.Repeat(lorem+" ", 8))}
	}
	next := len(msgs)
	h := newHost(1000, 900, chatView(&msgs, false))
	heap := func() float64 {
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return float64(m.HeapAlloc) / 1e6
	}
	start := heap()
	for i := 0; i < 1500; i++ {
		msgs = append(msgs[1:], msg{id: next, text: fmt.Sprintf("Message %d. %s", next, strings.Repeat(lorem+" ", 8))})
		next++
		h.frame()
	}
	t.Logf("after 1500 frames of new 1.5 KB messages: heap %.1f MB (start %.1f MB)", heap(), start)
}

// TestRetainedMemoryAfterIdle runs a burst of long, distinct messages,
// then lets the window go idle: the idle sweep, run on the "UI
// goroutine" before one more frame as App runs it, lets go of what
// isn't on screen.
func TestRetainedMemoryAfterIdle(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	msgs := make([]msg, 50)
	for i := range msgs {
		msgs[i] = msg{id: i, text: fmt.Sprintf("Message %d. %s", i, strings.Repeat(lorem+" ", 8))}
	}
	next := len(msgs)
	h := newHost(1000, 900, chatView(&msgs, false))
	heap := func() float64 {
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return float64(m.HeapAlloc) / 1e6
	}
	for i := 0; i < 1500; i++ {
		msgs = append(msgs[1:], msg{id: next, text: fmt.Sprintf("Message %d. %s", next, strings.Repeat(lorem+" ", 8))})
		next++
		h.frame()
	}
	busy := heap()
	ui := make(chan func(), 1)
	textrender.SetUIRunner(func(fn func()) {
		select {
		case ui <- fn:
		default:
		}
	})
	defer textrender.SetUIRunner(nil)
	defer textrender.SetIdleSweep(textrender.SetIdleSweep(50 * time.Millisecond))
	select {
	case fn := <-ui:
		fn()
		h.frame()
	case <-time.After(5 * time.Second):
		t.Fatal("no idle sweep")
	}
	t.Logf("heap after the burst %.1f MB, after going idle %.1f MB", busy, heap())
}
