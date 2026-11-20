# Command palette

`CommandPalette` is a search box over the app's commands, opened with Mod+K (Command+K on
Apple systems, Ctrl+K elsewhere). Matching is fuzzy, recent commands come first, and each
command's shortcut shows beside it.

```go
pal := giode.CommandPalette(
	giode.PaletteCommand{ID: "new", Title: "New document", Shortcut: "Mod+N", Run: newDoc},
	giode.PaletteCommand{ID: "close", Title: "Close window", Keywords: []string{"quit"}, Run: closeWin},
	giode.PaletteCommand{ID: "dark", Title: "Toggle dark mode", Group: "View", Run: toggleDark},
)

app.Run(func() giode.Element {
	giode.Shortcuts(pal.Shortcuts()...) // Mod+K, and every command's own shortcut
	return giode.Stack(st, content, pal)
})
```

The palette takes no space while closed; open, it dims the window and shows near the top.
Enter runs the highlighted command, Escape or a click outside closes it. `Shortcuts` returns
`Mod+K` and each command's `Shortcut`, which run the command and count it as recent; `Key`
changes `Mod+K`. `Open`, `Close`, `Toggle` and `Run(id)` drive it from code; save
`Recents()` and restore it with `SetRecents`. `Add` and `SetCommands` change the commands, for
context-dependent ones.
