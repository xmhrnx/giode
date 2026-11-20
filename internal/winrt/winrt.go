// Package winrt calls the few Windows Runtime APIs giode needs (toast
// notifications, push channels) without a projection: activation
// factories, HSTRINGs, vtable calls and polling of async operations.
package winrt

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"strings"
)

// GUID is a COM GUID, laid out as Windows' GUID.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// String formats g in lower case, without braces.
func (g GUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x", g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

// ParseGUID parses "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", with or
// without braces.
func ParseGUID(s string) GUID {
	s = strings.Trim(s, "{}")
	var b [16]byte
	j := 0
	for i := 0; i < len(s) && j < 32; i++ {
		c := s[i]
		var v byte
		switch {
		case c >= '0' && c <= '9':
			v = c - '0'
		case c >= 'a' && c <= 'f':
			v = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			v = c - 'A' + 10
		default:
			continue
		}
		if j%2 == 0 {
			b[j/2] = v << 4
		} else {
			b[j/2] |= v
		}
		j++
	}
	return guidFromBytes(b)
}

func guidFromBytes(b [16]byte) GUID {
	g := GUID{
		Data1: binary.BigEndian.Uint32(b[0:4]),
		Data2: binary.BigEndian.Uint16(b[4:6]),
		Data3: binary.BigEndian.Uint16(b[6:8]),
	}
	copy(g.Data4[:], b[8:])
	return g
}

func guidString(g GUID) string { return g.String() }

// pinterfaceNamespace is the namespace of the name-based GUIDs of
// parameterized WinRT types.
var pinterfaceNamespace = [16]byte{0x11, 0xf4, 0x7a, 0xd5, 0x7b, 0x73, 0x42, 0xc0, 0xab, 0xae, 0x87, 0x8b, 0x1e, 0x16, 0xad, 0xee}

// SignatureGUID is the IID of a parameterized interface or delegate,
// from its type signature: a version 5 UUID of the signature in the
// WinRT namespace.
func SignatureGUID(sig string) GUID {
	h := sha1.New()
	h.Write(pinterfaceNamespace[:])
	h.Write([]byte(sig))
	var b [16]byte
	copy(b[:], h.Sum(nil))
	b[6] = b[6]&0x0f | 0x50
	b[8] = b[8]&0x3f | 0x80
	return guidFromBytes(b)
}

// RuntimeClassSignature is a runtime class's part of a type signature:
// its name and default interface.
func RuntimeClassSignature(class string, defaultIID GUID) string {
	return "rc(" + class + ";{" + guidString(defaultIID) + "})"
}

// TypedEventHandlerIID is the IID of TypedEventHandler<sender, args>,
// given their signatures.
func TypedEventHandlerIID(sender, args string) GUID {
	return SignatureGUID("pinterface({9de1c534-6ae1-11e0-84e1-18a905bcc53f};" + sender + ";" + args + ")")
}

// NameGUID is a stable version 5 UUID for a name (an app's ID), in a
// giode namespace.
func NameGUID(name string) GUID {
	h := sha1.New()
	h.Write([]byte("giode:"))
	h.Write([]byte(name))
	var b [16]byte
	copy(b[:], h.Sum(nil))
	b[6] = b[6]&0x0f | 0x50
	b[8] = b[8]&0x3f | 0x80
	return guidFromBytes(b)
}

// ToastCLSID is the COM class of an app's toast activator, from its ID;
// giode build declares the same in the app's MSIX manifest.
func ToastCLSID(appID string) GUID { return NameGUID("toast:" + appID) }
