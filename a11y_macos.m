//go:build darwin && !ios

@import AppKit;
#include <objc/runtime.h>
#include "_cgo_export.h"

// VoiceOver: Gio's view gets accessibilityChildren, hit testing and the
// focused element; each node of Giode's tree is a GiodeAXElement, which
// asks Go for the node (as JSON, cached until the tree changes) when
// VoiceOver asks for its role, label, value or frame.

@class GiodeAXElement;

// GiodeAXView is a view's elements by node number.
@interface GiodeAXView : NSObject
@property (weak) NSView *view;
@property (strong) NSMutableDictionary<NSNumber *, GiodeAXElement *> *elements;
@end

@implementation GiodeAXView
@end

static NSMapTable<NSView *, GiodeAXView *> *axViews;

static GiodeAXView *axFor(NSView *view) {
	return axViews != nil ? [axViews objectForKey:view] : nil;
}

static NSDictionary *axNode(NSView *view, int32_t ident) {
	char *s = giodeAXNode((uintptr_t)(__bridge void *)view, ident);
	if (s == NULL) {
		return nil;
	}
	NSData *data = [NSData dataWithBytesNoCopy:s length:strlen(s) freeWhenDone:YES];
	id obj = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
	return [obj isKindOfClass:[NSDictionary class]] ? obj : nil;
}

static id axElement(NSView *view, int32_t ident);

// axFrame turns a frame in the view's pixels, from the top-left, into
// screen points.
static NSRect axFrame(NSView *view, double x, double y, double w, double h) {
	NSWindow *win = view.window;
	CGFloat scale = win != nil ? win.backingScaleFactor : 1;
	NSRect r = NSMakeRect(x / scale, y / scale, w / scale, h / scale);
	if (!view.isFlipped) {
		r.origin.y = view.bounds.size.height - r.origin.y - r.size.height;
	}
	r = [view convertRect:r toView:nil];
	return win != nil ? [win convertRectToScreen:r] : r;
}

@interface GiodeAXElement : NSAccessibilityElement
@property (weak) NSView *view;
@property int32_t ident;
@property int64_t gen;
@property (strong) NSDictionary *node;
@end

@implementation GiodeAXElement

- (NSDictionary *)current {
	int64_t gen = giodeAXGeneration();
	if (self.node == nil || self.gen != gen) {
		NSDictionary *n = axNode(self.view, self.ident);
		self.node = n;
		self.gen = gen;
	}
	return self.node;
}

- (BOOL)has:(NSString *)action {
	NSArray *actions = self.current[@"actions"];
	return [actions isKindOfClass:[NSArray class]] && [actions containsObject:action];
}

- (BOOL)perform:(NSString *)action arg:(NSString *)arg {
	return giodeAXAction((uintptr_t)(__bridge void *)self.view, self.ident, (char *)action.UTF8String, arg != nil ? (char *)arg.UTF8String : NULL) != 0;
}

- (NSString *)str:(NSString *)key {
	id v = self.current[key];
	return [v isKindOfClass:[NSString class]] && [v length] > 0 ? v : nil;
}

- (BOOL)flag:(NSString *)key {
	return [self.current[key] boolValue];
}

- (BOOL)isAccessibilityElement {
	return self.current != nil && [self flag:@"element"];
}

- (NSAccessibilityRole)accessibilityRole {
	return [self str:@"role"] ?: NSAccessibilityUnknownRole;
}

- (NSAccessibilitySubrole)accessibilitySubrole {
	return [self str:@"subrole"];
}

- (NSString *)accessibilityRoleDescription {
	NSString *d = [self str:@"roledesc"];
	if (d != nil) {
		return d;
	}
	return NSAccessibilityRoleDescription(self.accessibilityRole, self.accessibilitySubrole);
}

- (NSString *)accessibilityLabel {
	return [self str:@"label"];
}

- (id)accessibilityValue {
	id v = self.current[@"value"];
	return v == [NSNull null] ? nil : v;
}

- (void)setAccessibilityValue:(id)value {
	if ([value isKindOfClass:[NSString class]] && [self has:@"setvalue"]) {
		[self perform:@"setvalue" arg:value];
	}
}

- (NSString *)accessibilityHelp {
	return [self str:@"help"];
}

