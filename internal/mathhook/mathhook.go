// Package mathhook connects Markdown to the formula package without
// importing it: package formula registers its renderer when imported,
// so apps that don't set formulas don't carry the math font.
package mathhook

import (
	"sync/atomic"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/styles"
)

// Element is a laid-out formula.
type Element interface {
	Layout(gtx layout.Context) layout.Dimensions
}

// Renderer returns the formula src, in display style or inline, styled
// with st (FontSize, Color), or an error when it doesn't read.
type Renderer func(src string, display bool, st styles.Styles) (Element, error)

var renderer atomic.Pointer[Renderer]

// Set registers r.
func Set(r Renderer) { renderer.Store(&r) }

// Get returns the registered renderer, or nil.
func Get() Renderer {
	if r := renderer.Load(); r != nil {
		return *r
	}
	return nil
}
