//go:build linux && !android

package a11y

import (
	"errors"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/godbus/dbus/v5"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/internal/platform"
	"github.com/xmhrnx/giode/internal/platform/window"
)

// Screen readers on Linux (Orca) through AT-SPI: the app connects to the
// accessibility bus, exports an application object, a frame for its
// window and an object per tree node (a11y_tree.go), and embeds itself
// in the registry. Objects answer the Accessible, Component, Action,
// Value, Text and EditableText interfaces from the tree; focus, state,
// value changes and announcements go out as AT-SPI events after each
// frame. Tree recording starts when a screen reader is enabled or a
// client first asks for the window's contents.

const (
	atspiBase      = dbus.ObjectPath("/org/a11y/atspi/accessible")
	atspiRootPath  = dbus.ObjectPath("/org/a11y/atspi/accessible/root")
	atspiFramePath = dbus.ObjectPath("/org/a11y/atspi/accessible/frame")
	atspiNullPath  = dbus.ObjectPath("/org/a11y/atspi/null")

	atspiCoordScreen = 0
	atspiCoordWindow = 1
	atspiCoordParent = 2
)

// AT-SPI roles (atspi-constants.h).
const (
	roleAlert        = 2
	roleCheckBox     = 7
	roleColumnHeader = 10
	roleComboBox     = 11
	roleDialog       = 16
	roleFrame        = 23
	roleImage        = 27
	roleLabel        = 29
	roleList         = 31
	roleListItem     = 32
	roleMenu         = 33
	roleMenuBar      = 34
	roleMenuItem     = 35
	rolePageTab      = 37
	rolePageTabList  = 38
	rolePanel        = 39
	roleProgressBar  = 42
	rolePushButton   = 43
	roleRadioButton  = 44
	roleScrollPane   = 49
	roleSlider       = 51
	roleStatusBar    = 54
	roleTable        = 55
	roleTableCell    = 56
	roleToggleButton = 62
	roleToolBar      = 63
	roleTree         = 65
	roleApplication  = 75
	roleEntry        = 79
	roleHeading      = 83
	roleLink         = 88
	roleTableRow     = 90
	roleTreeItem     = 91
	roleGrouping     = 99
	roleNotification = 101
	roleLandmark     = 110
)

// AT-SPI states, as bit numbers.
const (
	stateActive        = 1
	stateBusy          = 3
	stateChecked       = 4
	stateCollapsed     = 5
	stateEditable      = 7
	stateEnabled       = 8
	stateExpandable    = 9
	stateExpanded      = 10
	stateFocusable     = 11
	stateFocused       = 12
	stateModal         = 16
	stateSelectable    = 22
	stateSelected      = 23
	stateSensitive     = 24
	stateShowing       = 25
	stateSingleLine    = 26
	stateVisible       = 30
	stateIndeterminate = 32
	stateRequired      = 33
	stateInvalidEntry  = 36
	stateCheckable     = 41
	stateReadOnly      = 43
)

type atspiRef struct {
	Name string
	Path dbus.ObjectPath
}

type atspiRelation struct {
	Type    uint32
	Targets []atspiRef
}

type atspiRect struct{ X, Y, W, H int32 }

// atspiBridge is the app's AT-SPI objects on the accessibility bus.
type atspiBridge struct {
	a    Host
	t    *Tree
	conn *dbus.Conn
	// desktop is the registry's root, the application's parent.
	desktop atspiRef
	appID   atomic.Int32
	on      atomic.Bool
	focus   int32
	// caret is the focused node's caret and anchor last frame.
	caret [2]int
	text  string
	mu    sync.Mutex
	// origin returns the window's client area on the screen; ok is false
	// where it isn't known (Wayland).
	origin func() (x, y int, ok bool)
}

// Install sets the tree up for the platform's bridge.
func Install(a Host) *Tree {
	t := NewTree()
	setTree(a, t)
	if os.Getenv("NO_AT_BRIDGE") == "1" {
		return t
	}
	var bridge atomic.Pointer[atspiBridge]
	t.Notify = func() {
		if b := bridge.Load(); b != nil {
			b.events()
		} else {
			t.takeChanges()
		}
	}
	go func() {
		b, err := startATSPI(a, t)
		if err != nil {
			a.Logf(platform.Debug, "giode: no AT-SPI bridge: %v", err)
			return
		}
		bridge.Store(b)
	}()
	return t
}

// atspiAddress returns the accessibility bus's address.
func atspiAddress() (string, error) {
	if addr := os.Getenv("AT_SPI_BUS_ADDRESS"); addr != "" {
		return addr, nil
	}
	if !sessionBusKnown() {
		return "", errors.New("no session bus")
	}
	session, err := dbus.SessionBus()
	if err != nil {
		return "", err
	}
	var addr string
	err = session.Object("org.a11y.Bus", "/org/a11y/bus").Call("org.a11y.Bus.GetAddress", 0).Store(&addr)
	return addr, err
}

// sessionBusKnown reports whether there's a session bus to ask, without
// godbus starting one (dbus-launch) where there isn't.
func sessionBusKnown() bool {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return true
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		if _, err := os.Stat(dir + "/bus"); err == nil {
			return true
		}
	}
	return false
}

// screenReaderOn reports whether the desktop says a screen reader runs.
func screenReaderOn() bool {
	if !sessionBusKnown() {
		return false
	}
	session, err := dbus.SessionBus()
	if err != nil {
		return false
	}
	v, err := session.Object("org.a11y.Bus", "/org/a11y/bus").GetProperty("org.a11y.Status.ScreenReaderEnabled")
	if err != nil {
		return false
	}
	on, _ := v.Value().(bool)
	return on
}

