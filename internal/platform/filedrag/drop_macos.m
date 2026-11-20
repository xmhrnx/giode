//go:build darwin && !ios

@import AppKit;
#include <objc/runtime.h>
#include "_cgo_export.h"

// File drags onto Gio's view: NSDraggingDestination methods added to
// its class at run time, and the view registered for file URLs.

static NSPoint dragPixels(id self, id<NSDraggingInfo> info) {
	NSView *v = self;
	NSPoint p = [v convertPoint:info.draggingLocation fromView:nil];
	if (!v.isFlipped) {
		p.y = v.bounds.size.height - p.y;
	}
	CGFloat s = v.window.backingScaleFactor;
	return NSMakePoint(p.x * s, p.y * s);
}

static BOOL hasFiles(id<NSDraggingInfo> info) {
	return [info.draggingPasteboard canReadObjectForClasses:@[NSURL.class] options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
}

static NSDragOperation giodeDraggingEntered(id self, SEL _cmd, id<NSDraggingInfo> info) {
	if (!hasFiles(info)) {
		return NSDragOperationNone;
	}
	NSPoint p = dragPixels(self, info);
	giodeFileDragEvent(1, p.x, p.y);
	return NSDragOperationCopy;
}

static NSDragOperation giodeDraggingUpdated(id self, SEL _cmd, id<NSDraggingInfo> info) {
	if (!hasFiles(info)) {
		return NSDragOperationNone;
	}
	NSPoint p = dragPixels(self, info);
	giodeFileDragEvent(2, p.x, p.y);
	return NSDragOperationCopy;
}

static void giodeDraggingExited(id self, SEL _cmd, id<NSDraggingInfo> info) {
	NSPoint p = info ? dragPixels(self, info) : NSZeroPoint;
	giodeFileDragEvent(3, p.x, p.y);
}

static BOOL giodePerformDragOperation(id self, SEL _cmd, id<NSDraggingInfo> info) {
	NSArray<NSURL *> *urls = [info.draggingPasteboard readObjectsForClasses:@[NSURL.class] options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
	NSMutableString *paths = [NSMutableString string];
	for (NSURL *u in urls) {
		if (u.fileURL && u.path) {
			[paths appendString:u.path];
			[paths appendString:@"\n"];
		}
	}
	NSPoint p = dragPixels(self, info);
	giodeFileDrop((char *)paths.UTF8String, p.x, p.y);
	return urls.count > 0;
}

void giodeEnableFileDrop(uintptr_t view) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSView *v = (__bridge NSView *)(void *)view;
		Class c = v.class;
		char op[16], ok[16];
		snprintf(op, sizeof(op), "%s@:@", @encode(NSDragOperation));
		snprintf(ok, sizeof(ok), "%s@:@", @encode(BOOL));
		class_addMethod(c, @selector(draggingEntered:), (IMP)giodeDraggingEntered, op);
		class_addMethod(c, @selector(draggingUpdated:), (IMP)giodeDraggingUpdated, op);
		class_addMethod(c, @selector(draggingExited:), (IMP)giodeDraggingExited, "v@:@");
		class_addMethod(c, @selector(performDragOperation:), (IMP)giodePerformDragOperation, ok);
		[v registerForDraggedTypes:@[NSPasteboardTypeFileURL]];
	});
}
