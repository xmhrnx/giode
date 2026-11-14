# Elements

Elements are the stateless building blocks. They are cheap to create inside the view
function every frame.

## Box

The flex container. Children are laid along `FlexDirection` (row by default) with
`Align`, `Justify`, `Gap` and `Wrap`:

```go
giode.Box(giode.Styles{
	FlexDirection: giode.FlexDirectionColumn,
	Align:         giode.AlignCenter,
	Gap:           12,
	Padding:       giode.UniformInset(16),
}, child1, child2)
```

Box also paints the background stack (color, image, border, shadow) and acts as the
containing block for absolutely positioned children.

## Row and Stack

Shorthand containers with the flex direction preset:

```go
giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter}, a, b, c) // horizontal
giode.Stack(giode.Styles{Gap: 8}, a, b, c)                        // vertical
```

`Row` sets `FlexDirection` to row, `Stack` to column.

## Hover

Every stateless element with something to paint supports `OnHover`: `Box` (and `Row`/
`Stack`), `Text`, `Divider`, `Image`, `Progress` and `Spinner`. Chain it to apply styles
while the pointer is over the element:

```go
card := giode.Box(base, children...).OnHover(giode.Styles{
	Background:   "#1e40af",
	BorderColor:  "#3b82f6",
	BorderWidth:  1,
	BoxShadow:    giode.BoxShadow{Y: 4, Blur: 12, Color: "#00000060"},
	Cursor:       giode.CursorPointer,
})

label := giode.Text("Docs", giode.Styles{Color: "#94a3b8"}).OnHover(giode.Styles{
	Color:          "#f8fafc",
	TextDecoration: giode.TextDecorationUnderline,
	Cursor:         giode.CursorPointer,
})
```

Hover styles merge on top of the base styles for both layout and painting, and can set
anything the element's own `Styles` support — colors, borders, shadows, padding,
cursor, and so on (see each element's own page for which properties it actually
paints). The hover state lives in the element, so **create it once and reuse it** like
a stateful component; rebuilding it inside the view function resets the hover each
frame. `Hovered()` reports the state observed on the last frame.

`Responsive` and `Raw` don't support `OnHover` - they're structural (they just delegate
to another element or to raw Gio), not something that paints its own styles to merge
into. The stateful components (`Button`, `Input.*`, `Tabs`, ...) already shade
themselves on hover internally instead of going through `OnHover`.

Component authors: `OnHover` is built on `kit.HoverTracker`, an embeddable building
block for adding the same capability to a custom stateless element - see
[Extending the library](extending.md).

## Wrap

By default a `Row` wraps children onto new lines when they overflow (`WrapWrap`), while
a `Stack` keeps them on a single line (`WrapNoWrap`). Override with the `Wrap` property:

```go
giode.Row(giode.Styles{Wrap: giode.WrapNoWrap, Gap: 8}, chips...)
giode.Stack(giode.Styles{Wrap: giode.WrapWrap, Gap: 8}, chips...)
```

Wrapped lines are stacked on the cross axis and separated by `Gap`.

## Text

```go
giode.Text("Hello", giode.Styles{Color: "#fff", FontSize: 18})
```

The styles argument is optional. Supports text alignment, weight, style, family,
line-height, max-lines, tab width and decoration.

### Selectable text

```go
giode.Text(reply, styles).Selectable(msg.ID)
```

`Selectable(key)` lets the user select and copy the text. With a mouse: drag to select,
double-click for a word, Ctrl+C (Cmd+C) to copy, or right-click to select it all and get a
Copy button. On a touch screen: long-press to select it all and get a Copy button; dragging
over the text still scrolls whatever it's in. `key` identifies the text across frames (a
message ID), since `Text` is rebuilt every frame and the selection has to live somewhere.

### SelectionArea

```go
giode.SelectionArea("chat", giode.Box(giode.Styles{FlexDirection: giode.FlexDirectionColumn},
	giode.Text(title, titleStyles),
	giode.Markdown(reply, opts),
	giode.Text(footer, footerStyles),
))
```

