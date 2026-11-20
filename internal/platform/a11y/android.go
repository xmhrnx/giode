//go:build android && giode_gio6

package a11y

import (
	"image"

	"gioui.org/app"

	"github.com/xmhrnx/giode/elements"
)

// TalkBack on Android: giode's patched GioView fills its virtual
// accessibility nodes from the tree (see tree.go), and runs their
// actions here.
// Install sets the tree up for the platform's bridge.
func Install(a Host) *Tree {
	t := NewTree()
	setTree(a, t)
	app.GiodeSetA11yProvider(&androidA11y{a: a, t: t})
	return t
}

type androidA11y struct {
	a Host
	t *Tree
	// sel is the selection A11ySelection last reported.
	sel [4]int
}

func (p *androidA11y) A11yActive(on bool) {
	if on != elements.SemanticsWanted() {
		elements.SetSemanticsWanted(on)
		p.a.Invalidate()
	}
}

func (p *androidA11y) A11yNode(id int) string {
	if id < 0 {
		id = 0
	}
	return p.t.androidJSON(int32(id))
}

func (p *androidA11y) A11yAt(x, y float32) int {
	return int(p.t.at(image.Pt(int(x), int(y))))
}

func (p *androidA11y) A11yAction(id, action int, arg string) bool {
	act := androidAction(action)
	if act == 0 {
		return false
	}
	if act == a11ySetSelection {
		// The range comes in UTF-16 units, as Android counts.
		arg = utf16RangeToRunes(p.t, int32(id), arg)
	}
	return p.t.perform(p.a, int32(id), act, arg)
}

// A11ySelection reports the focused text field's selection when it
// changed since the last call, for giode's Gio patches (revision 21 on)
// to send TalkBack a text selection event.
func (p *androidA11y) A11ySelection() (id, start, end, count int, ok bool) {
	k, s, e, c, found := p.t.androidSelection()
	if !found {
		p.sel = [4]int{}
		return 0, 0, 0, 0, false
	}
	now := [4]int{int(k), s, e, c}
	if now == p.sel {
		return 0, 0, 0, 0, false
	}
	p.sel = now
	return int(k), s, e, c, true
}

func (p *androidA11y) A11yChanged() (ids []int, announce []string, assertive []bool) {
	ch, ann := p.t.takeChanges()
	for _, k := range ch {
		if k == 0 {
			ids = append(ids, -1)
		} else {
			ids = append(ids, int(k))
		}
	}
	for _, a := range ann {
		announce = append(announce, a.text)
		assertive = append(assertive, a.assertive)
	}
	return ids, announce, assertive
}
