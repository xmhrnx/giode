# Combobox, MultiSelect, mentions and emoji

Searchable inputs for picking from many options: one (`Combobox`), several as chips
(`MultiSelect`, a tag input with `Creatable`), people and channels at the caret of a message
(`Mentions`), and emoji (`EmojiPicker`). They share one option list: fuzzy matching ("nwtb"
finds "New tab", matched letters in bold), group headings, Up and Down to move, Enter to pick,
Escape to close, and loading, error and empty states.

## Combobox

```go
city := giode.Combobox("City").
	Options(
		giode.ComboOption{Value: "ber", Label: "Berlin", Group: "Germany"},
		giode.ComboOption{Value: "mun", Label: "Munich", Group: "Germany"},
		giode.ComboOption{Value: "par", Label: "Paris", Group: "France", Detail: "FR"},
	).
	OnSelect(func(o giode.ComboOption) { save(o.Value) })
```

Options from a server: `Source` is called as the user types, after a pause (`Debounce`,
250 ms by default) and from `MinChars` characters; a newer query cancels the older one's
context, and its late answer is dropped.

```go
user := giode.Combobox("Assign to").MinChars(1).
	Source(func(ctx context.Context, q string) ([]giode.ComboOption, error) {
		return api.SearchUsers(ctx, q) // shows "Loading…", then the results or the error
	}).
	Messages("Searching…", "Nobody by that name")
```

`Value`, `SetValue` and `Clear` read and set the pick; `Input()` is the underlying field.
An `Option` has a `Value`, a `Label` (the value when empty), a dimmed `Detail`, a `Group`, an
`Icon`, search `Keywords` and `Disabled`.

## MultiSelect and tag input

```go
langs := giode.MultiSelect("Languages").Strings("Go", "Rust", "Zig", "Swift").
	OnChange(func(v []giode.ComboOption) { ... })

tags := giode.MultiSelect("Add tags").Creatable(true).Max(10)
```

Picked options show as chips before the field, each with a remove button, and leave the list.
Backspace in the empty field removes the last chip. With `Creatable`, Enter, a comma or a
semicolon turns what's typed into a tag, and pasting `go, rust, zig` adds three. `Values`,
`ValueStrings` and `SetValues` read and set them; `Source` works as for `Combobox`.

## Mentions

```go
msg := giode.Input.Text("Message")
composer := giode.Mentions(msg).
	Trigger('@', people...).          // static options
	TriggerSource('#', searchChannels). // or asynchronous
	OnPick(func(trigger rune, o giode.ComboOption) { mentioned = append(mentioned, o.Value) })
// Lay out composer in place of msg.
```

Typing a trigger at the start or after a space opens the list at the caret (`Input.CaretRect`);
Up, Down and Enter or Tab pick, inserting `@Label ` by default (`Format` changes it). For rich
text, the rich editor has its own suggestions (see [Rich text editor](richeditor.md)).

## Emoji picker

```go
picker := giode.EmojiPicker().OnPick(func(e string) { msg.InsertText(e) })
```

A search field, a tab per category, recently used emoji first and a skin tone row. Its set is
Unicode's (about 1,900 emoji, 300 with skin tones); `emojipicker.Search` and `All` give the
data for your own UI. Save `Recents()` and `Tone()` and restore them with `SetRecents` and
`SkinTone`. Show it in a popover or a sheet beside the message field.

Emoji draw in color from the platform's emoji font (Noto Color Emoji, Apple Color Emoji,
Segoe UI Emoji). A line with emoji stays a cached image: only the emoji themselves are drawn
separately.
