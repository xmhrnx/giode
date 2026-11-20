# Rich text editor

`giode.RichEditor()` edits formatted text: chat composers, documents, notes and ticket
descriptions. It holds paragraphs, headings, bulleted, numbered and task lists, quotes, code
blocks, tables, images and horizontal rules, with bold, italic, underline, strikethrough,
inline code, color, highlight and links, and atoms such as @mentions that edit as one
character.

```go
editor := giode.RichEditor("Write a note…").
	OnChange(func() { saveDraft(editor.Markdown()) })

giode.Box(giode.Styles{FlexDirection: giode.FlexDirectionColumn, Gap: 8},
	editor.Toolbar(),
	editor,
)
```

It grows with its content from `MinHeight` up to the height it's given, then scrolls. Only
the blocks in view are laid out, so long documents stay fast to type in.

## Content in and out

| | |
|---|---|
| `SetMarkdown(md)`, `Markdown()` | GitHub-flavored Markdown; underline, color and highlight as `<u>`, `<span style="color: …">` and `<mark>` |
| `SetDoc(doc)`, `Doc()` | the `richdoc.Doc` itself; it marshals to and from JSON |
| `SetText(s)`, `Text()` | plain text, a paragraph a line |
| `HTML()` | HTML, for e-mail and other apps |
| `InsertMarkdown`, `InsertHTML`, `InsertText` | insert at the caret; HTML is sanitized (no scripts, styles, handlers or unsafe links) |
| `Clear()` | empties it, undoably, as sending a message does |

Markdown and JSON round-trip, so the same content moves between a Giode app and a web client.

## Typing

Markdown typed into a paragraph becomes formatting as you type, and the first undo turns it
back into the characters typed:

| Type | Gets |
|---|---|
| `- `, `* `, `+ ` | a bulleted list |
| `1. ` | a numbered list |
| `[ ] `, `[x] ` | a checklist item |
| `> ` | a quote |
| `# ` to `###### ` | a heading |
| ```` ``` ```` | a code block (```` ```go ```` and Enter sets its language) |
| `---` | a horizontal rule |
| `**bold**`, `*italic*`, `` `code` ``, `~~strike~~` | marks |

`MarkdownShortcuts(false)` turns this off. Enter continues a list, and on an empty item ends
it; Enter twice at the end of a code block leaves it; Shift+Enter breaks a line inside a
block.

| Keys | |
|---|---|
| Ctrl+B, I, U, E | bold, italic, underline, inline code |
| Ctrl+Shift+X | strikethrough |
| Ctrl+K | link, through `OnLinkEdit` |
| Ctrl+Alt+1–6, Ctrl+Alt+0 | heading level, paragraph |
| Ctrl+Shift+7, 8, 9 | numbered, bulleted, checklist |
| Tab, Shift+Tab | nest and unnest list items; a tab in code |
| Ctrl+Z, Ctrl+Shift+Z or Ctrl+Y | undo, redo |
| arrows, Home, End, Page Up/Down | move, with Ctrl (Alt on Apple) by word, with Shift selecting |

On Apple platforms Ctrl is Cmd. Input methods compose inside it (Chinese, Japanese, Korean,
dictation), and screen readers see its text, caret and selection.

## Chat composers

```go
composer := giode.RichEditor("Message #general").
	SubmitOn(giode.RichSubmitEnter). // Enter sends; Shift+Enter starts a new paragraph
	OnSubmit(func() {
		send(composer.Markdown())
		composer.Clear()
	})
```

`RichSubmitShortcut` sends with Ctrl+Enter instead.

## Mentions, references and slash commands

`Suggest` shows a list while text follows a trigger character; picking an entry inserts its
atom (or text, or runs its command) in place of what was typed:

```go
editor.Suggest('@', func(query string) []giode.RichSuggestion {
	var out []giode.RichSuggestion
	for _, u := range users.Search(query) {
		out = append(out, giode.RichSuggestion{
			Label: u.Name, Detail: u.Title,
			Atom: &richdoc.Atom{Kind: "mention", ID: u.ID, Label: "@" + u.Name},
		})
	}
	return out
})
editor.Suggest('#', ticketSuggestions) // Kind: "ref"
editor.Suggest(':', emojiSuggestions)  // Kind: "emoji", Text: "🎉"
```

Atoms show as chips (emoji as text), delete as one character, and are written to Markdown as
`[@Ada](mention:42)`, which reads back as the same atom. Up and Down move through the list;
Enter or Tab picks; Escape closes it.

Typing `/` at the start of a word offers block types, a table and a divider;
`SlashCommands(false)` turns that off.

## Tables, images and checklists

Click a table cell to edit it: Tab and Shift+Tab move between cells (Tab past the last adds a
row), Enter moves down, and Escape leaves the cell with the table selected. Its context menu
inserts and deletes rows and columns. `InsertTable(rows, cols)` adds one.

`InsertImage(src, alt)` adds an image; `LoadImage(func(src string) image.Image)` gives the
editor the pictures, which otherwise show as placeholders. A checklist item's box toggles it.

## Toolbar and menu

`Toolbar()` is a row of buttons for marks, headings, lists, quotes, code, links, undo and redo,
which keep the keyboard focus in the editor. Right-click (or long-press) opens a menu with
cut, copy, paste, marks, "Turn into" and "Insert". To build your own, use the commands:
`ToggleMark`, `HasMark`, `SetKind`, `Kind`, `Indent`, `SetLink`, `SetColor`,
`ClearFormatting`, `InsertAtom`, `Undo`, `Redo`, `CanUndo`, `CanRedo`; `Context(menu)` replaces
the menu.

## Clipboard

Copying writes plain text; pasting it back into an editor keeps its formatting. Pasted text
that looks like Markdown is read as Markdown (`PasteMarkdown(false)` turns that off).

## Collaboration

Every edit is a `richdoc.Tx` of operations: inserting, deleting or formatting a range of a
block, splitting and joining blocks, and inserting, removing or changing blocks. `OnOps`
receives each one (they marshal to JSON) and `Apply` applies a collaborator's, keeping the
local caret in place, so a CRDT or OT layer can sync documents:

```go
editor.OnOps(func(tx *richdoc.Tx) { session.Send(tx.Ops) })
session.OnReceive(func(ops []richdoc.Op) { app.Post(func() { editor.Apply(ops) }) })
```

`richdoc.Op.Map` moves a position through an operation, and `Invert` undoes it.

## The document model

Package `richdoc` is usable without the editor, on a server too: `ParseMarkdown`,
`ParseHTML`, `Doc.Markdown`, `Doc.HTML`, `Doc.Text`, JSON, and `State`, which runs every
editing command headlessly.
