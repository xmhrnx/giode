# Switch, Segmented, Chip, Badge, Avatar

## Switch

An on/off switch with an animated knob, the standard settings control on phones.

```go
wifi := giode.Input.Switch("Wi-Fi").OnChange(func(on bool) { ... })
```

With a label it fills the row, label on the left and switch on the right, as in a settings
list; the whole row is tappable. `Checked()`, `SetChecked(v)` (without calling OnChange),
`Toggle()`, `Disabled(v)`. Styles: `Background` is the track when on (default
`theme.primary`), `BorderColor` the track when off (default `theme.border`), `Color` and
`FontSize` the label. `KnobColor(off, on)` colors the knob (white by default), for a light
"on" track.

## Segmented

Mutually exclusive options in one rounded track, for switching views or modes:

```go
mode := giode.Input.Segmented("Chat", "Deep research").OnChange(func(i int) { ... })
mode.Styles(giode.Styles{Width: -1}) // fill the row with equal segments (or Width: 240)
mode.Icons(chatIcon, searchIcon)      // optional, before each label
```

`Selected()`, `SetSelected(i)` (without calling OnChange), `Choose(i)`. The highlight
slides to the picked option. Styles: `Background` the track (default a faint tint of
`theme.text`, mixed opaque so it stays faint in dark mode), `BorderColor` the selected
segment (default `theme.surface`), `Color`, `FontSize`, and `Width`: -1 fills the row, a
positive width is shared equally by the segments.

## Chip

A small rounded label for filters, tags and choices:

```go
giode.Chip("Go")                                         // plain, build it every frame
filter := giode.Chip("Unread").Selected(on).OnClick(toggleUnread) // create once
tag := giode.Chip("draft").OnRemove(func() { removeTag("draft") })
```

`Icon(el)` shows an icon before the label; `Selected(true)` draws it filled in
`theme.primary` with `theme.onprimary` text. The chip's styles apply to the unselected
chip and don't override the selected colors; set those with
`SelectedStyles(giode.Styles{Background: ..., Color: ...})`. A chip with `OnClick` or `OnRemove` holds press state, so create it once
(or with `Keyed`).

## Badge

A counter or a dot, alone or pinned to the corner of another element:

```go
giode.Badge("12")                       // a pill
giode.Badge("3").On(giode.Icon("inbox")) // over an icon's top-right corner
giode.Badge("").On(avatar)             // a dot
```

Styles: `Background` (default `theme.danger`), `Color` (default white), `FontSize`. The
badge doesn't change its target's size.

## Avatar

A picture, or initials on a color derived from the name, in a circle:

```go
giode.Avatar("Ada Lovelace")                          // "AL" on a stable color
giode.Avatar("Ada Lovelace").Image(photo)             // the photo, cropped round
giode.Avatar("Claude", giode.Styles{Width: 28, Background: "theme.primary"})
```

`Width` sets the diameter in dp (default 40). A name always gets the same color.
