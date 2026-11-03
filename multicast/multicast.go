// Package multicast lets an Android app receive multicast and
// broadcast packets: LAN discovery (mDNS, SSDP, LocalSend's multicast
// announcements) that Android otherwise filters out on Wi-Fi to save
// battery.
//
//	release, err := multicast.Acquire("nearby discovery")
//	defer release()
//
// On Android it holds a WifiManager.MulticastLock until released; the
// package adds the CHANGE_WIFI_MULTICAST_STATE permission to the
// manifest (a normal permission, granted at install). Hold it only while
// discovering, as it costs battery. Elsewhere multicast needs no lock:
// Acquire returns a release function that does nothing.
package multicast

// Acquire starts receiving multicast on Android until release is
// called. tag names the lock in battery statistics. Calling release
// more than once is harmless.
func Acquire(tag string) (release func(), err error) {
	if tag == "" {
		tag = "giode"
	}
	return acquire(tag)
}