- (BOOL)isAccessibilityEnabled {
	return ![self flag:@"disabled"];
}

- (BOOL)isAccessibilityFocused {
	return [self flag:@"focused"];
}

- (void)setAccessibilityFocused:(BOOL)focused {
	if (focused && [self has:@"focus"]) {
		[self perform:@"focus" arg:nil];
	}
}

- (BOOL)isAccessibilitySelected {
	return [self flag:@"selected"];
}

- (BOOL)isAccessibilityExpanded {
	return [self.current[@"expanded"] intValue] == 1;
}

- (void)setAccessibilityExpanded:(BOOL)expanded {
	int now = [self.current[@"expanded"] intValue];
	if (now >= 0 && (now == 1) != expanded) {
		[self perform:expanded ? @"expand" : @"collapse" arg:nil];
	}
}

- (BOOL)isAccessibilityRequired {
	return [self flag:@"required"];
}

- (BOOL)isAccessibilityModal {
	return [self flag:@"modal"];
}

- (id)accessibilityMinValue {
	return [self flag:@"range"] ? (self.current[@"min"] ?: @0) : nil;
}

- (id)accessibilityMaxValue {
	return [self flag:@"range"] ? (self.current[@"max"] ?: @0) : nil;
}

- (NSInteger)accessibilityIndex {
	return [self.current[@"index"] integerValue];
}

- (NSInteger)accessibilityDisclosureLevel {
	return [self.current[@"level"] integerValue];
}

- (NSRect)accessibilityFrame {
	NSDictionary *n = self.current;
	NSView *view = self.view;
	if (n == nil || view == nil) {
		return NSZeroRect;
	}
	return axFrame(view, [n[@"x"] doubleValue], [n[@"y"] doubleValue], [n[@"w"] doubleValue], [n[@"h"] doubleValue]);
}

- (id)accessibilityParent {
	int32_t parent = [self.current[@"parent"] intValue];
	if (parent == 0) {
		return NSAccessibilityUnignoredAncestor(self.view);
	}
	return axElement(self.view, parent);
}

- (NSArray *)accessibilityChildren {
	NSMutableArray *out = [NSMutableArray array];
	for (NSNumber *c in self.current[@"children"]) {
		id e = axElement(self.view, c.intValue);
		if (e != nil) {
			[out addObject:e];
		}
	}
	return out;
}

- (id)accessibilityWindow {
	return self.view.window;
}

- (id)accessibilityTopLevelUIElement {
	return self.view.window;
}

- (BOOL)accessibilityPerformPress {
	return [self has:@"press"] && [self perform:@"press" arg:nil];
}

- (BOOL)accessibilityPerformIncrement {
	return [self has:@"increment"] && [self perform:@"increment" arg:nil];
}

- (BOOL)accessibilityPerformDecrement {
	return [self has:@"decrement"] && [self perform:@"decrement" arg:nil];
}

- (BOOL)isAccessibilitySelectorAllowed:(SEL)selector {
	if (selector == @selector(accessibilityPerformPress)) {
		return [self has:@"press"];
	}
	if (selector == @selector(accessibilityPerformIncrement) || selector == @selector(accessibilityPerformDecrement)) {
		return [self has:@"increment"];
	}
	if (selector == @selector(setAccessibilityValue:)) {
		return [self has:@"setvalue"];
	}
	if (selector == @selector(setAccessibilityFocused:)) {
		return [self has:@"focus"];
	}
	if (selector == @selector(setAccessibilityExpanded:)) {
		return [self.current[@"expanded"] intValue] >= 0;
	}
	return [super isAccessibilitySelectorAllowed:selector];
}

@end

// axElement returns node ident's element, made on first use; nil when
// the node is gone.
static id axElement(NSView *view, int32_t ident) {
	GiodeAXView *ax = axFor(view);
	if (ax == nil || ident <= 0) {
		return nil;
	}
	NSNumber *key = @(ident);
	GiodeAXElement *e = ax.elements[key];
	if (e == nil) {
		e = [[GiodeAXElement alloc] init];
		e.view = view;
		e.ident = ident;
		if (e.current == nil) {
			return nil;
		}
		ax.elements[key] = e;
	}
	return e;
}

// The view's own methods, added to Gio's view class.

