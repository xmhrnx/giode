//go:build darwin

package giode

/*
#include <pthread.h>
*/
import "C"

// onMainThread reports whether the caller runs on the main thread, as
// AppKit and UIKit callbacks (the Dock, the window's buttons,
// Services) do.
func onMainThread() bool {
	return C.pthread_main_np() != 0
}
