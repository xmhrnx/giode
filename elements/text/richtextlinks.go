package text

import (
	"gioui.org/gesture"
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements/internal/base"
)

// handleLinks opens the links tapped since the last frame.
func (r *RichTextElement) handleLinks(gtx layout.Context, links []resolvedSpan) {
	for _, s := range links {
		for {
			e, ok := s.click.Update(gtx.Source)
			if !ok {
				break
			}
			if e.Kind == gesture.KindClick {
				r.openLink(s.url)
			}
		}
	}
}

func (r *RichTextElement) openLink(url string) {
	if r.onLink != nil {
		r.onLink(url)
		return
	}
	if fn := linkOpener.Load(); fn != nil {
		(*fn)(url)
	}
}

func (r *RichTextElement) selection(gtx layout.Context) *selectState {
	if r.selKey == nil {
		return nil
	}
	s := base.Keyed("giode.selectable", r.selKey, newSelectState)
	s.update(gtx)
	kit.HandleInput(s.passFunc())
	return s
}
