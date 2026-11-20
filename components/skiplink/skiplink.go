// Package skiplink provides landmarks - the large parts of a window, such
// as the navigation and the main content - and skip links, which let a
// keyboard user jump past the navigation straight to a landmark.
package skiplink

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/state"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Landmark is a large part of the window a skip link jumps to. Screen
// readers list it as a region with its name, to move between regions.
// Create it once and wrap its content every frame:
//
//	main := giode.NewLandmark("Main content")
//	...
//	main.Wrap(content)
type Landmark struct {
	name  string
	scope kit.FocusScope
	// seen is set once it has been laid out, so its scope's list is
	// current.
	seen bool
}

// NewLandmark returns a Landmark named name.
func NewLandmark(name string) *Landmark {
	return &Landmark{name: name}
}

// Name returns the landmark's name.
func (l *Landmark) Name() string { return l.name }

// Wrap returns el as the landmark's content.
func (l *Landmark) Wrap(el elements.Element) elements.Element {
	return landmarkElement{l: l, el: el}
}

type landmarkElement struct {
	l  *Landmark
	el elements.Element
}

func (w landmarkElement) Layout(gtx layout.Context) layout.Dimensions {
	pop := kit.PushFocusScope(&w.l.scope)
	g := elements.BeginSemanticGroup(elements.SemanticNode{Kind: elements.SemanticRegion, Label: w.l.name})
	dims := w.el.Layout(gtx)
	g.End(gtx, dims.Size)
	pop()
	w.l.seen = true
	return dims
}

// First returns the focus tag of the first control in the landmark, or
// nil when it has none.
func (l *Landmark) First() any {
	if !l.seen {
		return nil
	}
	return l.scope.First()
}

// SkipLink is a link that stays out of sight until it takes keyboard
// focus - the first Tab press in a window, when it's the first control -
// and then shows in the corner where it sits. Enter or Space moves the
// focus to the first control of its landmark, skipping everything
// between. Create it once and lay it out first in the view:
//
//	skip := giode.NewSkipLink("Skip to main content", main)
type SkipLink struct {
	label  string
	target *Landmark
	st     styles.Styles
	// jump is set by Click, for the next layout.
	jump     bool
	focusReq bool
	focused  bool
	pill     image.Point
}

// New returns a skip link labelled label, jumping to target. The styles
// argument is optional: Color, Background, Padding, BorderRadius and
// FontSize of the link when it shows.
func New(label string, target *Landmark, st ...styles.Styles) *SkipLink {
	s := &SkipLink{label: label, target: target}
	if len(st) > 0 {
		s.st = st[0]
	}
	return s
}

// Click jumps to the landmark, as Enter does.
func (s *SkipLink) Click() {
	s.jump = true
	state.RequestRedraw()
}

// Focus gives the link keyboard focus, which shows it.
func (s *SkipLink) Focus() {
	s.jump = false
	s.focusReq = true
	state.RequestRedraw()
}

// Focused reports whether the link had keyboard focus last frame.
func (s *SkipLink) Focused() bool { return s.focused }

// Layout lays the link out. It takes no space: when focused, it shows
// over what comes after it.
func (s *SkipLink) Layout(gtx layout.Context) layout.Dimensions {
	if s.focusReq {
		s.focusReq = false
		gtx.Execute(key.FocusCmd{Tag: s})
	}
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: s},
			key.Filter{Focus: s, Name: key.NameReturn},
			key.Filter{Focus: s, Name: key.NameEnter},
			key.Filter{Focus: s, Name: key.NameSpace},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case key.FocusEvent:
			s.focused = e.Focus
		case key.Event:
			if e.State == key.Press {
				s.jump = true
			}
		}
	}
	if s.jump {
		s.jump = false
		if t := s.target.First(); t != nil {
			gtx.Execute(key.FocusCmd{Tag: t})
			s.focused = false
		}
	}
	// A tiny area to hold focus: Tab goes through controls in the order
	// they're laid out.
	area := clip.Rect{Max: image.Pt(1, 1)}.Push(gtx.Ops)
	event.Op(gtx.Ops, s)
	area.Pop()
	kit.FocusVisible(gtx, s)
	focused := gtx.Focused(s)
	if focused {
		margin := gtx.Dp(unit.Dp(8))
		anchor := elements.AbsolutePosition().Add(image.Pt(margin, margin))
		elements.RegisterPortal(gtx, anchor, s.build)
	}
	size := s.pill
	if size == (image.Point{}) {
		size = image.Pt(1, 1)
	}
	elements.RecordSemantic(gtx, elements.SemanticNode{
		Kind: elements.SemanticLink, Label: s.label, Target: s, Focused: focused,
	}, size)
	return layout.Dimensions{}
}

// build is the link as it shows while focused.
func (s *SkipLink) build() elements.Element {
	st := s.st
	if st.Background == "" {
		st.Background = "theme.primary"
	}
	if st.Color == "" {
		st.Color = "theme.onprimary"
	}
	if st.Padding == (properties.Inset{}) {
		st.Padding = properties.Inset{Top: 8, Bottom: 8, Left: 14, Right: 14}
	}
	if st.BorderRadius == 0 {
		st.BorderRadius = 6
	}
	text := elements.Text(s.label, styles.Styles{Color: st.Color, FontSize: st.FontSize, FontWeight: properties.FontWeightSemiBold})
	pill := elements.Box(styles.Styles{Background: st.Background, Padding: st.Padding, BorderRadius: st.BorderRadius}, text)
	return ringed{s: s, el: pill}
}

// ringed draws the focus ring around the shown link.
type ringed struct {
	s  *SkipLink
	el elements.Element
}

func (r ringed) Layout(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = image.Point{}
	dims := r.el.Layout(gtx)
	r.s.pill = dims.Size
	kit.PaintFocusRing(gtx, dims.Size, gtx.Dp(unit.Dp(6)))
	return dims
}
