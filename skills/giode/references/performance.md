# Rendering performance

Giode is immediate mode: the view function describes the whole UI every frame. That keeps
the API simple, and Gio only produces a frame when something happens — input, a resize, a
`State` change, an animation. Within a frame, Giode avoids redoing work for the parts of the
UI that didn't change.

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

## Long lists: LazyList

[LazyList](components/lazylist.md) builds and lays out only the rows near the viewport,
whatever their heights. Give it `Version` and each visible row is a memo: scrolling replays
rows instead of rebuilding them, and offscreen rows keep their measured heights until their
version changes. Rows are identified by key, so the list keeps its place when rows above
are added or resize, and `AnchorBottom` keeps a chat pinned to its newest message.

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
with the [giode CLI](https://github.com/mehran9675/giode-cli) (`giode build android`)
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
- `State.SetIfChanged` skips redraws when a value didn't actually change.
