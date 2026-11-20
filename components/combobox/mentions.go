package combobox

import (
	"image"
	"time"
	"unicode"

	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
)

// Mentions adds suggestions to a text field as the user types a
// trigger: "@" for people, "#" for channels, ":" for emoji. The list
// opens at the caret, filtered by what follows the trigger; Up, Down
// and Enter or Tab pick, Escape dismisses. Picking replaces the trigger
// and query with the option (by default the trigger, its label and a
// space). It takes over the field's OnKeys while the list is open. Lay
// it out in place of the field.
//
//	m := combobox.NewMentions(field).
//		Trigger('@', people...).
//		TriggerSource('#', searchChannels)
//
// For mentions in rich text, the rich editor has its own (Suggest).
type Mentions struct {
	field   *input.Input
	static  map[rune][]Option
	sources map[rune]Source
	list    List
	src     fetcher
	open    bool
	trig    rune
	start   int // the trigger's rune index
	query   string
	// dismissed is the trigger position Escape closed the list for.
	dismissed int
	format    func(trigger rune, o Option) string
	onPick    func(trigger rune, o Option)
	lastText  string
	lastCaret int
	now       time.Time
	outside   int
	panel     image.Rectangle
}

// NewMentions adds suggestions to field.
func NewMentions(field *input.Input) *Mentions {
	m := &Mentions{field: field, static: map[rune][]Option{}, sources: map[rune]Source{}, dismissed: -1}
	m.list.OnPick = m.pick
	m.src.debounce = 150 * time.Millisecond
	return m
}

// Trigger suggests opts after r.
func (m *Mentions) Trigger(r rune, opts ...Option) *Mentions {
	m.static[r] = opts
	return m
}

// TriggerSource suggests what src finds after r.
func (m *Mentions) TriggerSource(r rune, src Source) *Mentions {
	m.sources[r] = src
	return m
}

// Format sets the text a pick inserts.
func (m *Mentions) Format(fn func(trigger rune, o Option) string) *Mentions {
	m.format = fn
	return m
}

// OnPick sets a handler called with each pick, to record who was
// mentioned.
func (m *Mentions) OnPick(fn func(trigger rune, o Option)) *Mentions {
	m.onPick = fn
	return m
}

// Opened reports whether the list shows.
func (m *Mentions) Opened() bool { return m.open }

// find looks for a trigger before the caret: at the start or after a
// space, with no space between it and the caret.
func (m *Mentions) find() (trig rune, start int, query string, ok bool) {
	a, b := m.field.Selection()
	if a != b {
		return 0, 0, "", false
	}
	rs := []rune(m.field.Text())
	if b > len(rs) {
		return 0, 0, "", false
	}
	for i := b - 1; i >= 0 && b-i <= 64; i-- {
		r := rs[i]
		if unicode.IsSpace(r) {
			return 0, 0, "", false
		}
		_, st := m.static[r]
		_, sr := m.sources[r]
		if (st || sr) && (i == 0 || unicode.IsSpace(rs[i-1])) {
			return r, i, string(rs[i+1 : b]), true
		}
	}
	return 0, 0, "", false
}

func (m *Mentions) update() {
	trig, start, q, ok := m.find()
	if !ok || start == m.dismissed {
		m.setOpen(false)
		if !ok {
			m.dismissed = -1
		}
		return
	}
	if m.open && trig == m.trig && start == m.start && q == m.query {
		return
	}
	m.trig, m.start, m.query = trig, start, q
	if src := m.sources[trig]; src != nil {
		m.src.fn = src
		m.src.request(q, true, m.now)
	} else {
		m.src.fn = nil
		m.list.Loading, m.list.Err = false, nil
		m.list.SetItems(Filter(m.static[trig], q))
	}
	m.setOpen(true)
}

func (m *Mentions) setOpen(on bool) {
	if on == m.open {
		return
	}
	m.open = on
	if on {
		m.field.OnKeys(m.key, key.NameUpArrow, key.NameDownArrow, key.NameReturn, key.NameEnter, key.NameTab, key.NameEscape)
	} else {
		m.field.OnKeys(nil)
	}
}

func (m *Mentions) key(e key.Event) {
	switch e.Name {
	case key.NameUpArrow:
		m.list.Move(-1)
	case key.NameDownArrow:
		m.list.Move(1)
	case key.NameReturn, key.NameEnter, key.NameTab:
		if o, ok := m.list.Selected(); ok {
			m.pick(o)
		} else {
			m.setOpen(false)
		}
	case key.NameEscape:
		m.dismissed = m.start
		m.setOpen(false)
	}
}

func (m *Mentions) pick(o Option) {
	text := string(m.trig) + o.text() + " "
	if m.format != nil {
		text = m.format(m.trig, o)
	}
	_, caret := m.field.Selection()
	m.field.SetSelection(m.start, caret)
	m.field.InsertText(text)
	n := m.start + len([]rune(text))
	m.field.SetSelection(n, n)
	m.setOpen(false)
	m.lastText = m.field.Text()
	m.lastCaret = n
	if m.onPick != nil {
		m.onPick(m.trig, o)
	}
}

// Layout lays out the field and, while open, the suggestions at the
// caret.
func (m *Mentions) Layout(gtx layout.Context) layout.Dimensions {
	m.now = gtx.Now
	if m.now.IsZero() {
		m.now = time.Now()
	}
	dims := m.field.Layout(gtx)
	_, caret := m.field.Selection()
	if text := m.field.Text(); text != m.lastText || caret != m.lastCaret {
		m.lastText, m.lastCaret = text, caret
		if m.field.Focused() {
			m.update()
		}
	}
	if !m.field.Focused() {
		m.setOpen(false)
	}
	if m.src.fn != nil && m.open {
		m.src.poll(gtx, &m.list)
	}
	if m.open && (len(m.list.Items) > 0 || m.list.Loading) {
		elements.RegisterOverlay(func() elements.Element {
			return elements.Raw(func(gtx layout.Context) layout.Dimensions {
				open := m.open
				closeOnPress(gtx, &m.outside, &open, m.panel)
				if !open {
					m.dismissed = m.start
					m.setOpen(false)
				}
				return layout.Dimensions{}
			})
		})
		r := m.field.CaretRect()
		elements.RegisterPortalAround(gtx, r, 4, elements.PlaceAuto, func() elements.Element {
			return elements.Raw(func(gtx layout.Context) layout.Dimensions {
				d := m.list.Layout(gtx, gtx.Dp(240))
				at := elements.AbsolutePosition()
				m.panel = image.Rectangle{Min: at, Max: at.Add(d.Size)}
				kit.BlockDrag(gtx, d.Size)
				return d
			})
		})
	}
	return dims
}
