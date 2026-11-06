// Package markdown renders Markdown (CommonMark with the GitHub
// extensions: tables, task lists, strikethrough, autolinks) as giode
// elements: paragraphs and headings as RichText, code blocks with a
// copy button, lists, task lists, block quotes, tables and rules.

package markdown

import (
	"fmt"
	"strings"

	"github.com/xmhrnx/giode/components/copybutton"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
	gast "github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// codeBlock renders a code block: a tinted box with the language and a
// copy button above the code.
func (r *renderer) codeBlock(n gast.Node, lang string) elements.Element {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(r.src))
	}
	code := strings.TrimRight(b.String(), "\n")
	fontSize := properties.FontSize(float32(properties.ResolveFontSize(r.base.FontSize)) * 0.92)
	copyBtn := elements.Keyed("giode.markdown.copy", code, func() *copybutton.CopyButton {
		return copybutton.New(func() string { return code }, styles.Styles{FontSize: 12, Color: "theme.muted"})
	})
	header := elements.Row(styles.Styles{Width: -1, Align: properties.AlignCenter, Wrap: properties.WrapNoWrap},
		// The language label isn't part of a selection of the code.
		elements.Box(styles.Styles{FlexGrow: 1}, elements.Raw(elements.RawFunc(kit.Unselectable(elements.Text(lang, styles.Styles{Color: "theme.muted", FontSize: 12}).Layout)))),
		copyBtn,
	)
	body := elements.Text(code, styles.Styles{FontFamily: r.o.CodeFont, FontSize: fontSize, Color: r.base.Color, TabWidth: 4})
	return elements.Stack(styles.Styles{
		Width: -1, Background: r.o.CodeBackground, BorderRadius: 8,
		Padding: properties.Inset{Top: 6, Right: 10, Bottom: 10, Left: 12}, Gap: 4,
	}, header, sideways("code\x00"+code, body))
}

// list renders a bulleted, numbered or task list.
func (r *renderer) list(n *gast.List) elements.Element {
	var items []elements.Element
	num := n.Start
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		marker := r.marker(n, c, num)
		num++
		items = append(items, elements.Row(styles.Styles{Gap: 8, Width: -1, Wrap: properties.WrapNoWrap},
			marker,
			elements.Box(styles.Styles{FlexGrow: 1}, r.blocks(c)),
		))
	}
	gap := 4
	if !n.IsTight {
		gap = r.o.Gap
	}
	return elements.Stack(styles.Styles{Gap: properties.Gap(gap), Width: -1}, items...)
}

// marker returns a list item's bullet, number or task box.
func (r *renderer) marker(list *gast.List, item gast.Node, num int) elements.Element {
	if box := taskBox(item); box != nil {
		st := styles.Styles{Width: 16, Height: 16, BorderWidth: 1, BorderColor: "theme.muted", BorderRadius: 4, Justify: properties.JustifyCenter, Align: properties.AlignCenter, Margin: properties.Inset{Top: 2}}
		if box.IsChecked {
			st.Background = "theme.primary"
			st.BorderColor = "theme.primary"
			return elements.Box(st, elements.Glyph(kit.GlyphCheck, styles.Styles{Color: "theme.onprimary", Width: 12}))
		}
		return elements.Box(st)
	}
	label := "•"
	if list.IsOrdered() {
		label = fmt.Sprintf("%d.", num)
	}
	return elements.Box(styles.Styles{MinWidth: 16}, elements.Text(label, styles.Styles{Color: "theme.muted", FontSize: r.base.FontSize, LineHeight: r.base.LineHeight}))
}

// taskBox returns the task check box at the start of a list item.
func taskBox(item gast.Node) *east.TaskCheckBox {
	first := item.FirstChild()
	if first == nil {
		return nil
	}
	box, _ := first.FirstChild().(*east.TaskCheckBox)
	return box
}

// table renders a GFM table.
func (r *renderer) table(t *east.Table) elements.Element {
	var rows []elements.Element
	for c := t.FirstChild(); c != nil; c = c.NextSibling() {
		var cells []elements.Element
		for cell := c.FirstChild(); cell != nil; cell = cell.NextSibling() {
			st := r.base
			if tc, ok := cell.(*east.TableCell); ok {
				switch tc.Alignment {
				case east.AlignCenter:
					st.TextAlign = properties.TextAlignCenter
				case east.AlignRight:
					st.TextAlign = properties.TextAlignEnd
				}
			}
			var spans []elements.Span
			r.inlines(cell, styles.Styles{}, &spans)
			rt := elements.RichText(st, spans...)
			if r.o.OnLink != nil {
				rt.OnLink(r.o.OnLink)
			}
			cells = append(cells, elements.TableCell(styles.Styles{}, rt))
		}
		if _, ok := c.(*east.TableHeader); ok {
			rows = append(rows, elements.TableHeader(styles.Styles{Background: r.o.CodeBackground}, cells...))
		} else {
			rows = append(rows, elements.TableRow(styles.Styles{}, cells...))
		}
	}
	return sideways("table\x00"+string(t.Text(r.src)), elements.Table(styles.Styles{BorderWidth: 1, BorderColor: "theme.border", BorderRadius: 6}, rows...))
}