func startATSPI(a Host, t *Tree) (*atspiBridge, error) {
	addr, err := atspiAddress()
	if err != nil {
		return nil, err
	}
	b, err := connectATSPI(a, t, addr)
	if err != nil {
		return nil, err
	}
	if screenReaderOn() {
		b.enable()
	}
	return b, nil
}

// connectATSPI connects to the accessibility bus at addr, exports the
// objects and embeds the application in the registry.
func connectATSPI(a Host, t *Tree, addr string) (*atspiBridge, error) {
	conn, err := dbus.Dial(addr)
	if err != nil {
		return nil, err
	}
	if err := conn.Auth(nil); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.Hello(); err != nil {
		conn.Close()
		return nil, err
	}
	b := newATSPIBridge(a, t, conn)
	if a != nil {
		b.origin = func() (int, int, bool) { return window.ClientOrigin(a.NativeView()) }
	}
	if err := b.export(); err != nil {
		conn.Close()
		return nil, err
	}
	err = conn.Object("org.a11y.atspi.Registry", atspiRootPath).
		Call("org.a11y.atspi.Socket.Embed", 0, atspiRef{conn.Names()[0], atspiRootPath}).Store(&b.desktop)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return b, nil
}

func newATSPIBridge(a Host, t *Tree, conn *dbus.Conn) *atspiBridge {
	b := &atspiBridge{a: a, t: t, conn: conn, desktop: atspiRef{"", atspiNullPath}}
	b.origin = func() (int, int, bool) { return 0, 0, false }
	return b
}

func (b *atspiBridge) export() error {
	for iface, v := range map[string]any{
		"org.a11y.atspi.Accessible":       atspiAccessible{b},
		"org.a11y.atspi.Component":        atspiComponent{b},
		"org.a11y.atspi.Action":           atspiAction{b},
		"org.a11y.atspi.Text":             atspiText{b},
		"org.a11y.atspi.EditableText":     atspiEditable{b},
		"org.a11y.atspi.Application":      atspiApplication{b},
		"org.freedesktop.DBus.Properties": atspiProps{b},
	} {
		if err := b.conn.ExportSubtree(v, atspiBase, iface); err != nil {
			return err
		}
	}
	// An empty cache: clients then ask the objects themselves. Exported
	// last, at a path of its own: adding the path takes godbus's handler
	// lock, which orders the subtree exports above before any call that
	// looks them up.
	return b.conn.Export(atspiCache{}, "/org/a11y/atspi/cache", "org.a11y.atspi.Cache")
}

// enable starts recording the tree, once a client wants it.
func (b *atspiBridge) enable() {
	if b.on.Swap(true) {
		return
	}
	if b.a != nil && !elements.SemanticsWanted() {
		elements.SetSemanticsWanted(true)
		b.a.Invalidate()
	}
	b.emit(atspiFramePath, "org.a11y.atspi.Event.Window.Activate", "", 0, 0, "")
}

// The objects: the application, its frame, and node k.
type atspiObj struct {
	root, frame bool
	k           int32
}

func atspiPath(o atspiObj) dbus.ObjectPath {
	switch {
	case o.root:
		return atspiRootPath
	case o.frame || o.k == 0:
		return atspiFramePath
	}
	return dbus.ObjectPath(fmt.Sprintf("%s/n%d", atspiBase, o.k))
}

func objOf(msg dbus.Message) (atspiObj, bool) {
	p, _ := msg.Headers[dbus.FieldPath].Value().(dbus.ObjectPath)
	switch p {
	case atspiRootPath:
		return atspiObj{root: true}, true
	case atspiFramePath:
		return atspiObj{frame: true}, true
	}
	s, ok := strings.CutPrefix(string(p), string(atspiBase)+"/n")
	if !ok {
		return atspiObj{}, false
	}
	k, err := strconv.ParseInt(s, 10, 32)
	if err != nil || k <= 0 {
		return atspiObj{}, false
	}
	return atspiObj{k: int32(k)}, true
}

func (b *atspiBridge) ref(o atspiObj) atspiRef {
	return atspiRef{b.conn.Names()[0], atspiPath(o)}
}

var errUnknownObject = dbus.NewError("org.freedesktop.DBus.Error.UnknownObject", []any{"no such object"})

// resolve returns the message's object and, for a node, the node.
func (b *atspiBridge) resolve(msg dbus.Message) (atspiObj, elements.SemanticNode, *dbus.Error) {
	o, ok := objOf(msg)
	if !ok {
		return o, elements.SemanticNode{}, errUnknownObject
	}
	if o.root || o.frame {
		return o, elements.SemanticNode{}, nil
	}
	n, _, _, ok := b.t.node(o.k)
	if !ok {
		return o, n, errUnknownObject
	}
	return o, n, nil
}

func (b *atspiBridge) children(o atspiObj) []atspiRef {
	if o.root {
		return []atspiRef{b.ref(atspiObj{frame: true})}
	}
	k := o.k
	if o.frame {
		k = 0
		b.enable()
	}
	_, _, kids, _ := b.t.node(k)
	out := make([]atspiRef, len(kids))
	for i, c := range kids {
		out[i] = b.ref(atspiObj{k: c})
	}
	return out
}

func (b *atspiBridge) parent(o atspiObj) atspiRef {
	switch {
	case o.root:
		return b.desktop
	case o.frame:
		return b.ref(atspiObj{root: true})
	}
	_, p, _, _ := b.t.node(o.k)
	if p == 0 {
		return b.ref(atspiObj{frame: true})
	}
	return b.ref(atspiObj{k: p})
}

