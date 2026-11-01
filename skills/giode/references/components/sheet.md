# Sheet

A modal panel over a dimmed scrim: a bottom sheet that slides up from the bottom edge, or
with `Adaptive` a centered dialog on wide windows - the usual phone pattern for pickers,
menus and short forms. The open state lives in a `*bool` you own.

```go
open := new(bool)
picker := giode.Sheet(open, func() giode.Element {
	return giode.Stack(giode.Styles{Gap: 4}, rows...)
}).Title("Choose a model").
	Footer(func() giode.Element { return giode.Row(giode.Styles{Justify: giode.JustifyEnd}, done) }).
	Adaptive(true)

*open = true

// in the view, anywhere: it takes no space and paints over the window
giode.Stack(giode.Styles{}, page, picker)
```

The sheet has a drag handle, an optional title, a body that scrolls once it outgrows the
window, and an optional footer pinned below the body. By default it's as tall as its
content, up to nearly the full window. The scrim, Escape and the Back button close it,
and so does dragging the handle or title down.

| Member | Description |
| --- | --- |
| `Sheet(open *bool, content func() Element, st ...Styles) *Sheet` | Creates the sheet. Styles apply to the panel: `Background` (default `theme.surface`), `Color`, `Padding` (default 16dp on the sides). |
| `Title(t string) *Sheet` | Heading above the body. |
| `Footer(f func() Element) *Sheet` | Content pinned below the scrolling body, such as action buttons. |
| `Detents(d ...Detent) *Sheet` | Heights it rests at, smallest first: `DetentFit` (default), `DetentMedium` (half the window), `DetentLarge`. It opens at the first; dragging the handle moves between them, and below the first closes it. |
| `Adaptive(on bool) *Sheet` | From 600dp wide, show as a centered dialog (at most 560dp wide) instead. |
| `Dismissible(on bool) *Sheet` | Whether the scrim, Back, Escape and dragging close it (on by default). |
| `Styles(st Styles) *Sheet` | Replaces the panel styles from the next frame. |
| `Opened() bool` | Whether it's open. |

On a bottom sheet the panel is at most 640dp wide and centered, so it stays a sheet on a
tablet in portrait.
