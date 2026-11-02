// Package copybutton provides a button that copies text to the
// clipboard and briefly confirms it with a check mark.
package copybutton

import (
	"time"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// DefaultFeedback is how long the check mark shows after copying.
const DefaultFeedback = 1500 * time.Millisecond

// CopyButton copies text to the clipboard when clicked and shows a
// check mark (and its "copied" label) for a moment. Create it once and
// lay it out every frame:
//
//	copy := giode.CopyButton(func() string { return apiKey.Text() })
type CopyButton struct {
	text     func() string
	label    string
	copied   string
	st       styles.Styles
	feedback time.Duration
	onCopy   func(text string)

	press    *elements.Pressable
	pending  *string
	copiedAt time.Time
	showing  bool
}

// New returns a button copying text() when clicked. The styles
// argument is optional: Color (glyph and label, default theme.text),
// Background, Padding, BorderRadius, FontSize.
func New(text func() string, st ...styles.Styles) *CopyButton {
	c := &CopyButton{text: text, label: builtinLabel, copied: builtinLabel, feedback: DefaultFeedback}
	if len(st) > 0 {
		c.st = st[0]
	}
	c.press = elements.NewPressable(c.content).OnClick(func(elements.ClickEvent) {
		if c.text != nil {
			t := c.text()
			c.pending = &t
		}
	}).Label(i18n.T("giode.copy"))
	return c
}

// builtinLabel stands for giode's own, translated label.
const builtinLabel = "\x00giode"

// Labels sets the text shown before and after copying; empty strings
// show the glyph alone. It returns c for chaining.
func (c *CopyButton) Labels(label, copied string) *CopyButton {
	c.label, c.copied = label, copied
	if label != "" {
		c.press.Label(label)
	}
	return c
}

// Feedback sets how long the check mark shows. It returns c for
// chaining.
func (c *CopyButton) Feedback(d time.Duration) *CopyButton {
	c.feedback = d
	return c
}

// OnCopy sets a handler run with the copied text. It returns c for
// chaining.
func (c *CopyButton) OnCopy(fn func(text string)) *CopyButton {
	c.onCopy = fn
	return c
}

// Copied reports whether the confirmation is showing.
func (c *CopyButton) Copied() bool {
	return c.showing
}

// Click copies as a click would; call it on the UI goroutine.
func (c *CopyButton) Click() {
	c.press.Click()
}

func (c *CopyButton) content(s elements.PressState) elements.Element {
	st := styles.Merge(styles.Styles{
		Color:        "theme.text",
		Padding:      properties.SymmetricInset(10, 6),
		BorderRadius: 6,
		Gap:          6,
		Align:        properties.AlignCenter,
		FontSize:     13,
	}, c.st)
	st.FlexDirection = properties.FlexDirectionRow
	switch {
	case s.Pressed:
		st.Background = kit.Alpha(st.Color, 0.16)
	case s.Hovered:
		st.Background = kit.Alpha(st.Color, 0.08)
	}
	glyph, label := kit.GlyphCopy, c.label
	if label == builtinLabel {
		label = i18n.T("giode.copy")
	}
	if c.showing {
		glyph, label = kit.GlyphCheck, c.copied
		if label == builtinLabel {
			label = i18n.T("giode.copied")
		}
	}
	children := []elements.Element{elements.Glyph(glyph, styles.Styles{Color: st.Color, Width: 14})}
	if label != "" {
		children = append(children, elements.Text(label, styles.Styles{Color: st.Color, FontSize: st.FontSize}))
	}
	return elements.Box(st, children...)
}

// Layout lays out and updates the button.
func (c *CopyButton) Layout(gtx layout.Context) layout.Dimensions {
	c.showing = !c.copiedAt.IsZero() && gtx.Now.Before(c.copiedAt.Add(c.feedback))
	dims := c.press.Layout(gtx)
	if c.pending != nil {
		text := *c.pending
		c.pending = nil
		kit.WriteClipboard(gtx, text)
		c.copiedAt = gtx.Now
		if c.onCopy != nil {
			c.onCopy(text)
		}
		kit.RequestFrame(gtx)
	}
	if !c.copiedAt.IsZero() {
		end := c.copiedAt.Add(c.feedback)
		if gtx.Now.Before(end) {
			kit.RequestFrameAt(gtx, end)
		} else {
			c.copiedAt = time.Time{}
			if c.showing {
				kit.RequestFrame(gtx)
			}
		}
	}
	return dims
}
