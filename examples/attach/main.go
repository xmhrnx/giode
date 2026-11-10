// Command attach is an "attach a photo" row: take one with the camera
// (package takephoto, which an app imports only when it takes photos)
// or pick one from the library (core giode). On iOS taking a photo
// needs NSCameraUsageDescription (ios.usage.camera in giode.toml).
package main

import (
	"context"
	"fmt"
	"image"
	"io"
	"log"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/takephoto"
)

func main() {
	var app *giode.App
	status := "No photo yet."
	show := func(f giode.PickedFile, err error) {
		app.Post(func() {
			switch {
			case err == giode.ErrCanceled:
				status = "Canceled."
			case err != nil:
				status = err.Error()
			default:
				n, _ := io.Copy(io.Discard, f)
				f.Close()
				status = fmt.Sprintf("Attached %s, %d KB.", f.Name, n/1024)
			}
		})
	}
	take := giode.Button("Take a photo", giode.Styles{BorderRadius: 8}).OnClick(func() {
		go func() { show(takephoto.Take(context.Background(), app, 2048)) }()
	})
	pick := giode.Button("Choose a photo", giode.Styles{BorderRadius: 8}).OnClick(func() {
		go func() { show(app.PickPhoto(context.Background(), giode.PhotoOptions{MaxSide: 2048})) }()
	})

	app = giode.New(giode.Config{Title: "Attach", Size: image.Pt(420, 240), Background: "theme.background"})
	err := app.Run(func() giode.Element {
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(20), Gap: 12},
			giode.Row(giode.Styles{Gap: 8}, take, pick),
			giode.Text(status, giode.Styles{Color: "theme.muted"}),
		)
	})
	if err != nil {
		log.Fatal(err)
	}
}
