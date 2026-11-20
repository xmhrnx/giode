#import <UIKit/UIKit.h>
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