func atspiRole(n elements.SemanticNode) uint32 {
	switch n.Kind {
	case elements.SemanticButton:
		return rolePushButton
	case elements.SemanticLink:
		return roleLink
	case elements.SemanticCheckbox:
		return roleCheckBox
	case elements.SemanticSwitch:
		return roleToggleButton
	case elements.SemanticRadio:
		return roleRadioButton
	case elements.SemanticInput:
		return roleEntry
	case elements.SemanticSlider:
		return roleSlider
	case elements.SemanticProgress:
		return roleProgressBar
	case elements.SemanticSelect, elements.SemanticComboBox:
		return roleComboBox
	case elements.SemanticTab:
		return rolePageTab
	case elements.SemanticTabList:
		return rolePageTabList
	case elements.SemanticTabPanel, elements.SemanticGroup:
		return rolePanel
	case elements.SemanticImage:
		return roleImage
	case elements.SemanticScroll:
		return roleScrollPane
	case elements.SemanticList:
		return roleList
	case elements.SemanticListItem:
		return roleListItem
	case elements.SemanticTable:
		return roleTable
	case elements.SemanticRow:
		return roleTableRow
	case elements.SemanticCell:
		return roleTableCell
	case elements.SemanticColumnHeader:
		return roleColumnHeader
	case elements.SemanticTree:
		return roleTree
	case elements.SemanticTreeItem:
		return roleTreeItem
	case elements.SemanticMenu:
		return roleMenu
	case elements.SemanticMenuBar:
		return roleMenuBar
	case elements.SemanticMenuItem:
		return roleMenuItem
	case elements.SemanticToolbar:
		return roleToolBar
	case elements.SemanticHeading:
		return roleHeading
	case elements.SemanticDialog:
		return roleDialog
	case elements.SemanticAlert:
		return roleNotification
	case elements.SemanticStatus:
		return roleStatusBar
	case elements.SemanticRegion:
		return roleLandmark
	}
	return roleLabel
}

var atspiRoleNames = map[uint32]string{
	roleAlert: "alert", roleCheckBox: "check box", roleColumnHeader: "column header", roleComboBox: "combo box",
	roleDialog: "dialog", roleFrame: "frame", roleImage: "image", roleLabel: "label", roleList: "list",
	roleListItem: "list item", roleMenu: "menu", roleMenuBar: "menu bar", roleMenuItem: "menu item",
	rolePageTab: "page tab", rolePageTabList: "page tab list", rolePanel: "panel", roleProgressBar: "progress bar",
	rolePushButton: "push button", roleRadioButton: "radio button", roleScrollPane: "scroll pane", roleSlider: "slider",
	roleStatusBar: "status bar", roleTable: "table", roleTableCell: "table cell", roleToggleButton: "toggle button",
	roleToolBar: "tool bar", roleTree: "tree", roleApplication: "application", roleEntry: "entry", roleHeading: "heading",
	roleLink: "link", roleTableRow: "table row", roleTreeItem: "tree item", roleGrouping: "grouping",
	roleNotification: "notification", roleLandmark: "landmark",
}

func (b *atspiBridge) role(o atspiObj, n elements.SemanticNode) uint32 {
	switch {
	case o.root:
		return roleApplication
	case o.frame:
		return roleFrame
	}
	return atspiRole(n)
}

func (b *atspiBridge) name(o atspiObj, n elements.SemanticNode) string {
	switch {
	case o.root, o.frame:
		if b.a != nil {
			return b.a.Title()
		}
		return ""
	}
	if n.Label != "" {
		return n.Label
	}
	switch n.Kind {
	case elements.SemanticInput, elements.SemanticSlider, elements.SemanticProgress, elements.SemanticSelect, elements.SemanticComboBox:
		return ""
	}
	return n.Value
}

func atspiStates(n elements.SemanticNode, focusable bool) []uint32 {
	var s [2]uint32
	set := func(bit int) { s[bit/32] |= 1 << (bit % 32) }
	set(stateVisible)
	set(stateShowing)
	if !n.Disabled {
		set(stateEnabled)
		set(stateSensitive)
	}
	_, _, setText, focus := a11yActions(n)
	if focus || (focusable && actionable(n) && !n.Disabled) {
		set(stateFocusable)
	}
	if n.Focused {
		set(stateFocused)
	}
	switch n.Checked {
	case elements.Checked:
		set(stateCheckable)
		set(stateChecked)
	case elements.Unchecked:
		set(stateCheckable)
	case elements.Mixed:
		set(stateCheckable)
		set(stateIndeterminate)
	}
	switch n.Kind {
	case elements.SemanticRadio, elements.SemanticTab, elements.SemanticTreeItem, elements.SemanticListItem, elements.SemanticRow:
		set(stateSelectable)
	}
	if n.Selected {
		set(stateSelected)
		if n.Kind == elements.SemanticRadio {
			set(stateChecked)
		}
	}
	switch n.Expanded {
	case elements.Expanded:
		set(stateExpandable)
		set(stateExpanded)
	case elements.Collapsed:
		set(stateExpandable)
		set(stateCollapsed)
	}
	if n.Kind == elements.SemanticInput {
		set(stateSingleLine)
		if setText {
			set(stateEditable)
		} else {
			set(stateReadOnly)
		}
	}
	if n.Required {
		set(stateRequired)
	}
	if n.Invalid {
		set(stateInvalidEntry)
	}
	if n.Busy {
		set(stateBusy)
	}
	if n.Kind == elements.SemanticDialog {
		set(stateModal)
	}
	return s[:]
}

