//go:build android && giode_gio6

package giode

import (
	"image"

	"gioui.org/app"

	"github.com/xmhrnx/giode/elements"
)

// TalkBack on Android: giode's patched GioView fills its virtual
// accessibility nodes from the tree (see a11y_tree.go), and runs their
// actions here.
func installA11y(a *App) {
	t := newA11yTree()
	a.a11y = t
	app.GiodeSetA11yProvider(androidA11y{a: a, t: t})
}

type androidA11y struct {
	a *App
	t *a11yTree
}

func (p androidA11y) A11yActive(on bool) {
	if on != elements.SemanticsWanted() {
		elements.SetSemanticsWanted(on)
		p.a.invalidate()
	}
}

func (p androidA11y) A11yNode(id int) string {
	if id < 0 {
		id = 0
	}
	return p.t.androidJSON(int32(id))
}

func (p androidA11y) A11yAt(x, y float32) int {
	return int(p.t.at(image.Pt(int(x), int(y))))
}

func (p androidA11y) A11yAction(id, action int, arg string) bool {
	act := androidAction(action)
	if act == 0 {
		return false
	}
	return p.t.perform(p.a, int32(id), act, arg)
}

func (p androidA11y) A11yChanged() (ids []int, announce []string, assertive []bool) {
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
