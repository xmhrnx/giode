// Package doctabs provides a strip of document tabs, as editors and
// browsers show: tabs the app adds, removes and renames as documents
// open and close, each with a close button and a dot for unsaved
// changes, an optional "+" button, and tabs that shrink and then
// scroll when they don't fit.

package doctabs

import (
	"image"
	"slices"
	"time"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/menu"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

// Tab is one document's tab.
type Tab struct {
	// ID identifies the tab to the app, as a file's path would. IDs are
	// unique within a strip.
	ID string
	// Title is the name the tab shows, cut off with "…" when the tab is
	// narrower.
	Title string
	// Tooltip shows when the pointer rests on the tab, such as the
	// file's full path. Empty shows the title when it's cut off.
	Tooltip string
	// Modified shows a dot for unsaved changes, in the close button's
	// place until the pointer is over the tab.
	Modified bool
}

// DocTabs is a stateful strip of document tabs. Create it once with
// New, change its tabs as documents open and close, and lay it out
// every frame above the selected document:
//
//	tabs := giode.DocTabs().
//		OnSelect(func(id string) { showDocument(id) }).
//		OnClose(func(id string) { closeDocument(id) }). // asks, then tabs.Remove(id)
//		OnNew(newDocument)
//	tabs.Add(giode.DocTab{ID: path, Title: filepath.Base(path), Tooltip: path})
//	tabs.Select(path)
//
// Clicking a tab selects it; dragging it with the mouse moves it;
// a middle click or its close button closes it, and a right-click (or a
// long-press) opens its menu: Close, Close Others, Close to the Right,
// Close All. When the tabs don't fit they shrink to TabWidth's minimum,
// then scroll with the wheel or a swipe.
type DocTabs struct {
	// Contextual gives it a context menu (Context), over the whole
	// strip; a tab's own menu opens over a tab.
	elements.Contextual

	tabs     []Tab
	selected string
	st       styles.Styles
	minW     int
	maxW     int
	closable bool

	onSelect    func(id string)
	onClose     func(id string)
	onCloseTabs func(ids []string)
	onNew       func()
	onReorder   func(ids []string)
	tabMenu     func(id string, m *menu.Menu)
	menuOff     bool

	// Layout of the last frame, in strip pixels: hit tests use it.
	slots   []slot
	newRect image.Rectangle
	tabsW   int // the width the tabs show in
	total   int // the tabs' whole width
	height  int
	scrollX int
	reveal  bool

	// Pointer state.
	hover      string // the tab under the pointer
	hoverClose bool   // over its close button
	hoverNew   bool
	hoverAt    time.Time
	press      pressState
	pass       func(gtx layout.Context) bool

	// The tabs' menu, rebuilt for the tab it opens on; tabCtx anchors
	// it to each tab.
	menu    *menu.Menu
	menuFor string
	tabCtx  elements.Contextual
}

// slot is where a tab was laid out.
type slot struct {
	id        string
	r         image.Rectangle
	close     image.Rectangle
	truncated bool
}

type pressState struct {
	id     pointer.ID
	active bool
	button pointer.Buttons
	touch  bool
	tab    string
	close  bool
	newBtn bool
	at     f32.Point
	// grab is where in the tab the pointer took it, for a drag.
	grab     float32
	dragging bool
	dragX    float32
	moved    bool
	scroll0  int
}

// New returns an empty strip. The styles argument is optional:
// Background fills the strip, Color is the selected tab's text (the
// others are muted), BorderColor the lines between tabs and under the
// strip, FontSize the titles' size (13 by default) and Height the
// strip's height in dp (34 by default).
func New(st ...styles.Styles) *DocTabs {
	d := &DocTabs{minW: 96, maxW: 220, closable: true, menu: menu.New()}
	if len(st) > 0 {
		d.st = st[0]
	}
	d.tabCtx.Context(d.menu)
	return d
}

// Add appends tab, or updates the tab with its ID. It doesn't select
// it: call Select. It returns d for chaining.
func (d *DocTabs) Add(tab Tab) *DocTabs {
	if i := d.Index(tab.ID); i >= 0 {
		d.tabs[i] = tab
		return d
	}
	d.tabs = append(d.tabs, tab)
	return d
}

// Insert puts tab at index i (clamped to the tabs there are), or moves
// and updates the tab with its ID there.
func (d *DocTabs) Insert(i int, tab Tab) {
	if j := d.Index(tab.ID); j >= 0 {
		d.tabs = slices.Delete(d.tabs, j, j+1)
	}
	i = min(max(i, 0), len(d.tabs))
	d.tabs = slices.Insert(d.tabs, i, tab)
}

// Remove takes the tab with id away. When it was selected, the tab
// after it (or else before it) is selected and reported to OnSelect,
// with "" when it was the last tab.
func (d *DocTabs) Remove(id string) {
	i := d.Index(id)
	if i < 0 {
		return
	}
	d.tabs = slices.Delete(d.tabs, i, i+1)
	if d.hover == id {
		d.hover = ""
	}
	if d.selected != id {
		return
	}
	next := ""
	if len(d.tabs) > 0 {
		next = d.tabs[min(i, len(d.tabs)-1)].ID
	}
	d.selected = next
	d.reveal = true
	if d.onSelect != nil {
		d.onSelect(next)
	}
}

// SetTitle renames the tab with id.
func (d *DocTabs) SetTitle(id, title string) {
	if i := d.Index(id); i >= 0 {
		d.tabs[i].Title = title
	}
}

// SetTooltip sets the text shown when the pointer rests on the tab.
func (d *DocTabs) SetTooltip(id, tooltip string) {
	if i := d.Index(id); i >= 0 {
		d.tabs[i].Tooltip = tooltip
	}
}

// SetModified shows or hides the tab's unsaved-changes dot.
func (d *DocTabs) SetModified(id string, modified bool) {
	if i := d.Index(id); i >= 0 {
		d.tabs[i].Modified = modified
	}
}

// Rename changes a tab's ID, as when a file is saved under a new name,
// keeping its place and selection.
func (d *DocTabs) Rename(id, newID string) {
	i := d.Index(id)
	if i < 0 || (newID != id && d.Index(newID) >= 0) {
		return
	}
	d.tabs[i].ID = newID
	if d.selected == id {
		d.selected = newID
	}
}

// Move puts the tab with id at index to.
func (d *DocTabs) Move(id string, to int) {
	i := d.Index(id)
	if i < 0 {
		return
	}
	tab := d.tabs[i]
	d.tabs = slices.Delete(d.tabs, i, i+1)
	to = min(max(to, 0), len(d.tabs))
	d.tabs = slices.Insert(d.tabs, to, tab)
}

// Select selects the tab with id, scrolling it into view, without
// calling OnSelect.
func (d *DocTabs) Select(id string) {
	if d.Index(id) >= 0 {
		d.selected = id
		d.reveal = true
	}
}

// SelectNext selects the tab step places after the selected one (before
// it for a negative step), wrapping around, and reports it to OnSelect:
// for Ctrl+Tab and Ctrl+Shift+Tab.
func (d *DocTabs) SelectNext(step int) {
	n := len(d.tabs)
	if n == 0 {
		return
	}
	i := d.Index(d.selected)
	if i < 0 {
		i = 0
	} else {
		i = ((i+step)%n + n) % n
	}
	d.selectTab(d.tabs[i].ID)
}

// Selected returns the selected tab's ID, or "".
func (d *DocTabs) Selected() string {
	return d.selected
}

// Tabs returns the tabs, in order.
func (d *DocTabs) Tabs() []Tab {
	return slices.Clone(d.tabs)
}

// Tab returns the tab with id.
func (d *DocTabs) Tab(id string) (Tab, bool) {
	if i := d.Index(id); i >= 0 {
		return d.tabs[i], true
	}
	return Tab{}, false
}

// Len returns the number of tabs.
func (d *DocTabs) Len() int {
	return len(d.tabs)
}

// Index returns the position of the tab with id, or -1.
func (d *DocTabs) Index(id string) int {
	for i := range d.tabs {
		if d.tabs[i].ID == id {
			return i
		}
	}
	return -1
}

// OnSelect sets the handler called when the user selects a tab, and
// when Remove or a close moves the selection. It returns d for
// chaining.
func (d *DocTabs) OnSelect(fn func(id string)) *DocTabs {
	d.onSelect = fn
	return d
}

// OnClose sets the handler called when the user closes a tab: its close
// button, a middle click or its menu's Close. The tab stays until the
// handler calls Remove, so it can ask about unsaved changes first.
// Without a handler, closing removes the tab. It returns d for
// chaining.
func (d *DocTabs) OnClose(fn func(id string)) *DocTabs {
	d.onClose = fn
	return d
}

// OnCloseTabs sets the handler for the menu's Close Others, Close to
// the Right and Close All, given the tabs to close in order, so the app
// can ask about all their unsaved changes at once. Without it, OnClose
// is called for each. It returns d for chaining.
func (d *DocTabs) OnCloseTabs(fn func(ids []string)) *DocTabs {
	d.onCloseTabs = fn
	return d
}

// OnNew shows a "+" button after the tabs, calling fn when clicked. It
// returns d for chaining.
func (d *DocTabs) OnNew(fn func()) *DocTabs {
	d.onNew = fn
	return d
}

// OnReorder sets the handler called after the user drags a tab to a
// new place, with the tabs' IDs in their new order. It returns d for
// chaining.
func (d *DocTabs) OnReorder(fn func(ids []string)) *DocTabs {
	d.onReorder = fn
	return d
}

// TabMenu lets fn add items to a tab's menu after the built-in close
// items, each time it opens on the tab with id (start with
// m.Separator()): Copy Path, Reveal in Folder. It returns d for
// chaining.
func (d *DocTabs) TabMenu(fn func(id string, m *menu.Menu)) *DocTabs {
	d.tabMenu = fn
	return d
}

// NoTabMenu turns the tabs' menu off. It returns d for chaining.
func (d *DocTabs) NoTabMenu() *DocTabs {
	d.menuOff = true
	d.tabCtx.Context(nil)
	return d
}

// Closable shows or hides the tabs' close buttons (and turns middle
// clicks off); true by default. It returns d for chaining.
func (d *DocTabs) Closable(on bool) *DocTabs {
	d.closable = on
	return d
}

// TabWidth sets the narrowest and widest a tab gets, in dp: 96 and 220
// by default. Tabs shrink toward min before the strip scrolls. It
// returns d for chaining.
func (d *DocTabs) TabWidth(min, max int) *DocTabs {
	if min > 0 {
		d.minW = min
	}
	if max >= d.minW {
		d.maxW = max
	}
	return d
}

// selectTab selects id as the user did, reporting it.
func (d *DocTabs) selectTab(id string) {
	d.reveal = true
	if d.selected == id {
		return
	}
	d.selected = id
	if d.onSelect != nil {
		d.onSelect(id)
	}
}

// requestClose closes ids as the user asked.
func (d *DocTabs) requestClose(ids []string) {
	switch {
	case len(ids) == 0:
	case len(ids) > 1 && d.onCloseTabs != nil:
		d.onCloseTabs(ids)
	case d.onClose != nil:
		for _, id := range ids {
			d.onClose(id)
		}
	default:
		for _, id := range ids {
			d.Remove(id)
		}
	}
}

// buildMenu fills the tabs' menu for the tab with id.
func (d *DocTabs) buildMenu(id string) {
	d.menuFor = id
	m := d.menu.Clear()
	i := d.Index(id)
	if i < 0 {
		return
	}
	ids := func(keep func(j int) bool) []string {
		var out []string
		for j, t := range d.tabs {
			if keep(j) {
				out = append(out, t.ID)
			}
		}
		return out
	}
	if d.closable {
		m.Item("Close", func() { d.requestClose([]string{id}) })
		if len(d.tabs) > 1 {
			m.Item("Close Others", func() {
				d.requestClose(ids(func(j int) bool { return d.tabs[j].ID != id }))
			})
		} else {
			m.DisabledItem("Close Others")
		}
		if i < len(d.tabs)-1 {
			m.Item("Close to the Right", func() {
				k := d.Index(id)
				d.requestClose(ids(func(j int) bool { return j > k }))
			})
		} else {
			m.DisabledItem("Close to the Right")
		}
		m.Item("Close All", func() { d.requestClose(ids(func(int) bool { return true })) })
	}
	if d.tabMenu != nil {
		d.tabMenu(id, m)
	}
}
