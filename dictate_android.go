package giode

import (
	"context"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

func (a *App) dictate(ctx context.Context, lang string) (<-chan Transcript, error) {
	var id int
	var in <-chan mobile.Speech
	err := a.onAndroidMain(ctx, func(view uintptr) error {
		var err error
		id, in, err = mobile.Dictate(app.JavaVM(), app.AppContext(), view, lang)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make(chan Transcript, 16)
	go func() {
		defer close(out)
		stopped := false
		for {
			select {
			case s := <-in:
				out <- Transcript{Text: s.Text, Final: s.Done, Err: s.Err}
				if s.Done {
					return
				}
			case <-ctx.Done():
				if !stopped {
					stopped = true
					a.window.Run(func() { mobile.StopDictation(app.JavaVM(), app.AppContext(), id) })
				}
				ctx = context.Background() // keep waiting for the final result
			}
		}
	}()
	return out, nil
}
