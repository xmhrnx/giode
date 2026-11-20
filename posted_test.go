package giode

import "testing"

// TestNestedPost: a call posted by a posted call runs in the same pass,
// as its wakeup can be lost.
func TestNestedPost(t *testing.T) {
	a := &App{}
	ran := 0
	a.uiCalls = append(a.uiCalls, func() {
		ran++
		a.uiMu.Lock()
		a.uiCalls = append(a.uiCalls, func() { ran++ })
		a.uiMu.Unlock()
	})
	if !a.runUICalls() || ran != 2 {
		t.Errorf("ran %d calls", ran)
	}
	if a.runUICalls() {
		t.Error("calls left over")
	}
}
