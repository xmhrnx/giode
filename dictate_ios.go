package giode

import (
	"context"

	"github.com/xmhrnx/giode/internal/dictation"
)

// dictate runs package dictate's speech recognizer (SFSpeechRecognizer).
func (a *App) dictate(ctx context.Context, lang string) (<-chan Transcript, error) {
	if dictation.Start == nil {
		return nil, errDictateNotLinked
	}
	in := make(chan Transcript, 16)
	stop, err := dictation.Start(lang, func(text string, final bool, err error) {
		t := Transcript{Text: text, Final: final, Err: err}
		if final {
			// Never block the main thread on a slow reader; the final
			// result still arrives.
			go func() { in <- t }()
			return
		}
		select {
		case in <- t:
		default: // a slow reader misses partial results
		}
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
			case t := <-in:
				out <- t
				if t.Final {
					return
				}
			case <-ctx.Done():
				if !stopped {
					stopped = true
					stop()
				}
				ctx = context.Background() // keep waiting for the final result
			}
		}
	}()
	return out, nil
}
