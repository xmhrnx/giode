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
| `SetText(s string)` | Replaces the content, leaving the caret at the end (as if typed), without calling OnChange. |
| `Caret() int` / `SetCaret(pos int)` | The caret's position in runes from the start; setting it clears the selection. |
| `Selection() (start, end int)` / `SetSelection(start, end int)` | The selected runes, `[start, end)`; setting puts the caret at `end`. Select a word and `InsertText` to replace it - a completion at the caret, mid-line. |
| `Submit(fn func(text string))` | Handler for Enter. |
| `Focus()` | Gives the field keyboard focus in the next frame, which it requests. Works from anywhere on the UI goroutine, including right after the field first appears. |
| `Blur()` | Takes keyboard focus away from the field (if it has it) and closes the on-screen keyboard. |
| `OnKeys(fn func(key.Event), names...)` | `fn` takes presses of the named keys while the field has focus, before the editor (with any modifiers). |
| `OnFocus(fn func())` / `OnBlur(fn func())` | Called when the field gains or loses focus. |
| `Multiline(on bool)` | Several lines; Enter inserts a newline. |
| `Lines(min, max int)` | Multiline, growing with the content from `min` to `max` lines, then scrolling; `max` 0 means no limit. |
| `SubmitOn(m SubmitMode)` | `SubmitDefault` (Enter submits a single-line field), `SubmitEnter`, `SubmitEnterUnlessShift` (Enter sends, Shift+Enter inserts a newline; on a touch screen the on-screen Enter key inserts a newline, since phone apps send with a button), `SubmitNever`. |
| `SelectAll()` | Selects the whole content. |
| `InsertText(s string)` | Inserts at the caret (replacing the selection) and calls OnChange, as typing would. |
| `OnPaste(fn func(pasteText func()))` | Runs `fn` in place of pasting when the user pastes (Ctrl/Cmd+V, or Paste in the editing menu); calling `pasteText` pastes the clipboard's text as the field would have. See below. |
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

**Pasting images.** `OnPaste` takes over pasting, so a chat composer can attach an image
the user copied (a screenshot, an image from a browser) and paste text otherwise. `fn` runs
on the UI goroutine; `pasteText` can be called then, or later through `App.Post`:

```go
composer.OnPaste(func(pasteText func()) {
	go func() {
		img, err := app.ReadClipboardImage(ctx)
		app.Post(func() {
			if err == nil && img != nil {
				attach(img)
			} else {
				pasteText()
			}
		})
	}()
})
```

A read-only field doesn't paste and doesn't call `fn`.

**Editing.** Ctrl+Z (Command+Z) undoes and Ctrl+Shift+Z or Ctrl+Y redoes, a word at a time
for typing and deleting; `Undo`, `Redo`, `CanUndo` and `CanRedo` do the same from code, and
the touch edit menu offers them. `SetText` starts the history over. Ctrl+Left and Ctrl+Right
(Option on Apple systems) move by word, with Shift to select, and Ctrl+Backspace and
Ctrl+Delete delete a word; words are letters of any script, so Persian, Arabic and CJK text
step as English does. Input methods work as the platform's own fields: Chinese, Japanese and
Korean composition, Persian and Arabic keyboards, and Android and iOS predictive text and
autocorrect replacing the word being typed. Mixed right-to-left and left-to-right text
moves the caret through every position and selects by logical order.

**Spell checking.** `SpellCheck(true)` underlines misspelled words in red once typing pauses,
and the edit menu (right-click, or a long press) offers corrections and Add to dictionary.
It uses the platform's checker: NSSpellChecker on macOS, UITextChecker on iOS, the Windows
spell checker, Android's spell checker service and Hunspell on Linux, with the dictionaries
installed (`hunspell-en-us` and the like). `SpellLanguage("fa")` picks the language, the
user's by default; `Misspellings()` lists what's found. Package `spellcheck` offers the same
checks for your own text (`Check`, `Suggest`, `Learn`, `Ignore`). On phones the keyboard's own
corrections and predictions work too: text fields get them, numbers, addresses, passwords and
`Autocorrect(false)` fields don't.

**Autofill.** `Autofill(giode.AutofillUsername)`, `AutofillPassword`, `AutofillNewPassword`,
`AutofillEmail`, `AutofillOneTimeCode`, `AutofillName`, `AutofillPhone`, `AutofillAddress` or
`AutofillPostalCode` tells password managers and the keyboard what the field holds: iOS offers
saved passwords, strong new ones and codes from messages above the keyboard, and Android's
autofill service (Google Password Manager, 1Password, Bitwarden) fills the focused field. A
`Password` field is `AutofillPassword` unless set otherwise. Builds made with the Giode CLI
have it.

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
| `Outlined(on bool)` | Draws a checked box as an outline with a tick in the fill colour, instead of a solid fill. |

Styles: `Color` (label), `Background` (checked fill; outline and tick when outlined), `BorderColor` (unchecked border).

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
| `SetOptions(options ...string)` | Replaces the options; the selection stays on the same text when it's still there, and otherwise moves to the first option (no OnChange). |
| `OnChange(fn func(int))` | Change handler. |
| `Opened() bool` | Whether the panel is open. |

Styles: `Background` (field and panel fill), `Color` (text), `BorderColor` (field
border), `BorderRadius` (corners), `Width` (fixed field width). The panel opens below
the field via a portal (see [extending.md](../extending.md#rendering-above-everything-portals)),
so it paints above everything else regardless of tree position, and lands below the field
wherever it sits (in a row, a scroll, a dialog), or above it when there's more room there.
A list longer than that room scrolls, opening with the selected option in view. Escape and
the system Back button (Android) close it.

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