func (b *atspiBridge) states(o atspiObj, n elements.SemanticNode) []uint32 {
	if o.root || o.frame {
		var s [2]uint32
		for _, bit := range []int{stateVisible, stateShowing, stateEnabled, stateSensitive} {
			s[bit/32] |= 1 << (bit % 32)
		}
		if o.frame && (b.a == nil || b.a.Focused()) {
			s[0] |= 1 << stateActive
		}
		return s[:]
	}
	return atspiStates(n, b.t.isFocusable(n, o.k))
}

// extents returns a node's rectangle in coords.
func (b *atspiBridge) extents(o atspiObj, n elements.SemanticNode, coords uint32) atspiRect {
	r := n.Bounds
	if o.root || o.frame {
		r = image.Rectangle{}
		if b.a != nil {
			if sz, ok := b.a.Size(); ok {
				r = image.Rectangle{Max: sz}
			}
		}
	}
	switch coords {
	case atspiCoordScreen:
		if x, y, ok := b.origin(); ok {
			r = r.Add(image.Pt(x, y))
		}
	case atspiCoordParent:
		if !o.root && !o.frame {
			if _, p, _, _ := b.t.node(o.k); p != 0 {
				pn, _, _, _ := b.t.node(p)
				r = r.Sub(pn.Bounds.Min)
			}
		}
	}
	return atspiRect{int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy())}
}

// toWindow turns a point in coords into window pixels.
func (b *atspiBridge) toWindow(x, y int32, coords uint32) image.Point {
	p := image.Pt(int(x), int(y))
	if coords == atspiCoordScreen {
		if ox, oy, ok := b.origin(); ok {
			p = p.Sub(image.Pt(ox, oy))
		}
	}
	return p
}

func (b *atspiBridge) interfaces(o atspiObj, n elements.SemanticNode) []string {
	out := []string{"org.a11y.atspi.Accessible", "org.a11y.atspi.Component"}
	if o.root {
		return []string{"org.a11y.atspi.Accessible", "org.a11y.atspi.Application"}
	}
	if o.frame {
		return out
	}
	if len(atspiActions(n)) > 0 {
		out = append(out, "org.a11y.atspi.Action")
	}
	if atspiHasValue(n) {
		out = append(out, "org.a11y.atspi.Value")
	}
	if atspiHasText(n) {
		out = append(out, "org.a11y.atspi.Text")
	}
	if _, _, setText, _ := a11yActions(n); setText && n.Kind == elements.SemanticInput {
		out = append(out, "org.a11y.atspi.EditableText")
	}
	return out
}

func atspiHasValue(n elements.SemanticNode) bool {
	return n.Kind == elements.SemanticSlider || (n.Kind == elements.SemanticProgress && !n.Busy)
}

func atspiHasText(n elements.SemanticNode) bool {
	return n.Kind == elements.SemanticInput || textual(n) || n.Kind == elements.SemanticHeading
}

// atspiTextOf is the text a node's Text interface holds.
func atspiTextOf(n elements.SemanticNode) string {
	if n.Kind == elements.SemanticInput {
		return n.Value
	}
	return a11yText(n)
}

// atspiActions are the actions a node takes, by AT-SPI name.
func atspiActions(n elements.SemanticNode) []string {
	activate, _, _, _ := a11yActions(n)
	var out []string
	if activate {
		out = append(out, "click")
	}
	switch n.Expanded {
	case elements.Collapsed:
		out = append(out, "expand")
	case elements.Expanded:
		out = append(out, "collapse")
	}
	return out
}

func atspiDo(name string) a11yAction {
	switch name {
	case "click":
		return a11yActivate
	case "expand":
		return a11yExpand
	case "collapse":
		return a11yCollapse
	}
	return 0
}

// --- org.a11y.atspi.Accessible ---

type atspiAccessible struct{ b *atspiBridge }

func (x atspiAccessible) GetChildAtIndex(msg dbus.Message, i int32) (atspiRef, *dbus.Error) {
	o, _, err := x.b.resolve(msg)
	if err != nil {
		return atspiRef{}, err
	}
	kids := x.b.children(o)
	if i < 0 || int(i) >= len(kids) {
		return atspiRef{"", atspiNullPath}, nil
	}
	return kids[i], nil
}

func (x atspiAccessible) GetChildren(msg dbus.Message) ([]atspiRef, *dbus.Error) {
	o, _, err := x.b.resolve(msg)
	if err != nil {
		return nil, err
	}
	return x.b.children(o), nil
}

func (x atspiAccessible) GetIndexInParent(msg dbus.Message) (int32, *dbus.Error) {
	o, _, err := x.b.resolve(msg)
	if err != nil {
		return -1, err
	}
	switch {
	case o.root:
		return -1, nil
	case o.frame:
		return 0, nil
	}
	_, p, _, _ := x.b.t.node(o.k)
	_, _, siblings, _ := x.b.t.node(p)
	for i, s := range siblings {
		if s == o.k {
			return int32(i), nil
		}
	}
	return -1, nil
}

func (x atspiAccessible) GetRelationSet(msg dbus.Message) ([]atspiRelation, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil || o.root || o.frame {
		return []atspiRelation{}, err
	}
	out := []atspiRelation{}
	for _, rel := range []struct {
		kind uint32
		ids  []string
	}{{2, n.LabelledBy}, {18, n.DescribedBy}, {3, n.Controls}} {
		var targets []atspiRef
		for _, id := range rel.ids {
			if k := x.b.t.number(id); k != 0 {
				targets = append(targets, x.b.ref(atspiObj{k: k}))
			}
		}
		if len(targets) > 0 {
			out = append(out, atspiRelation{rel.kind, targets})
		}
	}
	return out, nil
}

