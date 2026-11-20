//go:build darwin

@import AuthenticationServices;
#include <TargetConditionals.h>
#include "_cgo_export.h"
#if TARGET_OS_IPHONE
@import UIKit;
#else
@import AppKit;
#endif

// GiodeAuthAnchor shows a sign-in sheet over the app's window.
@interface GiodeAuthAnchor : NSObject <ASWebAuthenticationPresentationContextProviding>
@property (weak) ASPresentationAnchor window;
@end

@implementation GiodeAuthAnchor
- (ASPresentationAnchor)presentationAnchorForWebAuthenticationSession:(ASWebAuthenticationSession *)session {
	if (self.window != nil) {
		return self.window;
	}
#if TARGET_OS_IPHONE
	for (UIScene *scene in UIApplication.sharedApplication.connectedScenes) {
		if ([scene isKindOfClass:UIWindowScene.class]) {
			for (UIWindow *w in ((UIWindowScene *)scene).windows) {
				if (w.isKeyWindow) {
					return w;
				}
			}
		}
	}
	return UIApplication.sharedApplication.windows.firstObject;
#else
	return NSApp.keyWindow ?: NSApp.mainWindow ?: NSApp.windows.firstObject;
#endif
}
@end

// sessions keeps the running sessions, by id, for cancelling.
static NSMutableDictionary<NSNumber *, ASWebAuthenticationSession *> *sessions;

void giodeAuthStart(uintptr_t viewRef, int ident, char *url, char *scheme, int ephemeral) {
	NSString *u = @(url);
	NSString *s = @(scheme);
	dispatch_async(dispatch_get_main_queue(), ^{
		if (sessions == nil) {
			sessions = [NSMutableDictionary dictionary];
		}
		GiodeAuthAnchor *anchor = [GiodeAuthAnchor new];
		if (viewRef) {
#if TARGET_OS_IPHONE
			anchor.window = ((__bridge UIViewController *)(void *)viewRef).view.window;
#else
			anchor.window = ((__bridge NSView *)(void *)viewRef).window;
#endif
		}
		NSURL *authURL = [NSURL URLWithString:u];
		if (authURL == nil) {
			giodeAuthDone(ident, (char *)"invalid authorization URL", 2);
			return;
		}
		ASWebAuthenticationSession *as = [[ASWebAuthenticationSession alloc]
			initWithURL:authURL
			callbackURLScheme:s
			completionHandler:^(NSURL *callback, NSError *err) {
				[sessions removeObjectForKey:@(ident)];
				(void)anchor; // the session holds its provider weakly
				if (callback != nil) {
					giodeAuthDone(ident, (char *)callback.absoluteString.UTF8String, 0);
				} else if ([err.domain isEqualToString:ASWebAuthenticationSessionErrorDomain] &&
					err.code == ASWebAuthenticationSessionErrorCodeCanceledLogin) {
					giodeAuthDone(ident, NULL, 1);
				} else {
					NSString *msg = err.localizedDescription ?: @"sign-in failed";
					giodeAuthDone(ident, (char *)msg.UTF8String, 2);
				}
			}];
		as.presentationContextProvider = anchor;
		as.prefersEphemeralWebBrowserSession = ephemeral != 0;
		sessions[@(ident)] = as;
		if (![as start]) {
			[sessions removeObjectForKey:@(ident)];
			giodeAuthDone(ident, (char *)"couldn't show the sign-in sheet", 2);
		}
	});
}

void giodeAuthCancel(int ident) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[sessions[@(ident)] cancel];
	});
}