`SelectionArea(key, child)` makes all the `Text`, `RichText` and `Markdown` inside `child`
selectable as one, the way text on a web page is: a selection can start in one paragraph and
end in another, and copies as their text joined by newlines. With a mouse: drag to select,
double-click for a word, triple-click for a whole paragraph, Shift+click to extend,
Ctrl+C (Cmd+C) to copy, Ctrl+A (Cmd+A) for everything, Escape to clear, right-click for a
Copy button. On a touch screen: long-press selects a word and dragging the held finger
extends it; letting go shows a Copy button. A finger that moves before the long-press
scrolls as usual.

Each element draws its part of the selection in its own text color at 30% opacity. The text
of buttons, menus, tabs, inputs and other interactive components is left out (custom
components can do the same with `kit.Unselectable`), as are elements that are
`Selectable` themselves. Text inside a `Memo` or a `LazyList` row takes part too: the cached
output is replayed until the selection changes, then rebuilt once. The boxes and components
that cache their own output (see [performance.md](performance.md)) don't inside an area:
they're laid out every frame.

## RichText

One wrapping paragraph of differently styled runs:

```go
giode.RichText(giode.Styles{FontSize: 15, LineHeight: 22},
	giode.Span("Run "),
	giode.Span("go test", giode.Styles{FontFamily: "JetBrains Mono", Background: "theme.surface"}),
	giode.Span(" or read the "),
	giode.Link("docs", "https://gioui.org"),
).OnLink(func(url string) { ... })
```

The paragraph's styles (`Color`, `FontSize`, `FontWeight`, `FontStyle`, `FontFamily`) are
the defaults each span inherits; `TextAlign`, `LineHeight` and `TabWidth` apply to the
paragraph. A span can also set `TextDecoration` (underline, line-through) and
`Background`, a rounded highlight behind it as for inline code. `Link` is underlined in
`theme.primary`; tapping it calls `OnLink`, or opens the URL in the browser when no
handler is set.

`Selectable(key)` works as for `Text`, except that the whole paragraph is selected and
copied at once. To select part of a paragraph, or across paragraphs, put it in a
`SelectionArea` instead.

## Markdown

```go
giode.Markdown(reply, giode.MarkdownOptions{
	Styles:     giode.Styles{FontSize: 15, LineHeight: 22},
	CodeFont:   "JetBrains Mono",
	OnLink:     openURL,
	Selectable: msg.ID,
})
```

Renders CommonMark with the GitHub extensions: headings, paragraphs with bold, italic,
strikethrough, inline code and links, bulleted, numbered and task lists, block quotes,
fenced code blocks (with the language and a copy button), tables and rules. Parsing is
cached per source, so a document laid out every frame is parsed once, and a streaming
reply only when it changes. Raw HTML is dropped and images show their alt text.

`Selectable` puts the document in its own `SelectionArea`, keyed by its value: a drag
selects across paragraphs, headings, lists, tables and code, and copies their text joined
by newlines. A document already inside a `SelectionArea` is part of that area's selection,
with or without `Selectable`.

## Divider

A thin horizontal line. Color defaults to a subtle gray, height to 1px:

```go
giode.Divider(giode.Styles{Color: "#334155"})
```

## Table

Rows of cells with column widths aligned across all of them - see
[table.md](components/table.md):

```go
giode.Table(giode.Styles{},
	giode.TableHeader(giode.Styles{}, giode.TableCell(giode.Styles{}, giode.Text("Name"))),
	giode.TableRow(giode.Styles{}, giode.TableCell(giode.Styles{}, giode.Text("Ada"))),
)
```

## Raw

Drops down to plain Gio at any point of the tree:

```go
giode.Raw(func(gtx giode.Context) giode.Dimensions {
	return giode.Dimensions{Size: image.Pt(10, 10)}
})
```

## Canvas

Custom drawing - progress rings, gauges, charts, maps - without raw Gio ops. The canvas
is sized like a Box (`Width`/`Height`, -1 to fill; without a size it takes what its parent
offers) and its function runs every frame with a `*giode.Painter`:

