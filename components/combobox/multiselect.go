package combobox

import (
	"image"
	"strings"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/chip"
	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// MultiSelect picks several options, shown as removable chips before a
// search field. With Creatable it's a tag input: Enter, a comma or a
// paste of comma-separated text adds what's typed as new tags.
// Backspace in the empty field removes the last chip. Create it once
// with NewMulti and lay it out every frame.
type MultiSelect struct {
	elements.Contextual

	field     *input.Input
	list      List
	options   []Option
	src       fetcher
	values    []Option
	chips     map[string]*chip.Chip
	creatable bool
	max       int
	onChange  func([]Option)
	open      bool
	lastText  string
	outside   int
	now       time.Time
	panel     image.Rectangle
	st        styles.Styles
}

// NewMulti returns an empty MultiSelect.
func NewMulti(placeholder string, st ...styles.Styles) *MultiSelect {
	m := &MultiSelect{chips: map[string]*chip.Chip{}}
	if len(st) > 0 {
		m.st = st[0]
	}
	m.field = input.New(placeholder, styles.Styles{Padding: properties.SymmetricInset(4, 4), BorderColor: "#00000000", Width: -1})
	m.field.OnKeys(m.key, key.NameUpArrow, key.NameDownArrow, key.NameReturn, key.NameEnter, key.NameEscape, key.NameDeleteBackward)
	m.field.OnFocus(func() { m.show() })
	m.list.OnPick = m.add
	m.src.debounce = 250 * time.Millisecond
	return m
}

// Options sets the options to pick from, filtered as the user types.
func (m *MultiSelect) Options(opts ...Option) *MultiSelect {
	m.options = opts
	return m
}

// Strings sets options from plain strings.
func (m *MultiSelect) Strings(values ...string) *MultiSelect {
	opts := make([]Option, len(values))
	for i, v := range values {
		opts[i] = Option{Value: v}
	}
	return m.Options(opts...)
}

// Source fetches options as the user types; see Combobox.Source.
func (m *MultiSelect) Source(fn Source) *MultiSelect {
	m.src.fn = fn
	return m
}

// Creatable lets the user add values that aren't options: a tag input.
func (m *MultiSelect) Creatable(on bool) *MultiSelect {
	m.creatable = on
	return m
}

// Max caps how many values may be picked; 0 for no cap.
func (m *MultiSelect) Max(n int) *MultiSelect {
	m.max = n
	return m
}

// OnChange sets the handler called with the values after each change.
func (m *MultiSelect) OnChange(fn func([]Option)) *MultiSelect {
	m.onChange = fn
	return m
}

// Values are the picked options, in order.
func (m *MultiSelect) Values() []Option { return append([]Option(nil), m.values...) }

// Strings are the picked values.
func (m *MultiSelect) ValueStrings() []string {
	out := make([]string, len(m.values))
	for i, v := range m.values {
		out[i] = v.Value
	}
	return out
}

// SetValues replaces the picked options without calling OnChange.
func (m *MultiSelect) SetValues(vals ...Option) {
	m.values = nil
	for _, v := range vals {
		if !m.has(v.Value) {
			m.values = append(m.values, v)
		}
	}
}

// Input is the search field.
func (m *MultiSelect) Input() *input.Input { return m.field }

// Opened reports whether the option list shows.
func (m *MultiSelect) Opened() bool { return m.open }

func (m *MultiSelect) has(v string) bool {
	for _, x := range m.values {
		if x.Value == v {
			return true
		}
	}
	return false
}

func (m *MultiSelect) full() bool { return m.max > 0 && len(m.values) >= m.max }

// available are the options not picked yet, matching the query.
func (m *MultiSelect) refilter() {
	var opts []Option
	for _, o := range m.options {
		if !m.has(o.Value) {
			opts = append(opts, o)
		}
	}
	m.list.SetItems(Filter(opts, m.field.Text()))
}

func (m *MultiSelect) show() {
	if m.full() {
		return
	}
	m.open = true
	if m.src.fn == nil {
		m.refilter()
	} else {
		m.src.request(m.field.Text(), m.src.minChars == 0, m.now)
	}
}

func (m *MultiSelect) add(o Option) {
	if m.has(o.Value) || m.full() || strings.TrimSpace(o.Value) == "" {
		return
	}
	m.values = append(m.values, o)
	m.field.SetText("")
	m.lastText = ""
	m.changed()
	if m.full() {
		m.open = false
	} else if m.open {
		m.show()
	}
}

func (m *MultiSelect) remove(v string) {
	for i, x := range m.values {
		if x.Value == v {
			m.values = append(m.values[:i], m.values[i+1:]...)
			delete(m.chips, v)
			m.changed()
			if m.open && m.src.fn == nil {
				m.refilter()
			}
			return
		}
	}
}

func (m *MultiSelect) changed() {
	if m.onChange != nil {
		m.onChange(m.Values())
	}
}

// create adds typed text as a value: the matching option, or with
// Creatable a new one.
func (m *MultiSelect) create(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for _, o := range m.options {
		if strings.EqualFold(o.text(), text) || strings.EqualFold(o.Value, text) {
			m.add(o)
			return true
		}
	}
	if m.creatable {
		m.add(Option{Value: text})
		return true
	}
	return false
}

func (m *MultiSelect) key(e key.Event) {
	switch e.Name {
	case key.NameUpArrow, key.NameDownArrow:
		if !m.open {
			m.show()
		} else if e.Name == key.NameUpArrow {
			m.list.Move(-1)
		} else {
			m.list.Move(1)
		}
	case key.NameReturn, key.NameEnter:
		if o, ok := m.list.Selected(); ok && m.open && m.field.Text() != "" || ok && m.open && !m.creatable {
			m.add(o)
		} else {
			m.create(m.field.Text())
		}
	case key.NameEscape:
		m.open = false
	case key.NameDeleteBackward:
		text := m.field.Text()
		if text == "" {
			if n := len(m.values); n > 0 {
				m.remove(m.values[n-1].Value)
			}
			return
		}
		// The editor didn't see the key: delete as it would.
		start, end := m.field.Selection()
		if start > end {
			start, end = end, start
		}
		if start == end {
			if start == 0 {
				return
			}
			start--
			if e.Modifiers.Contain(key.ModShortcut) || e.Modifiers.Contain(key.ModAlt) {
				rs := []rune(text)
				for start > 0 && rs[start-1] == ' ' {
					start--
				}
				for start > 0 && rs[start-1] != ' ' {
					start--
				}
			}
		}
		m.field.SetSelection(start, end)
		m.field.InsertText("")
	}
}

// Layout lays out the chips, the field and, while open, the list.
func (m *MultiSelect) Layout(gtx layout.Context) layout.Dimensions {
	kit.OpaqueInput()
	m.now = gtx.Now
	if m.now.IsZero() {
		m.now = time.Now()
	}
	// Typed or pasted separators make values.
	if text := m.field.Text(); text != m.lastText {
		if strings.ContainsAny(text, ",;\n\t") {
			parts := strings.FieldsFunc(text, func(r rune) bool { return strings.ContainsRune(",;\n\t", r) })
			rest := ""
			if !strings.ContainsAny(text[len(text)-1:], ",;\n\t") && len(parts) > 0 {
				rest, parts = parts[len(parts)-1], parts[:len(parts)-1]
			}
			for _, p := range parts {
				m.create(p)
			}
			m.field.SetText(rest)
			text = rest
		}
		m.lastText = text
		m.open = m.field.Focused() && !m.full()
		if m.src.fn == nil {
			m.refilter()
		} else {
			m.src.request(text, len([]rune(text)) >= m.src.minChars, m.now)
		}
	}
	if m.src.fn != nil {
		m.src.poll(gtx, &m.list)
		// Drop picked values from fetched results.
		if !m.list.Loading {
			kept := m.list.Items[:0:0]
			for _, o := range m.list.Items {
				if !m.has(o.Value) {
					kept = append(kept, o)
				}
			}
			if len(kept) != len(m.list.Items) {
				m.list.Items = kept
				m.list.Index = min(m.list.Index, max(len(kept)-1, 0))
			}
		}
	}

	kids := make([]elements.Element, 0, len(m.values)+1)
	for _, v := range m.values {
		c := m.chips[v.Value]
		if c == nil {
			value := v.Value
			c = chip.New(v.text()).OnRemove(func() { m.remove(value) })
			m.chips[v.Value] = c
		}
		kids = append(kids, c)
	}
	kids = append(kids, elements.Box(styles.Styles{FlexGrow: 1, MinWidth: 80}, m.field))
	box := styles.Merge(styles.Styles{
		FlexDirection: properties.FlexDirectionRow, Wrap: properties.WrapWrap, Gap: 6, Align: properties.AlignCenter,
		Padding: properties.SymmetricInset(6, 4), BorderWidth: 1, BorderRadius: 6, BorderColor: "theme.border", Width: -1,
	}, m.st)
	if m.field.Focused() {
		box.BorderColor = "theme.primary"
	}
	dims := elements.Box(box, kids...).Layout(gtx)
	kit.BlockDrag(gtx, dims.Size)
	m.AnchorContext(gtx, dims.Size)

	showList := m.open && (len(m.list.Items) > 0 || m.list.Loading || m.list.Err != nil || !m.creatable)
	if showList {
		elements.HandleDismiss(func() { m.open = false })
		elements.RegisterOverlay(func() elements.Element {
			return elements.Raw(func(gtx layout.Context) layout.Dimensions {
				closeOnPress(gtx, &m.outside, &m.open, m.panel)
				return layout.Dimensions{}
			})
		})
		trigger, width := anchorBelow(dims.Size), dims.Size.X
		elements.RegisterPortalAround(gtx, trigger, 4, elements.PlaceAuto, func() elements.Element {
			return elements.Raw(func(gtx layout.Context) layout.Dimensions {
				d := m.list.Layout(gtx, width)
				at := elements.AbsolutePosition()
				m.panel = image.Rectangle{Min: at, Max: at.Add(d.Size)}
				return d
			})
		})
	}
	return dims
}
