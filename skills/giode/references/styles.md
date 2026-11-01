# Style properties

Every element accepts a `giode.Styles{...}` struct. Zero values mean "unset": the element
falls back to its default. Each property is defined in its own file under
`styles/properties/`.

```go
card := giode.Styles{
	FlexDirection: giode.FlexDirectionColumn,
	Gap:           12,
	Padding:       giode.UniformInset(16),
	Background:    "#1e293b",
	BorderRadius:  12,
	BoxShadow:     giode.BoxShadow{Y: 4, Blur: 12, Color: "#00000060"},
}
```

## Units

Lengths — sizes, paddings, margins, gaps, radii, border widths, offsets, shadows — are in
**dp** (device-independent pixels): they're scaled by the display density, so a layout
keeps its physical size on HiDPI screens, in step with font sizes (sp). On a standard
display 1dp is 1px. Custom elements resolve a `Styles` value to pixels with
`st.Px(gtx.Metric)`.

## Colors

Colors are strings in any CSS syntax: hex (`"#fff"`, `"#1e293b"`, `"#00000060"`),
functional (`"rgb(59 130 246)"`, `"rgba(59, 130, 246, 0.5)"`, `"hsl(217 91% 60%)"`), any of
the 148 CSS named colors (`"rebeccapurple"`) or `"transparent"`. Theme tokens
(`"theme.primary"`, `"theme.surface"`, `"theme.text"`, `"theme.border"`, `"theme.muted"`,
`"theme.accent"`, `"theme.onprimary"`, `"theme.success"`, `"theme.warning"`,
`"theme.danger"`, `"theme.info"`, …) follow `SetMode`/`SetPalette`, and take an alpha:
`"theme.primary/0.5"` or `"theme.primary/50%"`. An empty string means unset; an
unparseable value renders white so a typo is visible — validate input with `ParseColor`.
Parsed colors are cached.

**Translucent colors.** Gio composites translucent colors in linear light, which is
physically correct but differs from browsers and design tools, which blend in sRGB. A
faint overlay comes out much brighter than in CSS: `"#ffffff0d"` (5% white) over
`"#101115"` renders close to `"#434344"`, where a browser shows about `"#1c1d21"`. For
tints, hover and pressed shades, use an opaque color mixed in sRGB space instead:
`giode.Mix("#ffffff", "#101115", 0.05)` returns `"#1C1D21FF"`. With a theme token on
either side, `Mix` returns a CSS `color-mix()` value instead
(`giode.Mix("theme.text", "theme.background", 0.08)` is
`"color-mix(in srgb, theme.text 8%, theme.background)"`), which resolves against the
palette whenever it's painted, so a mixed color made once, at construction, follows
light/dark switches. `color-mix(in srgb, …)` can also be written by hand.

**Inheritance.** Like CSS, a Box's `Color` is inherited: Text, icons and glyphs inside it
that set no `Color` of their own use it, and the value `"currentColor"` refers to it
explicitly. A button's icon therefore follows the label color.

Styles compose with `Merge(base, overrides...)` or the method form
`base.With(override)`.

## Layout