```go
giode.Canvas(giode.Styles{Width: 96, Height: 96}, func(p *giode.Painter) {
	c := p.Width() / 2
	p.Arc(c, c, c-6, 8, 0, 360, "theme.border")              // track
	p.Arc(c, c, c-6, 8, 0, 360*progress, "theme.primary")    // progress, round ends
	p.Sector(c, c, 20, 30, 90, 45, "#e5484d")                 // a donut segment
})
```

Coordinates and lengths are in dp from the top-left corner; angles are degrees clockwise
from 12 o'clock. Colors are any `Color`, theme tokens included (`""` is the inherited text
color), so a canvas follows light/dark switches.

Labels go on the drawing itself:

```go
giode.Canvas(giode.Styles{Width: 240, Height: 160}, func(p *giode.Painter) {
	p.Line(20, 140, 220, 140, 1, "theme.border") // x axis
	p.Circle(120, 60, 4, "theme.primary")
	p.Text(128, 60, "M", giode.TextAnchorLeft, giode.Styles{FontWeight: giode.FontWeightBold})
	p.Text(120, 150, "r (km)", giode.TextAnchorTop, giode.Styles{FontSize: 12, Color: "theme.muted"})
	p.TextRotated(8, 80, "v (km/s)", giode.TextAnchorCenter, -90, giode.Styles{FontSize: 12})
})
```

| Painter method | Draws |
| --- | --- |
| `Rect(x, y, w, h, radius, fill)` | a rectangle, rounded |
| `Circle(cx, cy, r, fill)` | a filled circle |
| `Line(x1, y1, x2, y2, width, color)` | a straight line |
| `Arc(cx, cy, r, width, start, sweep, color)` | a stroked arc with round ends; a sweep of 360 is a ring |
| `Sector(cx, cy, inner, outer, start, sweep, fill)` | a pie slice (inner 0) or donut segment |
| `Text(x, y, s, anchor, styles)` | text with its anchor point at (x, y): `TextAnchorCenter`, `TextAnchorLeft`, `TextAnchorBottom`, … Styled like a `Text` element (`Color`, `FontSize`, `FontWeight`, `FontFamily`, `Background`, `Padding`; `MaxWidth` wraps it). Returns its size in dp. |
| `TextRotated(x, y, s, anchor, deg, styles)` | the same, turned `deg` degrees clockwise around the anchor: `-90` for a y axis label |
| `MeasureText(s, styles)` | the size `Text` would draw `s` at, in dp |
| `Path()` | a `*giode.CanvasPath`: `MoveTo`, `LineTo`, `QuadTo`, `CubeTo`, `ArcAround`, `Close`, then `Fill(color)` or `Stroke(width, color)` |
| `Width()`, `Height()` | the canvas size in dp |
| `Context()` | the Gio context, for raw drawing in pixels |

## Formulas

Package `formula` typesets mathematics written in LaTeX notation, for apps built on
equations and derivations:

```go
import "github.com/xmhrnx/giode/formula"

formula.New(`v = \sqrt{\frac{2GM}{r}}`, giode.Styles{FontSize: 20})
formula.New(`\oint \vec{E}\cdot d\vec{A} = \frac{Q}{\varepsilon_0}`)
formula.New(`i\hbar\frac{\partial}{\partial t}\Psi = -\frac{\hbar^2}{2m}\nabla^2\Psi + V\Psi`)

// A derivation: lines broken with \\, aligned at &.
formula.New(`E &= \tfrac{1}{2}mv^2 \\ &= \frac{p^2}{2m}`)

// Within a line of text: smaller fractions, limits beside the sum.
giode.Row(giode.Styles{Align: giode.AlignBaseline},
	giode.Text("The series "),
	formula.Inline(`\sum_{n=1}^\infty \frac{1}{n^2} = \frac{\pi^2}{6}`),
	giode.Text(" converges."),
)
```

