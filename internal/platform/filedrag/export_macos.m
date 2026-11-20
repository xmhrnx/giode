//go:build darwin && !ios

@import AppKit;
#include "_cgo_export.h"

// The drag's source: copies, reports the end to Go, and gives Gio's view
// the mouse-up the drag took, or Gio takes the button for still held.
@interface GiodeExportSource : NSObject <NSDraggingSource>
@property (weak) NSView *view;
@end

@implementation GiodeExportSource
- (NSDragOperation)draggingSession:(NSDraggingSession *)session sourceOperationMaskForDraggingContext:(NSDraggingContext)context {
	return NSDragOperationCopy;
}

- (void)draggingSession:(NSDraggingSession *)session endedAtPoint:(NSPoint)screenPoint operation:(NSDragOperation)operation {
	NSView *v = self.view;
	NSWindow *w = v.window;
	if (w != nil) {
		NSEvent *up = [NSEvent mouseEventWithType:NSEventTypeLeftMouseUp
		                                 location:[w mouseLocationOutsideOfEventStream]
		                            modifierFlags:0
		                                timestamp:NSProcessInfo.processInfo.systemUptime
		                             windowNumber:w.windowNumber
		                                  context:nil
		                              eventNumber:0
		                               clickCount:1
		                                 pressure:0];
		[v mouseUp:up];
	}
	giodeExportEnded(operation != NSDragOperationNone);
}
@end

static GiodeExportSource *exportSource;

void giodeStartExport(uintptr_t view, char *files, char *text) {
	NSString *fileList = [NSString stringWithUTF8String:files];
	NSString *str = [NSString stringWithUTF8String:text];
	dispatch_async(dispatch_get_main_queue(), ^{
		NSView *v = (__bridge NSView *)(void *)view;
		NSWindow *w = v.window;
		if (w == nil) {
			giodeExportEnded(0);
			return;
		}
		NSPoint at = [v convertPoint:[w mouseLocationOutsideOfEventStream] fromView:nil];
		NSMutableArray<NSDraggingItem *> *items = [NSMutableArray array];
		NSRect frame = NSMakeRect(at.x - 16, at.y - 16, 32, 32);
		for (NSString *p in [fileList componentsSeparatedByString:@"\n"]) {
			if (p.length == 0) {
				continue;
			}
			NSDraggingItem *it = [[NSDraggingItem alloc] initWithPasteboardWriter:[NSURL fileURLWithPath:p]];
			[it setDraggingFrame:frame contents:[[NSWorkspace sharedWorkspace] iconForFile:p]];
			frame = NSOffsetRect(frame, 4, -4);
			[items addObject:it];
		}
		if (str.length > 0) {
			NSDraggingItem *it = [[NSDraggingItem alloc] initWithPasteboardWriter:str];
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
			NSImage *icon = [[NSWorkspace sharedWorkspace] iconForFileType:@"txt"];
#pragma clang diagnostic pop
			[it setDraggingFrame:frame contents:icon];
			[items addObject:it];
		}
		if (items.count == 0) {
			giodeExportEnded(0);
			return;
		}
		// The drag continues the button press: the current event when
		// it's the mouse's, else one made for it.
		NSEvent *ev = NSApp.currentEvent;
		if (ev == nil || (ev.type != NSEventTypeLeftMouseDragged && ev.type != NSEventTypeLeftMouseDown)) {
			ev = [NSEvent mouseEventWithType:NSEventTypeLeftMouseDragged
			                        location:[w mouseLocationOutsideOfEventStream]
			                   modifierFlags:0
			                       timestamp:NSProcessInfo.processInfo.systemUptime
			                    windowNumber:w.windowNumber
			                         context:nil
			                     eventNumber:0
			                      clickCount:1
			                        pressure:1];
		}
		exportSource = [GiodeExportSource new];
		exportSource.view = v;
		[v beginDraggingSessionWithItems:items event:ev source:exportSource];
	});
}
