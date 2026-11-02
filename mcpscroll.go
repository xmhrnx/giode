package giode

import (
	"context"
	"fmt"
	"image"
	"strings"

	"gioui.org/op"

	"github.com/xmhrnx/giode/elements"
)

// scroller is a scroll container automation can move (Scroll,
// VirtualList).
type scroller interface {
	ScrollBy(dx, dy float32)
	ScrollOffset() (x, y float32)
}

// maxScrollSteps bounds how many times ui_scroll moves while looking
// for a node.
const maxScrollSteps = 60

// scroll implements the ui_scroll tool.
func (b *mcpBridge) scroll(ctx context.Context, a *App, id string, dx, dy int, until string) (any, error) {
	nodes, size, err := b.snapshot(ctx, a)
	if err != nil {
		return nil, err
	}
	win := image.Rectangle{Max: size}
	if until == "" {
		c, err := scrollArea(nodes, id)
		if err != nil {
			return nil, err
		}
		if dx == 0 && dy == 0 {
			return nil, fmt.Errorf("give dx or dy to scroll by, or until to scroll to a node")
		}
		if err := b.scrollBy(ctx, a, c, float32(dx), float32(dy)); err != nil {
			return nil, err
		}
		// The offset is clamped to the content on the next layout.
		if err := b.afterFrame(ctx, a, func(*op.Ops, image.Point) {}); err != nil {
			return nil, err
		}
		x, y, err := b.scrollOffset(ctx, a, c)
		if err != nil {
			return nil, err
		}
		return map[string]any{"scroll": c.ID, "offset": [2]int{int(x), int(y)}}, nil
	}

	// Scroll until the node is visible. A node that isn't laid out (a
	// virtual list row) is searched for a page at a time.
	dir := 1
	if dy < 0 {
		dir = -1
	}
	for step := 0; ; step++ {
		target, found := findNode(nodes, until)
		if found {
			c, delta, ok := revealStep(nodes, target, win)
			if !ok {
				return map[string]any{"visible": true, "node": reportNode(target), "scroll": target.Scroll}, nil
			}
			if c.ID == "" || step >= maxScrollSteps {
				return map[string]any{"visible": false, "node": reportNode(target),
					"reason": "it's outside the window and no scroll area can bring it in"}, nil
			}
			moved, err := b.scrollAndCheck(ctx, a, c, delta)
			if err != nil {
				return nil, err
			}
			if !moved {
				return map[string]any{"visible": false, "node": reportNode(target), "scroll": c.ID,
					"reason": "the scroll area can't scroll further"}, nil
			}
		} else {
			c, err := scrollArea(nodes, id)
			if err != nil {
				return nil, fmt.Errorf("no node matches %q, and %w", until, err)
			}
			page := float32(dir) * float32(c.Bounds.Intersect(win).Dy()) * 0.8
			moved, err := b.scrollAndCheck(ctx, a, c, image.Pt(0, int(page)))
			if err != nil {
				return nil, err
			}
			if !moved || step >= maxScrollSteps {
				where := "end"
				if dir < 0 {
					where = "top"
				}
				return map[string]any{"visible": false, "scroll": c.ID,
					"reason": fmt.Sprintf("no node matches %q; scrolled to the %s without finding one", until, where)}, nil
			}
		}
		if nodes, size, err = b.snapshot(ctx, a); err != nil {
			return nil, err
		}
		win = image.Rectangle{Max: size}
	}
}

// scrollAndCheck scrolls c by delta and reports whether it moved.
func (b *mcpBridge) scrollAndCheck(ctx context.Context, a *App, c elements.SemanticNode, delta image.Point) (bool, error) {
	x0, y0, err := b.scrollOffset(ctx, a, c)
	if err != nil {
		return false, err
	}
	if err := b.scrollBy(ctx, a, c, float32(delta.X), float32(delta.Y)); err != nil {
		return false, err
	}
	// The offset is clamped on the next layout.
	if err := b.afterFrame(ctx, a, func(*op.Ops, image.Point) {}); err != nil {
		return false, err
	}
	x1, y1, err := b.scrollOffset(ctx, a, c)
	if err != nil {
		return false, err
	}
	return int(x1) != int(x0) || int(y1) != int(y0), nil
}

func (b *mcpBridge) scrollBy(ctx context.Context, a *App, c elements.SemanticNode, dx, dy float32) error {
	s, ok := c.Target.(scroller)
	if !ok {
		return fmt.Errorf("scroll area %q can't be scrolled by automation", c.ID)
	}
	return a.RunOnUI(ctx, func() { s.ScrollBy(dx, dy) })
}