It sets fractions (`\frac`, `\dfrac`, `\tfrac`, `\binom`), roots (`\sqrt`, `\sqrt[3]`),
superscripts and subscripts, large operators with limits (`\sum`, `\prod`, `\int`, `\iint`,
`\oint`, with `\limits`/`\nolimits`), `\lim`, `\max` and the named functions (`\sin`,
`\ln`, `\operatorname{…}`), delimiters sized to what they enclose (`\left( … \right)`,
with `[`, `\{`, `|`, `\|`, `\langle`, and `.` for none) or by hand (`\big`, `\Big`,
`\bigg`, `\Bigg`), accents (`\vec`, `\hat`, `\bar`, `\overline`, `\dot`, `\ddot`,
`\tilde`, `\underline`), `\text`, `\mathrm`, `\mathbf`, Greek letters, primes, spaces
(`\,`, `\;`, `\quad`), and the usual operators, relations and arrows (`\nabla`,
`\partial`, `\hbar`, `\propto`, `\Rightarrow`, `\approx`, `\leq`, `\pm`, `\times`,
`\cdot`, `\infty`, `\in`, `\to` …), spaced as TeX spaces them.

Also: letter styles `\mathcal` and `\mathscr` (ℰ, ℒ), `\mathbb` (ℝ, ℤ), `\mathfrak` (𝔤),
`\mathsf`, `\mathtt`, `\mathbf` and `\boldsymbol` (bold italic, Greek too); matrices
(`\begin{pmatrix}`, `bmatrix`, `Bmatrix`, `vmatrix`, `Vmatrix`, `matrix`, `smallmatrix`) and
`cases`, with `&` between cells and `\\` between rows; `aligned` inside other math;
`\overset`, `\underset` and `\stackrel` (`\stackrel{\text{def}}{=}`); `\boxed`; `\textcolor{c}{…}`
and `\color{c}` with any color `Styles` take, theme tokens included; `\phantom`, `\hphantom` and
`\vphantom`; and a long list of symbols past the usual - `\odot` (so `M_\odot` is M☉),
`\astrosun` ☉, `\therefore`, `\leqslant`, `\prec`, `\hookrightarrow`, `\Longrightarrow`,
`\nexists`, `\wp`, `\aleph`, the negated relations (`\nleq`, `\nsim`), harpoons and more.
Any symbol can also be typed as itself: `ℰ`, `☉` and `∯` set like their commands.

- Letters are variables, set in italic; write `\mathrm{d}x` for an upright d and `\text{km/s}`
  for units.
- `New` sets a formula in display style, as on a line of its own; `Inline` in text style.
- Styles: `FontSize` and `Color` (theme tokens follow the theme), and the box styles
  (`Padding`, `Margin`, `Background`). Without box styles a formula keeps its baseline, so
  `AlignBaseline` lines it up with text.
- A command it doesn't know is shown as written, so a typo is visible rather than lost.
- Formulas are set in STIX Two Math (SIL Open Font License), which importing the package
  adds to the app's fonts: about 0.8 MB, only in apps that import it. Its symbols then also
  show in ordinary text, so `giode.Text("∇ ℏ ⇒ ∝")` works with any font.

## Responsive

Simplified media queries: renders different views depending on the available width.

```go
giode.Responsive(
	giode.Breakpoint{MaxWidth: 600, Name: "compact", View: func() giode.Element { return phoneView }},
	giode.Breakpoint{MaxWidth: 1024, Name: "medium", View: func() giode.Element { return tabletView }},
	giode.Breakpoint{View: func() giode.Element { return desktopView }}, // fallback
)
```

Breakpoints are evaluated in order; the first whose `MaxWidth` is at least the available
width matches, so list the narrowest first. A breakpoint without `MaxWidth` always matches.
Only the matching view is built each frame.

`MaxWidth` is in dp, like every length in `Styles`, so a layout switches at the same
physical size on a desktop and on a 3x phone. It's compared with the width the parent
container makes available, not the window's. The older `Width` field is in physical
pixels and is deprecated; it applies only when `MaxWidth` is 0.

Content inside the chosen view can ask which breakpoint matched, without a flag passed
through every page:

```go
if giode.ActiveBreakpoint() == "medium" { ... }
if giode.Compact() { ... } // the innermost Responsive matched a breakpoint named "compact"
```

Both answer for the innermost enclosing `Responsive`. Outside one, `ActiveBreakpoint`
returns `""` and `Compact` reports whether the window is narrower than
`giode.CompactWidth` (600 dp).
Like `AbsolutePosition`, they're only meaningful while the view is being laid out. A
`Memo` inside the view rebuilds when the active breakpoint changes.
