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

## Styles

| Property | Role |
| --- | --- |
| `Color` | Active underline. |
| `BorderColor` | Active label color. |
