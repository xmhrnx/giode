//go:build darwin && !ios

@import AppKit;
#include "_cgo_export.h"

// giodeOpenFiles shows an NSOpenPanel for files, several when multiple
// is set, limited to the newline-separated extensions exts when there
// are any, as a sheet on the app's window when there is one. It answers
// giodeFilesPicked with the paths, each ended by a NUL, or NULL when the
// user cancels.
void giodeOpenFiles(uintptr_t viewRef, char *exts, int multiple) {
	NSMutableArray<NSString *> *types = [NSMutableArray array];
	for (NSString *e in [@(exts) componentsSeparatedByString:@"\n"]) {
		if (e.length > 0) {
			[types addObject:e];
		}
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		NSOpenPanel *panel = [NSOpenPanel openPanel];
		panel.canChooseFiles = YES;
		panel.canChooseDirectories = NO;
		panel.allowsMultipleSelection = multiple != 0;
		panel.resolvesAliases = YES;
		if (types.count > 0) {
			// allowedContentTypes needs UniformTypeIdentifiers, which
			// macOS 10.15 lacks; allowedFileTypes still works.
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
			panel.allowedFileTypes = types;
#pragma clang diagnostic pop
		}
		void (^done)(NSModalResponse) = ^(NSModalResponse r) {
			if (r != NSModalResponseOK || panel.URLs.count == 0) {
				giodeFilesPicked(NULL, 0);
				return;
			}
			NSMutableData *buf = [NSMutableData data];
			for (NSURL *u in panel.URLs) {
				const char *p = u.path.fileSystemRepresentation;
				[buf appendBytes:p length:strlen(p) + 1];
			}
			giodeFilesPicked((char *)buf.bytes, (int)buf.length);
		};
		NSWindow *win = viewRef ? ((__bridge NSView *)(void *)viewRef).window : nil;
		if (win != nil) {
			[panel beginSheetModalForWindow:win completionHandler:done];
		} else {
			done([panel runModal]);
		}
	});
}
