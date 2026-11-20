//go:build !android && !ios

package background

import (
	"context"
	"sync"
	"time"
)

// On computers a task runs on a timer while the app runs: apps there
// stay running in the tray or the background without the system's
// help.

var timers = struct {
	sync.Mutex
	stop map[string]context.CancelFunc
}{stop: map[string]context.CancelFunc{}}

func schedule(name string, o Options) error {
	ctx, stop := context.WithCancel(context.Background())
	timers.Lock()
	if old := timers.stop[name]; old != nil {
		old()
	}
	timers.stop[name] = stop
	timers.Unlock()
	go func() {
		wait := o.Delay
		for {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return
			}
			run(ctx, name)
			if o.Interval <= 0 {
				timers.Lock()
				delete(timers.stop, name)
				timers.Unlock()
				return
			}
			wait = o.Interval
		}
	}()
	return nil
}

func cancel(name string) error {
	timers.Lock()
	defer timers.Unlock()
	if stop := timers.stop[name]; stop != nil {
		stop()
		delete(timers.stop, name)
	}
	return nil
}
