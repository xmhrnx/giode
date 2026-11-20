//go:build (linux && !android) || windows

package camera

import (
	"context"
	"testing"
)

// TestDevices lists cameras; CI runners have none, so it only checks
// listing works.
func TestDevices(t *testing.T) {
	ds, err := Devices(context.Background())
	if err != nil {
		t.Skipf("listing cameras: %v", err)
	}
	for _, d := range ds {
		if d.ID == "" {
			t.Errorf("a camera without an ID: %+v", d)
		}
	}
	t.Logf("%d cameras", len(ds))
}
