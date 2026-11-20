// SPDX-License-Identifier: Unlicense OR MIT

#include <TargetConditionals.h>
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#include <objc/runtime.h>
#if TARGET_OS_IPHONE
#import <UIKit/UIKit.h>
#define GIODE_APP_DELEGATE "_gioAppDelegate"
#else
#import <AppKit/AppKit.h>
#define GIODE_APP_DELEGATE "GioAppDelegate"
#endif
#include "_cgo_export.h"

// giode/push adds the remote notification methods to Gio's application
// delegate, and becomes the notification center's delegate to see
// notifications arriving in the foreground and taps on them. Other
// notifications (App.Notify's) go to the delegate it replaced.

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

static NSString *messageJSON(NSDictionary *userInfo, NSString *title, NSString *body) {
	NSMutableDictionary *data = [NSMutableDictionary dictionary];
	for (id k in userInfo) {
		if (![k isKindOfClass:[NSString class]] || [k isEqualToString:@"aps"]) {
			continue;
		}
		data[k] = userInfo[k];
	}
	if (title == nil || body == nil) {
		id alert = userInfo[@"aps"][@"alert"];
		if ([alert isKindOfClass:[NSString class]]) {
			body = body ?: alert;
		} else if ([alert isKindOfClass:[NSDictionary class]]) {
			title = title ?: alert[@"title"];
			body = body ?: alert[@"body"];
		}
	}
	NSMutableDictionary *m = [NSMutableDictionary dictionary];
	m[@"title"] = title ?: @"";
	m[@"body"] = body ?: @"";
	m[@"data"] = [NSJSONSerialization isValidJSONObject:data] ? data : @{};
	NSData *js = [NSJSONSerialization dataWithJSONObject:m options:0 error:nil];
	return js != nil ? [[NSString alloc] initWithData:js encoding:NSUTF8StringEncoding] : @"{}";
}

static void sendMessage(NSDictionary *userInfo, NSString *title, NSString *body, int opened) {
	giodePushMessage((char *)[messageJSON(userInfo, title, body) UTF8String], opened);
}

static BOOL isRemote(UNNotification *n) {
	return [n.request.trigger isKindOfClass:[UNPushNotificationTrigger class]];
}

@interface GiodePushCenterDelegate : NSObject <UNUserNotificationCenterDelegate>
@property (strong) id<UNUserNotificationCenterDelegate> next;
@end

