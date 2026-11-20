package notification

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"gioui.org/app"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/winrt"
)

// Windows notifications are toasts. An unpackaged app registers its
// AppUserModelID and toast activator class under HKEY_CURRENT_USER the
// first time; an MSIX package declares them in its manifest (giode
// build does). Clicks and buttons come to the activator, which Windows
// starts the app for when it isn't running.

var (
	iidXmlDocument          = winrt.ParseGUID("f7f3a506-1e87-42d6-bcfb-b8c809fa5494")
	iidXmlDocumentIO        = winrt.ParseGUID("6cd0e74e-ee65-4489-9ebf-ca43e87ba637")
	iidToastFactory         = winrt.ParseGUID("04124b20-82c6-4229-b109-fd9ed4662b53")
	iidToastNotification2   = winrt.ParseGUID("9dfb9fd1-143a-490e-90bf-b9fba7132de7")
	iidToastManagerStatics  = winrt.ParseGUID("50ac103f-d235-4598-bbef-98fe4d1a3ad4")
	iidToastManagerStatics2 = winrt.ParseGUID("7ab93c52-0e48-4750-ba9d-1a4113981847")
)

const (
	classToast        = "Windows.UI.Notifications.ToastNotification"
	classToastManager = "Windows.UI.Notifications.ToastNotificationManager"
	argsPrefix        = "giode.n:"
	defaultGroup      = "giode"
)

var win struct {
	sync.Mutex
	once   sync.Once
	aumid  string
	err    error
	groups map[string]string // notification ID → toast group
}

var (
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procGetPackageFullName = kernel32.NewProc("GetCurrentPackageFullName")
	procGetAppUserModelID  = kernel32.NewProc("GetCurrentApplicationUserModelId")
	errInsufficientBuffer  = uintptr(122)
	appModelErrorNoPackage = uintptr(15700)
	activationHandlerOnce  sync.Once
)

// appID is the ID giode build gives the app, else its executable's name.
func appID() string {
	if app.ID != "" {
		return app.ID
	}
	exe, _ := os.Executable()
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}

// ActivatorCLSID is the COM class of an app's toast activator, derived
// from its ID: giode build puts the same in the MSIX manifest.
func activatorCLSID() winrt.GUID { return winrt.NameGUID("toast:" + appID()) }

// packagedAUMID returns the AppUserModelID of a packaged app, or "".
func packagedAUMID() string {
	var n uint32
	if r, _, _ := procGetPackageFullName.Call(uintptr(unsafe.Pointer(&n)), 0); r != errInsufficientBuffer {
		return ""
	}
	n = 0
	procGetAppUserModelID.Call(uintptr(unsafe.Pointer(&n)), 0)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n)
	if r, _, _ := procGetAppUserModelID.Call(uintptr(unsafe.Pointer(&n)), uintptr(unsafe.Pointer(&buf[0]))); r != 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}

// setup finds the app's AppUserModelID, registering it when unpackaged,
// and starts the activator.
func setup() (string, error) {
	win.once.Do(func() {
		win.groups = map[string]string{}
		clsid := activatorCLSID()
		if id := packagedAUMID(); id != "" {
			win.aumid = id
		} else {
			win.aumid = appID()
			win.err = register(win.aumid, clsid)
		}
		activationHandlerOnce.Do(func() { winrt.OnActivate(activated) })
		if err := winrt.ServeActivator(clsid); err != nil && win.err == nil {
			win.err = err
		}
	})
	return win.aumid, win.err
}

// register writes the unpackaged app's toast registration: its name and
// activator, and the activator's executable.
func register(aumid string, clsid winrt.GUID) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\AppUserModelId\`+aumid, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	name := strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
	if i := strings.LastIndexByte(aumid, '.'); app.ID != "" && i >= 0 && i < len(aumid)-1 {
		name = aumid[i+1:]
	}
	k.SetStringValue("DisplayName", name)
	k.SetStringValue("CustomActivator", "{"+strings.ToUpper(clsid.String())+"}")
	c, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\CLSID\{`+strings.ToUpper(clsid.String())+`}\LocalServer32`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.SetStringValue("", `"`+exe+`"`)
}

func listen(*giode.App) error {
	_, err := setup()
	return err
}

type winArgs struct {
	payload
	Action string `json:"a,omitempty"`
}

// activated takes a toast activation meant for this package.
func activated(args string, inputs map[string]string) bool {
	js, ok := strings.CutPrefix(args, argsPrefix)
	if !ok {
		return false
	}
	var a winArgs
	if json.Unmarshal([]byte(js), &a) != nil || a.ID == "" {
		return true
	}
	respond(Response{ID: a.ID, Action: a.Action, Reply: inputs["r"], Data: a.Data})
	return true
}

func attr(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return strings.ReplaceAll(b.String(), `"`, "&quot;")
}

