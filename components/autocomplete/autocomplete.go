// Package autocomplete provides a stateful text field with a
// filtered, portal-rendered list of suggestions.
package autocomplete

import (
	"image"
	"strings"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

var (
	defaultBackground = "#1e293b"
	defaultText       = "#e2e8f0"
	defaultHover      = "#334155"
)

// AutoComplete is a stateful text field with a filtered suggestion
// list, shown in a portal-rendered panel below it (see
// elements.RegisterPortal) while the current text matches at least
// one option. Create it once with New and lay it out every frame.
type AutoComplete struct {
	field    *input.Input
	options  []string
	onSelect func(value string)
	st       styles.Styles

	lastText   string
	suppressed bool
	filtered   []string
	itemClicks []*widget.Clickable
}

// New returns an AutoComplete offering options, filtered as the user
// types (case-insensitive substring match). The styles argument is
// optional and styles both the field and the suggestion panel.
func New(placeholder string, options []string, st ...styles.Styles) *AutoComplete {
	a := &AutoComplete{options: options}
	if len(st) > 0 {
		a.st = st[0]
	}
	a.field = input.New(placeholder, a.st)
	return a
}

// Text returns the field's current content.
func (a *AutoComplete) Text() string {
	return a.field.Text()
}

// SetText replaces the field's content without reopening the
// suggestion panel.
func (a *AutoComplete) SetText(s string) {
	a.field.SetText(s)
	a.lastText = s
	a.suppressed = true
}

// SetOptions replaces the suggestion source.
func (a *AutoComplete) SetOptions(options []string) {
	a.options = options
}

// OnSelect sets the handler invoked when a suggestion is picked
// (clicked). It returns a for chaining.
func (a *AutoComplete) OnSelect(fn func(value string)) *AutoComplete {
	a.onSelect = fn
	return a
}

// Layout lays out the field and, while it has matching suggestions,
// queues the suggestion panel as a portal anchored just below it.
func (a *AutoComplete) Layout(gtx layout.Context) layout.Dimensions {
	// Handles its input without an input pass; see kit.OpaqueInput.
	kit.OpaqueInput()
	dims := a.field.Layout(gtx)
	// Keep the field clickable even inside a frameless window's
	// whole-window drag area - see button.Layout's identical comment.
	kit.BlockDrag(gtx, dims.Size)

	// A change the user typed (as opposed to one SetText/a selection
	// just made) re-enables the panel if it was suppressed.
	if text := a.field.Text(); text != a.lastText {
		a.lastText = text
		a.suppressed = false
	}

	a.filtered = a.filtered[:0]
	if a.lastText != "" && !a.suppressed {
		needle := strings.ToLower(a.lastText)
		for _, opt := range a.options {
			if strings.Contains(strings.ToLower(opt), needle) {
				a.filtered = append(a.filtered, opt)
			}
		}
	}
	for len(a.itemClicks) < len(a.filtered) {
		a.itemClicks = append(a.itemClicks, new(widget.Clickable))
	}

	if len(a.filtered) > 0 {
		at := elements.AbsolutePosition()
		trigger := image.Rectangle{Min: at, Max: at.Add(dims.Size)}
		filtered, width := a.filtered, dims.Size.X
		elements.RegisterPortalAround(gtx, trigger, 4, elements.PlaceAuto, func() elements.Element {
			return elements.Raw(func(gtx layout.Context) layout.Dimensions {
				return a.panel(gtx, filtered, width)
			})
		})
	}
	return dims
}

// panel lays out the suggestion list and processes item clicks.
func (a *AutoComplete) panel(gtx layout.Context, filtered []string, width int) layout.Dimensions {
	st := styles.Merge(styles.Styles{
		Background:   defaultBackground,
		BorderRadius: 8,
		Padding:      properties.UniformInset(4),
	}, a.st)
	st.FlexDirection = properties.FlexDirectionColumn
	gtx.Constraints.Min.X = width
	gtx.Constraints.Max.X = width

	children := make([]elements.Element, 0, len(filtered))
	for i, opt := range filtered {
		children = append(children, a.row(i, opt))
	}
	return elements.Box(st, children...).Layout(gtx)
}

// row renders one hoverable, clickable suggestion.
func (a *AutoComplete) row(i int, opt string) elements.Element {
	click := a.itemClicks[i]
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		if click.Clicked(gtx) {
			a.field.SetText(opt)
			a.lastText = opt
			a.suppressed = true
			if a.onSelect != nil {
				a.onSelect(opt)
			}
			kit.RequestFrame(gtx)
		}
		dims := click.Layout(gtx, kit.Unselectable(func(gtx layout.Context) layout.Dimensions {
			textColor := a.st.Color
			if textColor == "" {
				textColor = defaultText
			}
			rowSt := styles.Styles{
				Padding:      properties.SymmetricInset(12, 8),
				BorderRadius: 4,
			}
			if click.Hovered() {
				rowSt.Background = defaultHover
				pointer.CursorPointer.Add(gtx.Ops)
			}
			return elements.Box(rowSt, elements.Text(opt, styles.Styles{Color: textColor})).Layout(gtx)
		}))
		// Keep the suggestion clickable even when a frameless window's
		// whole-window drag area happens to reach this portal - see
		// button.Layout's identical comment.
		kit.BlockDrag(gtx, dims.Size)
		return dims
	})
}
