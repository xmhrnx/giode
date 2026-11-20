//go:build linux && !android

package push

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"gioui.org/app"
	"github.com/godbus/dbus/v5"

	"github.com/xmhrnx/giode"
)

// Linux push goes through UnifiedPush: a distributor on the session bus
// (KDE's, ntfy's, NextPush...) keeps one connection to its push server
// for all apps, gives each app an endpoint URL, and calls the app's
// connector object with its messages. When the app isn't running, the
// distributor's call starts it through D-Bus activation: the package
// installs a service file for the app's bus name.

const (
	connectorPath    = "/org/unifiedpush/Connector"
	distributorPath  = "/org/unifiedpush/Distributor"
	distributorName  = "org.unifiedpush.Distributor."
	registerOK       = "REGISTRATION_SUCCEEDED"
	connectorIface1  = "org.unifiedpush.Connector1"
	connectorIface2  = "org.unifiedpush.Connector2"
	distributorIface = "org.unifiedpush.Distributor"
)

// ErrNoDistributor is returned by Token on Linux when no UnifiedPush
// distributor runs: the user installs one (KUnifiedPush, ntfy's
// desktop app...).
var ErrNoDistributor = errors.New("push: no UnifiedPush distributor on the session bus")

// state is what the package keeps between runs.
type state struct {
	Token    string `json:"token"`
	Endpoint string `json:"endpoint,omitempty"`
	Private  []byte `json:"key"`
	Auth     []byte `json:"auth"`
}

var lin struct {
	sync.Mutex
	conn     *dbus.Conn
	st       state
	key      webPushKey
	endpoint chan string
	failed   chan string
	// dataDir overrides where state is kept, in tests.
	dataDir string
}

func statePath() (string, error) {
	dir := lin.dataDir
	if dir == "" {
		d, err := app.DataDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(d, appName())
	}
	return filepath.Join(dir, "push.json"), nil
}

func appName() string {
	if app.ID != "" {
		return app.ID
	}
	exe, _ := os.Executable()
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}

var badBusChars = regexp.MustCompile(`[^A-Za-z0-9_.]`)

// busName is the app's well-known name on the session bus, which the
// distributor calls: its ID.
func busName() string {
	n := badBusChars.ReplaceAllString(appName(), "_")
	parts := strings.Split(n, ".")
	for i, p := range parts {
		if p == "" || (p[0] >= '0' && p[0] <= '9') {
			parts[i] = "_" + p
		}
	}
	if len(parts) < 2 {
		parts = append([]string{"org", "giode", "app"}, parts...)
	}
	return strings.Join(parts, ".")
}

// load reads or makes the package's state; lin is locked.
func load() error {
	if lin.st.Token != "" {
		return nil
	}
	p, err := statePath()
	if err != nil {
		return err
	}
	if b, err := os.ReadFile(p); err == nil && json.Unmarshal(b, &lin.st) == nil && lin.st.Token != "" {
		if k, err := ecdhP256Private(lin.st.Private); err == nil {
			lin.key = webPushKey{private: k, auth: lin.st.Auth}
			return nil
		}
	}
	k, err := newWebPushKey()
	if err != nil {
		return err
	}
	var tok [16]byte
	rand.Read(tok[:])
	lin.key = k
	lin.st = state{Token: hex.EncodeToString(tok[:]), Private: k.private.Bytes(), Auth: k.auth}
	return save()
}

// save writes the state, readable only by the user; lin is locked.
func save() error {
	p, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(lin.st)
	return os.WriteFile(p, b, 0o600)
}

// connector is the app's object the distributor calls, in both
// versions of the interface.
type connector struct{}

type connector1 struct{}

func (connector) Message(args map[string]dbus.Variant) (map[string]dbus.Variant, *dbus.Error) {
	tok, _ := args["token"].Value().(string)
	msg, _ := args["message"].Value().([]byte)
	id, _ := args["id"].Value().(string)
	onMessage(tok, msg, id)
	res := map[string]dbus.Variant{}
	if id != "" {
		res["id"] = dbus.MakeVariant(id)
	}
	return res, nil
}

