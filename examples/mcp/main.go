// Command mcp is a small todo app exposed to AI agents through Giode's
// built-in MCP server: one option turns it on, and one call adds a
// custom tool next to the built-in ones (ui_snapshot, ui_click,
// ui_type, ui_set_value, screenshot, state_list, state_set, ...).
//
// Run it, then point an MCP client at http://127.0.0.1:7600/mcp.
package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/mcp"
)

// todos is named so the MCP state tools can read and set it.
var todos = giode.NamedState("todos", []string{"Try the MCP server"})

func main() {
	giode.UseGofont()

	app := giode.New(giode.Config{
		Title:      "Todos",
		Size:       image.Pt(420, 360),
		Resizable:  true,
		Background: "#0f172a",
		Logger:     log.Default(),
		LogLevel:   giode.LogLevelInfo,
		MCP:        true,
	})

	app.MCPTool(mcp.Func("add_todo", "Adds an item to the todo list.",
		func(ctx context.Context, in struct {
			Title string `json:"title" description:"The todo's title"`
		}) (any, error) {
			title := strings.TrimSpace(in.Title)
			if title == "" {
				return nil, fmt.Errorf("title is empty")
			}
			todos.Update(func(t []string) []string { return append(append([]string(nil), t...), title) })
			return "added " + title, nil
		}))

	field := giode.Input.Text("What needs doing?", giode.Styles{Color: "#e2e8f0", BorderColor: "#334155"}).Label("New todo")
	add := giode.Button("Add", giode.Styles{Background: "theme.primary", Color: "white", BorderRadius: 6})
	submit := func() {
		if t := strings.TrimSpace(field.Text()); t != "" {
			todos.Update(func(l []string) []string { return append(append([]string(nil), l...), t) })
			field.SetText("")
		}
	}
	field.Submit(func(string) { submit() })
	add.OnClick(submit)
	clear := giode.Button("Clear all", giode.Styles{Background: "#94a3b8", Color: "#e2e8f0", BorderRadius: 6}).Variant(giode.VariantOutlined).OnClick(func() { todos.Set(nil) })

	if err := app.Run(func() giode.Element {
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(16), Gap: 12},
			giode.Text("Todos", giode.Styles{Color: "white", FontSize: 22, FontWeight: giode.FontWeightBold}),
			giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap},
				giode.Box(giode.Styles{FlexGrow: 1}, field),
				add,
			),
			// Rebuilt only when todos changes: the Memo tracks the
			// State read inside it.
			giode.Memo("list", func() giode.Element {
				items := todos.Get()
				rows := make([]giode.Element, 0, len(items))
				for _, t := range items {
					rows = append(rows, giode.Text("• "+t, giode.Styles{Color: "#e2e8f0"}))
				}
				if len(rows) == 0 {
					rows = append(rows, giode.Text("Nothing to do", giode.Styles{Color: "#64748b"}))
				}
				return giode.Stack(giode.Styles{Gap: 6}, rows...)
			}),
			clear,
		)
	}); err != nil {
		log.Fatal(err)
	}
}
