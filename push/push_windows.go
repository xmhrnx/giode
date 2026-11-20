package push

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"gioui.org/app"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/winrt"
)

// Windows push goes through WNS: the app asks for a push channel, whose
// URI the server sends to. Toasts sent to an app that isn't running are
// shown by Windows, and clicking one starts the app through its toast
// activator (as giode/notification's toasts do); messages arriving while
// it runs come to Handlers.Message instead, Windows showing nothing.
// Channels need package identity: an MSIX build.

var (
	iidChannelManagerStatics = winrt.ParseGUID("8baf9b65-77a1-4588-bd19-861529a9dcf0")
	iidChannel               = winrt.ParseGUID("2b28102e-ef0b-4f39-9b8a-a3c194de7081")
	iidReceivedArgs          = winrt.ParseGUID("d1065e0c-36cd-484c-b935-0a99b753cf00")
	iidRawNotification       = winrt.ParseGUID("1a227281-3b79-42ac-9963-22ab00d4f0b7")
	iidXmlNodeSerializer     = winrt.ParseGUID("5cc5b382-e6dd-4991-abef-06d8d2e7bd0c")
	receivedIID              = winrt.TypedEventHandlerIID(
		winrt.RuntimeClassSignature("Windows.Networking.PushNotifications.PushNotificationChannel", iidChannel),
		winrt.RuntimeClassSignature("Windows.Networking.PushNotifications.PushNotificationReceivedEventArgs", iidReceivedArgs))
)

const classChannelManager = "Windows.Networking.PushNotifications.PushNotificationChannelManager"

var wns struct {
	sync.Mutex
	channel  *winrt.Obj
	uri      string
	expires  time.Time
	received *winrt.Delegate
}

func appID() string {
	if app.ID != "" {
		return app.ID
	}
	exe, _ := os.Executable()
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}

func start(a *giode.App) error {
	// Clicks on pushed toasts: giode/notification takes its own; the
	// rest are pushes the server sent.
	winrt.OnActivate(func(args string, inputs map[string]string) bool {
		if strings.HasPrefix(args, "giode.n:") {
			return false
		}
		m := Message{Data: map[string]string{}}
		if args != "" {
			m = parseRaw([]byte(args))
			if _, ok := m.Data["message"]; ok {
				m.Data = map[string]string{"launch": args}
			}
		}
		for k, v := range inputs {
			m.Data[k] = v
		}
		deliver(event{kind: evOpen, msg: m})
		return true
	})
	return winrt.ServeActivator(winrt.ToastCLSID(appID()))
}

func token(ctx context.Context) (string, error) {
	wns.Lock()
	if wns.channel != nil && time.Until(wns.expires) > time.Hour {
		uri := wns.uri
		wns.Unlock()
		return uri, nil
	}
	wns.Unlock()
	timeout := 30 * time.Second
	if d, ok := ctx.Deadline(); ok {
		timeout = time.Until(d)
	}
	var uri string
	var err error
	derr := winrt.Do(func() { uri, err = openChannel(timeout) })
	if derr != nil {
		return "", derr
	}
	if err != nil {
		var hr winrt.HRESULT
		if errors.As(err, &hr) && (hr == 0x80073D54 || hr == 0x803E0105) {
			// APPMODEL_ERROR_NO_PACKAGE, WPN_E_..: no package identity.
			return "", errors.New("push: WNS needs the app packaged as MSIX (giode build -target msix): " + err.Error())
		}
		return "", err
	}
	return uri, nil
}

// openChannel creates the app's channel and subscribes to its pushes;
// on the Windows Runtime thread.
func openChannel(timeout time.Duration) (string, error) {
	mgr, err := winrt.Factory(classChannelManager, iidChannelManagerStatics)
	if err != nil {
		return "", err
	}
	defer mgr.Release()
	op, err := mgr.Out(6) // CreatePushNotificationChannelForApplicationAsync
	if err != nil {
		return "", err
	}
	ch, err := winrt.Await(op, timeout)
	if err != nil {
		return "", err
	}
	var h winrt.HString
	if err := ch.Call(6, uintptr(unsafe.Pointer(&h))); err != nil { // Uri
		ch.Release()
		return "", err
	}
	uri := h.String()
	h.Delete()
	var expires int64 // 100 ns ticks since 1601
	ch.Call(7, uintptr(unsafe.Pointer(&expires)))
	wns.Lock()
	if wns.received == nil {
		wns.received = winrt.NewDelegate(receivedIID, onReceived)
	}
	old := wns.channel
	wns.channel, wns.uri = ch, uri
	wns.expires = time.Unix(expires/1e7-11644473600, 0)
	d := wns.received
	wns.Unlock()
	var cookie int64
	ch.Call(9, d.Ptr(), uintptr(unsafe.Pointer(&cookie))) // add_PushNotificationReceived
	old.Release()
	return uri, nil
}

// onReceived takes a push arriving while the app runs.
func onReceived(_, args *winrt.Obj) {
	a, err := args.Query(iidReceivedArgs)
	if err != nil {
		return
	}
	defer a.Release()
	var kind int32
	a.Call(8, uintptr(unsafe.Pointer(&kind)))
	switch kind {
	case 3: // Raw
		raw, err := a.Out(12)
		if err != nil {
			return
		}
		defer raw.Release()
		r, err := raw.Query(iidRawNotification)
		if err != nil {
			return
		}
		defer r.Release()
		var h winrt.HString
		r.Call(6, uintptr(unsafe.Pointer(&h)))
		body := h.String()
		h.Delete()
		deliver(event{kind: evMessage, msg: parseRaw([]byte(body))})
	case 0: // Toast
		// The app decides what to show while it runs, as elsewhere.
		a.Call(6, 1) // put_Cancel
		t, err := a.Out(9)
		if err != nil {
			return
		}
		defer t.Release()
		doc, err := t.Out(6) // Content
		if err != nil {
			return
		}
		defer doc.Release()
		ser, err := doc.Query(iidXmlNodeSerializer)
		if err != nil {
			return
		}
		defer ser.Release()
		var h winrt.HString
		ser.Call(6, uintptr(unsafe.Pointer(&h))) // GetXml
		x := h.String()
		h.Delete()
		deliver(event{kind: evMessage, msg: parseToast(x)})
	}
}

// parseToast reads a toast's XML: its first two texts as the title and
// body, its launch arguments as Data["launch"].
func parseToast(s string) Message {
	m := Message{Data: map[string]string{}}
	var t struct {
		Launch string   `xml:"launch,attr"`
		Texts  []string `xml:"visual>binding>text"`
	}
	if xml.Unmarshal([]byte(s), &t) != nil {
		return m
	}
	if len(t.Texts) > 0 {
		m.Title = t.Texts[0]
	}
	if len(t.Texts) > 1 {
		m.Body = t.Texts[1]
	}
	if t.Launch != "" {
		m.Data["launch"] = t.Launch
	}
	return m
}

func requestPermission(ctx context.Context, a *giode.App) (bool, error) { return true, nil }

func unregister() error {
	wns.Lock()
	ch := wns.channel
	wns.channel, wns.uri = nil, ""
	wns.Unlock()
	if ch == nil {
		return nil
	}
	return winrt.Do(func() {
		ch.Call(8) // Close
		ch.Release()
	})
}

func keys() (WebPushKeys, error) { return WebPushKeys{}, giode.ErrNotSupported }
