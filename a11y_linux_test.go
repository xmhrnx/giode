//go:build linux && !android

package giode

import (
	"bufio"
	"image"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/xmhrnx/giode/elements"
)

// privateBus starts a D-Bus daemon for the test and returns its address.
func privateBus(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("dbus-daemon"); err != nil {
		t.Skip("no dbus-daemon")
	}
	cmd := exec.Command("dbus-daemon", "--session", "--nofork", "--print-address")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skip("dbus-daemon:", err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(line)
}

func dialBus(t *testing.T, addr string) *dbus.Conn {
	t.Helper()
	c, err := dbus.Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Auth(nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Hello(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func atspiTestTree() *a11yTree {
	tree := newA11yTree()
	tree.update([]elements.SemanticNode{
		{ID: "name", Kind: elements.SemanticInput, Label: "Name", Value: "Ada Lovelace", Required: true, Bounds: image.Rect(10, 10, 210, 40), Focused: true, Target: &atspiTextTarget{}},
		{ID: "dark", Kind: elements.SemanticSwitch, Label: "Dark mode", Checked: elements.Checked, Bounds: image.Rect(10, 50, 110, 80)},
		{ID: "vol", Kind: elements.SemanticSlider, Label: "Volume", Min: 0, Max: 1, Now: 0.25, Bounds: image.Rect(10, 90, 210, 110)},
	})
	return tree
}

type atspiTextTarget struct{ text string }

func (c *atspiTextTarget) SetText(s string) { c.text = s }

func TestATSPIObjects(t *testing.T) {
	addr := privateBus(t)
	server := dialBus(t, addr)
	tree := atspiTestTree()
	b := newATSPIBridge(nil, tree, server)
	if err := b.export(); err != nil {
		t.Fatal(err)
	}
	client := dialBus(t, addr)
	obj := func(p dbus.ObjectPath) dbus.BusObject { return client.Object(server.Names()[0], p) }

	var kids []atspiRef
	if err := obj(atspiRootPath).Call("org.a11y.atspi.Accessible.GetChildren", 0).Store(&kids); err != nil {
		t.Fatal(err)
	}
	if len(kids) != 1 || kids[0].Path != atspiFramePath {
		t.Fatalf("root children = %v", kids)
	}
	if err := obj(atspiFramePath).Call("org.a11y.atspi.Accessible.GetChildren", 0).Store(&kids); err != nil {
		t.Fatal(err)
	}
	if len(kids) != 3 {
		t.Fatalf("frame children = %v", kids)
	}
	if !b.on.Load() {
		t.Error("asking for the frame's children didn't turn recording on")
	}
	entry := obj(kids[0].Path)
	var role uint32
	entry.Call("org.a11y.atspi.Accessible.GetRole", 0).Store(&role)
	if role != roleEntry {
		t.Errorf("entry role = %d", role)
	}
	name, err := entry.GetProperty("org.a11y.atspi.Accessible.Name")
	if err != nil || name.Value() != "Name" {
		t.Errorf("entry name = %v, %v", name, err)
	}
	var text string
	entry.Call("org.a11y.atspi.Text.GetText", 0, int32(0), int32(-1)).Store(&text)
	if text != "Ada Lovelace" {
		t.Errorf("entry text = %q", text)
	}
	var word string
	var ws, we int32
	entry.Call("org.a11y.atspi.Text.GetStringAtOffset", 0, int32(5), uint32(1)).Store(&word, &ws, &we)
	if word != "Lovelace" || ws != 4 {
		t.Errorf("word at 5 = %q from %d", word, ws)
	}
	var states []uint32
	entry.Call("org.a11y.atspi.Accessible.GetState", 0).Store(&states)
	has := func(bit int) bool { return states[bit/32]&(1<<(bit%32)) != 0 }
	if !has(stateFocused) || !has(stateEditable) || !has(stateRequired) || !has(stateEnabled) {
		t.Errorf("entry states = %b", states)
	}
	var ifaces []string
	entry.Call("org.a11y.atspi.Accessible.GetInterfaces", 0).Store(&ifaces)
	if !strings.Contains(strings.Join(ifaces, " "), "EditableText") {
		t.Errorf("entry interfaces = %v", ifaces)
	}

	sw := obj(kids[1].Path)
	sw.Call("org.a11y.atspi.Accessible.GetState", 0).Store(&states)
	if !has(stateChecked) || !has(stateCheckable) {
		t.Errorf("switch states = %b", states)
	}
	var parent atspiRef
	v, _ := sw.GetProperty("org.a11y.atspi.Accessible.Parent")
	v.Store(&parent)
	if parent.Path != atspiFramePath {
		t.Errorf("switch parent = %v", parent)
	}

	vol := obj(kids[2].Path)
	cur, _ := vol.GetProperty("org.a11y.atspi.Value.CurrentValue")
	if cur.Value() != 0.25 {
		t.Errorf("slider value = %v", cur)
	}
	var at atspiRef
	obj(atspiFramePath).Call("org.a11y.atspi.Component.GetAccessibleAtPoint", 0, int32(50), int32(60), uint32(atspiCoordWindow)).Store(&at)
	if at.Path != kids[1].Path {
		t.Errorf("at (50, 60) = %v, want the switch", at)
	}
	var ext atspiRect
	vol.Call("org.a11y.atspi.Component.GetExtents", 0, uint32(atspiCoordWindow)).Store(&ext)
	if ext != (atspiRect{10, 90, 200, 20}) {
		t.Errorf("slider extents = %v", ext)
	}

	// Focus moving emits state-changed:focused.
	if err := client.AddMatchSignal(dbus.WithMatchInterface("org.a11y.atspi.Event.Object")); err != nil {
		t.Fatal(err)
	}
	sig := make(chan *dbus.Signal, 16)
	client.Signal(sig)
	tree.update([]elements.SemanticNode{
		{ID: "name", Kind: elements.SemanticInput, Label: "Name", Value: "Ada Lovelace", Bounds: image.Rect(10, 10, 210, 40)},
		{ID: "dark", Kind: elements.SemanticSwitch, Label: "Dark mode", Checked: elements.Checked, Bounds: image.Rect(10, 50, 110, 80), Focused: true},
		{ID: "vol", Kind: elements.SemanticSlider, Label: "Volume", Min: 0, Max: 1, Now: 0.25, Bounds: image.Rect(10, 90, 210, 110)},
	})
	b.focus = tree.number("name")
	b.events()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case s := <-sig:
			if s.Name == "org.a11y.atspi.Event.Object.StateChanged" && s.Path == kids[1].Path && s.Body[0] == "focused" && s.Body[1] == int32(1) {
				return
			}
		case <-deadline:
			t.Fatal("no focus event for the switch")
		}
	}
}

// TestATSPIRegistry embeds a tree in a running AT-SPI registry and keeps
// it there for GIODE_ATSPI_HOLD, for inspecting it with a screen reader
// or pyatspi. It runs only with GIODE_ATSPI_E2E set.
func TestATSPIRegistry(t *testing.T) {
	if os.Getenv("GIODE_ATSPI_E2E") == "" {
		t.Skip("set GIODE_ATSPI_E2E to run against the session's AT-SPI registry")
	}
	addr, err := atspiAddress()
	if err != nil {
		t.Fatal(err)
	}
	b, err := connectATSPI(nil, atspiTestTree(), addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("embedded as %s under %v", b.conn.Names()[0], b.desktop)
	hold, _ := time.ParseDuration(os.Getenv("GIODE_ATSPI_HOLD"))
	time.Sleep(hold)
}
