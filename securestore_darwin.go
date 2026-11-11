//go:build darwin

package giode

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <stdlib.h>
#include <string.h>
#include <TargetConditionals.h>
#include <Security/Security.h>

static CFMutableDictionaryRef giodeQuery(const char *service, const char *account) {
	CFMutableDictionaryRef q = CFDictionaryCreateMutable(NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFStringRef s = CFStringCreateWithCString(NULL, service, kCFStringEncodingUTF8);
	CFStringRef a = CFStringCreateWithCString(NULL, account, kCFStringEncodingUTF8);
	CFDictionarySetValue(q, kSecClass, kSecClassGenericPassword);
	// Strings that aren't UTF-8 give NULL, which neither the dictionary
	// nor CFRelease takes.
	if (s != NULL) {
		CFDictionarySetValue(q, kSecAttrService, s);
		CFRelease(s);
	}
	if (a != NULL) {
		CFDictionarySetValue(q, kSecAttrAccount, a);
		CFRelease(a);
	}
	return q;
}

static int giodeKeychainSet(const char *service, const char *account, const void *data, int len) {
	CFMutableDictionaryRef q = giodeQuery(service, account);
	SecItemDelete(q);
	CFDataRef d = CFDataCreate(NULL, data, len);
	CFDictionarySetValue(q, kSecValueData, d);
#if TARGET_OS_IPHONE
	CFDictionarySetValue(q, kSecAttrAccessible, kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly);
#endif
	OSStatus st = SecItemAdd(q, NULL);
	CFRelease(d);
	CFRelease(q);
	return (int)st;
}

static int giodeKeychainGet(const char *service, const char *account, void **out, int *outLen) {
	CFMutableDictionaryRef q = giodeQuery(service, account);
	CFDictionarySetValue(q, kSecReturnData, kCFBooleanTrue);
	CFDictionarySetValue(q, kSecMatchLimit, kSecMatchLimitOne);
	CFTypeRef res = NULL;
	OSStatus st = SecItemCopyMatching(q, &res);
	CFRelease(q);
	if (st != errSecSuccess) {
		return (int)st;
	}
	CFDataRef d = (CFDataRef)res;
	*outLen = (int)CFDataGetLength(d);
	*out = malloc(*outLen > 0 ? *outLen : 1);
	memcpy(*out, CFDataGetBytePtr(d), *outLen);
	CFRelease(res);
	return 0;
}

static int giodeKeychainDelete(const char *service, const char *account) {
	CFMutableDictionaryRef q = giodeQuery(service, account);
	OSStatus st = SecItemDelete(q);
	CFRelease(q);
	return (int)st;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Security framework status codes.
const (
	errSecSuccess      = 0
	errSecItemNotFound = -25300
)

func keychainErr(st C.int) error {
	return fmt.Errorf("giode: keychain error %d", int(st))
}

func secureSet(service, name string, value []byte) error {
	cs, cn := C.CString(service), C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	defer C.free(unsafe.Pointer(cn))
	var p unsafe.Pointer
	if len(value) > 0 {
		p = C.CBytes(value)
		defer C.free(p)
	}
	if st := C.giodeKeychainSet(cs, cn, p, C.int(len(value))); st != errSecSuccess {
		return keychainErr(st)
	}
	return nil
}

func secureGet(service, name string) ([]byte, error) {
	cs, cn := C.CString(service), C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	defer C.free(unsafe.Pointer(cn))
	var out unsafe.Pointer
	var n C.int
	switch st := C.giodeKeychainGet(cs, cn, &out, &n); st {
	case errSecSuccess:
		defer C.free(out)
		return C.GoBytes(out, n), nil
	case errSecItemNotFound:
		return nil, ErrNotFound
	default:
		return nil, keychainErr(st)
	}
}

func secureDelete(service, name string) error {
	cs, cn := C.CString(service), C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	defer C.free(unsafe.Pointer(cn))
	if st := C.giodeKeychainDelete(cs, cn); st != errSecSuccess && st != errSecItemNotFound {
		return keychainErr(st)
	}
	return nil
}
