//go:build darwin && !ios

@import AppKit;

// giodeWindowButtons takes the miniaturize and close abilities from the
// window's style and disables the zoom button.
void giodeWindowButtons(uintptr_t view, int noMin, int noMax, int noClose) {
	NSView *v = (__bridge NSView *)(void *)view;
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *w = v.window;
		NSWindowStyleMask m = w.styleMask;
		if (noMin) {
			m &= ~NSWindowStyleMaskMiniaturizable;
		}
		if (noClose) {
			m &= ~NSWindowStyleMaskClosable;
		}
		w.styleMask = m;
		if (noMax) {
			[w standardWindowButton:NSWindowZoomButton].enabled = NO;
			w.collectionBehavior |= NSWindowCollectionBehaviorFullScreenNone;
		}
	});
}
