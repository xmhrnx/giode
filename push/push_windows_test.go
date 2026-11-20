package push

import (
	"context"
	"testing"
	"time"
)

func TestParseToast(t *testing.T) {
	m := parseToast(`<toast launch="chat=42"><visual><binding template="ToastGeneric"><text>Ana</text><text>Lunch?</text></binding></visual></toast>`)
	if m.Title != "Ana" || m.Body != "Lunch?" || m.Data["launch"] != "chat=42" {
		t.Errorf("%+v", m)
	}
}

// TestChannel asks for a WNS channel: without package identity (go test)
// it must fail with an error, not crash or hang.
func TestChannel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	uri, err := Token(ctx)
	t.Logf("uri %q, err %v", uri, err)
	if err == nil && uri == "" {
		t.Error("no URI and no error")
	}
}
