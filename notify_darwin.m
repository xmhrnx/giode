//go:build darwin

@import Foundation;
@import UserNotifications;
#include "_cgo_export.h"

// GiodeNotifyDelegate shows the app's notifications while it's in
// front: without a delegate saying so, iOS and macOS hide them.
@interface GiodeNotifyDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation GiodeNotifyDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center willPresentNotification:(UNNotification *)n withCompletionHandler:(void (^)(UNNotificationPresentationOptions))done {
	if (@available(iOS 14.0, macOS 11.0, *)) {
		done(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList | UNNotificationPresentationOptionSound);
	} else {
		done(UNNotificationPresentationOptionAlert | UNNotificationPresentationOptionSound);
	}
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center didReceiveNotificationResponse:(UNNotificationResponse *)r withCompletionHandler:(void (^)(void))done {
	done();
}
@end

// The notification center holds its delegate weakly.
static GiodeNotifyDelegate *notifyDelegate;

// giodeNotifyAvailable reports whether the app has a bundle, without
// which UNUserNotificationCenter throws.
int giodeNotifyAvailable(void) {
	return [NSBundle mainBundle].bundleIdentifier != nil;
}

// giodeNotifyInstall makes GiodeNotifyDelegate the notification
// center's delegate, unless the app has one (giode/push chains to it).
void giodeNotifyInstall(void) {
	UNUserNotificationCenter *nc = [UNUserNotificationCenter currentNotificationCenter];
	if (notifyDelegate == nil) {
		notifyDelegate = [[GiodeNotifyDelegate alloc] init];
	}
	if (nc.delegate == nil) {
		nc.delegate = notifyDelegate;
	}
}

// giodeNotify asks for permission (the system prompts only the first
// time) and shows the notification, answering giodeNotified.
void giodeNotify(int rid, const char *title, const char *body) {
	NSString *t = [NSString stringWithUTF8String:title];
	NSString *b = [NSString stringWithUTF8String:body];
	UNUserNotificationCenter *nc = [UNUserNotificationCenter currentNotificationCenter];
	UNAuthorizationOptions opts = UNAuthorizationOptionAlert | UNAuthorizationOptionSound | UNAuthorizationOptionBadge;
	[nc requestAuthorizationWithOptions:opts completionHandler:^(BOOL granted, NSError *err) {
		if (!granted) {
			NSString *msg = err != nil ? err.localizedDescription : @"notifications aren't allowed for this app (turn them on in the system's notification settings)";
			giodeNotified(rid, (char *)msg.UTF8String);
			return;
		}
		UNMutableNotificationContent *c = [[UNMutableNotificationContent alloc] init];
		c.title = t;
		c.body = b;
		c.sound = [UNNotificationSound defaultSound];
		UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:[NSUUID UUID].UUIDString content:c trigger:nil];
		[nc addNotificationRequest:req withCompletionHandler:^(NSError *e) {
			giodeNotified(rid, e != nil ? (char *)e.localizedDescription.UTF8String : NULL);
		}];
	}];
}
