// Command screenshare captures a display and shows it live, the way a
// call would share it: pick a display, start, and frames stream in at
// the chosen rate. It uses the screencap package; on macOS the user
// allows screen recording in System Settings, on Android they confirm
// each time, and on iOS it captures the app's own screen.
package main

import (
	"context"
	"fmt"
	"image"
	"log"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/screencap"
	"github.com/xmhrnx/giode/video"
)

func main() {
	var (
		app     *giode.App
		surface *video.Surface
		capture *screencap.Capture
		sources []screencap.Source
		srcIdx  int
		status  = "Pick a display and start sharing."
	)
	picker := giode.Input.Select(giode.Styles{}, "Primary display")
	go func() {
		ss, err := screencap.Sources(context.Background())
		if err != nil {
			app.Post(func() { status = err.Error() })
			return
		}
		app.Post(func() {
			sources = ss
			names := []string{"Primary display"}
			for _, s := range ss {
				names = append(names, fmt.Sprintf("%s (%d×%d)", s.Name, s.Bounds.Dx(), s.Bounds.Dy()))
			}
			picker.SetOptions(names...)
		})
	}()
	picker.OnChange(func(i int) { srcIdx = i })

	var share *giode.ButtonElement
	share = giode.Button("Share screen").OnClick(func() {
		if capture != nil {
			capture.Stop()
			capture = nil
			surface.Clear()
			share.Label("Share screen")
			status = "Stopped."
			return
		}
		opts := screencap.Options{FPS: 15, MaxSize: 1280}
		if srcIdx > 0 && srcIdx <= len(sources) {
			opts.Source = sources[srcIdx-1].ID
		}
		status = "Starting…"
		go func() {
			c, err := screencap.Start(context.Background(), app, opts, func(f screencap.Frame) {
				surface.Push(f.Image)
			})
			app.Post(func() {
				if err != nil {
					status = err.Error()
					return
				}
				capture = c
				share.Label("Stop")
				status = "Sharing."
			})
		}()
	})

	app = giode.New(giode.Config{Title: "Screen share", Size: image.Pt(720, 520), Background: "theme.background"})
	surface = video.NewSurface(app)
	err := app.Run(func() giode.Element {
		line := status
		if sz := surface.Size(); capture != nil && sz.X > 0 {
			line = fmt.Sprintf("Sharing: %d×%d, %.1f frames a second.", sz.X, sz.Y, surface.FPS())
		}
		preview := giode.Box(giode.Styles{Width: -1, Height: -1, BorderRadius: 8, Overflow: giode.OverflowHidden},
			surface.View(video.ViewOptions{Label: "Shared screen"}))
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(20), Gap: 14},
			giode.Row(giode.Styles{Gap: 8}, picker, share),
			giode.Text(line, giode.Styles{Color: "theme.muted"}),
			preview,
		)
	})
	if err != nil {
		log.Fatal(err)
	}
}
