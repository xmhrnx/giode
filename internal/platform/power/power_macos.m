//go:build darwin && !ios

@import AppKit;
#include "_cgo_export.h"

// The workspace's sleep and wake notifications.
void giodeWatchSleep(void) {
	static BOOL watching;
	if (watching) {
		return;
	}
	watching = YES;
	NSNotificationCenter *nc = [[NSWorkspace sharedWorkspace] notificationCenter];
	[nc addObserverForName:NSWorkspaceWillSleepNotification object:nil queue:nil usingBlock:^(NSNotification *n) {
		giodeSleepEvent(1);
	}];
	[nc addObserverForName:NSWorkspaceDidWakeNotification object:nil queue:nil usingBlock:^(NSNotification *n) {
		giodeSleepEvent(0);
	}];
}
