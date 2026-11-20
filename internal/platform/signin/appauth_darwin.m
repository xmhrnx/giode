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
	UIWindow *any = nil;
	for (UIScene *scene in UIApplication.sharedApplication.connectedScenes) {
		if ([scene isKindOfClass:UIWindowScene.class]) {
			for (UIWindow *w in ((UIWindowScene *)scene).windows) {
				if (w.isKeyWindow) {
					return w;
				}
				if (any == nil) {
					any = w;
				}
			}
		}
	}
	return any;
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
		// initWithURL:callbackURLScheme: is deprecated from iOS 17.4,
		// whose replacement older systems lack; apps building with -Werror
		// and a recent deployment target mustn't fail on it.
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
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
#pragma clang diagnostic pop
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

// Passkeys: ASAuthorizationPlatformPublicKeyCredentialProvider, iOS 15
// and macOS 12. The app needs the webcredentials associated domain of
// the relying party.

static void giodePasskeySend(int ident, int kind, NSArray<NSData *> *parts, NSString *msg) {
	void *bufs[5] = {0};
	int lens[5] = {0};
	int n = parts.count < 5 ? (int)parts.count : 5;
	for (int i = 0; i < n; i++) {
		bufs[i] = (void *)parts[i].bytes;
		lens[i] = (int)parts[i].length;
	}
	giodePasskeyDone(ident, kind, bufs, lens, n, (char *)(msg ?: @"").UTF8String);
}

@interface GiodePasskeyDelegate : NSObject <ASAuthorizationControllerDelegate, ASAuthorizationControllerPresentationContextProviding>
@property int ident;
@property (weak) ASPresentationAnchor window;
@property (strong) ASAuthorizationController *controller;
// keepAlive holds the delegate, which the controller holds weakly, until
// the request ends.
@property (strong) GiodePasskeyDelegate *keepAlive;
@end

@implementation GiodePasskeyDelegate
- (ASPresentationAnchor)presentationAnchorForAuthorizationController:(ASAuthorizationController *)controller {
	GiodeAuthAnchor *a = [GiodeAuthAnchor new];
	a.window = self.window;
	return [a presentationAnchorForWebAuthenticationSession:(ASWebAuthenticationSession *_Nonnull)nil];
}

- (void)finish {
	self.controller = nil;
	self.keepAlive = nil;
}

- (void)authorizationController:(ASAuthorizationController *)controller didCompleteWithAuthorization:(ASAuthorization *)authorization {
	id cred = authorization.credential;
	NSData *empty = [NSData data];
	if (@available(iOS 15.0, macOS 12.0, *)) {
		if ([cred isKindOfClass:ASAuthorizationPlatformPublicKeyCredentialRegistration.class]) {
			ASAuthorizationPlatformPublicKeyCredentialRegistration *r = cred;
			giodePasskeySend(self.ident, 0, @[r.credentialID ?: empty, r.rawClientDataJSON ?: empty, r.rawAttestationObject ?: empty], nil);
			[self finish];
			return;
		}
		if ([cred isKindOfClass:ASAuthorizationPlatformPublicKeyCredentialAssertion.class]) {
			ASAuthorizationPlatformPublicKeyCredentialAssertion *r = cred;
			giodePasskeySend(self.ident, 1, @[r.credentialID ?: empty, r.rawClientDataJSON ?: empty,
				r.rawAuthenticatorData ?: empty, r.signature ?: empty, r.userID ?: empty], nil);
			[self finish];
			return;
		}
	}
	giodePasskeySend(self.ident, 4, @[], @"unexpected credential type");
	[self finish];
}

- (void)authorizationController:(ASAuthorizationController *)controller didCompleteWithError:(NSError *)error {
	int kind = 4;
	if ([error.domain isEqualToString:ASAuthorizationErrorDomain] && error.code == ASAuthorizationErrorCanceled) {
		kind = 2;
	}
	giodePasskeySend(self.ident, kind, @[], error.localizedDescription);
	[self finish];
}
@end

void giodePasskeyStart(uintptr_t viewRef, int ident, int create, char *rpID, void *challenge, int challengeLen,
	char *userName, void *userID, int userIDLen, void *ids, int *idLens, int idCount, char *uv) {
	NSString *rp = @(rpID);
	NSData *chal = [NSData dataWithBytes:challenge length:challengeLen];
	NSString *name = @(userName);
	NSData *uid = userIDLen > 0 ? [NSData dataWithBytes:userID length:userIDLen] : [NSData data];
	NSMutableArray<NSData *> *creds = [NSMutableArray array];
	char *p = ids;
	for (int i = 0; i < idCount; i++) {
		[creds addObject:[NSData dataWithBytes:p length:idLens[i]]];
		p += idLens[i];
	}
	NSString *uvs = @(uv);
	dispatch_async(dispatch_get_main_queue(), ^{
		if (@available(iOS 15.0, macOS 12.0, *)) {
			ASAuthorizationPlatformPublicKeyCredentialProvider *prov =
				[[ASAuthorizationPlatformPublicKeyCredentialProvider alloc] initWithRelyingPartyIdentifier:rp];
			ASAuthorizationPublicKeyCredentialUserVerificationPreference pref = ASAuthorizationPublicKeyCredentialUserVerificationPreferencePreferred;
			if ([uvs isEqualToString:@"required"]) {
				pref = ASAuthorizationPublicKeyCredentialUserVerificationPreferenceRequired;
			} else if ([uvs isEqualToString:@"discouraged"]) {
				pref = ASAuthorizationPublicKeyCredentialUserVerificationPreferenceDiscouraged;
			}
			NSMutableArray *descs = [NSMutableArray array];
			for (NSData *d in creds) {
				[descs addObject:[[ASAuthorizationPlatformPublicKeyCredentialDescriptor alloc] initWithCredentialID:d]];
			}
			ASAuthorizationRequest *req;
			if (create) {
				ASAuthorizationPlatformPublicKeyCredentialRegistrationRequest *r =
					[prov createCredentialRegistrationRequestWithChallenge:chal name:name userID:uid];
				r.userVerificationPreference = pref;
				if (descs.count > 0 && [r respondsToSelector:NSSelectorFromString(@"setExcludedCredentials:")]) {
					// iOS 17.4 and macOS 14.4; by name, for older SDKs.
					[r setValue:descs forKey:@"excludedCredentials"];
				}
				req = r;
			} else {
				ASAuthorizationPlatformPublicKeyCredentialAssertionRequest *r = [prov createCredentialAssertionRequestWithChallenge:chal];
				r.userVerificationPreference = pref;
				if (descs.count > 0) {
					r.allowedCredentials = descs;
				}
				req = r;
			}
			ASAuthorizationController *c = [[ASAuthorizationController alloc] initWithAuthorizationRequests:@[req]];
			GiodePasskeyDelegate *del = [GiodePasskeyDelegate new];
			del.ident = ident;
			if (viewRef) {
#if TARGET_OS_IPHONE
				del.window = ((__bridge UIViewController *)(void *)viewRef).view.window;
#else
				del.window = ((__bridge NSView *)(void *)viewRef).window;
#endif
			}
			del.controller = c;
			del.keepAlive = del;
			c.delegate = del;
			c.presentationContextProvider = del;
			[c performRequests];
		} else {
			giodePasskeySend(ident, 5, @[], @"passkeys need iOS 15 or macOS 12");
		}
	});
}
