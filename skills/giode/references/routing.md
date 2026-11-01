# Routing

`giode/router` is a simple navigation layer. Views are functions that build elements; the
router renders the active one.

```go
router := giode.NewRouter(
	giode.Route{Path: "/", View: func() giode.Element { return home() }},
	giode.Route{Path: "/settings", View: func() giode.Element { return settings() }},
)

// in the view (the router is an element):
giode.Box(giode.Styles{}, router)
```

## API

| Member | Description |
| --- | --- |
| `NewRouter(routes ...Route) *Router` | Creates the router; the first route is shown initially. |
| `Navigate(path string)` | Switches to a path, replacing the current history entry. |
| `Push(path string)` | Navigates, keeping the current path on the history stack. |
| `Back() bool` | Returns to the previous path. While there is history, the system Back button (Android) does the same. |
| `Current() string` | Active path. |
| `NotFound(v View) *Router` | View for unknown paths. |
| `Link(path, text string, st ...Styles) *Router` | Element that navigates when clicked. |

```go
// navigate from any callback
router.Push("/settings")

// or render a link
giode.Box(giode.Styles{}, router.Link("/settings", "Settings"))
```

Views are built only when their route is active. Stateful components used by a view must
live outside the view closure.
