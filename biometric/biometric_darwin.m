// SPDX-License-Identifier: Unlicense OR MIT

#include <TargetConditionals.h>
#import <Foundation/Foundation.h>
#import <LocalAuthentication/LocalAuthentication.h>
#include "_cgo_export.h"

// Result codes, as in biometric.go.
enum { resOK, resCanceled, resNotEnrolled, resLockout, resFailed, resUnavailable };

// giodeBiometryKind returns biometric.go's Kind for the enrolled
// biometry: 0 none, 1 fingerprint, 2 face, 3 iris.
int giodeBiometryKind(void) {
	LAContext *ctx = [[LAContext alloc] init];
	if (![ctx canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:nil]) {
		return 0;
	}
	switch (ctx.biometryType) {
	case LABiometryTypeTouchID:
		return 1;
	case LABiometryTypeFaceID:
		return 2;
	default:
		if (@available(iOS 17.0, macOS 14.0, *)) {
			if (ctx.biometryType == LABiometryTypeOpticID) {
				return 3;
			}
		}
		return 0;
	}
}

static int giodeLACode(NSError *err) {
	switch (err.code) {
	case LAErrorUserCancel:
	case LAErrorSystemCancel:
	case LAErrorAppCancel:
	case LAErrorUserFallback:
		return resCanceled;
	case LAErrorBiometryNotEnrolled:
	case LAErrorPasscodeNotSet:
		return resNotEnrolled;
	case LAErrorBiometryLockout:
		return resLockout;
	case LAErrorBiometryNotAvailable:
		return resUnavailable;
	default:
		return resFailed;
	}
}

void giodeBiometricAuthenticate(int rid, const char *reason, int passcode) {
	NSString *why = [NSString stringWithUTF8String:reason];
	LAContext *ctx = [[LAContext alloc] init];
	LAPolicy policy = passcode ? LAPolicyDeviceOwnerAuthentication : LAPolicyDeviceOwnerAuthenticationWithBiometrics;
	if (!passcode) {
		// Without the passcode fallback, hide the "Enter Password" button.
		ctx.localizedFallbackTitle = @"";
	}
	NSError *err = nil;
	if (![ctx canEvaluatePolicy:policy error:&err]) {
		giodeBiometricDone(rid, giodeLACode(err), (char *)err.localizedDescription.UTF8String);
		return;
	}
#if TARGET_OS_IPHONE
	// Face ID without its usage text ends the app: refuse instead.
	if (ctx.biometryType == LABiometryTypeFaceID && [[NSBundle mainBundle] objectForInfoDictionaryKey:@"NSFaceIDUsageDescription"] == nil) {
		giodeBiometricDone(rid, resUnavailable, "set ios.usage.faceid in giode.toml (NSFaceIDUsageDescription) to use Face ID");
		return;
	}
#endif
	[ctx evaluatePolicy:policy localizedReason:why reply:^(BOOL ok, NSError *e) {
		if (ok) {
			giodeBiometricDone(rid, resOK, NULL);
		} else {
			giodeBiometricDone(rid, giodeLACode(e), (char *)e.localizedDescription.UTF8String);
		}
	}];
}