func (x atspiAccessible) GetRole(msg dbus.Message) (uint32, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return 0, err
	}
	return x.b.role(o, n), nil
}

func (x atspiAccessible) GetRoleName(msg dbus.Message) (string, *dbus.Error) {
	r, err := x.GetRole(msg)
	return atspiRoleNames[r], err
}

func (x atspiAccessible) GetLocalizedRoleName(msg dbus.Message) (string, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return "", err
	}
	if n.Kind == elements.SemanticSwitch {
		return "switch", nil
	}
	if n.Kind == elements.SemanticRegion {
		return "region", nil
	}
	return atspiRoleNames[x.b.role(o, n)], nil
}

func (x atspiAccessible) GetState(msg dbus.Message) ([]uint32, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return []uint32{0, 0}, err
	}
	return x.b.states(o, n), nil
}

func (x atspiAccessible) GetAttributes(msg dbus.Message) (map[string]string, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return map[string]string{}, err
	}
	attrs := map[string]string{}
	if o.root || o.frame {
		attrs["toolkit"] = "Giode"
		return attrs, nil
	}
	if n.Kind == elements.SemanticHeading || n.Level > 0 {
		attrs["level"] = strconv.Itoa(max(n.Level, 1))
	}
	if n.Pos > 0 {
		attrs["posinset"] = strconv.Itoa(n.Pos)
	}
	if n.Count > 0 {
		attrs["setsize"] = strconv.Itoa(n.Count)
	}
	switch {
	case n.Live == elements.LiveAssertive || n.Kind == elements.SemanticAlert:
		attrs["live"] = "assertive"
	case n.Live == elements.LivePolite || n.Kind == elements.SemanticStatus:
		attrs["live"] = "polite"
	}
	if n.Kind == elements.SemanticRegion {
		attrs["xml-roles"] = "region"
	}
	if n.ID != "" {
		attrs["id"] = n.ID
	}
	return attrs, nil
}

func (x atspiAccessible) GetApplication(msg dbus.Message) (atspiRef, *dbus.Error) {
	return x.b.ref(atspiObj{root: true}), nil
}

func (x atspiAccessible) GetInterfaces(msg dbus.Message) ([]string, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return nil, err
	}
	return x.b.interfaces(o, n), nil
}

// --- org.a11y.atspi.Component ---

type atspiComponent struct{ b *atspiBridge }

func (x atspiComponent) Contains(msg dbus.Message, px, py int32, coords uint32) (bool, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return false, err
	}
	r := x.b.extents(o, n, coords)
	return px >= r.X && py >= r.Y && px < r.X+r.W && py < r.Y+r.H, nil
}

func (x atspiComponent) GetAccessibleAtPoint(msg dbus.Message, px, py int32, coords uint32) (atspiRef, *dbus.Error) {
	if _, _, err := x.b.resolve(msg); err != nil {
		return atspiRef{}, err
	}
	if k := x.b.t.at(x.b.toWindow(px, py, coords)); k != 0 {
		return x.b.ref(atspiObj{k: k}), nil
	}
	return atspiRef{"", atspiNullPath}, nil
}

func (x atspiComponent) GetExtents(msg dbus.Message, coords uint32) (atspiRect, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return atspiRect{}, err
	}
	return x.b.extents(o, n, coords), nil
}

func (x atspiComponent) GetPosition(msg dbus.Message, coords uint32) (int32, int32, *dbus.Error) {
	r, err := x.GetExtents(msg, coords)
	return r.X, r.Y, err
}

func (x atspiComponent) GetSize(msg dbus.Message) (int32, int32, *dbus.Error) {
	r, err := x.GetExtents(msg, atspiCoordWindow)
	return r.W, r.H, err
}

func (x atspiComponent) GetLayer(msg dbus.Message) (uint32, *dbus.Error) {
	o, _, err := x.b.resolve(msg)
	if o.frame {
		return 7, err // a window
	}
	return 3, err // a widget
}

func (x atspiComponent) GetMDIZOrder(msg dbus.Message) (int16, *dbus.Error) { return 0, nil }
func (x atspiComponent) GetAlpha(msg dbus.Message) (float64, *dbus.Error)   { return 1, nil }

func (x atspiComponent) GrabFocus(msg dbus.Message) (bool, *dbus.Error) {
	o, _, err := x.b.resolve(msg)
	if err != nil || o.root || o.frame || x.b.a == nil {
		return false, err
	}
	return x.b.t.perform(x.b.a, o.k, a11yFocus, ""), nil
}

func (x atspiComponent) ScrollTo(msg dbus.Message, kind uint32) (bool, *dbus.Error) {
	return false, nil
}

func (x atspiComponent) SetExtents(msg dbus.Message, px, py, w, h int32, coords uint32) (bool, *dbus.Error) {
	return false, nil
}

func (x atspiComponent) SetPosition(msg dbus.Message, px, py int32, coords uint32) (bool, *dbus.Error) {
	return false, nil
}

func (x atspiComponent) SetSize(msg dbus.Message, w, h int32) (bool, *dbus.Error) { return false, nil }

// --- org.a11y.atspi.Action ---

type atspiAction struct{ b *atspiBridge }

func (x atspiAction) actions(msg dbus.Message) []string {
	_, n, err := x.b.resolve(msg)
	if err != nil {
		return nil
	}
	return atspiActions(n)
}

