//go:build !android && !ios

package background

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduleRuns(t *testing.T) {
	var n atomic.Int32
	Register("tick", func(ctx context.Context) error { n.Add(1); return nil })
	if err := Schedule("nope", Options{}); err != ErrUnknown {
		t.Error(err)
	}
	if err := Schedule("tick", Options{Delay: 10 * time.Millisecond, Interval: 20 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(120 * time.Millisecond)
	Cancel("tick")
	got := n.Load()
	if got < 3 {
		t.Errorf("ran %d times", got)
	}
	time.Sleep(60 * time.Millisecond)
	if n.Load() > got+1 {
		t.Error("still running after Cancel")
	}
}

// TestRunWaitsForRegister: the system can start a task before main
// registered it.
func TestRunWaitsForRegister(t *testing.T) {
	done := make(chan error, 1)
	go func() { done <- run(context.Background(), "late") }()
	time.Sleep(30 * time.Millisecond)
	Register("late", func(ctx context.Context) error { return context.Canceled })
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Error(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run didn't find the late task")
	}
	old := timeout
	timeout = 50 * time.Millisecond
	defer func() { timeout = old }()
	if err := run(context.Background(), "never"); err != ErrUnknown {
		t.Error(err)
	}
}
