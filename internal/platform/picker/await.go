package picker

import "context"

// await runs f on its own goroutine and returns its result, or
// ctx.Err() if ctx ends first, in which case a result arriving later
// is passed to discard (when f succeeded).
func await[T any](ctx context.Context, f func() (T, error), discard func(T)) (T, error) {
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := f()
		done <- result{v, err}
	}()
	select {
	case r := <-done:
		return r.v, r.err
	case <-ctx.Done():
		go func() {
			if r := <-done; r.err == nil {
				discard(r.v)
			}
		}()
		var zero T
		return zero, ctx.Err()
	}
}
