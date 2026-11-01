# TreeView

A tree of expandable rows: file browsers, outlines, settings categories. The tree owns
expansion and selection (by node key) and draws the indentation, expand arrows, and
selection and hover backgrounds; your row function draws only the content. It is built on
`VirtualList`, so only visible rows are built and trees of any size stay fast.

```go
type Folder struct {
	Path     string
	children []giode.TreeNode
	loaded   bool
}

func (f *Folder) Key() any                   { return f.Path }
func (f *Folder) Children() []giode.TreeNode { return f.children }
func (f *Folder) HasChildren() bool          { return !f.loaded || len(f.children) > 0 }
func (f *Folder) Label() string              { return filepath.Base(f.Path) } // optional: names rows for automation

tree := giode.TreeView(
	func() []giode.TreeNode { return roots },
	func(n giode.TreeNode, s giode.TreeRowState) giode.Element {
		return giode.Text(filepath.Base(n.Key().(string)), giode.Styles{Color: "theme.text"})
	},
	giode.Styles{Height: 26},
).
	OnExpand(func(n giode.TreeNode) { go load(n.(*Folder)) }). // lazy loading
	OnSelect(func(n giode.TreeNode) { preview(n) }).
	OnActivate(func(n giode.TreeNode) { open(n) })              // double click or Enter
```

`HasChildren()` lets a node show its arrow before its children are loaded. An expanded node
with no children yet is drawn in its loading state (a spinner in place of the arrow) until
`Children()` returns some; load them in `OnExpand`, and set them with `app.Post` when the
loading happens on another goroutine.

## TreeRowState

| Field | Meaning |
| --- | --- |
| `Depth` | Nesting level, 0 for roots. |
| `Expanded` | The node is open. |
| `Selected` | The node is the selection. |
| `Hovered` | The pointer is over the row. |
| `Loading` | Expanded, but children not loaded yet. |
| `Focused` | The tree has keyboard focus. |

## Methods

| Method | Effect |
| --- | --- |
| `Expand(key, open)`, `Toggle(key)`, `Expanded(key)` | Open and close nodes. |
| `Select(key)`, `Selected()` | The selection. |
| `Reveal(key)` | Opens every ancestor, selects the node and scrolls it into view. |
| `Focus()` | Gives the tree keyboard focus. |
| `OnSelect`, `OnExpand`, `OnCollapse`, `OnActivate` | Callbacks. |

## Interaction

- Click selects; double click activates; click on the arrow toggles without selecting.
- Up/Down move the selection, Right expands a node (or moves into it when open), Left
  collapses it (or moves to the parent), Enter activates, PageUp/PageDown and Home/End jump.
- Styles: `Height` (row height, dp), `Background`, `Color` (arrows), `BorderColor`
  (selection color, default `theme.primary`), `ScrollBar`.