@implementation GiodePushCenterDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center willPresentNotification:(UNNotification *)n withCompletionHandler:(void (^)(UNNotificationPresentationOptions))done {
	if (isRemote(n)) {
		// The app decides what to show while it's on screen, as on
		// Android.
		UNNotificationContent *c = n.request.content;
		sendMessage(c.userInfo, c.title, c.body, 0);
		done(UNNotificationPresentationOptionNone);
		return;
	}
	if ([self.next respondsToSelector:_cmd]) {
		[self.next userNotificationCenter:center willPresentNotification:n withCompletionHandler:done];
		return;
	}
	if (@available(iOS 14.0, macOS 11.0, *)) {
		done(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList | UNNotificationPresentationOptionSound);
	} else {
		done(UNNotificationPresentationOptionAlert | UNNotificationPresentationOptionSound);
	}
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center didReceiveNotificationResponse:(UNNotificationResponse *)r withCompletionHandler:(void (^)(void))done {
	BOOL custom = ![r.actionIdentifier isEqualToString:UNNotificationDefaultActionIdentifier];
	if (isRemote(r.notification) && custom && [self.next respondsToSelector:_cmd]) {
		// A button of a category from giode/notification.
		[self.next userNotificationCenter:center didReceiveNotificationResponse:r withCompletionHandler:done];
		return;
	}
	if (isRemote(r.notification)) {
		UNNotificationContent *c = r.notification.request.content;
		sendMessage(c.userInfo, c.title, c.body, 1);
		done();
		return;
	}
	if ([self.next respondsToSelector:_cmd]) {
		[self.next userNotificationCenter:center didReceiveNotificationResponse:r withCompletionHandler:done];
		return;
	}
	done();
}
@end

static GiodePushCenterDelegate *centerDelegate;

// reaches reports whether following the next links from d gets to target
// (giode/notification's delegate chains the same way).
static BOOL reaches(id d, id target) {
	for (int i = 0; d != nil && i < 8; i++) {
		if (d == target) {
			return YES;
		}
		d = [d respondsToSelector:@selector(next)] ? [d performSelector:@selector(next)] : nil;
	}
	return NO;
}

static void installCenterDelegate(void) {
	UNUserNotificationCenter *nc = [UNUserNotificationCenter currentNotificationCenter];
	if (centerDelegate == nil) {
		centerDelegate = [[GiodePushCenterDelegate alloc] init];
	}
	id<UNUserNotificationCenterDelegate> cur = nc.delegate;
	if (reaches(cur, centerDelegate)) {
		return;
	}
	if (cur != nil) {
		centerDelegate.next = cur;
	}
	nc.delegate = centerDelegate;
}

void giodePushReclaimDelegate(void) {
	onMain(^{
		installCenterDelegate();
	});
}

static void tokenReceived(id self, SEL _cmd, id app, NSData *token) {
	const unsigned char *b = token.bytes;
	NSMutableString *hex = [NSMutableString stringWithCapacity:token.length * 2];
	for (NSUInteger i = 0; i < token.length; i++) {
		[hex appendFormat:@"%02x", b[i]];
	}
	giodePushToken((char *)[hex UTF8String], NULL);
}

static void tokenFailed(id self, SEL _cmd, id app, NSError *err) {
	giodePushToken(NULL, (char *)[err.localizedDescription UTF8String]);
}

#if TARGET_OS_IPHONE
static void remoteReceived(id self, SEL _cmd, UIApplication *app, NSDictionary *info, void (^done)(UIBackgroundFetchResult)) {
	// An alert arriving in the foreground also reaches
	// willPresentNotification; deliver it once.
	BOOL alert = info[@"aps"][@"alert"] != nil;
	if (!(alert && app.applicationState == UIApplicationStateActive)) {
		sendMessage(info, nil, nil, 0);
	}
	done(UIBackgroundFetchResultNewData);
}

static BOOL willFinishLaunching(id self, SEL _cmd, UIApplication *app, NSDictionary *opts) {
	installCenterDelegate();
	return YES;
}
#else
static void remoteReceived(id self, SEL _cmd, NSApplication *app, NSDictionary *info) {
	BOOL alert = info[@"aps"][@"alert"] != nil;
	if (!(alert && app.isActive)) {
		sendMessage(info, nil, nil, 0);
	}
}

static void willFinishLaunching(id self, SEL _cmd, NSNotification *n) {
	installCenterDelegate();
}
#endif

__attribute__((constructor)) static void giodeInstallPush(void) {
	Class c = objc_getClass(GIODE_APP_DELEGATE);
	if (c == nil) {
		return;
	}
	class_addMethod(c, @selector(application:didRegisterForRemoteNotificationsWithDeviceToken:), (IMP)tokenReceived, "v@:@@");
	class_addMethod(c, @selector(application:didFailToRegisterForRemoteNotificationsWithError:), (IMP)tokenFailed, "v@:@@");
	BOOL added;
#if TARGET_OS_IPHONE
	class_addMethod(c, @selector(application:didReceiveRemoteNotification:fetchCompletionHandler:), (IMP)remoteReceived, "v@:@@@?");
	added = class_addMethod(c, @selector(application:willFinishLaunchingWithOptions:), (IMP)willFinishLaunching, "B@:@@");
#else
	class_addMethod(c, @selector(application:didReceiveRemoteNotification:), (IMP)remoteReceived, "v@:@@");
	added = class_addMethod(c, @selector(applicationWillFinishLaunching:), (IMP)willFinishLaunching, "v@:@");
#endif
	if (!added) {
		// Gio's delegate grew the method: set the delegate as soon as
		// the main queue runs.
		dispatch_async(dispatch_get_main_queue(), ^{
			installCenterDelegate();
		});
	}
}

void giodePushStart(void) {
	onMain(^{
		installCenterDelegate();
	});
}

void giodePushRegister(void) {
	onMain(^{
#if TARGET_OS_IPHONE
		[[UIApplication sharedApplication] registerForRemoteNotifications];
#else
		[NSApp registerForRemoteNotifications];
#endif
	});
}

void giodePushUnregister(void) {
	onMain(^{
#if TARGET_OS_IPHONE
		[[UIApplication sharedApplication] unregisterForRemoteNotifications];
#else
		[NSApp unregisterForRemoteNotifications];
#endif
	});
}

void giodePushRequestPermission(int rid) {
	UNAuthorizationOptions opts = UNAuthorizationOptionAlert | UNAuthorizationOptionSound | UNAuthorizationOptionBadge;
	[[UNUserNotificationCenter currentNotificationCenter] requestAuthorizationWithOptions:opts completionHandler:^(BOOL granted, NSError *err) {
		giodePushPermission(rid, granted ? 1 : 0);
	}];
}