func (x atspiAction) name(msg dbus.Message, i int32) string {
	a := x.actions(msg)
	if i < 0 || int(i) >= len(a) {
		return ""
	}
	return a[i]
}

func (x atspiAction) GetName(msg dbus.Message, i int32) (string, *dbus.Error) {
	return x.name(msg, i), nil
}

func (x atspiAction) GetLocalizedName(msg dbus.Message, i int32) (string, *dbus.Error) {
	return x.name(msg, i), nil
}

func (x atspiAction) GetDescription(msg dbus.Message, i int32) (string, *dbus.Error) {
	return "", nil
}

func (x atspiAction) GetKeyBinding(msg dbus.Message, i int32) (string, *dbus.Error) {
	return "", nil
}

func (x atspiAction) GetActions(msg dbus.Message) ([]struct{ Name, Description, KeyBinding string }, *dbus.Error) {
	out := []struct{ Name, Description, KeyBinding string }{}
	for _, a := range x.actions(msg) {
		out = append(out, struct{ Name, Description, KeyBinding string }{a, "", ""})
	}
	return out, nil
}

func (x atspiAction) DoAction(msg dbus.Message, i int32) (bool, *dbus.Error) {
	o, _, err := x.b.resolve(msg)
	if err != nil || x.b.a == nil {
		return false, err
	}
	act := atspiDo(x.name(msg, i))
	if act == 0 {
		return false, nil
	}
	return x.b.t.perform(x.b.a, o.k, act, ""), nil
}

// --- org.a11y.atspi.Text ---

type atspiText struct{ b *atspiBridge }

func (x atspiText) text(msg dbus.Message) []rune {
	_, n, err := x.b.resolve(msg)
	if err != nil {
		return nil
	}
	return []rune(atspiTextOf(n))
}

func clampRange(start, end int32, n int) (int, int) {
	s, e := int(start), int(end)
	if e < 0 || e > n {
		e = n
	}
	s = max(0, min(s, e))
	return s, e
}

func (x atspiText) GetText(msg dbus.Message, start, end int32) (string, *dbus.Error) {
	t := x.text(msg)
	s, e := clampRange(start, end, len(t))
	return string(t[s:e]), nil
}

func (x atspiText) GetCharacterAtOffset(msg dbus.Message, off int32) (int32, *dbus.Error) {
	t := x.text(msg)
	if off < 0 || int(off) >= len(t) {
		return 0, nil
	}
	return t[off], nil
}

// atspiSpan returns the span of granularity g (0 char, 1 word, 2
// sentence, 3 line, 4 paragraph) around off in t.
func atspiSpan(t []rune, off int, g uint32) (int, int) {
	if len(t) == 0 {
		return 0, 0
	}
	off = max(0, min(off, len(t)-1))
	switch g {
	case 0:
		return off, off + 1
	case 1:
		s, e := off, off
		for s > 0 && !unicode.IsSpace(t[s-1]) {
			s--
		}
		for e < len(t) && !unicode.IsSpace(t[e]) {
			e++
		}
		for e < len(t) && unicode.IsSpace(t[e]) {
			e++
		}
		return s, e
	}
	// Single-line text: a sentence, a line and a paragraph are all of it.
	return 0, len(t)
}

func (x atspiText) GetStringAtOffset(msg dbus.Message, off int32, g uint32) (string, int32, int32, *dbus.Error) {
	t := x.text(msg)
	s, e := atspiSpan(t, int(off), g)
	return string(t[s:e]), int32(s), int32(e), nil
}

func (x atspiText) GetTextAtOffset(msg dbus.Message, off int32, boundary uint32) (string, int32, int32, *dbus.Error) {
	// Boundaries: 0 char, 1-2 word, 3-4 sentence, 5-6 line.
	g := map[uint32]uint32{0: 0, 1: 1, 2: 1, 3: 2, 4: 2, 5: 3, 6: 3}[boundary]
	return x.GetStringAtOffset(msg, off, g)
}

func (x atspiText) GetNSelections(msg dbus.Message) (int32, *dbus.Error) {
	_, n, err := x.b.resolve(msg)
	if err != nil {
		return 0, err
	}
	if start, end, ok := a11ySelection(n); ok && start != end {
		return 1, nil
	}
	return 0, nil
}

func (x atspiText) GetSelection(msg dbus.Message, i int32) (int32, int32, *dbus.Error) {
	_, n, err := x.b.resolve(msg)
	if err != nil {
		return 0, 0, err
	}
	if start, end, ok := a11ySelection(n); ok && i == 0 && start != end {
		return int32(start), int32(end), nil
	}
	return 0, 0, nil
}

// selectText selects [start, end) in the message's node.
func (x atspiText) selectText(msg dbus.Message, start, end int32) (bool, *dbus.Error) {
	o, _, err := x.b.resolve(msg)
	if err != nil || o.root || o.frame || x.b.a == nil {
		return false, err
	}
	return x.b.t.perform(x.b.a, o.k, a11ySetSelection, fmt.Sprintf("%d,%d", start, end)), nil
}

func (x atspiText) SetCaretOffset(msg dbus.Message, off int32) (bool, *dbus.Error) {
	return x.selectText(msg, off, off)
}

func (x atspiText) SetSelection(msg dbus.Message, i, start, end int32) (bool, *dbus.Error) {
	if i != 0 {
		return false, nil
	}
	return x.selectText(msg, start, end)
}

func (x atspiText) AddSelection(msg dbus.Message, start, end int32) (bool, *dbus.Error) {
	return x.selectText(msg, start, end)
}

