//go:build darwin && !ios

@import AppKit;
#include "_cgo_export.h"

// giodePickFolder shows an NSOpenPanel for one directory, as a sheet on
// the app's window when there is one, and answers giodeFolderPicked.
void giodePickFolder(uintptr_t viewRef, char *title, char *dir) {
	NSString *t = title[0] ? @(title) : nil;
	NSString *d = dir[0] ? @(dir) : nil;
	dispatch_async(dispatch_get_main_queue(), ^{
		NSOpenPanel *panel = [NSOpenPanel openPanel];
		panel.canChooseDirectories = YES;
		panel.canChooseFiles = NO;
		panel.canCreateDirectories = YES;
		panel.allowsMultipleSelection = NO;
		if (t != nil) {
			panel.message = t;
		}
		if (d != nil) {
			panel.directoryURL = [NSURL fileURLWithPath:d isDirectory:YES];
		}
		void (^done)(NSModalResponse) = ^(NSModalResponse r) {
			if (r == NSModalResponseOK && panel.URL != nil) {
				giodeFolderPicked((char *)panel.URL.path.fileSystemRepresentation, 1);
			} else {
				giodeFolderPicked(NULL, 0);
			}
		};
		NSWindow *win = viewRef ? ((__bridge NSView *)(void *)viewRef).window : nil;
		if (win != nil) {
			[panel beginSheetModalForWindow:win completionHandler:done];
		} else {
			done([panel runModal]);
		}
	});
}