func (b *mcpBridge) scrollOffset(ctx context.Context, a *App, c elements.SemanticNode) (x, y float32, err error) {
	s, ok := c.Target.(scroller)
	if !ok {
		return 0, 0, fmt.Errorf("scroll area %q can't be scrolled by automation", c.ID)
	}
	err = a.RunOnUI(ctx, func() { x, y = s.ScrollOffset() })
	return x, y, err
}

// scrollArea returns the scroll container to scroll for id: the node
// itself when it's one, the one it's in otherwise, or with no id the
// main one - the largest not inside another.
func scrollArea(nodes []elements.SemanticNode, id string) (elements.SemanticNode, error) {
	if id == "" {
		var best elements.SemanticNode
		area := -1
		for _, n := range nodes {
			if n.Kind != elements.SemanticScroll || n.Scroll != "" {
				continue
			}
			if a := n.Bounds.Dx() * n.Bounds.Dy(); a > area {
				best, area = n, a
			}
		}
		if area < 0 {
			return best, fmt.Errorf("there's no scroll area on screen")
		}
		return best, nil
	}
	n, ok := nodeByID(nodes, id)
	switch {
	case !ok:
		return n, fmt.Errorf("no node %q on screen; call ui_snapshot again", id)
	case n.Kind == elements.SemanticScroll:
		return n, nil
	case n.Scroll == "":
		return n, fmt.Errorf("node %q isn't in a scroll area", id)
	}
	c, ok := nodeByID(nodes, n.Scroll)
	if !ok {
		return c, fmt.Errorf("the scroll area of node %q isn't on screen", id)
	}
	return c, nil
}

// revealStep returns the scroll container to move next to bring target
// into view, and by how much; ok is false when target is already fully
// visible. The innermost container is moved until the target is within
// its viewport, then the one around that, and so on out to the window.
func revealStep(nodes []elements.SemanticNode, target elements.SemanticNode, win image.Rectangle) (c elements.SemanticNode, delta image.Point, ok bool) {
	visible := win
	for p := target.Scroll; p != ""; {
		n, found := nodeByID(nodes, p)
		if !found {
			break
		}
		visible = visible.Intersect(n.Bounds)
		p = n.Scroll
	}
	if target.Bounds.In(visible) || target.Bounds.Empty() {
		return c, delta, false
	}
	for p := target.Scroll; p != ""; {
		n, found := nodeByID(nodes, p)
		if !found {
			break
		}
		// An inner container scrolls within its own bounds, wherever
		// they are (the containers around it bring them into view next);
		// the outermost one within the window.
		view := n.Bounds
		if n.Scroll == "" {
			view = view.Intersect(win)
		}
		if !view.Empty() {
			if d := revealDelta(target.Bounds, view); d != (image.Point{}) {
				return n, d, true
			}
		}
		p = n.Scroll
	}
	// Every container shows it, but the window doesn't: not reachable.
	return elements.SemanticNode{}, delta, true
}

// revealDelta is how far to scroll so t lies within view: its start
// edge when it's larger than view.
func revealDelta(t, view image.Rectangle) image.Point {
	axis := func(tmin, tmax, vmin, vmax int) int {
		switch {
		case tmax-tmin > vmax-vmin || tmin < vmin:
			return tmin - vmin
		case tmax > vmax:
			return tmax - vmax
		}
		return 0
	}
	return image.Pt(axis(t.Min.X, t.Max.X, view.Min.X, view.Max.X), axis(t.Min.Y, t.Max.Y, view.Min.Y, view.Max.Y))
}

// findNode finds a node by id, or else the first whose label or value
// is q, or else the first containing q, ignoring case. Scroll
// containers themselves are skipped.
func findNode(nodes []elements.SemanticNode, q string) (elements.SemanticNode, bool) {
	if n, ok := nodeByID(nodes, q); ok {
		return n, true
	}
	lq := strings.ToLower(q)
	for _, exact := range []bool{true, false} {
		for _, n := range nodes {
			if n.Kind == elements.SemanticScroll {
				continue
			}
			l, v := strings.ToLower(n.Label), strings.ToLower(n.Value)
			if exact && (l == lq || v == lq) || !exact && (strings.Contains(l, lq) || strings.Contains(v, lq)) {
				return n, true
			}
		}
	}
	return elements.SemanticNode{}, false
}

func nodeByID(nodes []elements.SemanticNode, id string) (elements.SemanticNode, bool) {
	for _, n := range nodes {
		if n.ID == id {
			return n, true
		}
	}
	return elements.SemanticNode{}, false
}

func reportNode(n elements.SemanticNode) snapshotNode {
	return snapshotNode{
		ID: n.ID, Kind: string(n.Kind), Label: n.Label, Value: n.Value, Disabled: n.Disabled,
		Bounds:  [4]int{n.Bounds.Min.X, n.Bounds.Min.Y, n.Bounds.Dx(), n.Bounds.Dy()},
		Actions: actionsOf(n.Target),
		Scroll:  n.Scroll,
	}
}
