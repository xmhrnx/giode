// Command dictation is a note field with a microphone button: hold a
// thought, tap the button, speak. It uses the dictate package, which an
// app imports only when it dictates; on iOS it needs
// NSMicrophoneUsageDescription and NSSpeechRecognitionUsageDescription
// (ios.usage.microphone and ios.usage.speech in giode.toml).
package main

import (
	"context"
	"image"
	"log"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/dictate"
)

func main() {
	var app *giode.App
	status := "Tap the microphone and speak."
	var cancel context.CancelFunc
	note := giode.Input.Text("A note…")

	mic := giode.Button("Dictate", giode.Styles{BorderRadius: 8}).OnClick(func() {
		if cancel != nil {
			// Listening already: stop; the final result still comes.
			cancel()
			return
		}
		ctx, stop := context.WithCancel(context.Background())
		results, err := dictate.Start(ctx, app, "")
		if err != nil {
			stop()
			status = err.Error()
			return
		}
		cancel, status = stop, "Listening…"
		before := note.Text()
		if before != "" {
			before += " "
		}
		go func() {
			for t := range results {
				app.Post(func() {
					if t.Err != nil {
						status = t.Err.Error()
					} else {
						note.SetText(before + t.Text)
					}
					if t.Final {
						cancel()
						cancel = nil
						if t.Err == nil {
							status = "Done."
						}
					}
				})
			}
		}()
	})

	app = giode.New(giode.Config{Title: "Dictation", Size: image.Pt(420, 300), Background: "theme.background"})
	err := app.Run(func() giode.Element {
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(20), Gap: 12},
			giode.Row(giode.Styles{Width: -1, Gap: 8, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap},
				giode.Box(giode.Styles{FlexGrow: 1}, note), mic),
			giode.Text(status, giode.Styles{Color: "theme.muted"}),
		)
	})
	if err != nil {
		log.Fatal(err)
	}
}
