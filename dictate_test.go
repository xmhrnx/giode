package giode

import (
	"context"
	"errors"
	"testing"
)

// Without the dictate package, Dictate says what's missing.
func TestDictateNeedsPackage(t *testing.T) {
	var a *App
	if _, err := a.Dictate(context.Background(), ""); !errors.Is(err, ErrNotSupported) {
		t.Errorf("err = %v, want ErrNotSupported", err)
	}
}
