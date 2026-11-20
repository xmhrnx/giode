//go:build darwin && !ios

@import AppKit;
#include "_cgo_export.h"

// GiodeServices receives the app's Services (its context actions): the
// NSMessage of each NSServices entry is giodeContextAction and its
// NSUserData the action's ID.
@interface GiodeServices : NSObject
@end

@implementation GiodeServices
- (void)giodeContextAction:(NSPasteboard *)pboard userData:(NSString *)userData error:(NSString **)error {
	NSArray<NSURL *> *urls = [pboard readObjectsForClasses:@[[NSURL class]]
	                                               options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
	NSMutableArray<NSString *> *paths = [NSMutableArray array];
	for (NSURL *u in urls) {
		[paths addObject:u.path];
	}
	NSString *joined = [paths componentsJoinedByString:@"\n"];
	giodeServiceInvoked((char *)userData.UTF8String, (char *)joined.UTF8String);
}
@end

static GiodeServices *giodeServices;

void giodeRegisterServices(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		giodeServices = [GiodeServices new];
		[NSApp setServicesProvider:giodeServices];
		NSUpdateDynamicServices();
	});
}
