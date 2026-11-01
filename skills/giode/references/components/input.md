# Input

Form controls are grouped under the `Input` API:

| Constructor | Component |
| --- | --- |
| `giode.Input.Text(placeholder, st...)` | Single-line text field |
| `giode.Input.Checkbox(label, st...)` | Checkbox (label may be `""`) |
| `giode.Input.Slider(st...)` | Horizontal slider in [0, 1] |
| `giode.Input.Select(st, options...)` | Dropdown among options |
| `giode.Input.AutoComplete(placeholder, options, st...)` | Text field with filtered suggestions |

## Text field

```go
name := giode.Input.Text("Your name").
	Submit(func(text string) { save(text) })

// in the view:
giode.Stack(giode.Styles{}, name)
```

| Member | Description |
| --- | --- |
| `Text() string` | Current content. |
| `SetText(s string)` | Replaces the content. |
| `Submit(fn func(text string))` | Handler for Enter. |
| `Focus()` | Requests keyboard focus. |
| `OnFocus(fn func())` / `OnBlur(fn func())` | Called when the field gains or loses focus. |
| `Multiline(on bool)` | Several lines; Enter inserts a newline. |
| `Lines(min, max int)` | Multiline, growing with the content from `min` to `max` lines, then scrolling; `max` 0 means no limit. |
| `SubmitOn(m SubmitMode)` | `SubmitDefault` (Enter submits a single-line field), `SubmitEnter`, `SubmitEnterUnlessShift` (Enter sends, Shift+Enter inserts a newline; on a touch screen the on-screen Enter key inserts a newline, since phone apps send with a button), `SubmitNever`. |
| `SelectAll()` | Selects the whole content. |
| `InsertText(s string)` | Inserts at the caret (replacing the selection) and calls OnChange, as typing would. |
| `Placeholder(s string)` / `Styles(st Styles)` | Replace the placeholder or styles from the next frame, keeping the text and focus. A placeholder too long for the field is cut off with "…" (at one line, or at `Lines`' maximum). |
| `Password(on bool)` | Masks the content and asks for the password keyboard. |
| `Keyboard(k Keyboard)` | On-screen keyboard on touch devices: `KeyboardText` (default), `KeyboardNumeric`, `KeyboardEmail`, `KeyboardURL`, `KeyboardPhone`. |

```go
server := giode.Input.Text("Server URL").Keyboard(giode.KeyboardURL)
```

A press anywhere on the field - padding, border and blank space included - focuses it,
and the editor fills the field's width (and its height, when `Height` is set), so a tap
in an empty or multiline field lands where expected on a touch screen.

**Editing menu.** On a touch screen, holding a finger on the field selects the word under
it (or places the caret where there's no word) and shows a menu with **Cut**, **Copy**,
**Paste** and **Select all**, offering only what applies: no Cut in a read-only field, no Cut
or Copy in a password field, no Paste in a read-only one. Two handles below the selection
drag its ends. Tapping anywhere else closes the menu. A finger that moves before the
long-press scrolls as usual. With a mouse, a right-click opens the same menu. Keyboard
shortcuts (Ctrl/Cmd+X, C, V, A) work as before.

Styles: `Color` (text), `Background` (fill), `BorderWidth`/`BorderColor`/`BorderRadius`
(frame, 1px border by default), `Padding` (12x8 by default). Expands to the available
width; set `Width` to fix it.

## Checkbox

```go
notify := giode.Input.Checkbox("Enable notifications").
	OnChange(func(checked bool) { ... })
notify.SetChecked(true)
```

| Member | Description |
| --- | --- |
| `Checked() bool` | Current state. |
| `SetChecked(v bool)` | Sets the state without invoking OnChange. |
| `OnChange(fn func(bool))` | Change handler. |
| `Icon(name string, st ...Styles)` | Replaces the check mark with a material icon. |

Styles: `Color` (label), `Background` (checked fill), `BorderColor` (unchecked border).

## Slider

```go
volume := giode.Input.Slider().
	OnChange(func(v float32) { player.SetVolume(v) })
```

| Member | Description |
| --- | --- |
| `Value() float32` | Current value in [0, 1]. |
| `SetValue(v float32)` | Sets the value without invoking OnChange. |
| `OnChange(fn func(float32))` | Called while dragging. |

Styles: `Color` (fill), `Background` (track), `BorderColor` (thumb).

## Select

```go
size := giode.Input.Select(giode.Styles{}, "Small", "Medium", "Large").
	OnChange(func(i int) { ... })
```

| Member | Description |
| --- | --- |
| `Selected() int` | Selected index. |
| `Value() string` | Selected option text. |
| `SetSelected(i int)` | Changes the selection without invoking OnChange. |
| `OnChange(fn func(int))` | Change handler. |
| `Opened() bool` | Whether the panel is open. |

Styles: `Background` (field and panel fill), `Color` (text), `BorderColor` (field
border), `BorderRadius` (corners), `Width` (fixed field width). The panel opens below
the field via a portal (see [extending.md](../extending.md#rendering-above-everything-portals)),
so it paints above everything else regardless of tree position, and lands below the field
wherever it sits (in a row, a scroll, a dialog). Escape and the system Back button
(Android) close it.

On phones, `Sheet(giode.SheetCompact)` opens the options as a bottom sheet instead - a
panel along the bottom of the window over a dimmed scrim, with tall rows that are easy to
tap - inside a `Responsive` breakpoint named `"compact"` or in a window narrower than
600dp. `SheetAlways` always does; `SheetNever` is the default. Tapping the scrim or
pressing Back closes it.

## AutoComplete

```go
city := giode.Input.AutoComplete("City", []string{"Paris", "London", "Tokyo", ...}).
	OnSelect(func(value string) { ... })
```

| Member | Description |
| --- | --- |
| `Text() string` | Current field content. |
| `SetText(s string)` | Replaces the content without reopening the suggestion panel - for a programmatic reset, e.g. loading a saved value; not meant for what the user types. |
| `SetOptions(options []string)` | Replaces the suggestion source. |
| `OnSelect(fn func(value string))` | Handler for picking a suggestion (click). |

Styles: same as `Select` (`Background`, `Color`, `BorderColor`/`BorderRadius`/`Width` on
the field; `Background` also fills the panel). The panel shows below the field, via a
portal, whenever the current text matches at least one option (case-insensitive
substring match) - no keyboard navigation of the list yet, only clicking a suggestion.
