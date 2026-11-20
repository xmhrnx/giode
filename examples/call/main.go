// Command call rings itself and, once answered, runs a voice call over
// WebRTC between two peers inside the app: the microphone goes out
// through one, comes back through the other and plays on the speaker,
// so you hear yourself as the other side would. On phones the ring is
// the system's incoming call screen (telecom); on computers the app
// shows its own. Signaling is a function call here; a real app sends
// the offer, answer and candidates through its server.
package main

import (
	"context"
	"image"
	"log"
	"time"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/rtc"
	"github.com/xmhrnx/giode/telecom"
)

func main() {
	var (
		app     *giode.App
		caller  *rtc.Call // the far end: sends the microphone
		callee  *rtc.Call // this end: plays what arrives
		callID  string
		ringing bool
		status  = "Ring yourself, answer, and talk."
		muted   bool
		started time.Time
	)
	hangUp := func() {
		if caller != nil {
			caller.Close()
			callee.Close()
			caller, callee = nil, nil
		}
		ringing = false
		status = "Call ended."
	}
	connect := func() {
		ringing = false
		status = "Connecting…"
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			// Both peers are on this device: host candidates are enough,
			// with no STUN server.
			a, err := rtc.New(ctx, app, rtc.Config{Microphone: true})
			if err != nil {
				app.Post(func() { status = err.Error() })
				return
			}
			b, err := rtc.New(ctx, app, rtc.Config{Speaker: true, OnState: func(s rtc.State) {
				app.Post(func() {
					if s == rtc.StateConnected {
						started = time.Now()
						telecom.ReportConnected(callID)
					}
					status = "Call " + s.String() + "."
				})
			}})
			if err != nil {
				a.Close()
				app.Post(func() { status = err.Error() })
				return
			}
			// The signaling: here, function calls.
			offer, err := a.Offer(ctx)
			if err == nil {
				var answer string
				if answer, err = b.Answer(ctx, offer); err == nil {
					err = a.Accept(answer)
				}
			}
			app.Post(func() {
				if err != nil {
					a.Close()
					b.Close()
					status = err.Error()
					return
				}
				caller, callee = a, b
				caller.SetMuted(muted)
			})
		}()
	}

	ring := giode.Button("Ring me in 3 s").OnClick(func() {
		status = "Lock the phone or switch apps: it rings in 3 seconds."
		go func() {
			time.Sleep(3 * time.Second)
			app.Post(func() {
				callID = telecom.NewID()
				err := telecom.ReportIncoming(context.Background(), telecom.Call{ID: callID, Handle: "echo", Name: "Echo test"})
				if err != nil {
					// No system call screen: ring in the app.
					ringing = true
					status = "Echo test is calling."
				}
			})
		}()
	})
	answer := giode.Button("Answer").OnClick(connect)
	decline := giode.Button("Decline").OnClick(hangUp)
	mute := giode.Button("Mute").OnClick(func() {
		muted = !muted
		if caller != nil {
			caller.SetMuted(muted)
		}
		telecom.SetMuted(callID, muted)
	})
	end := giode.Button("Hang up").OnClick(func() {
		telecom.End(callID)
		hangUp()
	})

	app = giode.New(giode.Config{Title: "Call", Size: image.Pt(420, 300), Background: "theme.background"})
	telecom.Setup(app, telecom.Handler{
		OnAnswer: func(string) { connect() },
		OnEnd:    func(string) { hangUp() },
		OnMute: func(_ string, m bool) {
			muted = m
			if caller != nil {
				caller.SetMuted(m)
			}
		},
	})
	err := app.Run(func() giode.Element {
		row := []giode.Element{ring}
		switch {
		case ringing:
			row = []giode.Element{answer, decline}
		case caller != nil:
			if muted {
				mute.Label("Unmute")
			} else {
				mute.Label("Mute")
			}
			row = []giode.Element{mute, end}
		}
		line := status
		if caller != nil && !started.IsZero() && callee.State() == rtc.StateConnected {
			line = "Connected " + time.Since(started).Round(time.Second).String() + " · " + caller.AudioCodec()
		}
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(20), Gap: 14},
			giode.Text(line, giode.Styles{Color: "theme.muted"}),
			giode.Row(giode.Styles{Gap: 8}, row...),
		)
	})
	if err != nil {
		log.Fatal(err)
	}
}
