// Package background runs the app's work while it isn't on screen: a
// sync every hour or so, a cleanup at night while charging. The system
// decides when, within the app's constraints, and starts the app's
// process for it when it isn't running.
//
//   - iOS: BGTaskScheduler. A task is an app refresh (about 30 seconds)
//     or, when Long, a processing task (minutes, usually at night while
//     charging). Each task's name must be listed in giode.toml's
//     app.background_tasks, which permits its identifier.
//   - Android: JobScheduler, with network and charging constraints; a
//     job gets up to about ten minutes. Periodic jobs run at most every
//     15 minutes and survive reboots.
//   - Computers: the task runs on a timer while the app runs.
//
// Register each task's function early, in main before App.Run: the
// system can start the app's process just to run one.
package background

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Options say when a task may run.
type Options struct {
	// Interval runs the task again about this often; zero runs it once.
	// Android's minimum is 15 minutes; iOS treats it as the earliest
	// next run.
	Interval time.Duration
	// Delay is the earliest the first run starts.
	Delay time.Duration
	// Network runs the task only with a network connection.
	Network bool
	// Charging runs the task only while the device charges.
	Charging bool
	// Long asks for minutes rather than seconds: an iOS processing task,
	// which the system usually runs at night. Android jobs get minutes
	// either way.
	Long bool
}

// Func is a task's work. ctx is canceled when the system ends the task's
// time; return soon after. An error asks the system to try again later
// (Android) or reports a failed run (iOS).
type Func func(ctx context.Context) error

// ErrUnknown is returned by Schedule for a task without Register.
var ErrUnknown = errors.New("background: no task registered with this name")

var (
	mu    sync.Mutex
	funcs = map[string]Func{}
	// registered is signalled when Register adds a task, for runs that
	// started before main got there.
	registered = sync.NewCond(&mu)
	// timeout is how long a run waits for its task's Register.
	timeout = 10 * time.Second
)

// Register sets the work of the task with the given name.
func Register(name string, f Func) {
	mu.Lock()
	funcs[name] = f
	mu.Unlock()
	registered.Broadcast()
}

// Schedule asks the system to run the task, replacing an earlier
// schedule of it.
func Schedule(name string, o Options) error {
	mu.Lock()
	_, ok := funcs[name]
	mu.Unlock()
	if !ok {
		return ErrUnknown
	}
	return schedule(name, o)
}

// Cancel removes the task's schedule.
func Cancel(name string) error {
	return cancel(name)
}

// lookup waits for the task's Register, up to timeout.
func lookup(name string) Func {
	deadline := time.Now().Add(timeout)
	stop := time.AfterFunc(timeout, registered.Broadcast)
	defer stop.Stop()
	mu.Lock()
	defer mu.Unlock()
	for {
		if f, ok := funcs[name]; ok {
			return f
		}
		if time.Now().After(deadline) {
			return nil
		}
		registered.Wait()
	}
}

// run runs a task the system started, until it returns or ctx ends.
func run(ctx context.Context, name string) error {
	f := lookup(name)
	if f == nil {
		return ErrUnknown
	}
	return f(ctx)
}