func (connector) NewEndpoint(args map[string]dbus.Variant) (map[string]dbus.Variant, *dbus.Error) {
	tok, _ := args["token"].Value().(string)
	ep, _ := args["endpoint"].Value().(string)
	onEndpoint(tok, ep)
	return map[string]dbus.Variant{}, nil
}

func (connector) Unregistered(args map[string]dbus.Variant) (map[string]dbus.Variant, *dbus.Error) {
	tok, _ := args["token"].Value().(string)
	onUnregistered(tok)
	return map[string]dbus.Variant{}, nil
}

func (connector1) Message(tok string, msg []byte, id string) *dbus.Error {
	onMessage(tok, msg, id)
	return nil
}

func (connector1) NewEndpoint(tok, ep string) *dbus.Error {
	onEndpoint(tok, ep)
	return nil
}

func (connector1) Unregistered(tok string) *dbus.Error {
	onUnregistered(tok)
	return nil
}

func ours(tok string) bool {
	lin.Lock()
	defer lin.Unlock()
	return tok != "" && tok == lin.st.Token
}

func onMessage(tok string, body []byte, id string) {
	if !ours(tok) {
		return
	}
	lin.Lock()
	k := lin.key
	lin.Unlock()
	if plain, err := k.decrypt(body); err == nil {
		body = plain
	}
	m := parseRaw(body)
	if m.ID == "" {
		m.ID = id
	}
	deliver(event{kind: evMessage, msg: m})
}

func onEndpoint(tok, ep string) {
	if !ours(tok) || ep == "" {
		return
	}
	lin.Lock()
	changed := lin.st.Endpoint != ep
	lin.st.Endpoint = ep
	save()
	ch := lin.endpoint
	lin.Unlock()
	if ch != nil {
		select {
		case ch <- ep:
		default:
		}
	}
	if changed {
		deliver(event{kind: evToken, token: ep})
	}
}

func onUnregistered(tok string) {
	if !ours(tok) {
		return
	}
	lin.Lock()
	lin.st.Endpoint = ""
	save()
	ch := lin.failed
	lin.Unlock()
	if ch != nil {
		select {
		case ch <- "unregistered by the distributor":
		default:
		}
	}
}

// connect exports the connector and takes the app's bus name, once.
func connect() (*dbus.Conn, error) {
	lin.Lock()
	defer lin.Unlock()
	if lin.conn != nil {
		return lin.conn, nil
	}
	if err := load(); err != nil {
		return nil, err
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	if err := conn.Export(connector{}, connectorPath, connectorIface2); err != nil {
		conn.Close()
		return nil, err
	}
	conn.Export(connector1{}, connectorPath, connectorIface1)
	if _, err := conn.RequestName(busName(), dbus.NameFlagDoNotQueue); err != nil {
		conn.Close()
		return nil, err
	}
	lin.conn = conn
	installService()
	return conn, nil
}

// installService writes the D-Bus service file that lets the
// distributor start the app for a message.
func installService() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dir = filepath.Join(home, ".local", "share")
	}
	if os.Getenv("FLATPAK_ID") != "" || os.Getenv("SNAP") != "" {
		// Sandboxed: giode build's Flatpak declares the service; a snap's
		// pushes arrive while it runs.
		return
	}
	if _, err := os.Stat(filepath.Join("/usr/share/dbus-1/services", busName()+".service")); err == nil {
		return // installed by the app's package
	}
	p := filepath.Join(dir, "dbus-1", "services", busName()+".service")
	body := fmt.Sprintf("[D-BUS Service]\nName=%s\nExec=%s\n", busName(), quoteExec(exe))
	if b, err := os.ReadFile(p); err == nil && string(b) == body {
		return
	}
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(body), 0o644)
}

