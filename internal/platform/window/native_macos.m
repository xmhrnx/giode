//go:build darwin && !ios

@import AppKit;
#include <objc/runtime.h>
#include "_cgo_export.h"

static NSWindow *windowOf(uintptr_t view) {
	NSView *v = (__bridge NSView *)(void *)view;
	return v.window;
}

// giodeHidden is set while giode keeps the window hidden (HideWindow,
// CloseToTray). Gio goes on drawing into a window that is off screen,
// and its Metal layer never lets nextDrawable time out: a hidden window
// gets no drawables back, so after a few frames the main thread would
// wait forever, and the window could never be shown again. Gio's view
// doesn't draw while the window is hidden; showing it draws again.
static BOOL giodeHidden;

static void (*gioDisplayLayer)(id, SEL, CALayer *);
static void (*gioDrawRect)(id, SEL, NSRect);

static BOOL giodeSkipDraw(NSView *v) {
	return giodeHidden && !v.window.visible;
}

static void giodeDisplayLayer(id self, SEL _cmd, CALayer *layer) {
	if (!giodeSkipDraw(self)) {
		gioDisplayLayer(self, _cmd, layer);
	}
}

static void giodeDrawRect(id self, SEL _cmd, NSRect r) {
	if (!giodeSkipDraw(self)) {
		gioDrawRect(self, _cmd, r);
	}
}

__attribute__((constructor)) static void giodeInstallHiddenDraw(void) {
	Class c = objc_getClass("GioView");
	if (c == nil) {
		return;
	}
	Method m = class_getInstanceMethod(c, @selector(displayLayer:));
	if (m != NULL) {
		gioDisplayLayer = (void (*)(id, SEL, CALayer *))method_setImplementation(m, (IMP)giodeDisplayLayer);
	}
	m = class_getInstanceMethod(c, @selector(drawRect:));
	if (m != NULL) {
		gioDrawRect = (void (*)(id, SEL, NSRect))method_setImplementation(m, (IMP)giodeDrawRect);
	}
}

void giodeHideWindow(uintptr_t view) {
	dispatch_async(dispatch_get_main_queue(), ^{
		giodeHidden = YES;
		[windowOf(view) orderOut:nil];
	});
}

void giodeShowWindow(uintptr_t view) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSView *v = (__bridge NSView *)(void *)view;
		NSWindow *w = v.window;
		giodeHidden = NO;
		if (w.miniaturized) {
			[w deminiaturize:nil];
		}
		[NSApp activateIgnoringOtherApps:YES];
		[w makeKeyAndOrderFront:nil];
		// The frames skipped while hidden: draw the window as it is now.
		[v setNeedsDisplay:YES];
	});
}

static BOOL giodeWindowShouldClose(id self, SEL _cmd, NSWindow *sender) {
	if (giodeVetoClose()) {
		return NO;
	}
	if (giodeShouldHideOnClose()) {
		giodeHidden = YES;
		[sender orderOut:nil];
		giodeHiddenOnClose();
		return NO;
	}
	return YES;
}

// giodeInterceptClose gives Gio's window delegate a windowShouldClose:
// that asks Config.OnCloseRequest, and hides the window instead for
// CloseToTray.
void giodeInterceptClose(uintptr_t view) {
	dispatch_async(dispatch_get_main_queue(), ^{
		id del = windowOf(view).delegate;
		if (del == nil) {
			return;
		}
		struct objc_method_description d = protocol_getMethodDescription(@protocol(NSWindowDelegate), @selector(windowShouldClose:), NO, YES);
		class_replaceMethod([del class], @selector(windowShouldClose:), (IMP)giodeWindowShouldClose, d.types);
	});
}

static BOOL giodeShouldHandleReopen(id self, SEL _cmd, NSApplication *app, BOOL visible) {
	giodeReopen();
	return YES;
}

// giodeHandleReopen shows a hidden window when the Dock icon is
// clicked.
void giodeHandleReopen(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		id del = [NSApp delegate];
		if (del == nil) {
			return;
		}
		SEL sel = @selector(applicationShouldHandleReopen:hasVisibleWindows:);
		struct objc_method_description d = protocol_getMethodDescription(@protocol(NSApplicationDelegate), sel, NO, YES);
		class_addMethod([del class], sel, (IMP)giodeShouldHandleReopen, d.types);
	});
}

// giodeMenuBarApp removes the Dock icon and app menu. Gio's delegate
// makes the app a regular one when it finishes launching, so the
// policy is set again after that.
void giodeMenuBarApp(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
	});
	[[NSNotificationCenter defaultCenter] addObserverForName:NSApplicationDidFinishLaunchingNotification
	                                                  object:nil
	                                                   queue:[NSOperationQueue mainQueue]
	                                              usingBlock:^(NSNotification *note) {
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
	}];
}
