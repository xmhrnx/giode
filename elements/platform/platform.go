// Package platform shows elements only on some platforms:
//
//	giode.OnlyOn(giode.PlatformWindows, func() giode.Element { return windowsTitleBar() })
//
// The platform is fixed when the app is built (Current is a constant),
// so OnlyOn costs one comparison, and the builders for other platforms
// are never called.
package platform

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/elements/internal/base"
)

// Platform is a set of platforms. Combine them with |:
// Windows|Linux.
type Platform uint8

const (
	Windows Platform = 1 << iota
	// MacOS is macOS, not iOS.
	MacOS
	// Linux is Linux desktops (and other Unix desktops: FreeBSD,
	// OpenBSD), not Android.
	Linux
	Android
	IOS
	// Web is the browser (WebAssembly).
	Web

	// Desktop is Windows, macOS and Linux; Mobile is Android and iOS.
	Desktop = Windows | MacOS | Linux
	Mobile  = Android | IOS
	// Any is every platform.
	Any = Desktop | Mobile | Web
)

// Current is the platform the app was built for.
const Current = current

// Is reports whether the app runs on one of p's platforms.
func Is(p Platform) bool { return p&Current != 0 }

// String names the platform, or the set.
func (p Platform) String() string {
	names := []string{"windows", "macos", "linux", "android", "ios", "web"}
	s := ""
	for i, n := range names {
		if p&(1<<i) != 0 {
			if s != "" {
				s += "|"
			}
			s += n
		}
	}
	if s == "" {
		return "none"
	}
	return s
}

// OnlyOn returns the element build makes on p's platforms, and an
// empty element (no size, nothing drawn) elsewhere, where build isn't
// called.
func OnlyOn(p Platform, build func() base.Element) base.Element {
	if !Is(p) || build == nil {
		return empty{}
	}
	if el := build(); el != nil {
		return el
	}
	return empty{}
}

// Views is an element for each platform, for ByPlatform. A specific
// platform's builder wins over Desktop or Mobile, and those over
// Default; a platform with none of them shows nothing.
type Views struct {
	Windows, MacOS, Linux, Android, IOS, Web func() base.Element
	Desktop, Mobile                          func() base.Element
	Default                                  func() base.Element
}

// ByPlatform returns the element the current platform's builder in v
// makes (see Views).
func ByPlatform(v Views) base.Element {
	var specific func() base.Element
	switch Current {
	case Windows:
		specific = v.Windows
	case MacOS:
		specific = v.MacOS
	case Linux:
		specific = v.Linux
	case Android:
		specific = v.Android
	case IOS:
		specific = v.IOS
	case Web:
		specific = v.Web
	}
	for _, b := range []func() base.Element{specific, group(v), v.Default} {
		if b != nil {
			return OnlyOn(Any, b)
		}
	}
	return empty{}
}

// group returns v's builder for the current platform's group.
func group(v Views) func() base.Element {
	switch {
	case Is(Desktop):
		return v.Desktop
	case Is(Mobile):
		return v.Mobile
	}
	return nil
}

// empty is an element with no size that draws nothing.
type empty struct{}

func (empty) Layout(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }
