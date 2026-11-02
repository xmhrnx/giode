package router

import (
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/xmhrnx/giode/elements"
)

// Router is a stateful navigator. Create it once with New and lay it
// out every frame; it renders the active route's view.
type Router struct {
	routes   []Route
	path     string
	history  []string
	notFound View

	linkClicks map[string]*widget.Clickable
}

// New returns a Router with the given routes. The first route is
// shown initially.
func New(routes ...Route) *Router {
	r := &Router{
		routes:     routes,
		linkClicks: make(map[string]*widget.Clickable),
	}
	if len(routes) > 0 {
		r.path = routes[0].Path
	}
	return r
}

// NotFound sets the view rendered for unknown paths. It returns r for
// chaining.
func (r *Router) NotFound(v View) *Router {
	r.notFound = v
	return r
}

// Current returns the active path.
func (r *Router) Current() string {
	return r.path
}

// Navigate switches to path, replacing the current entry in the
// history.
func (r *Router) Navigate(path string) {
	if r.path == path {
		return
	}
	if len(r.history) > 0 {
		r.history[len(r.history)-1] = path
	} else {
		r.history = append(r.history, path)
	}
	r.path = path
}

// Push navigates to path, pushing the current path onto the history
// stack so Back returns to it.
func (r *Router) Push(path string) {
	if r.path == path {
		return
	}
	r.history = append(r.history, r.path)
	r.path = path
}

// Back pops the history stack. It reports whether there was anywhere
// to go back to.
func (r *Router) Back() bool {
	if len(r.history) == 0 {
		return false
	}
	r.path = r.history[len(r.history)-1]
	r.history = r.history[:len(r.history)-1]
	return true
}

// view returns the view for a path.
func (r *Router) view(path string) View {
	for _, rt := range r.routes {
		if rt.Path == path {
			return rt.View
		}
	}
	return r.notFound
}

// Layout renders the active route's view. While there is history to
// go back to, the system Back button (Android) pops it - see
// elements.HandleBack.
func (r *Router) Layout(gtx layout.Context) layout.Dimensions {
	if len(r.history) > 0 {
		elements.HandleBack(func() { r.Back() })
	}
	v := r.view(r.path)
	if v == nil {
		return layout.Dimensions{}
	}
	return v().Layout(gtx)
}
