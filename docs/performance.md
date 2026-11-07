# Rendering performance

Giode is immediate mode: the view function describes the whole UI every frame. That keeps
the API simple, and Gio only produces a frame when something happens — input, a resize, a
`State` change, an animation. Within a frame, Giode avoids redoing work for the parts of the
UI that didn't change.

## Stateful components cache themselves

Components you keep and lay out every frame - `Button`, `Checkbox`, `Switch`, `Slider`, and a
`Box` with `OnHover` or `OnClick` - cache their own output: a frame where nothing about them
changed replays what they drew last time instead of laying them out again. They lay out again
when something they draw changes:

- their own fields, including through setters (`Label`, `SetChecked`, `SetValue`, ...);
- their hover, press or drag state, and animations in progress;
- States they read, the constraints they're given, the theme, display density, touch input,
  the locale or the active breakpoint.

On a screen of 200 rows, each a hoverable row, a button, a checkbox and a switch, a frame
takes 2.7 ms instead of 8.5 ms with no other change to the app.

A `Box` caches only when everything inside it is safe to cache: text, rich text, glyphs,
dividers, other boxes and the components above. A `Raw` drawing function, a `Canvas` or
any element built by a function you pass may read any variable, so a box holding one lays
its content out every frame as before. Inside a `SelectionArea` these boxes and components
don't cache themselves and are laid out every frame; `Memo` regions and `LazyList` rows
there still replay their recorded output, and are rebuilt when the selection changes (see
[SelectionArea](elements.md#selectionarea)).

Elements made fresh in the view function each frame aren't cached by themselves; wrap a
large, mostly static region of them in `Memo` (below) to skip building them too.

### A component of your own

A component built from elements can cache itself the same way, with a `MemoCache` and a
`Watch` over the fields it draws from, gathered in a comparable struct:

```go
type Rating struct {
	stars, of int
	st        giode.Styles
	cache     *giode.MemoCache
	build     func() giode.Element
	watched   giode.Watch[ratingDrawn]
}

type ratingDrawn struct {
	stars, of int
	st        giode.Styles
}

func (r *Rating) Layout(gtx giode.Context) giode.Dimensions {
	if r.cache == nil {
		r.cache = giode.NewMemoCache()
		r.build = func() giode.Element { return r.view() } // made once, not every frame
		r.watched.Snap = func() ratingDrawn { return ratingDrawn{r.stars, r.of, r.st} }
	}
	return r.cache.LayoutOwn(gtx, r.build, &r.watched)
}
```

`LayoutOwn` replays the last drawing until the snapshot differs from the one it was drawn
from, or anything a `Memo` watches changes. Comparing the snapshot allocates nothing, so a
screen of such components costs little more than replaying their drawings.

## Memo: only rebuild what changed

Wrap a part of the UI in `Memo` and it's rebuilt only when something it depends on changed;
every other frame replays the drawing recorded last time, at almost no cost regardless of
its size:

```go
giode.Memo("sidebar", func() giode.Element {
	return sidebar(user.Name)
}, user.Name) // deps: plain values the subtree reads
```

A memo rebuilds when:

- one of its **deps** changed (compared with `==`, or `reflect.DeepEqual` for slices, maps
  and other uncomparable values);
- a **State** read inside it changed — tracked automatically, no deps needed;
- its **constraints** (available size) or the display density changed;
- the **theme** changed (`SetMode`, `SetPalette`);
- a widget inside is **live**: animating (a spinner, a hover transition), focused (a text
  field), being dragged;
- a widget inside **changed its look in response to input**: the pointer entered a button,
  a press started, a checkbox toggled;
- `InvalidateCaches()` was called.

Input alone doesn't rebuild a memo. Every interactive component registers an *input pass*
while it's laid out; on a frame the memo replays, it runs the passes of the widgets inside,
which handle their events, and rebuilds only if one reports a visible change. Moving the
mouse over a cached list, or scrolling past it, replays it; entering a button rebuilds it
once, for the hover color.

The key identifies the cache across frames and must be unique among the memos laid out in
a frame. A memo not laid out in a frame is released. Inside a stateful component, keep a
`*MemoCache` (from `NewMemoCache`) instead of using a key.

Memos nest: a list memo can contain one memo per row, so changing one row rebuilds only
that row.

### Rules for widgets inside a memo

Built-in components handle this themselves. A custom component that handles input inside a
memo registers an input pass (see [extending.md](extending.md#input-inside-a-memo)), or
calls `kit.OpaqueInput()`, which makes enclosing memos lay it out live while the pointer is
over them. A custom component that changes without input — an animation, a timer — must
say so:

```go
kit.RequestFrame(gtx)       // animate: next frame ASAP, and keep enclosing memos live
kit.RequestFrameAt(gtx, t)  // timed change
kit.MarkLive()              // changed for another reason
```

Use `kit.RequestFrame` instead of `gtx.Execute(op.InvalidateCmd{})`, which enclosing memos
can't see. Values read from plain variables must be passed as deps. Driving a component
programmatically from outside a frame (`Button.Click()`) inside a cached subtree needs
`InvalidateCaches()`; `App.RunOnUI` does that for you.

## Measuring frame times

To find what makes scrolling or a streaming reply stutter, measure on the device:

```go
app := giode.New(giode.Config{ShowFrameTimes: true})
```

draws a counter in the top right corner with the average build time, render time and
frame rate over the last frames. The same numbers, with 95th percentiles and maximums,
come from `app.FrameStats()` and the MCP server's `frame_stats` tool; `app.ResetFrameStats()`
(or `frame_stats` with `reset`) starts a fresh measurement, so you can time one
interaction on its own.

- **Build** is the view function and layout, on the UI goroutine. When it's high, memo
  the parts that didn't change (above).
- **Render** is handing the frame to the GPU and presenting it. Presenting waits when the
  GPU is behind, so a GPU-bound UI shows up here: many large rounded or clipped shapes,
  big blurred shadows, large images. It can also include waiting for the display's refresh,
  so on a vsynced display it can approach the refresh interval while the app is otherwise
  idle; compare it against `1000 / fps`.
- **Frame rate** counts only frames that follow each other closely, so an idle app
  doesn't drag it down.

Recording the semantic tree for the MCP server adds to the build time, so measure without
the MCP server for the final numbers.

## Capping the frame rate

`Config.MaxFPS` draws at most that many frames a second; by default (0) the app draws at the
display's refresh rate. A cap saves CPU, GPU and battery while something animates, scrolls
or plays, at the cost of smoothness; an idle app draws no frames either way. `SetMaxFPS`
changes it while the app runs, for a battery saver setting, say:

```go
app := giode.New(giode.Config{MaxFPS: 30})
app.SetMaxFPS(0) // back to the refresh rate
```

A frame that comes sooner than 1/MaxFPS after the last waits out the difference before
it's laid out, and is then shown at the next refresh: on a 60 Hz display 30 gives 30 fps,
but 50 gives 30 too.

## Long lists: LazyList

[LazyList](components/lazylist.md) builds and lays out only the rows near the viewport,
whatever their heights. Give it `Version` and each visible row is a memo: scrolling replays
rows instead of rebuilding them, and offscreen rows keep their measured heights until their
version changes. Rows are identified by key, so the list keeps its place when rows above
are added or resize, and `AnchorBottom` keeps a chat pinned to its newest message.

A row out of view for a second drops its recorded drawing (and the line images it holds),
keeping only its measured height, so memory follows what's on screen rather than what was
scrolled past. The scrollbar's estimate of the content height, which asks every row for its
key and version, is worked out again only when a row's height, the row count or the scroll
position changes (or every half second, for edits offscreen), so an idle list of thousands
of rows costs about as much as one of ten.

## Text

Text is drawn from images, not paths. Each line is rasterized once on the CPU (glyphs at
four subpixel positions, composed into one image per line and color) and drawn as a
textured quad afterwards, so a screen of text costs the GPU about as much as a few images.
Lines are cached by content, so the same line anywhere reuses its image. Glyphs the
rasterizer can't handle (color emoji, bitmap fonts) fall back to Gio's vector path.

Layouts are cached too: the same text, style and width isn't shaped again. Long text is
laid out a paragraph at a time, so a streaming reply that appends to its last paragraph
reshapes only that paragraph, and only the lines inside the clip are drawn. (Paragraphs are
split for start-aligned, left-to-right text; other text is laid out whole, still cached.)

Rich text (`RichText`, and so `Markdown`) shapes each run of text once however many lines it
wraps to. A long run that grows - a reply streaming in - keeps the lines it wrapped to before
and shapes only the text from its last lines on, so a frame of a streaming reply costs about
the same at its 5,000th character as at its 50th. (Text with right-to-left letters is
shaped whole, still once.) A line that wraps or ends with a newline ends at its last
visible glyph: the spaces before the break don't count toward its width, so centered lines
are centered on their text.

Line images are kept while drawn and dropped after about five seconds unused, within a 32 MB
budget; layouts are dropped after about two seconds unused. Text scrolled back into view is
drawn again from the cached glyphs.

Gio's text shaper keeps its own cache of the last thousand paragraphs it laid out, however
long they are: in a long chat, a hundred megabytes and more, kept after the chat is closed.
Giode keeps its own layouts, so it replaces the shaper with a fresh one, over the same
fonts, once a quarter of a megabyte of text has been shaped with it - which caps Gio's copy
at about 25 MB. A new shaper takes well under a millisecond.

These caches are swept as frames are drawn, and a window that stops drawing would keep
everything it last held. So once a window has drawn nothing for ten seconds, Giode drops
the text images and layouts its last frame didn't use, Gio's shaper cache if much text went
through it, and the drawings of every `LazyList` row that isn't on screen; then it draws one
frame, which lets Gio's GPU caches drop what nothing draws any more. An idle window keeps
what it shows: after a burst of long chat messages, the heap went from 63 MB to 19 MB once
the window was idle. (A custom host using `giode.Frame` gets the text part, not the lists'.)

`RegisterFont` registers each file under its own family name and maps the name you give to
it, so text in those fonts is drawn from the registered faces as bitmaps. A face renamed
in Gio itself would be found again as a copy, drawn as shapes.

Icons (SVG, path data, IconVG and the built-in glyphs) are rasterized the same way, once
per size and color.

## Shadows

A blurred shadow is computed once per size, radius, blur and color: a Gaussian-blurred
rounded rectangle, drawn as nine slices of one small image. The edges and the middle
stretch, so a large card's shadow costs no more than a small one's, and the GPU never
blurs anything.

## Android refresh rate

Gio as released renders Android apps at most at 60fps on 120Hz displays: it asks for the
next frame only after presenting the current one, which misses the next vsync. Apps built
with the [giode CLI](https://github.com/xmhrnx/giode-cli) (`giode build android`)
are built against a copy of Gio with that fixed and run at the display's full rate. The
CLI's README describes how, and how to turn it off.

## Built-in savings

Without any memo, Giode still avoids repeated work every frame:

- colors are parsed once and cached;
- SVG documents and path data are parsed once; IconVG is decoded once;
- images keep their GPU upload instead of being uploaded every frame;
- boxes without absolutely positioned or z-indexed children don't allocate to sort them;
- rounded backgrounds, borders and shadows are drawn as plain rectangles with small
  paths only at the corners, so their GPU cost doesn't grow with the box's size, and a
  hidden-overflow box clips with a plain rectangle when its content stays clear of the
  rounded corners;
- `State.SetIfChanged` skips redraws when a value didn't actually change;
- an element carries hover styles only when `OnHover` was called, so the many that don't
  are half the size;
- a stateful component checks whether its own fields changed by comparing them with what it
  last drew, without allocating; components of your own can do the same (see
  [A component of your own](#a-component-of-your-own)).

## Benchmarks

`internal/perf` lays out typical screens frame after frame with a real input router: 200
rows of controls, a page of text, a 1,000-message chat (idle, scrolling, streaming a reply,
in a `SelectionArea`) and a table. Run them with

```
go test ./internal/perf -bench . -benchmem
```

`TestRetainedMemory` in the same package runs a chat for 2,000 frames with a message
arriving every frame and logs the heap and the text caches' sizes, to catch a cache that
grows without bound.
