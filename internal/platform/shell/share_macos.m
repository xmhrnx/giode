//go:build darwin && !ios

@import AppKit;

// giodeSharePicker shows NSSharingServicePicker for text or files,
// anchored at the middle of the view's top edge.
void giodeSharePicker(uintptr_t view, const char *text, const char *paths) {
	NSView *v = (__bridge NSView *)(void *)view;
	NSMutableArray *items = [NSMutableArray array];
	NSString *t = [NSString stringWithUTF8String:text];
	if (t.length > 0) {
		[items addObject:t];
	}
	NSString *p = [NSString stringWithUTF8String:paths];
	if (p.length > 0) {
		for (NSString *path in [p componentsSeparatedByString:@"\n"]) {
			[items addObject:[NSURL fileURLWithPath:path]];
		}
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		NSSharingServicePicker *picker = [[NSSharingServicePicker alloc] initWithItems:items];
		NSRect r = NSMakeRect(NSMidX(v.bounds) - 1, v.isFlipped ? 0 : NSMaxY(v.bounds) - 2, 2, 2);
		[picker showRelativeToRect:r ofView:v preferredEdge:NSRectEdgeMinY];
	});
}
