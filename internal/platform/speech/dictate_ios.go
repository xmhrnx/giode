package speech

import (
	"context"

	"github.com/xmhrnx/giode/internal/optin"
	"github.com/xmhrnx/giode/internal/platform"
)

// Listen runs package dictate's speech recognizer (SFSpeechRecognizer).
func Listen(ctx context.Context, h platform.Host, lang string) (<-chan Result, error) {
	if optin.Dictate == nil {
		return nil, platform.NotLinked("dictate")
	}
	in := make(chan Result, 16)
	stop, err := optin.Dictate(lang, func(text string, final bool, err error) {
		t := Result{Text: text, Final: final, Err: err}
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
	out := make(chan Result, 16)
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