func (x atspiText) RemoveSelection(msg dbus.Message, i int32) (bool, *dbus.Error) {
	_, n, err := x.b.resolve(msg)
	if err != nil || !n.HasCaret {
		return false, err
	}
	return x.selectText(msg, int32(n.Caret), int32(n.Caret))
}

func (x atspiText) GetCharacterExtents(msg dbus.Message, off int32, coords uint32) (int32, int32, int32, int32, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	r := x.b.extents(o, n, coords)
	return r.X, r.Y, r.W, r.H, nil
}

func (x atspiText) GetRangeExtents(msg dbus.Message, start, end int32, coords uint32) (int32, int32, int32, int32, *dbus.Error) {
	return x.GetCharacterExtents(msg, start, coords)
}

func (x atspiText) GetOffsetAtPoint(msg dbus.Message, px, py int32, coords uint32) (int32, *dbus.Error) {
	return 0, nil
}

func (x atspiText) GetAttributeRun(msg dbus.Message, off int32, defaults bool) (map[string]string, int32, int32, *dbus.Error) {
	return map[string]string{}, 0, int32(len(x.text(msg))), nil
}

func (x atspiText) GetAttributes(msg dbus.Message, off int32) (map[string]string, int32, int32, *dbus.Error) {
	return map[string]string{}, 0, int32(len(x.text(msg))), nil
}

func (x atspiText) GetDefaultAttributes(msg dbus.Message) (map[string]string, *dbus.Error) {
	return map[string]string{}, nil
}

func (x atspiText) GetDefaultAttributeSet(msg dbus.Message) (map[string]string, *dbus.Error) {
	return map[string]string{}, nil
}

// --- org.a11y.atspi.EditableText ---

type atspiEditable struct{ b *atspiBridge }

func (x atspiEditable) SetTextContents(msg dbus.Message, s string) (bool, *dbus.Error) {
	o, _, err := x.b.resolve(msg)
	if err != nil || x.b.a == nil {
		return false, err
	}
	return x.b.t.perform(x.b.a, o.k, a11ySetText, s), nil
}

func (x atspiEditable) InsertText(msg dbus.Message, pos int32, s string, length int32) (bool, *dbus.Error) {
	_, n, err := x.b.resolve(msg)
	if err != nil {
		return false, err
	}
	t := []rune(n.Value)
	p := max(0, min(int(pos), len(t)))
	return x.SetTextContents(msg, string(t[:p])+s+string(t[p:]))
}

func (x atspiEditable) DeleteText(msg dbus.Message, start, end int32) (bool, *dbus.Error) {
	_, n, err := x.b.resolve(msg)
	if err != nil {
		return false, err
	}
	t := []rune(n.Value)
	s, e := clampRange(start, end, len(t))
	return x.SetTextContents(msg, string(t[:s])+string(t[e:]))
}

// --- org.a11y.atspi.Application ---

type atspiApplication struct{ b *atspiBridge }

func (x atspiApplication) GetLocale(msg dbus.Message, kind uint32) (string, *dbus.Error) {
	return atspiLocale(), nil
}

func atspiLocale() string {
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if s := os.Getenv(v); s != "" {
			return s
		}
	}
	return "C"
}

// --- org.a11y.atspi.Cache ---

type atspiCacheItem struct {
	Ref, App, Parent atspiRef
	Index, Count     int32
	Interfaces       []string
	Name             string
	Role             uint32
	Description      string
	States           []uint32
}

type atspiCache struct{}

func (atspiCache) GetItems() ([]atspiCacheItem, *dbus.Error) { return []atspiCacheItem{}, nil }

// --- org.freedesktop.DBus.Properties ---

type atspiProps struct{ b *atspiBridge }

func (x atspiProps) GetAll(msg dbus.Message, iface string) (map[string]dbus.Variant, *dbus.Error) {
	out := map[string]dbus.Variant{}
	names := map[string][]string{
		"org.a11y.atspi.Accessible":  {"Name", "Description", "Parent", "ChildCount", "Locale", "AccessibleId", "HelpText"},
		"org.a11y.atspi.Application": {"ToolkitName", "Version", "AtspiVersion", "Id"},
		"org.a11y.atspi.Value":       {"MinimumValue", "MaximumValue", "MinimumIncrement", "CurrentValue", "Text"},
		"org.a11y.atspi.Text":        {"CharacterCount", "CaretOffset"},
		"org.a11y.atspi.Action":      {"NActions"},
	}[iface]
	for _, name := range names {
		if v, err := x.Get(msg, iface, name); err == nil {
			out[name] = v
		}
	}
	return out, nil
}

