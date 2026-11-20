#import <UIKit/UIKit.h>
#include <stdint.h>

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

void giodeHaptic(int kind) {
	onMain(^{
		switch (kind) {
		case 0:
			[[[UIImpactFeedbackGenerator alloc] initWithStyle:UIImpactFeedbackStyleLight] impactOccurred];
			break;
		case 1:
			[[[UIImpactFeedbackGenerator alloc] initWithStyle:UIImpactFeedbackStyleMedium] impactOccurred];
			break;
		case 2:
			[[[UIImpactFeedbackGenerator alloc] initWithStyle:UIImpactFeedbackStyleHeavy] impactOccurred];
			break;
		case 3:
			[[[UINotificationFeedbackGenerator alloc] init] notificationOccurred:UINotificationFeedbackTypeSuccess];
			break;
		default:
			[[[UINotificationFeedbackGenerator alloc] init] notificationOccurred:UINotificationFeedbackTypeWarning];
			break;
		}
	});
}

void giodeIdleTimerDisabled(int disabled) {
	onMain(^{
		[UIApplication sharedApplication].idleTimerDisabled = disabled != 0;
	});
}

uintptr_t giodeBeginBackgroundTask(const char *name) {
	__block UIBackgroundTaskIdentifier task = UIBackgroundTaskInvalid;
	NSString *n = [NSString stringWithUTF8String:name];
	task = [[UIApplication sharedApplication] beginBackgroundTaskWithName:n expirationHandler:^{
		// Time's up: iOS suspends the app once the task ends.
		[[UIApplication sharedApplication] endBackgroundTask:task];
		task = UIBackgroundTaskInvalid;
	}];
	return (uintptr_t)task;
}

void giodeEndBackgroundTask(uintptr_t task) {
	if ((UIBackgroundTaskIdentifier)task != UIBackgroundTaskInvalid) {
		[[UIApplication sharedApplication] endBackgroundTask:(UIBackgroundTaskIdentifier)task];
	}
}
