//go:build android

package giode

import (
	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

// playSound plays the samples, as 16-bit PCM, through an AudioTrack.
func playSound(_ []byte, w wavInfo) error {
	return mobile.PlaySound(app.JavaVM(), app.AppContext(), w.pcm16(), w.rate, w.channels)
}
