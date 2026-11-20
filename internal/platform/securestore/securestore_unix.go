//go:build (linux && !android) || freebsd || openbsd || netbsd || dragonfly

package securestore

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/xmhrnx/giode/internal/platform"
)

// The freedesktop Secret Service API, implemented by GNOME Keyring and
// KWallet: https://specifications.freedesktop.org/secret-service/
const (
	secretsName       = "org.freedesktop.secrets"
	secretsPath       = dbus.ObjectPath("/org/freedesktop/secrets")
	secretsDefault    = dbus.ObjectPath("/org/freedesktop/secrets/aliases/default")
	secretsService    = "org.freedesktop.Secret.Service"
	secretsCollection = "org.freedesktop.Secret.Collection"
	secretsItem       = "org.freedesktop.Secret.Item"
	noPrompt          = dbus.ObjectPath("/")
)

// secretValue is the Secret Service's Secret struct.
type secretValue struct {
	Session     dbus.ObjectPath
	Params      []byte
	Value       []byte
	ContentType string
}

type secrets struct {
	conn    *dbus.Conn
	session dbus.ObjectPath
}

func openSecrets() (*secrets, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, errors.Join(platform.ErrNotSupported, err)
	}
	var out dbus.Variant
	var session dbus.ObjectPath
	// The plain algorithm: values travel unencrypted over the session
	// bus, which only this user's processes can reach.
	err = conn.Object(secretsName, secretsPath).Call(secretsService+".OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&out, &session)
	if err != nil {
		return nil, fmt.Errorf("giode: no Secret Service (GNOME Keyring or KWallet) running: %w", err)
	}
	return &secrets{conn: conn, session: session}, nil
}

func (s *secrets) close() {
	s.conn.Object(secretsName, s.session).Call("org.freedesktop.Secret.Session.Close", 0)
}

func secretAttrs(service, name string) map[string]string {
	return map[string]string{"xdg:schema": "org.giode.SecureStore", "service": service, "account": name}
}

// prompt runs a Secret Service prompt (to unlock the keyring, say) and
// waits for the user.
func (s *secrets) prompt(p dbus.ObjectPath) error {
	if p == noPrompt || p == "" {
		return nil
	}
	if err := s.conn.AddMatchSignal(dbus.WithMatchObjectPath(p), dbus.WithMatchInterface("org.freedesktop.Secret.Prompt"), dbus.WithMatchMember("Completed")); err != nil {
		return err
	}
	defer s.conn.RemoveMatchSignal(dbus.WithMatchObjectPath(p), dbus.WithMatchInterface("org.freedesktop.Secret.Prompt"), dbus.WithMatchMember("Completed"))
	ch := make(chan *dbus.Signal, 4)
	s.conn.Signal(ch)
	defer s.conn.RemoveSignal(ch)
	if err := s.conn.Object(secretsName, p).Call("org.freedesktop.Secret.Prompt.Prompt", 0, "").Err; err != nil {
		return err
	}
	for sig := range ch {
		if sig.Path != p || len(sig.Body) == 0 {
			continue
		}
		if dismissed, _ := sig.Body[0].(bool); dismissed {
			return errors.New("giode: the keyring prompt was dismissed")
		}
		return nil
	}
	return errors.New("giode: lost the Secret Service connection")
}

func (s *secrets) unlock(paths []dbus.ObjectPath) error {
	var unlocked []dbus.ObjectPath
	var p dbus.ObjectPath
	if err := s.conn.Object(secretsName, secretsPath).Call(secretsService+".Unlock", 0, paths).Store(&unlocked, &p); err != nil {
		return err
	}
	return s.prompt(p)
}

func (s *secrets) search(service, name string) ([]dbus.ObjectPath, error) {
	var unlocked, locked []dbus.ObjectPath
	if err := s.conn.Object(secretsName, secretsPath).Call(secretsService+".SearchItems", 0, secretAttrs(service, name)).Store(&unlocked, &locked); err != nil {
		return nil, err
	}
	if len(locked) > 0 {
		if err := s.unlock(locked); err != nil {
			return nil, err
		}
	}
	return append(unlocked, locked...), nil
}

func Set(service, name string, value []byte) error {
	s, err := openSecrets()
	if err != nil {
		return err
	}
	defer s.close()
	if err := s.unlock([]dbus.ObjectPath{secretsDefault}); err != nil {
		return err
	}
	props := map[string]dbus.Variant{
		"org.freedesktop.Secret.Item.Label":      dbus.MakeVariant(service + ": " + name),
		"org.freedesktop.Secret.Item.Attributes": dbus.MakeVariant(secretAttrs(service, name)),
	}
	secret := secretValue{Session: s.session, Params: []byte{}, Value: value, ContentType: "application/octet-stream"}
	var item, p dbus.ObjectPath
	if err := s.conn.Object(secretsName, secretsDefault).Call(secretsCollection+".CreateItem", 0, props, secret, true).Store(&item, &p); err != nil {
		return err
	}
	return s.prompt(p)
}

func Get(service, name string) ([]byte, error) {
	s, err := openSecrets()
	if err != nil {
		return nil, err
	}
	defer s.close()
	items, err := s.search(service, name)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, platform.ErrNotFound
	}
	var secret secretValue
	if err := s.conn.Object(secretsName, items[0]).Call(secretsItem+".GetSecret", 0, s.session).Store(&secret); err != nil {
		return nil, err
	}
	return secret.Value, nil
}

func Delete(service, name string) error {
	s, err := openSecrets()
	if err != nil {
		return err
	}
	defer s.close()
	items, err := s.search(service, name)
	if err != nil {
		return err
	}
	for _, it := range items {
		var p dbus.ObjectPath
		if err := s.conn.Object(secretsName, it).Call(secretsItem+".Delete", 0).Store(&p); err != nil {
			return err
		}
		if err := s.prompt(p); err != nil {
			return err
		}
	}
	return nil
}