// toastXML is the toast's content.
func toastXML(n Notification) string {
	args := func(action string) string {
		js, _ := json.Marshal(winArgs{payload: payload{ID: n.ID, Data: n.Data}, Action: action})
		return attr(argsPrefix + string(js))
	}
	var b strings.Builder
	b.WriteString(`<toast launch="` + args("") + `" activationType="foreground">`)
	b.WriteString(`<visual><binding template="ToastGeneric"><text>` + attr(n.Title) + `</text><text>` + attr(n.Body) + `</text></binding></visual>`)
	if len(n.Actions) > 0 {
		b.WriteString(`<actions>`)
		for _, a := range n.Actions {
			if a.Reply {
				b.WriteString(`<input id="r" type="text" placeHolderContent="` + attr(a.Placeholder) + `"/>`)
				break
			}
		}
		replied := false
		for _, a := range n.Actions {
			kind := "background"
			if a.Foreground {
				kind = "foreground"
			}
			b.WriteString(`<action content="`)
			if a.Reply && !replied {
				replied = true
				b.WriteString(attr(sendLabel(a)) + `" hint-inputId="r`)
			} else {
				b.WriteString(attr(a.Title))
			}
			b.WriteString(`" arguments="` + args(a.ID) + `" activationType="` + kind + `"/>`)
		}
		b.WriteString(`</actions>`)
	}
	if n.Silent {
		b.WriteString(`<audio silent="true"/>`)
	}
	b.WriteString(`</toast>`)
	return b.String()
}

// tagFor fits a notification ID into a toast tag (64 characters).
func tagFor(s string) string {
	if len(s) <= 64 {
		return s
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:16])
}

func show(n Notification, ch Channel) error {
	aumid, err := setup()
	if err != nil {
		return err
	}
	group := defaultGroup
	if n.Thread != "" {
		group = tagFor(n.Thread)
	}
	content := toastXML(n)
	err = winrt.Do(func() { err = showToast(aumid, content, tagFor(n.ID), group, ch.Importance == Low) })
	if err != nil {
		return err
	}
	win.Lock()
	win.groups[n.ID] = group
	win.Unlock()
	return nil
}

func showToast(aumid, content, tag, group string, quiet bool) error {
	doc, err := winrt.Activate("Windows.Data.Xml.Dom.XmlDocument")
	if err != nil {
		return err
	}
	defer doc.Release()
	io, err := doc.Query(iidXmlDocumentIO)
	if err != nil {
		return err
	}
	defer io.Release()
	h := winrt.NewHString(content)
	err = io.Call(6, uintptr(h))
	h.Delete()
	if err != nil {
		return err
	}
	xdoc, err := doc.Query(iidXmlDocument)
	if err != nil {
		return err
	}
	defer xdoc.Release()
	factory, err := winrt.Factory(classToast, iidToastFactory)
	if err != nil {
		return err
	}
	defer factory.Release()
	toast, err := factory.Out(6, uintptr(unsafe.Pointer(xdoc)))
	if err != nil {
		return err
	}
	defer toast.Release()
	if t2, err := toast.Query(iidToastNotification2); err == nil {
		ht, hg := winrt.NewHString(tag), winrt.NewHString(group)
		t2.Call(6, uintptr(ht))
		t2.Call(8, uintptr(hg))
		ht.Delete()
		hg.Delete()
		if quiet {
			t2.Call(10, 1)
		}
		t2.Release()
	}
	mgr, err := winrt.Factory(classToastManager, iidToastManagerStatics)
	if err != nil {
		return err
	}
	defer mgr.Release()
	ha := winrt.NewHString(aumid)
	notifier, err := mgr.Out(7, uintptr(ha))
	ha.Delete()
	if err != nil {
		return err
	}
	defer notifier.Release()
	return notifier.Call(6, uintptr(unsafe.Pointer(toast)))
}

// history calls fn with the toast history.
func history(fn func(h *winrt.Obj) error) error {
	var err error
	derr := winrt.Do(func() {
		var mgr *winrt.Obj
		mgr, err = winrt.Factory(classToastManager, iidToastManagerStatics2)
		if err != nil {
			return
		}
		defer mgr.Release()
		var h *winrt.Obj
		h, err = mgr.Out(6)
		if err != nil {
			return
		}
		defer h.Release()
		err = fn(h)
	})
	if derr != nil {
		return derr
	}
	return err
}

func remove(id string) error {
	aumid, err := setup()
	if err != nil {
		return err
	}
	win.Lock()
	group, ok := win.groups[id]
	delete(win.groups, id)
	win.Unlock()
	if !ok {
		group = defaultGroup
	}
	return history(func(h *winrt.Obj) error {
		ht, hg, ha := winrt.NewHString(tagFor(id)), winrt.NewHString(group), winrt.NewHString(aumid)
		defer ht.Delete()
		defer hg.Delete()
		defer ha.Delete()
		return h.Call(8, uintptr(ht), uintptr(hg), uintptr(ha)) // RemoveGroupedTagWithId
	})
}

func removeAll() error {
	aumid, err := setup()
	if err != nil {
		return err
	}
	win.Lock()
	win.groups = map[string]string{}
	win.Unlock()
	return history(func(h *winrt.Obj) error {
		ha := winrt.NewHString(aumid)
		defer ha.Delete()
		return h.Call(12, uintptr(ha)) // ClearWithId
	})
}

func defineChannel(Channel) error { return nil }

func defineCategory(string, []Action) error { return nil }
