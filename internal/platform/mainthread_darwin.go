//go:build darwin

package platform

/*
#include <pthread.h>
*/
import "C"

// OnMainThread reports whether the caller runs on the main thread, as
// AppKit and UIKit callbacks (the Dock, the window's buttons,
// Services) do.
func OnMainThread() bool {
	return C.pthread_main_np() != 0
}
