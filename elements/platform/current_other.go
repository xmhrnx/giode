//go:build !windows && !darwin && !linux && !js

package platform

// Other Unix desktops (FreeBSD, OpenBSD, ...) count as Linux.
const current = Linux
