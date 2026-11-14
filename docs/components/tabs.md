# Tabs

Tab bar with per-tab views. Only the selected view is built each frame.

```go
tabs := giode.Tabs(giode.Styles{},
	giode.Tab{Label: "Home", View: func() giode.Element { return homeView }},
	giode.Tab{Label: "Settings", View: func() giode.Element { return settingsView }},
).OnChange(func(i int) { ... })

// in the view:
giode.Stack(giode.Styles{}, tabs)
```

## API

| Member | Description |
| --- | --- |
| `Tabs(st Styles, tabs ...Tab) *Tabs` | Creates the bar. |
| `Selected() int` | Active tab index. |
| `SetSelected(i int)` | Switches tabs without invoking OnChange. |
| `OnChange(fn func(index int)) *Tabs` | Change handler. |
| `Add(tab Tab) *Tabs` | Appends a tab. |
| `Remove(i int)` | Removes a tab; the selection stays on its tab, or moves to the neighbour. |
| `SetLabel(i int, label string)` | Renames a tab. |
| `SetTabs(tabs ...Tab)` | Replaces the tabs. |
| `Len() int` | Number of tabs. |

For open documents, with close buttons, unsaved dots and tabs that shrink and scroll, use
[DocTabs](doctabs.md).

## Styles

| Property | Role |
| --- | --- |
| `Color` | Active underline. |
| `BorderColor` | Active label color. |
