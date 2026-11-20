//go:build darwin && !ios

@import AppKit;
#include <objc/runtime.h>
#include "_cgo_export.h"

// Universal links on macOS arrive as a browsing-web user activity;
// giode adds the delegate method Gio lacks and hands the URL to Gio's
// URL path (see lifecycle_ios.m).
extern void gio_onOpenURI(CFTypeRef uri);

static BOOL giodeContinueUserActivity(id self, SEL _cmd, NSApplication *app, NSUserActivity *activity, void (^handler)(NSArray *)) {
	if ([activity.activityType isEqualToString:NSUserActivityTypeBrowsingWeb] && activity.webpageURL != nil) {
		gio_onOpenURI((__bridge CFTypeRef)activity.webpageURL.absoluteString);
		return YES;
	}
	return NO;
}

__attribute__((constructor)) static void giodeInstallUserActivity(void) {
	Class c = objc_getClass("GioAppDelegate");
	if (c == nil) {
		return;
	}
	SEL sel = @selector(application:continueUserActivity:restorationHandler:);
	struct objc_method_description d = protocol_getMethodDescription(@protocol(NSApplicationDelegate), sel, NO, YES);
	class_addMethod(c, sel, (IMP)giodeContinueUserActivity, d.types);
}
