package giode

import (
	"runtime"

	"github.com/xmhrnx/giode/elements"
)

// DeviceClass is a device's form factor.
type DeviceClass string

const (
	DevicePhone   DeviceClass = "phone"
	DeviceTablet  DeviceClass = "tablet"
	DeviceDesktop DeviceClass = "desktop"
)

// DeviceInfo describes the device the app runs on.
type DeviceInfo struct {
	// TotalRAM is the physical memory in bytes, 0 if unknown.
	TotalRAM uint64
	// Class is the form factor: phone or tablet on Android and iOS (by
	// the window's shorter side, 600dp and up being a tablet), desktop
	// elsewhere.
	Class DeviceClass
	// OS is runtime.GOOS; OSVersion its version ("14", "10.0.22631",
	// "6.8.0-45-generic"), and Model the hardware model ("Pixel 8",
	// "MacBookPro18,3"), each "" when unknown.
	OS        string
	OSVersion string
	Model     string
}

// Device returns information about the device, for choosing a
// suitable on-device model or layout. Class uses the window size, so
// call it once the first frame has been laid out (in OnReady or the
// view).
func Device() DeviceInfo {
	d := DeviceInfo{OS: runtime.GOOS, Class: DeviceDesktop}
	d.TotalRAM, d.OSVersion, d.Model = deviceDetails()
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		d.Class = DevicePhone
		w := elements.WindowSize()
		if short := min(w.X, w.Y); short > 0 && elements.PxToDp(short) >= 600 {
			d.Class = DeviceTablet
		}
	}
	return d
}
