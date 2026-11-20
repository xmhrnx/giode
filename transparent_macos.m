//go:build darwin && !ios

@import AppKit;
@import QuartzCore;

// giodeMakeTransparent clears the window's background and the view's
// layer, so what the app leaves transparent shows the desktop.
void giodeMakeTransparent(uintptr_t view) {
	NSView *v = (__bridge NSView *)(void *)view;
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *w = v.window;
		w.opaque = NO;
		w.backgroundColor = [NSColor clearColor];
		v.wantsLayer = YES;
		v.layer.opaque = NO;
		v.layer.backgroundColor = [NSColor clearColor].CGColor;
	});
}