| Property | Type | Values | Notes |
| --- | --- | --- | --- |
| `FlexDirection` | string | `FlexDirectionRow` (default), `FlexDirectionColumn` | Main axis (CSS flex-direction). |
| `Align` | string | `AlignStart`, `AlignCenter`, `AlignEnd` | Cross-axis alignment (align-items). |
| `Justify` | string | `JustifyStart` (default), `JustifyCenter`, `JustifyEnd`, `JustifySpaceBetween`, `JustifySpaceAround`, `JustifySpaceEvenly` | Main-axis distribution. |
| `Wrap` | string | `WrapWrap`, `WrapNoWrap` | Whether children continue onto new lines. Unset: Row wraps, Stack does not. |
| `Gap` | int | dp | Space between children and between wrapped lines. |
| `FlexGrow` | int | ≥0 | Share of spare main-axis space. |
| `Width` / `Height` | int | `0` hug, `-1` expand, `>0` fixed dp | |
| `WidthPct` / `HeightPct` | int | 1–100 | Percent of available space; takes precedence over `Width`/`Height`. |
| `MinWidth` / `MaxWidth` / `MinHeight` / `MaxHeight` | int | dp | Clamp the resolved size, including for expanding boxes (`Width: -1`, `FlexGrow`): `Width: -1, MaxWidth: 760` is a column at most 760dp wide, placed by the parent's `Justify`/`Align`. A minimum beats a maximum. |
| `AspectRatio` | float | w/h | Derives the hugging axis from the resized axis. |
| `Padding` / `Margin` | Inset | dp per side | `UniformInset(v)`, `SymmetricInset(h, v)` or `giode.Inset{Top, Right, Bottom, Left}`. |
| `Position` | string | `PositionStatic` (default), `PositionRelative`, `PositionAbsolute` | Absolute removes the element from the flow; see below. |
| `Top` / `Right` / `Bottom` / `Left` | int | dp | Offsets of absolute children. |
| `ZIndex` | int | any | Stacking order of siblings, flow and positioned alike. |
| `Overflow` | string | `OverflowVisible` (default), `OverflowHidden` | Hidden clips content to the padding box, following rounded corners. |
| `ScrollBar` | struct | — | Customizes the scrollbar a `Scroll` draws: `Width`, `TrackColor` (zero alpha: no track), `ThumbColor`, `Radius`, `MinThumbLength`, `Inset`, `AutoHide` (fades out when idle); `Hidden`, `HiddenMaxWidth` (dp, window width) and `HiddenOn` (breakpoint names) hide it, see [scroll.md](components/scroll.md#hiding-the-scrollbar). |

## Paint

| Property | Type | Values | Notes |
| --- | --- | --- | --- |
| `Background` | string | any color | Fill color. |
| `BackgroundImage` | `image.Image` | — | Image behind the content. |
| `BackgroundFit` | string | `FitContain` (default), `FitCover`, `FitStretch`, `FitNone` | Scaling of the background image. |
| `BorderWidth` / `BorderColor` | int / string | — | Border around the box. |
| `BorderRadius` | int | dp | Rounds the box corners. |
| `BorderRadii` | `Corners` | dp per corner | `UniformCorners(r)`, `TopCorners(r)`, `BottomCorners(r)` or `giode.Corners{TopLeft, …}`; overrides `BorderRadius`. |
| `BackgroundGradient` | `*LinearGradient` | — | `giode.Linear(angle, colors...)` or explicit `Stops`; CSS angles (90 = left to right, 180 = top to bottom). Any number of stops. |
| `BoxShadow` | struct | `{X, Y, Blur, Spread int; Color string}` | Drop shadow behind the box. |
| `Opacity` | float | 0–1 | Transparency of the element and its children; 0 means unset. |
| `Transform` | struct | `{Rotate, Scale, ScaleX, ScaleY float32; TranslateX, TranslateY int; OriginX, OriginY float32}` | Paint-only transform around the center, like CSS: layout is unaffected. |
| `Display` | string | `DisplayNone` | Removes the element: no space, no paint. |
| `Visibility` | string | `VisibilityHidden` | Keeps the space, paints nothing, takes no input. |
| `Transition` | `StyleTransition` | `{Duration time.Duration; Easing}` | Animates `Background`, `Color`, `BorderColor`, shadow color and `Opacity` between a Box's base and `OnHover` styles. `EaseInOut` (default), `EaseLinear`, `EaseIn`, `EaseOut`. |

## Text

| Property | Type | Values | Notes |
| --- | --- | --- | --- |
| `Color` | string | any color | Text color; empty resolves to black. |
| `FontSize` | sp | — | Default 14sp. |
| `FontWeight` | string | `FontWeightThin`, `FontWeightExtraLight`, `FontWeightLight`, `FontWeightNormal`, `FontWeightMedium`, `FontWeightSemiBold`, `FontWeightBold`, `FontWeightExtraBold`, `FontWeightBlack`, or the CSS numbers `"100"`…`"900"` | Default normal. Register each weight's face (e.g. Inter SemiBold as `font.SemiBold`) with `RegisterFonts`. |
| `FontStyle` | string | `FontStyleNormal`, `FontStyleItalic` | |
| `FontFamily` | string | — | Typeface must exist in the registered fonts. |
| `TextAlign` | string | `TextAlignStart` (default), `TextAlignCenter`, `TextAlignEnd` | |
| `LineHeight` | sp | — | Distance between baselines. |
| `MaxLines` | int | — | Limits the number of lines. |
| `TabWidth` | int | columns | Distance between tab stops (default 4). Tabs are expanded to spaces, since fonts rarely have a tab glyph; in a monospaced font code lines up exactly. |
| `TextOverflow` | string | `TextOverflowClip` (default), `TextOverflowEllipsis` | Ends cut-off text with "…"; implies one line without `MaxLines`. |
| `TextTransform` | string | `TextTransformUppercase`, `TextTransformLowercase`, `TextTransformCapitalize` | |
| `TextDecoration` | string | `TextDecorationNone` (default), `TextDecorationUnderline`, `TextDecorationLineThrough` | |

## Interaction

| Property | Type | Values | Notes |
| --- | --- | --- | --- |
| `Cursor` | string | `CursorDefault`, `CursorPointer`, `CursorText`, `CursorGrab`, … | Cursor while hovering interactive components. |
| `Fit` | string | `FitContain`, `FitCover`, `FitStretch`, `FitNone` | Scaling of `Image` elements. |

## Absolute positioning

Children of a `Box` with `Position: PositionAbsolute` are removed from the flow and placed
at `Top`/`Left` (or anchored from `Right`/`Bottom`) of the parent's content box, painting
above the flow content. They do not affect the parent's size.

```go
giode.Box(giode.Styles{}, content,
	giode.Box(giode.Styles{
		Position: giode.PositionAbsolute,
		Top: 8, Right: 8, ZIndex: 10,
		Background: "#ef4444", BorderRadius: 99,
	}, giode.Text("3")),
)
```

Offsets are in dp, and **0 means "not set"**. To pin a box flush to an edge, use
`giode.Flush`, an explicit zero:

```go
// A composer pinned to the bottom, spanning the full width.
giode.Box(giode.Styles{
	Position: giode.PositionAbsolute,
	Left: giode.Flush, Right: giode.Flush, Bottom: giode.Flush,
}, composer)
```

`Left` wins over `Right` and `Top` over `Bottom`. With both of a pair set, a box without its
own `Width` (or `Height`) stretches between them, as in CSS. Negative offsets move a box
past the edge.

Notes:

- `ZIndex` orders siblings: higher values paint above and receive input first. It applies to
  flow and positioned children alike, so a normal-flow child with `ZIndex: 2` paints above a
  positioned one with `ZIndex: 1`. At equal z-index, positioned children paint above flow
  children; otherwise children paint in document order. It doesn't change where children
  are laid out.
- Absolute positioning applies to `Box` children; wrap other elements in a `Box`.

## Not supported (by design)

Complex or platform-specific CSS: transitions, animations, transforms, box-sizing,
grid, float, `position: sticky`, `white-space` (Gio's label always wraps), pseudo
elements, media-query orientation. Raise an issue if one of these matters to you.
