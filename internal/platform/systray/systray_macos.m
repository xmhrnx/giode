//go:build darwin && !ios

@import AppKit;
#include <objc/runtime.h>
#include "_cgo_export.h"

// The status item's owner in fyne.io/systray.
extern id owner;

void giodeTrayStartMain(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		giodeTrayStart();
	});
}

void giodeTrayHideItem(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (owner == nil) {
			return;
		}
		NSStatusItem *item = [owner valueForKey:@"statusItem"];
		item.visible = NO;
	});
}
