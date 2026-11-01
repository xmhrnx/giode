# Swipeable

A list row that slides sideways to reveal actions, like the delete and pin actions of
mobile mail and chat apps.

```go
row := giode.Keyed("chats", chat.ID, func() *swipeable.Swipeable {
	return giode.Swipeable(func() giode.Element { return chatRow(chat) },
		[]giode.SwipeAction{{Label: "Pin", Background: "theme.info", OnTap: func() { pin(chat) }}},
		[]giode.SwipeAction{{Label: "Delete", Background: "theme.danger", OnTap: func() { remove(chat) }}},
	)
})
```

Swiping right reveals the leading actions, swiping left the trailing ones; each is 80dp
wide. A swipe past half the actions' width opens the row, a shorter one springs back.
Tapping an action runs it and closes the row; tapping the open row closes it without
reaching the content. Vertical drags still scroll the list around it.

The content should paint an opaque background, since the actions sit behind it.

When an action depends on state, update it on the existing row rather than creating a new
one:

```go
label := "Pin"
if chat.Pinned {
	label = "Unpin"
}
row.SetActions([]giode.SwipeAction{{Label: label, OnTap: func() { togglePin(chat) }}}, deleteAction)
```

| Member | Description |
| --- | --- |
| `Swipeable(content func() Element, leading, trailing []SwipeAction) *Swipeable` | Creates the row; either action list may be nil. |
| `SetActions(leading, trailing []SwipeAction)` | Replaces the actions, for labels that follow state (Pin becomes Unpin). An open row stays open, resized, while its side still has actions. |
| `Close()` | Slides the row back. |
| `Opened() bool` | Whether the row is open or opening. |

`SwipeAction` has `Label`, an optional `Icon` shown above it, `Background` (default
`theme.primary`), `Color` (default `theme.onprimary`) and `OnTap`.
