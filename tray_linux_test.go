//go:build linux && !android

package giode

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// watcher stands in for a desktop's StatusNotifierWatcher, recording
// the items that register.
type watcher struct {
	mu    sync.Mutex
	items []string
}

func (w *watcher) RegisterStatusNotifierItem(service string, sender dbus.Sender) *dbus.Error {
	w.mu.Lock()
	w.items = append(w.items, string(sender)+service)
	w.mu.Unlock()
	return nil
}

func (w *watcher) RegisterStatusNotifierHost(string) *dbus.Error { return nil }

// menuLayout is com.canonical.dbusmenu's layout node.
type menuLayout struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

// TestTray_StatusNotifier checks the Linux tray end to end over D-Bus:
// the item registers, shows the title and tooltip, and its menu runs
// the item's action on the UI goroutine. It needs a session bus (run
// under dbus-run-session).
func TestTray_StatusNotifier(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus; run under dbus-run-session")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Skip(err)
	}
	defer conn.Close()
	w := &watcher{}
	if err := conn.Export(w, "/StatusNotifierWatcher", "org.kde.StatusNotifierWatcher"); err != nil {
		t.Fatal(err)
	}
	if r, err := conn.RequestName("org.kde.StatusNotifierWatcher", dbus.NameFlagDoNotQueue); err != nil || r != dbus.RequestNameReplyPrimaryOwner {
		t.Skipf("can't own the watcher name: %v %v", r, err)
	}

	a := New(Config{})
	clicked := 0
	a.Tray().
		SetIcon(testPNG(t)).
		SetTitle("3 unread").
		SetTooltip("Pocket Agent").
		SetMenu(
			TrayAction("Open", func() { clicked++ }),
			TraySeparator(),
			TrayCheckbox("Do not disturb", true, nil),
		)

	var item string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		w.mu.Lock()
		if len(w.items) > 0 {
			item = w.items[0]
		}
		w.mu.Unlock()
		if item != "" {
			break
		}
	}
	if item == "" {
		t.Fatal("the tray item never registered")
	}
	// The item registered with its path: the bus name is the sender.
	bus := item[:len(item)-len("/StatusNotifierItem")]
	obj := conn.Object(bus, "/StatusNotifierItem")

	// Title and tooltip are applied once the item is ready.
	var title string
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline) && title != "3 unread"; time.Sleep(50 * time.Millisecond) {
		v, err := obj.GetProperty("org.kde.StatusNotifierItem.Title")
		if err == nil {
			title, _ = v.Value().(string)
		}
	}
	if title != "3 unread" {
		t.Errorf("title %q, want %q", title, "3 unread")
	}

	menu := conn.Object(bus, "/StatusNotifierItem/menu")
	var labels map[string]int32
	// The labels come from GetGroupProperties: systray's first
	// GetLayout starts a goroutine that races with its own read of the
	// menu version (in the library), which -race reports.
	ids := make([]int32, 64)
	for i := range ids {
		ids[i] = int32(i)
	}
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		var props []struct {
			ID    int32
			Props map[string]dbus.Variant
		}
		if err := menu.Call("com.canonical.dbusmenu.GetGroupProperties", 0, ids, []string{}).Store(&props); err != nil {
			t.Fatal(err)
		}
		labels = map[string]int32{}
		for _, p := range props {
			if l, ok := p.Props["label"]; ok {
				if s, ok := l.Value().(string); ok {
					labels[s] = p.ID
				}
			}
		}
		if len(labels) >= 2 {
			break
		}
	}
	id, ok := labels["Open"]
	if !ok {
		t.Fatalf("menu labels %v, want Open", labels)
	}
	if _, ok := labels["Do not disturb"]; !ok {
		t.Errorf("menu labels %v, want Do not disturb", labels)
	}
	if err := menu.Call("com.canonical.dbusmenu.Event", 0, id, "clicked", dbus.MakeVariant(""), uint32(0)).Err; err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline) && clicked == 0; time.Sleep(20 * time.Millisecond) {
		a.runUICalls()
	}
	if clicked != 1 {
		t.Errorf("Open ran %d times, want 1", clicked)
	}
	a.Tray().Remove()
}
