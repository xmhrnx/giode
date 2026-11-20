package media

import (
	"fmt"
	"strings"
	"time"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/components/iconbutton"
	"github.com/xmhrnx/giode/components/slider"
	"github.com/xmhrnx/giode/styles/properties"
	"github.com/xmhrnx/giode/video"
)

// Material icon paths, 24×24.
const (
	iconPlay     = "M8 5v14l11-7z"
	iconPause    = "M6 19h4V5H6v14zm8-14v14h4V5h-4z"
	iconReplay   = "M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"
	iconVolume   = "M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"
	iconMuted    = "M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"
	iconCaptions = "M19 4H5c-1.11 0-2 .9-2 2v12c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm-8 7H9.5v-.5h-2v3h2V13H11v1c0 .55-.45 1-1 1H7c-.55 0-1-.45-1-1v-4c0-.55.45-1 1-1h3c.55 0 1 .45 1 1v1zm7 0h-1.5v-.5h-2v3h2V13H18v1c0 .55-.45 1-1 1h-3c-.55 0-1-.45-1-1v-4c0-.55.45-1 1-1h3c.55 0 1 .45 1 1v1z"
)

// Speeds are the rates the speed button steps through.
var Speeds = []float64{0.5, 0.75, 1, 1.25, 1.5, 2}

// uiState is the controls' widgets, made on first use.
type uiState struct {
	made     bool
	play     *iconbutton.Button
	mute     *iconbutton.Button
	captions *iconbutton.Button
	speed    *giode.ButtonElement
	seek     *slider.Slider
	volume   *slider.Slider
}

func (p *Player) widgets() *uiState {
	u := &p.ui
	if u.made {
		return u
	}
	u.made = true
	u.play = giode.IconButton(giode.IconSVG(iconPlay)).Variant(giode.IconButtonGhost).Label("Play").OnClick(p.Toggle)
	u.mute = giode.IconButton(giode.IconSVG(iconVolume)).Variant(giode.IconButtonGhost).Label("Mute").OnClick(func() {
		p.SetMuted(!p.Muted())
	})
	u.captions = giode.IconButton(giode.IconSVG(iconCaptions)).Variant(giode.IconButtonGhost).Label("Captions").OnClick(func() {
		// Step through the tracks, then off.
		n := len(p.Captions())
		t := p.CaptionTrack() + 1
		if t >= n {
			t = -1
		}
		p.SetCaptionTrack(t)
	})
	u.speed = giode.Button("1×").OnClick(func() {
		cur := p.Speed()
		next := Speeds[0]
		for _, s := range Speeds {
			if s > cur+0.001 {
				next = s
				break
			}
		}
		p.SetSpeed(next)
	})
	u.seek = giode.Input.Slider(giode.Styles{FlexGrow: 1}).OnChange(func(v float32) {
		if d := p.Duration(); d > 0 {
			p.Seek(time.Duration(float64(v) * float64(d)))
		}
	})
	u.volume = giode.Input.Slider(giode.Styles{Width: 80}).OnChange(func(v float32) {
		p.SetVolume(float64(v))
		if v > 0 && p.Muted() {
			p.SetMuted(false)
		}
	})
	u.volume.SetValue(float32(p.Volume()))
	return u
}

// ViewOptions configure View.
type ViewOptions struct {
	// Fit is how the picture fills the view: video.Contain by default.
	Fit video.Fit
	// NoControls hides the control bar.
	NoControls bool
	// CaptionSize is the captions' text size in sp: 18 by default.
	CaptionSize int
	// Label is read by screen readers for the picture.
	Label string
}

