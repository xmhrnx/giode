# Button

Clickable button.

```go
btn := giode.Button("Click me",
	giode.Styles{Background: "#3b82f6", BorderRadius: 8},
).OnClick(func() { count++ })

// in the view:
giode.Stack(giode.Styles{}, btn)
```

## API

| Member | Description |
| --- | --- |
| `Button(label string, st ...Styles) *Button` | Creates the button. |
| `OnClick(fn func()) *Button` | Click handler. |
| `Label(label string) *Button` | Replaces the label. |
| `Variant(v ButtonVariant) *Button` | Visual style: `VariantContained` (default), `VariantOutlined` or `VariantGhost`. |
| `Styles(st Styles) *Button` | Replaces the styles. |
| `Icon(el Element) *Button` | Element before the label. Icons without their own `Color` follow the label color. |
| `TrailingIcon(el Element) *Button` | Element after the label (a chevron, an external-link mark). |
| `Disabled(v bool) *Button` | Dimmed, default cursor, ignores input. |
| `Loading(v bool) *Button` | A spinner replaces the icon and clicks are ignored. |
| `IsDisabled()`, `IsLoading()` | Current state. |
| `Click()` | Simulates a click (ignored while disabled or loading). |

```go
save := giode.Button("Save").
	Icon(giode.Icon("save", giode.Styles{Width: 16})).
	OnClick(func() {
		save.Loading(true)
		go func() {
			err := store.Save()
			app.Post(func() { save.Loading(false); showResult(err) })
		}()
	})
```

## Touch screens

While the user is on a touch screen (see [extending.md](../extending.md#touch-input)),
`Button`, `IconButton` and `Checkbox` grow their hit area to at least 48x48dp, the Android
guideline, with the visual centered inside and unchanged. With a mouse they keep their
natural size.

## Variants

```go
giode.Button("Save", giode.Styles{Background: "#3b82f6", Color: "white"})                              // contained (default)
giode.Button("Cancel", giode.Styles{BorderColor: "#3b82f6", Color: "#3b82f6"}).Variant(giode.VariantOutlined)
giode.Button("Skip", giode.Styles{Background: "#3b82f6", Color: "white"}).Variant(giode.VariantGhost)
```

- `VariantContained` fills with `Background` (or `theme.primary` if unset), or with
  `BackgroundGradient` when set; hover and press shade every gradient stop.
- `VariantOutlined` is transparent with a visible border (`BorderColor`, or `Background`
  if unset; `BorderWidth` defaults to 1), filling faintly with the border color on
  hover/press.
- `VariantGhost` is fully transparent at rest - only the label is visible - filling
  faintly with `Background` on hover/press.

## Styles

| Property | Role |
| --- | --- |
| `Background` | Face color (`VariantContained`) or accent color (`VariantOutlined`/`VariantGhost`); pressed and hovered shades are derived automatically. |
| `Color` | Label color. |
| `FontSize` | Label size. |
| `Padding` | Face padding; defaults to 16x8. |
| `BorderRadius`, `BorderWidth`, `BorderColor`, `Opacity`, `Cursor` | As usual. |
