# AppShell

Full-window application layout: an optional fixed header, navbar, aside and footer
around a main content area that's automatically offset so it never sits under them.

```go
shell := giode.AppShell(giode.AppShellConfig{
	Header: giode.Header(60, giode.Text("My App", giode.Styles{Color: "white"})),
	Navbar: giode.Navbar(240, navLinks),
	Main:   giode.Main(pageContent),
})

app.Run(func() giode.Element { return shell })
```

Build it once (it's stateless itself, but its slots often aren't - see Section below) and
return it from the view, typically as the whole app's root element.

## Slots

| Constructor | Edge | Sized by |
| --- | --- | --- |
| `Header(height int, content Element, st ...Styles) *AppShellSlot` | Top, full width | height |
| `Navbar(width int, content Element, st ...Styles) *AppShellSlot` | Left (right in a right-to-left locale), between Header and Footer | width |
| `Aside(width int, content Element, st ...Styles) *AppShellSlot` | Right (left in a right-to-left locale), between Header and Footer | width |
| `Footer(height int, content Element, st ...Styles) *AppShellSlot` | Bottom, full width | height |
| `Main(content Element, st ...Styles) Element` | Fills whatever the other four leave | - |

Assign the slots to `AppShellConfig`'s fields (`Header`, `Navbar`, `Aside`, `Footer`,
`Main`) and pass it to `AppShell`. Header, Navbar, Aside and Footer are optional - leave
a field nil (or omit it) to skip that edge; `Main` fills the rest of the window either
way.

## Section

`Section(content func() Element, st ...Styles) Element` groups content inside a
Navbar/Aside - a scrollable area for a long list of nav links, for instance. It's a thin
wrapper over [Scroll](scroll.md), so build it **once** and reuse it (like any stateful
component) rather than rebuilding it inside the view function, or its scroll offset
resets every frame:

```go
navbar := giode.Section(func() giode.Element {
	return giode.Stack(giode.Styles{}, link1, link2, link3)
})

shell := giode.AppShell(giode.AppShellConfig{
	Navbar: giode.Navbar(240, navbar),
	Main:   giode.Main(content),
})
```

## Styles

Each slot constructor's `st` styles that region's own box (`Background` defaults to a
dark slate, `Padding`, `BorderRadius`, etc.); `AppShell`'s own `st` styles the root
container (defaults to filling the window, `Width`/`Height: -1`).