// View returns the picture with captions over it and the controls
// under it. It fills the space it's given.
func (p *Player) View(opts ViewOptions) giode.Element {
	if opts.CaptionSize == 0 {
		opts.CaptionSize = 18
	}
	label := opts.Label
	if label == "" {
		label = "Video"
	}
	children := []giode.Element{
		p.surface.View(video.ViewOptions{Fit: opts.Fit, Label: label, Placeholder: p.placeholder()}),
	}
	if c := p.Caption(); c != "" {
		children = append(children, giode.Box(giode.Styles{
			Position: giode.PositionAbsolute, Left: giode.Flush, Right: giode.Flush, Bottom: 16,
			Align: giode.AlignCenter,
		}, giode.Box(giode.Styles{Background: "#000000b0", Padding: giode.Inset{Top: 4, Bottom: 4, Left: 10, Right: 10}, BorderRadius: 4},
			giode.Accessible(giode.Text(c, giode.Styles{Color: "#ffffff", FontSize: properties.FontSize(opts.CaptionSize), TextAlign: giode.TextAlignCenter}),
				func(n *giode.SemanticNode) { n.Live = giode.LivePolite }))))
	}
	pic := giode.Box(giode.Styles{FlexGrow: 1, Width: -1, Background: "#000000"}, children...)
	if opts.NoControls {
		return giode.Stack(giode.Styles{Width: -1, Height: -1}, pic)
	}
	return giode.Stack(giode.Styles{Width: -1, Height: -1}, pic, p.Controls())
}

func (p *Player) placeholder() giode.Element {
	var msg string
	switch p.State() {
	case Loading:
		msg = "Loading…"
	case Failed:
		msg = "Can't play this: " + errText(p.Err())
	}
	if msg == "" {
		return nil
	}
	return giode.Box(giode.Styles{Width: -1, Height: -1, Align: giode.AlignCenter, Justify: giode.JustifyCenter},
		giode.Text(msg, giode.Styles{Color: "#ffffffb0"}))
}

func errText(err error) string {
	if err == nil {
		return "unknown error"
	}
	return strings.TrimPrefix(err.Error(), "media: ")
}

// Controls returns the control bar alone: play, the seek bar and
// times, speed, captions and volume. Use it for audio.
func (p *Player) Controls() giode.Element {
	u := p.widgets()
	st, pos, dur := p.State(), p.Position(), p.Duration()
	switch st {
	case Playing:
		u.play.Icon(giode.IconSVG(iconPause)).Label("Pause")
	case Ended:
		u.play.Icon(giode.IconSVG(iconReplay)).Label("Play again")
	default:
		u.play.Icon(giode.IconSVG(iconPlay)).Label("Play")
	}
	if p.Muted() {
		u.mute.Icon(giode.IconSVG(iconMuted)).Label("Unmute")
	} else {
		u.mute.Icon(giode.IconSVG(iconVolume)).Label("Mute")
	}
	if !u.seek.Dragging() && dur > 0 {
		u.seek.SetValue(float32(float64(pos) / float64(dur)))
	}
	if !u.volume.Dragging() {
		v := p.Volume()
		if p.Muted() {
			v = 0
		}
		u.volume.SetValue(float32(v))
	}
	u.speed.Label(speedLabel(p.Speed()))
	items := []giode.Element{u.play, u.seek,
		giode.Text(FormatTime(pos)+" / "+FormatTime(dur), giode.Styles{Color: "theme.muted", FontSize: 13}),
		u.speed,
	}
	if len(p.Captions()) > 0 {
		cc := u.captions
		if p.CaptionTrack() >= 0 {
			cc.Variant(giode.IconButtonContained).Label("Captions: " + p.Captions()[p.CaptionTrack()].Label)
		} else {
			cc.Variant(giode.IconButtonGhost).Label("Captions off")
		}
		items = append(items, cc)
	}
	items = append(items, u.mute, u.volume)
	return giode.Row(giode.Styles{Width: -1, Align: giode.AlignCenter, Gap: 6, Padding: giode.Inset{Top: 4, Bottom: 4, Left: 6, Right: 10}}, items...)
}

func speedLabel(s float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", s), "0"), ".") + "×"
}

// FormatTime formats d as m:ss, or h:mm:ss from an hour.
func FormatTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int(d / time.Second)
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
