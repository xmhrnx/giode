package speech

import (
	"context"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
	"github.com/xmhrnx/giode/internal/platform"
)

func Listen(ctx context.Context, h platform.Host, lang string) (<-chan Result, error) {
	var id int
	var in <-chan mobile.Speech
	err := platform.OnAndroidMain(ctx, h, func(view uintptr) error {
		var err error
		id, in, err = mobile.Dictate(app.JavaVM(), app.AppContext(), view, lang)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make(chan Result, 16)
	go func() {
		defer close(out)
		stopped := false
		for {
			select {
			case s := <-in:
				out <- Result{Text: s.Text, Final: s.Done, Err: s.Err}
				if s.Done {
					return
				}
			case <-ctx.Done():
				if !stopped {
					stopped = true
					h.RunOnWindow(func() { mobile.StopDictation(app.JavaVM(), app.AppContext(), id) })
				}
				ctx = context.Background() // keep waiting for the final result
			}
		}
	}()
	return out, nil
}