static NSArray *viewChildren(NSView *self, SEL _cmd) {
	giodeAXWanted((uintptr_t)(__bridge void *)self);
	NSMutableArray *out = [NSMutableArray array];
	NSDictionary *root = axNode(self, 0);
	for (NSNumber *c in root[@"children"]) {
		id e = axElement(self, c.intValue);
		if (e != nil) {
			[out addObject:e];
		}
	}
	return out;
}

static id viewHitTest(NSView *self, SEL _cmd, NSPoint point) {
	giodeAXWanted((uintptr_t)(__bridge void *)self);
	NSWindow *win = self.window;
	if (win == nil) {
		return self;
	}
	NSPoint p = [self convertPoint:[win convertPointFromScreen:point] fromView:nil];
	if (!self.isFlipped) {
		p.y = self.bounds.size.height - p.y;
	}
	CGFloat scale = win.backingScaleFactor;
	int32_t ident = giodeAXAt((uintptr_t)(__bridge void *)self, p.x * scale, p.y * scale);
	id e = axElement(self, ident);
	return e ?: self;
}

static id viewFocused(NSView *self, SEL _cmd) {
	giodeAXWanted((uintptr_t)(__bridge void *)self);
	id e = axElement(self, giodeAXFocused((uintptr_t)(__bridge void *)self));
	return e ?: self;
}

void giodeAXInstall(uintptr_t viewRef) {
	NSView *view = (__bridge NSView *)(void *)viewRef;
	dispatch_async(dispatch_get_main_queue(), ^{
		if (axViews == nil) {
			axViews = [NSMapTable weakToStrongObjectsMapTable];
		}
		if ([axViews objectForKey:view] != nil) {
			return;
		}
		GiodeAXView *ax = [[GiodeAXView alloc] init];
		ax.view = view;
		ax.elements = [NSMutableDictionary dictionary];
		[axViews setObject:ax forKey:view];
		Class cls = [view class];
		class_replaceMethod(cls, @selector(accessibilityChildren), (IMP)viewChildren, "@@:");
		class_replaceMethod(cls, @selector(accessibilityHitTest:), (IMP)viewHitTest, "@@:{CGPoint=dd}");
		class_replaceMethod(cls, @selector(accessibilityFocusedUIElement), (IMP)viewFocused, "@@:");
	});
}

// giodeAXChanged posts what changed to VoiceOver: the focus (focus >= 0),
// the changed nodes' values, a new layout, and text to announce. It
// frees changed and announce.
void giodeAXChanged(uintptr_t viewRef, int32_t focus, int32_t *changed, int n, char *announce, int assertive, int layout) {
	NSView *view = (__bridge NSView *)(void *)viewRef;
	NSMutableArray<NSNumber *> *ids = [NSMutableArray arrayWithCapacity:n];
	for (int i = 0; i < n; i++) {
		[ids addObject:@(changed[i])];
	}
	free(changed);
	NSString *text = announce != NULL ? [NSString stringWithUTF8String:announce] : nil;
	free(announce);
	dispatch_async(dispatch_get_main_queue(), ^{
		GiodeAXView *ax = axFor(view);
		if (ax == nil) {
			return;
		}
		if (layout) {
			// Forget elements of nodes that are gone.
			for (NSNumber *key in ax.elements.allKeys) {
				if (ax.elements[key].current == nil) {
					[ax.elements removeObjectForKey:key];
				}
			}
			NSAccessibilityPostNotification(view, NSAccessibilityLayoutChangedNotification);
		}
		for (NSNumber *k in ids) {
			GiodeAXElement *e = ax.elements[k];
			if (e != nil) {
				NSAccessibilityPostNotification(e, NSAccessibilityValueChangedNotification);
			}
		}
		if (focus > 0) {
			id e = axElement(view, focus);
			if (e != nil) {
				NSAccessibilityPostNotification(e, NSAccessibilityFocusedUIElementChangedNotification);
			}
		}
		if (text != nil) {
			NSDictionary *info = @{
				NSAccessibilityAnnouncementKey: text,
				NSAccessibilityPriorityKey: @(assertive ? NSAccessibilityPriorityHigh : NSAccessibilityPriorityMedium),
			};
			NSAccessibilityPostNotificationWithUserInfo(NSApp, NSAccessibilityAnnouncementRequestedNotification, info);
		}
	});
}
