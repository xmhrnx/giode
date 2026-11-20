// Command voicememo records the microphone and plays it back, with a
// level meter, a device picker and an echo test that hears itself
// through a voice stream's echo cancellation. It uses the audio
// package; on iOS and macOS it needs NSMicrophoneUsageDescription
// (ios.usage.microphone in giode.toml).
package main

import (
	"context"
	"image"
	"log"
	"sync"
	"time"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/audio"
	"github.com/xmhrnx/giode/styles/properties"
)

func main() {
	var (
		app       *giode.App
		mu        sync.Mutex
		recorded  []int16
		level     float64
		status    = "Record a memo, or try the echo test."
		in        *audio.Input
		out       *audio.Output
		echo      bool
		inputs    []audio.Device
		deviceIdx int
	)
	opts := func() audio.Options {
		o := audio.Options{SampleRate: 48000}
		if deviceIdx > 0 && deviceIdx <= len(inputs) {
			o.Device = inputs[deviceIdx-1].ID
		}
		return o
	}
	stop := func() {
		if in != nil {
			in.Close()
			in = nil
		}
		if out != nil {
			out.Close()
			out = nil
		}
		echo = false
	}
	record := giode.Button("Record").OnClick(func() {
		if in != nil {
			stop()
			status = "Recorded " + (time.Duration(len(recorded)) * time.Second / 48000).Round(100*time.Millisecond).String() + "."
			return
		}
		recorded = recorded[:0]
		var err error
		in, err = audio.OpenInput(context.Background(), app, opts(), func(s []int16) {
			mu.Lock()
			recorded = append(recorded, s...)
			level = audio.Level(s)
			mu.Unlock()
			app.Invalidate()
		})
		if err != nil {
			status = err.Error()
			return
		}
		status = "Recording…"
	})
	play := giode.Button("Play").OnClick(func() {
		stop()
		var err error
		out, err = audio.OpenOutput(context.Background(), app, audio.Options{SampleRate: 48000})
		if err != nil {
			status = err.Error()
			return
		}
		mu.Lock()
		s := append([]int16(nil), recorded...)
		mu.Unlock()
		o := out
		go o.Write(context.Background(), s)
		status = "Playing…"
	})
	echoTest := giode.Button("Echo test").OnClick(func() {
		if echo {
			stop()
			status = "Echo test stopped."
			return
		}
		stop()
		// A voice stream in and out: what's played back half a second
		// later is cancelled from the microphone, so it doesn't loop.
		o := opts()
		o.Voice = true
		var err error
		out, err = audio.OpenOutput(context.Background(), app, o)
		if err != nil {
			status = err.Error()
			return
		}
		delay := make([]int16, 24000)
		po := out
		in, err = audio.OpenInput(context.Background(), app, o, func(s []int16) {
			delay = append(delay, s...)
			po.Write(context.Background(), delay[:len(s)])
			delay = delay[len(s):]
			mu.Lock()
			level = audio.Level(s)
			mu.Unlock()
			app.Invalidate()
		})
		if err != nil {
			stop()
			status = err.Error()
			return
		}
		echo = true
		status = "Speak: you'll hear yourself half a second later."
	})
	devices := giode.Input.Select(giode.Styles{}, "Default microphone")
	go func() {
		ds, err := audio.Inputs(context.Background())
		if err != nil {
			return
		}
		app.Post(func() {
			inputs = ds
			names := []string{"Default microphone"}
			for _, d := range ds {
				names = append(names, d.Name)
			}
			devices.SetOptions(names...)
		})
	}()
	devices.OnChange(func(i int) { deviceIdx = i })

	app = giode.New(giode.Config{Title: "Voice memo", Size: image.Pt(460, 260), Background: "theme.background"})
	err := app.Run(func() giode.Element {
		mu.Lock()
		l := level
		mu.Unlock()
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(20), Gap: 14},
			devices,
			giode.Row(giode.Styles{Gap: 8}, record, play, echoTest),
			giode.Box(giode.Styles{Width: -1, Height: 8, Background: "theme.surface", BorderRadius: 4},
				giode.Box(giode.Styles{Width: properties.Width(max(1, int(320*l))), Height: 8, Background: "theme.success", BorderRadius: 4})),
			giode.Text(status, giode.Styles{Color: "theme.muted"}),
		)
	})
	if err != nil {
		log.Fatal(err)
	}
}
