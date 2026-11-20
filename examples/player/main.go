// Command player plays a video or audio file, or a URL, with the media
// package's controls: play and pause, seeking, speed, volume and SRT or
// WebVTT captions.
package main

import (
	"context"
	"image"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/media"
)

func main() {
	var (
		app    *giode.App
		player *media.Player
		status = "Open a video or audio file, or paste a URL."
	)
	open := func(src string) {
		if player != nil {
			player.Close()
			player = nil
		}
		p, err := media.Open(context.Background(), app, src, media.Options{
			Autoplay: true,
			OnError:  func(err error) { app.Post(func() { status = err.Error() }) },
		})
		if err != nil {
			status = err.Error()
			return
		}
		player = p
		status = filepath.Base(src)
	}
	// local returns a picked file's path, copying it where it has none
	// (Android's picker gives a stream).
	local := func(f giode.PickedFile) (string, error) {
		defer f.Close()
		if f.Path != "" {
			return f.Path, nil
		}
		out, err := os.Create(filepath.Join(os.TempDir(), filepath.Base(f.Name)))
		if err != nil {
			return "", err
		}
		defer out.Close()
		_, err = io.Copy(out, f)
		return out.Name(), err
	}
	pickVideo := giode.Button("Open file").OnClick(func() {
		go func() {
			files, err := app.PickFiles(context.Background(), giode.PickOptions{Types: []string{"video/*", "audio/*"}})
			if err != nil || len(files) == 0 {
				return
			}
			path, err := local(files[0])
			app.Post(func() {
				if err != nil {
					status = err.Error()
					return
				}
				open(path)
			})
		}()
	})
	pickCaptions := giode.Button("Add captions").OnClick(func() {
		go func() {
			files, err := app.PickFiles(context.Background(), giode.PickOptions{Types: []string{"srt", "vtt"}})
			if err != nil || len(files) == 0 {
				return
			}
			defer files[0].Close()
			cues, err := media.ParseCaptions(files[0])
			app.Post(func() {
				if err != nil {
					status = err.Error()
					return
				}
				if player != nil {
					name := strings.TrimSuffix(files[0].Name, filepath.Ext(files[0].Name))
					player.SetCaptionTrack(player.AddCaptions(media.Track{Label: name, Cues: cues}))
				}
			})
		}()
	})
	url := giode.Input.Text("https://…", giode.Styles{FlexGrow: 1})
	play := giode.Button("Play URL").OnClick(func() {
		if u := strings.TrimSpace(url.Text()); u != "" {
			open(u)
		}
	})
	app = giode.New(giode.Config{Title: "Player", Size: image.Pt(800, 560), Background: "theme.background"})
	if len(os.Args) > 1 {
		app.Post(func() { open(os.Args[1]) })
	}
	err := app.Run(func() giode.Element {
		var body giode.Element = giode.Box(giode.Styles{FlexGrow: 1, Width: -1, Background: "theme.surface", BorderRadius: 8,
			Align: giode.AlignCenter, Justify: giode.JustifyCenter}, giode.Text(status, giode.Styles{Color: "theme.muted"}))
		if player != nil {
			body = giode.Box(giode.Styles{FlexGrow: 1, Width: -1, BorderRadius: 8, Overflow: giode.OverflowHidden},
				player.View(media.ViewOptions{}))
		}
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(16), Gap: 12},
			giode.Row(giode.Styles{Width: -1, Gap: 8, Align: giode.AlignCenter}, pickVideo, pickCaptions, url, play),
			body,
		)
	})
	if err != nil {
		log.Fatal(err)
	}
}
