// Package router provides a simple navigation layer: define views as
// routes, navigate between them with Push/Navigate/Back and link to
// them with Link.
package router

import "github.com/xmhrnx/giode/elements"

// View builds the element of a route on every frame it is shown.
type View func() elements.Element

// Route maps a path to its view.
type Route struct {
	Path string
	View View
}
