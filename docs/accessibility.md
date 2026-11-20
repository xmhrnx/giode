# Accessibility

Giode describes the UI as a tree of nodes - what each element is, what it says, what state
it's in - and hands it to the platform's screen reader. The same tree serves automation
(the [MCP server](mcp.md) and tests).

| Platform | Screen reader | Status |
| --- | --- | --- |
| Android | TalkBack | Built with the giode CLI (its Gio patches, revision 20 on): every node, its actions and announcements reach TalkBack. Not yet checked on a device. |
| Windows | Narrator, NVDA, JAWS (UI Automation) | Every node, its patterns (invoke, toggle, value, range, expand and collapse, selection), focus changes and announcements. Asked for a point, it answers for the mouse cursor's. Not yet checked with a screen reader. |
| macOS | VoiceOver (NSAccessibility) | Every node as an accessibility element with its role, label, value, state and frame; press, increment and decrement, setting text, focus; focus and value changes and announcements. The first request from VoiceOver or another assistive app turns the tree on. Not yet checked with VoiceOver. |
| iOS | VoiceOver, Switch Control (UIAccessibility) | The nodes a screen reader stops on, in reading order, with labels, values, hints and traits; activate, adjust (sliders); focus moves and announcements. Recorded while VoiceOver or Switch Control runs. Not yet checked on a device. |
| Linux | Orca (AT-SPI) | The application, its window and every node on the accessibility bus, with roles, states, names, descriptions, relations, text (entries and labels), values (sliders, progress), actions and extents (screen coordinates on X11, window coordinates on Wayland); focus, state, value and text changes and announcements as events. Checked against the AT-SPI registry with pyatspi; not yet with Orca. Set `NO_AT_BRIDGE=1` to turn it off. |

## What components report

Built-in components fill the tree in themselves:

- **Roles**: buttons, links, checkboxes, switches (`Switch`), radio buttons (`Segmented`'s
  segments), text fields, sliders, progress bars and spinners, selects, tabs, menu items,
  tree items (`TreeView`), headings (`title.H1`-`H6`, Markdown headings), lists (Markdown
  lists), dialogs (`Dialog`, `Sheet`, `Drawer`), status messages and alerts (toasts),
  images, scroll areas.
- **States**: checked, unchecked or mixed; selected (tabs, segments, tree items);
  expanded or collapsed (selects, submenus, accordion sections, tree items); disabled;
  focused; busy (spinners); a slider's or progress bar's value and range; a list item's
  place in its list.
- **Announcements**: a toast is read when it appears, after what's being read; an error
  toast at once.

Some need a name from you:

```go
giode.Dialog(&open, content).Label("Delete project?")   // read when it opens
giode.Drawer(&open, giode.SideStart, 280, nav).Label("Navigation")
giode.Progress(0.4).Label("Uploading report.pdf")         // "Uploading report.pdf, 40%"
giode.Input.Text("you@example.com").Label("Email").Required(true)
input.Invalid("Enter an email address")                   // red border, read with the field
giode.IconButton(icon).Label("Delete")                     // an icon has no text to read
```

## Keyboard

Every control takes keyboard focus with Tab and Shift+Tab, in the order it's laid out, and
shows a focus ring while focused from the keyboard. A click or a tap focuses without a ring,
as browsers do (`:focus-visible`).

`Dialog`, `Sheet` and `Drawer` trap the focus while open: opening moves it to their first
control, Tab and Shift+Tab wrap around inside, and closing gives it back to what had it
before. A dialog opened from another dialog traps it in turn.

### Skip links and landmarks

A long navigation before the content costs a keyboard user a Tab for every link on every
screen. A skip link is the first Tab stop: hidden until focused, then shown in the corner;
Enter moves the focus to the first control of a landmark.

```go
main := giode.NewLandmark("Main content")
nav := giode.NewLandmark("Navigation")
skip := giode.SkipLink("Skip to main content", main)

func view() giode.Element {
	return giode.Row(giode.Styles{},
		skip, // first: the first Tab stop, taking no space
		nav.Wrap(sidebar()),
		main.Wrap(content()),
	)
}
```

Landmarks are regions to screen readers too, which list them by name to jump between.

## Custom elements

`Pressable` reports a button with its `Label`. `Semantics` makes it something else:

```go
row := giode.Pressable(rowView).Label(contact.Name).
	Semantics(func(n *giode.SemanticNode) {
		n.Kind = giode.SemanticListItem
		n.Selected = contact.ID == selectedID
		n.Pos, n.Count = i+1, len(contacts) // "3 of 120"
	})
```

`Accessible(element, func(n *SemanticNode))` changes the nodes any element reports - a
text as a heading, a line as a status message that's announced when it changes:

```go
giode.Accessible(giode.Text("Settings", big), func(n *giode.SemanticNode) {
	n.Kind, n.Level = giode.SemanticHeading, 1
})
giode.Accessible(giode.Text(syncState), func(n *giode.SemanticNode) {
	n.Kind, n.Live = giode.SemanticStatus, giode.LivePolite
})
```

`AccessibleGroup(node, element)` makes an element a container - a region, a list, a table,
a toolbar - whose nodes become its children, so a screen reader can tell what they belong
to and jump between regions:

```go
giode.AccessibleGroup(giode.SemanticNode{Kind: giode.SemanticRegion, Label: "Inbox"}, inbox)
```

A custom component records its own node with `RecordSemantic` (see
[extending.md](extending.md)). Set `Target` to the component: when it has a `Click()`
method, a screen reader's activate calls it; `ScrollBy(dx, dy)` lets it scroll;
`SetText(string)` lets it be typed into by voice; `Focus()` lets it take focus. A node
without a `Click` is activated with a tap at its middle.

## SemanticNode fields

| Field | Meaning |
| --- | --- |
| `Kind` | The role: `SemanticButton`, `SemanticHeading`, `SemanticListItem`, `SemanticDialog`, `SemanticAlert`... |
| `Label`, `Value`, `Description` | The name; the current value (a field's text); longer help or an error. |
| `Level` | A heading's level, a tree item's depth. |
| `Checked` | `Checked`, `Unchecked`, `Mixed`, or `NotCheckable`. |
| `Selected`, `Expanded` | Selection; `Expanded`, `Collapsed` or `NotExpandable`. |
| `Required`, `Invalid`, `Busy`, `Disabled`, `Focused` | States. |
| `Live` | `LivePolite` or `LiveAssertive`: changes are announced. |
| `Min`, `Max`, `Now` | A slider's or progress bar's range and value. |
| `Pos`, `Count` | An item's place in its list or set, from 1. |
| `Parent`, `LabelledBy`, `DescribedBy`, `Controls` | Relations to other nodes, by ID. |

## Motion

The system's reduce-motion setting is followed; see [styles.md](styles.md#animation).
