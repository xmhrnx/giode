// SPDX-License-Identifier: Unlicense OR MIT

//go:build darwin

#include <TargetConditionals.h>
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#if TARGET_OS_IPHONE
#import <UIKit/UIKit.h>
#else
#import <AppKit/AppKit.h>
#endif
#include "_cgo_export.h"

// giode/notification becomes the notification center's delegate for its
// own notifications and categories; others go to the delegate it
// replaced (App.Notify's, giode/push's).

#define KEY @"giode.notification"

static NSMutableSet<NSString *> *ourCategories;

static BOOL isOurs(UNNotification *n) {
	UNNotificationContent *c = n.request.content;
	return c.userInfo[KEY] != nil || [ourCategories containsObject:c.categoryIdentifier];
}

static BOOL isRemote(UNNotification *n) {
	return [n.request.trigger isKindOfClass:[UNPushNotificationTrigger class]];
}

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

@interface GiodeNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@property (strong) id<UNUserNotificationCenterDelegate> next;
@end

@implementation GiodeNotificationDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center willPresentNotification:(UNNotification *)n withCompletionHandler:(void (^)(UNNotificationPresentationOptions))done {
	if ((!isOurs(n) || isRemote(n)) && [self.next respondsToSelector:_cmd]) {
		// giode/push decides what to show of a push.
		[self.next userNotificationCenter:center willPresentNotification:n withCompletionHandler:done];
		return;
	}
	UNNotificationPresentationOptions sound = n.request.content.sound != nil ? UNNotificationPresentationOptionSound : 0;
	if (@available(iOS 14.0, macOS 11.0, *)) {
		done(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList | sound);
	} else {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
		done(UNNotificationPresentationOptionAlert | sound);
#pragma clang diagnostic pop
	}
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center didReceiveNotificationResponse:(UNNotificationResponse *)r withCompletionHandler:(void (^)(void))done {
	BOOL tap = [r.actionIdentifier isEqualToString:UNNotificationDefaultActionIdentifier];
	if (!isOurs(r.notification) || (isRemote(r.notification) && tap && [self.next respondsToSelector:_cmd])) {
		// Taps on pushes go to giode/push's Open.
		if ([self.next respondsToSelector:_cmd]) {
			[self.next userNotificationCenter:center didReceiveNotificationResponse:r withCompletionHandler:done];
		} else {
			done();
		}
		return;
	}
	NSString *action = r.actionIdentifier;
	if ([action isEqualToString:UNNotificationDismissActionIdentifier]) {
		done();
		return;
	}
	if ([action isEqualToString:UNNotificationDefaultActionIdentifier]) {
		action = @"";
	}
	NSString *reply = @"";
	if ([r isKindOfClass:[UNTextInputNotificationResponse class]]) {
		reply = ((UNTextInputNotificationResponse *)r).userText ?: @"";
	}
	UNNotificationContent *c = r.notification.request.content;
	NSString *payload = c.userInfo[KEY];
	if (![payload isKindOfClass:[NSString class]]) {
		// A push with one of the app's categories: its custom keys are
		// the data.
		NSMutableDictionary *data = [NSMutableDictionary dictionary];
		for (id k in c.userInfo) {
			id v = c.userInfo[k];
			if ([k isKindOfClass:[NSString class]] && ![k isEqualToString:@"aps"] && [v isKindOfClass:[NSString class]]) {
				data[k] = v;
			}
		}
		NSData *js = [NSJSONSerialization dataWithJSONObject:@{@"i": r.notification.request.identifier, @"d": data} options:0 error:nil];
		payload = js != nil ? [[NSString alloc] initWithData:js encoding:NSUTF8StringEncoding] : @"";
	}
	giodeNotificationResponse((char *)payload.UTF8String, (char *)action.UTF8String, (char *)reply.UTF8String);
	done();
}
@end

static GiodeNotificationDelegate *delegate;

// reaches reports whether following the next links from d gets to target.
static BOOL reaches(id d, id target) {
	for (int i = 0; d != nil && i < 8; i++) {
		if (d == target) {
			return YES;
		}
		d = [d respondsToSelector:@selector(next)] ? [d performSelector:@selector(next)] : nil;
	}
	return NO;
}

static void install(void) {
	UNUserNotificationCenter *nc = [UNUserNotificationCenter currentNotificationCenter];
	if (delegate == nil) {
		delegate = [[GiodeNotificationDelegate alloc] init];
		ourCategories = [NSMutableSet set];
	}
	id cur = nc.delegate;
	if (reaches(cur, delegate)) {
		return;
	}
	delegate.next = cur;
	nc.delegate = delegate;
}

int giodeNotificationAvailable(void) {
	return [NSBundle mainBundle].bundleIdentifier != nil;
}

void giodeNotificationInstall(void) {
	if (!giodeNotificationAvailable()) {
		return;
	}
	onMain(^{
		install();
	});
}

// The delegate must be set before launching ends to see the response
// that launched the app.
__attribute__((constructor)) static void giodeNotificationEarly(void) {
	if (!giodeNotificationAvailable()) {
		return;
	}
#if TARGET_OS_IPHONE
	NSString *name = UIApplicationDidFinishLaunchingNotification;
#else
	NSString *name = NSApplicationWillFinishLaunchingNotification;
#endif
	[[NSNotificationCenter defaultCenter] addObserverForName:name object:nil queue:nil usingBlock:^(NSNotification *n) {
		install();
	}];
}

static NSArray<UNNotificationAction *> *makeActions(NSArray *spec) {
	NSMutableArray *out = [NSMutableArray array];
	for (NSDictionary *a in spec) {
		UNNotificationActionOptions opts = 0;
		if ([a[@"destructive"] boolValue]) {
			opts |= UNNotificationActionOptionDestructive;
		}
		if ([a[@"foreground"] boolValue]) {
			opts |= UNNotificationActionOptionForeground;
		}
		if ([a[@"reply"] boolValue]) {
			[out addObject:[UNTextInputNotificationAction actionWithIdentifier:a[@"id"] title:a[@"title"] options:opts
				textInputButtonTitle:a[@"send"] textInputPlaceholder:a[@"placeholder"]]];
		} else {
			[out addObject:[UNNotificationAction actionWithIdentifier:a[@"id"] title:a[@"title"] options:opts]];
		}
	}
	return out;
}

// giodeNotificationCategory adds a category of actions, given as JSON,
// to the app's categories, then answers giodeNotificationDone(rid).
void giodeNotificationCategory(int rid, const char *cid, const char *actionsJSON) {
	NSString *ident = [NSString stringWithUTF8String:cid];
	NSData *js = [NSData dataWithBytes:actionsJSON length:strlen(actionsJSON)];
	NSArray *spec = [NSJSONSerialization JSONObjectWithData:js options:0 error:nil];
	UNNotificationCategory *cat = [UNNotificationCategory categoryWithIdentifier:ident actions:makeActions(spec) intentIdentifiers:@[] options:UNNotificationCategoryOptionCustomDismissAction];
	UNUserNotificationCenter *nc = [UNUserNotificationCenter currentNotificationCenter];
	onMain(^{
		install();
		[ourCategories addObject:ident];
	});
	[nc getNotificationCategoriesWithCompletionHandler:^(NSSet<UNNotificationCategory *> *cats) {
		NSMutableSet *all = [NSMutableSet set];
		for (UNNotificationCategory *c in cats) {
			if (![c.identifier isEqualToString:ident]) {
				[all addObject:c];
			}
		}
		[all addObject:cat];
		[nc setNotificationCategories:all];
		giodeNotificationDone(rid, NULL);
	}];
}

// giodeNotificationShow shows a notification when they're allowed and
// answers giodeNotificationDone(rid) with an error message, or "off".
void giodeNotificationShow(int rid, const char *ident, const char *title, const char *body, const char *thread, const char *category, const char *payload, int badge, int silent, int level) {
	NSString *i = [NSString stringWithUTF8String:ident];
	UNMutableNotificationContent *c = [[UNMutableNotificationContent alloc] init];
	c.title = [NSString stringWithUTF8String:title];
	c.body = [NSString stringWithUTF8String:body];
	c.threadIdentifier = [NSString stringWithUTF8String:thread];
	c.categoryIdentifier = [NSString stringWithUTF8String:category];
	c.userInfo = @{KEY: [NSString stringWithUTF8String:payload]};
	if (!silent) {
		c.sound = [UNNotificationSound defaultSound];
	}
	if (badge > 0) {
		c.badge = @(badge);
	}
	if (@available(iOS 15.0, macOS 12.0, *)) {
		switch (level) {
		case 1:
			c.interruptionLevel = UNNotificationInterruptionLevelPassive;
			break;
		case 2:
			c.interruptionLevel = UNNotificationInterruptionLevelTimeSensitive;
			break;
		}
	}
	UNUserNotificationCenter *nc = [UNUserNotificationCenter currentNotificationCenter];
	onMain(^{
		install();
	});
	[nc getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings *s) {
		if (s.authorizationStatus == UNAuthorizationStatusDenied || s.authorizationStatus == UNAuthorizationStatusNotDetermined) {
			giodeNotificationDone(rid, "off");
			return;
		}
		UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:i content:c trigger:nil];
		[nc addNotificationRequest:req withCompletionHandler:^(NSError *e) {
			giodeNotificationDone(rid, e != nil ? (char *)e.localizedDescription.UTF8String : NULL);
		}];
	}];
}

void giodeNotificationRemove(const char *ident) {
	NSArray *ids = @[[NSString stringWithUTF8String:ident]];
	UNUserNotificationCenter *nc = [UNUserNotificationCenter currentNotificationCenter];
	[nc removeDeliveredNotificationsWithIdentifiers:ids];
	[nc removePendingNotificationRequestsWithIdentifiers:ids];
}

void giodeNotificationRemoveAll(void) {
	UNUserNotificationCenter *nc = [UNUserNotificationCenter currentNotificationCenter];
	[nc removeAllDeliveredNotifications];
	[nc removeAllPendingNotificationRequests];
}

void giodeNotificationBadge(int n) {
	onMain(^{
#if TARGET_OS_IPHONE
		if (@available(iOS 16.0, *)) {
			[[UNUserNotificationCenter currentNotificationCenter] setBadgeCount:n withCompletionHandler:nil];
		} else {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
			[UIApplication sharedApplication].applicationIconBadgeNumber = n;
#pragma clang diagnostic pop
		}
#else
		NSApp.dockTile.badgeLabel = n > 0 ? [NSString stringWithFormat:@"%d", n] : nil;
#endif
	});
}