func (x atspiProps) Get(msg dbus.Message, iface, prop string) (dbus.Variant, *dbus.Error) {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return dbus.Variant{}, err
	}
	v := func(x any) (dbus.Variant, *dbus.Error) { return dbus.MakeVariant(x), nil }
	switch iface + "." + prop {
	case "org.a11y.atspi.Accessible.Name":
		return v(x.b.name(o, n))
	case "org.a11y.atspi.Accessible.Description":
		return v(n.Description)
	case "org.a11y.atspi.Accessible.HelpText":
		return v(n.Description)
	case "org.a11y.atspi.Accessible.Parent":
		return v(x.b.parent(o))
	case "org.a11y.atspi.Accessible.ChildCount":
		return v(int32(len(x.b.children(o))))
	case "org.a11y.atspi.Accessible.Locale":
		return v(atspiLocale())
	case "org.a11y.atspi.Accessible.AccessibleId":
		return v(n.ID)
	case "org.a11y.atspi.Application.ToolkitName":
		return v("Giode")
	case "org.a11y.atspi.Application.Version":
		return v("1")
	case "org.a11y.atspi.Application.AtspiVersion":
		return v("2.1")
	case "org.a11y.atspi.Application.Id":
		return v(x.b.appID.Load())
	case "org.a11y.atspi.Value.MinimumValue":
		return v(n.Min)
	case "org.a11y.atspi.Value.MaximumValue":
		return v(n.Max)
	case "org.a11y.atspi.Value.MinimumIncrement":
		return v((n.Max - n.Min) / 100)
	case "org.a11y.atspi.Value.CurrentValue":
		return v(n.Now)
	case "org.a11y.atspi.Value.Text":
		return v("")
	case "org.a11y.atspi.Text.CharacterCount":
		return v(int32(utf8.RuneCountInString(atspiTextOf(n))))
	case "org.a11y.atspi.Text.CaretOffset":
		if n.HasCaret {
			return v(int32(n.Caret))
		}
		return v(int32(-1))
	case "org.a11y.atspi.Action.NActions":
		return v(int32(len(atspiActions(n))))
	}
	return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{prop})
}

func (x atspiProps) Set(msg dbus.Message, iface, prop string, val dbus.Variant) *dbus.Error {
	o, n, err := x.b.resolve(msg)
	if err != nil {
		return err
	}
	switch iface + "." + prop {
	case "org.a11y.atspi.Application.Id":
		if id, ok := val.Value().(int32); ok {
			x.b.appID.Store(id)
		}
		return nil
	case "org.a11y.atspi.Value.CurrentValue":
		f, ok := val.Value().(float64)
		st, isStep := n.Target.(a11yStepper)
		if !ok || !isStep || o.root || o.frame || x.b.a == nil || n.Disabled {
			return dbus.NewError("org.freedesktop.DBus.Error.NotSupported", []any{prop})
		}
		x.b.a.Post(func() { st.Change(float32(f)) })
		return nil
	}
	return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{prop})
}

// --- Events ---

// textChanged reports whether focused node n's text differs from last
// frame's, for TextChanged events that only typing sends (not a caret
// move).
func (b *atspiBridge) textChanged(n elements.SemanticNode) bool {
	changed := n.Value != b.text
	b.text = n.Value
	return changed
}

func (b *atspiBridge) emit(path dbus.ObjectPath, event, detail string, d1, d2 int32, data any) {
	b.conn.Emit(path, event, detail, d1, d2, dbus.MakeVariant(data), map[string]dbus.Variant{})
}

// events reports the frame's changes: focus moving, the focused node's
// state and value, and announcements. It runs on the UI goroutine after
// each tree update.
func (b *atspiBridge) events() {
	changed, announce := b.t.takeChanges()
	if !b.on.Load() {
		return
	}
	focus := b.t.focusedNode()
	b.mu.Lock()
	old := b.focus
	b.focus = focus
	b.mu.Unlock()
	if focus != old {
		if old != 0 {
			b.emit(atspiPath(atspiObj{k: old}), "org.a11y.atspi.Event.Object.StateChanged", "focused", 0, 0, int32(0))
		}
		if focus != 0 {
			b.emit(atspiPath(atspiObj{k: focus}), "org.a11y.atspi.Event.Object.StateChanged", "focused", 1, 0, int32(0))
		}
	}
	for _, k := range changed {
		if k == 0 {
			b.emit(atspiFramePath, "org.a11y.atspi.Event.Object.ChildrenChanged", "add", -1, 0, b.ref(atspiObj{frame: true}))
			continue
		}
		n, _, _, ok := b.t.node(k)
		if !ok || !n.Focused {
			continue
		}
		path := atspiPath(atspiObj{k: k})
		if n.Checked != elements.NotCheckable {
			b.emit(path, "org.a11y.atspi.Event.Object.StateChanged", "checked", boolInt(n.Checked == elements.Checked), 0, int32(0))
		}
		if n.Expanded != elements.NotExpandable {
			b.emit(path, "org.a11y.atspi.Event.Object.StateChanged", "expanded", boolInt(n.Expanded == elements.Expanded), 0, int32(0))
		}
		if atspiHasValue(n) {
			b.emit(path, "org.a11y.atspi.Event.Object.PropertyChange", "accessible-value", 0, 0, n.Now)
		}
		if n.Kind == elements.SemanticInput && b.textChanged(n) {
			b.emit(path, "org.a11y.atspi.Event.Object.TextChanged", "insert", 0, int32(utf8.RuneCountInString(n.Value)), n.Value)
		}
		if n.HasCaret {
			caret := [2]int{n.Caret, n.Anchor}
			if caret != b.caret || k != old {
				b.emit(path, "org.a11y.atspi.Event.Object.TextCaretMoved", "", int32(n.Caret), 0, int32(0))
				if n.Anchor != n.Caret || b.caret[0] != b.caret[1] {
					b.emit(path, "org.a11y.atspi.Event.Object.TextSelectionChanged", "", 0, 0, int32(0))
				}
			}
			b.caret = caret
		}
		b.emit(path, "org.a11y.atspi.Event.Object.PropertyChange", "accessible-name", 0, 0, b.name(atspiObj{k: k}, n))
	}
	for _, an := range announce {
		politeness := int32(1)
		if an.assertive {
			politeness = 2
		}
		b.emit(atspiFramePath, "org.a11y.atspi.Event.Object.Announcement", "", politeness, 0, an.text)
	}
}
