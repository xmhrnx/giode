#import <UIKit/UIKit.h>
#include <objc/runtime.h>
#include "_cgo_export.h"

void giodeWatchLifecycle(void) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		NSNotificationCenter *nc = [NSNotificationCenter defaultCenter];
		[nc addObserverForName:UIApplicationDidEnterBackgroundNotification object:nil queue:nil usingBlock:^(NSNotification *n) {
			giodeOnBackground();
		}];
		[nc addObserverForName:UIApplicationWillEnterForegroundNotification object:nil queue:nil usingBlock:^(NSNotification *n) {
			giodeOnForeground();
		}];
	});
}

// Universal links arrive as a browsing-web user activity, which Gio's
// app delegate doesn't handle: giode adds the method and hands the URL
// to Gio's URL path, which reaches Config.OnOpenURL.
extern void gio_onOpenURI(CFTypeRef uri);

static BOOL giodeContinueUserActivity(id self, SEL _cmd, UIApplication *app, NSUserActivity *activity, void (^handler)(NSArray *)) {
	if ([activity.activityType isEqualToString:NSUserActivityTypeBrowsingWeb] && activity.webpageURL != nil) {
		gio_onOpenURI((__bridge CFTypeRef)activity.webpageURL.absoluteString);
		return YES;
	}
	return NO;
}

__attribute__((constructor)) static void giodeInstallUserActivity(void) {
	Class c = objc_getClass("_gioAppDelegate");
	if (c == nil) {
		return;
	}
	SEL sel = @selector(application:continueUserActivity:restorationHandler:);
	struct objc_method_description d = protocol_getMethodDescription(@protocol(UIApplicationDelegate), sel, NO, YES);
	class_addMethod(c, sel, (IMP)giodeContinueUserActivity, d.types);
}
