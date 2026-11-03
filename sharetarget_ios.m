#import <UIKit/UIKit.h>
#include <stdlib.h>
#include <string.h>
#include "_cgo_export.h"

// giodeInfoString returns an Info.plist string (malloc'ed) or NULL.
static char *giodeInfoString(NSString *key) {
	id v = [[NSBundle mainBundle] objectForInfoDictionaryKey:key];
	if (![v isKindOfClass:[NSString class]] || [(NSString *)v length] == 0) {
		return NULL;
	}
	return strdup([(NSString *)v UTF8String]);
}

// giodeShareInbox returns the folder in the App Group container where
// the share extension leaves shares (malloc'ed), or NULL when the app
// isn't a share target.
char *giodeShareInbox(void) {
	char *group = giodeInfoString(@"GiodeAppGroup");
	if (group == NULL) {
		return NULL;
	}
	NSString *g = [NSString stringWithUTF8String:group];
	free(group);
	NSURL *u = [[NSFileManager defaultManager] containerURLForSecurityApplicationGroupIdentifier:g];
	if (u == nil) {
		return NULL;
	}
	return strdup([[u URLByAppendingPathComponent:@"GiodeShare" isDirectory:YES].path UTF8String]);
}

// giodeShareScheme returns the URL scheme the share extension opens
// the app with (malloc'ed), or NULL.
char *giodeShareScheme(void) {
	return giodeInfoString(@"GiodeShareScheme");
}

// giodeWatchShares checks the inbox each time the app becomes active:
// at launch, and when the share extension opens it.
void giodeWatchShares(void) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		[[NSNotificationCenter defaultCenter] addObserverForName:UIApplicationDidBecomeActiveNotification object:nil queue:nil usingBlock:^(NSNotification *n) {
			giodeOnSharesMaybe();
		}];
	});
}
