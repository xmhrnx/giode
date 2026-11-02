// Package review asks the user to rate the app without leaving it:
// Google Play's in-app review on Android, the App Store's rating prompt
// on iOS and macOS. Elsewhere Request returns giode.ErrNotSupported.
//
// The stores limit how often the prompt appears (the App Store three
// times a year) and don't report whether it did or what the user chose,
// so ask at a natural pause after the user did something worthwhile,
// not from a button: for a "Rate us" button, open the store page with
// App.OpenURL.
package review

import (
	"context"

	"github.com/xmhrnx/giode"
)

// Request asks the store to show its rating prompt. It returns once
// the store is done, whether or not it showed anything.
func Request(ctx context.Context, a *giode.App) error {
	return request(ctx, a)
}