func quoteExec(s string) string {
	if !strings.ContainsAny(s, " \t\"'\\") {
		return s
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// distributor picks the distributor: UNIFIEDPUSH_DISTRIBUTOR's, else
// the first by name.
func distributor(conn *dbus.Conn) (string, error) {
	var names []string
	if err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return "", err
	}
	var ds []string
	for _, n := range names {
		if strings.HasPrefix(n, distributorName) {
			ds = append(ds, n)
		}
	}
	sort.Strings(ds)
	if want := os.Getenv("UNIFIEDPUSH_DISTRIBUTOR"); want != "" {
		for _, d := range ds {
			if d == distributorName+want {
				return d, nil
			}
		}
		return "", ErrNoDistributor
	}
	if len(ds) == 0 {
		return "", ErrNoDistributor
	}
	return ds[0], nil
}

func start(a *giode.App) error {
	if _, err := connect(); err != nil {
		return err
	}
	// Registered before: register again, as distributors expect at
	// start, so a new endpoint reaches Token.
	lin.Lock()
	registered := lin.st.Endpoint != ""
	lin.Unlock()
	if registered {
		go register(context.Background())
	}
	return nil
}

// register asks the distributor for an endpoint and waits for it.
func register(ctx context.Context) (string, error) {
	conn, err := connect()
	if err != nil {
		return "", err
	}
	d, err := distributor(conn)
	if err != nil {
		return "", err
	}
	lin.Lock()
	tok := lin.st.Token
	epc, failc := make(chan string, 1), make(chan string, 1)
	lin.endpoint, lin.failed = epc, failc
	lin.Unlock()
	defer func() {
		lin.Lock()
		lin.endpoint, lin.failed = nil, nil
		lin.Unlock()
	}()
	obj := conn.Object(d, distributorPath)
	desc := appName()
	var res map[string]dbus.Variant
	call := obj.CallWithContext(ctx, distributorIface+"2.Register", 0, map[string]dbus.Variant{
		"service":     dbus.MakeVariant(busName()),
		"token":       dbus.MakeVariant(tok),
		"description": dbus.MakeVariant(desc),
	})
	if call.Err == nil {
		call.Store(&res)
		if s, _ := res["success"].Value().(string); s != registerOK {
			reason, _ := res["reason"].Value().(string)
			return "", fmt.Errorf("push: registration failed: %s", reason)
		}
	} else {
		// A version 1 distributor.
		var result, reason string
		if err := obj.CallWithContext(ctx, distributorIface+"1.Register", 0, busName(), tok, desc).Store(&result, &reason); err != nil {
			return "", err
		}
		if result != registerOK {
			return "", fmt.Errorf("push: registration failed: %s", reason)
		}
	}
	lin.Lock()
	ep := lin.st.Endpoint
	lin.Unlock()
	select {
	case e := <-epc:
		return e, nil
	case msg := <-failc:
		return "", errors.New("push: " + msg)
	case <-ctx.Done():
		if ep != "" {
			return ep, nil
		}
		return "", ctx.Err()
	}
}

func token(ctx context.Context) (string, error) {
	return register(ctx)
}

func requestPermission(ctx context.Context, a *giode.App) (bool, error) { return true, nil }

func unregister() error {
	conn, err := connect()
	if err != nil {
		return err
	}
	d, err := distributor(conn)
	if err != nil {
		return err
	}
	lin.Lock()
	tok := lin.st.Token
	lin.st.Endpoint = ""
	save()
	lin.Unlock()
	obj := conn.Object(d, distributorPath)
	if err := obj.Call(distributorIface+"2.Unregister", 0, map[string]dbus.Variant{"token": dbus.MakeVariant(tok)}).Err; err != nil {
		return obj.Call(distributorIface+"1.Unregister", 0, tok).Err
	}
	return nil
}

func keys() (WebPushKeys, error) {
	lin.Lock()
	defer lin.Unlock()
	if err := load(); err != nil {
		return WebPushKeys{}, err
	}
	return lin.key.public(), nil
}
