//go:build ios

#import <UIKit/UIKit.h>
#import <PhotosUI/PhotosUI.h>
#include "_cgo_export.h"

// GiodeMediaDelegate copies the picked originals into a temporary
// directory, in the order picked, and answers giodeMediaPicked.
API_AVAILABLE(ios(14))
@interface GiodeMediaDelegate : NSObject <PHPickerViewControllerDelegate>
@end

// The picker holds its delegate weakly; one picker shows at a time.
static id mediaDelegate;

@implementation GiodeMediaDelegate
- (void)picker:(PHPickerViewController *)picker didFinishPicking:(NSArray<PHPickerResult *> *)results {
	[picker dismissViewControllerAnimated:YES completion:nil];
	NSUInteger n = results.count;
	if (n == 0) {
		giodeMediaPicked(NULL, 0, NULL);
		return;
	}
	NSFileManager *fm = [NSFileManager defaultManager];
	NSString *dir = [NSTemporaryDirectory() stringByAppendingPathComponent:[@"giode-media-" stringByAppendingString:[NSUUID UUID].UUIDString]];
	NSMutableArray *paths = [NSMutableArray arrayWithCapacity:n];
	for (NSUInteger i = 0; i < n; i++) {
		[paths addObject:[NSNull null]];
	}
	dispatch_group_t group = dispatch_group_create();
	__block NSString *failure = nil;
	for (NSUInteger i = 0; i < n; i++) {
		NSItemProvider *p = results[i].itemProvider;
		NSString *type = nil;
		if ([p hasItemConformingToTypeIdentifier:@"public.movie"]) {
			type = @"public.movie";
		} else if ([p hasItemConformingToTypeIdentifier:@"public.image"]) {
			type = @"public.image";
		} else {
			continue;
		}
		NSString *suggested = p.suggestedName;
		dispatch_group_enter(group);
		// The file exists only during the handler: copy it out.
		[p loadFileRepresentationForTypeIdentifier:type completionHandler:^(NSURL *url, NSError *err) {
			NSString *dest = nil;
			NSError *copyErr = err;
			if (url != nil) {
				NSString *name = url.lastPathComponent;
				if (suggested.length > 0 && url.pathExtension.length > 0) {
					name = [suggested stringByAppendingPathExtension:url.pathExtension];
				}
				// A folder per item: names can repeat.
				NSString *sub = [dir stringByAppendingPathComponent:[NSString stringWithFormat:@"%lu", (unsigned long)i]];
				[fm createDirectoryAtPath:sub withIntermediateDirectories:YES attributes:nil error:nil];
				NSString *path = [sub stringByAppendingPathComponent:name];
				if ([fm copyItemAtURL:url toURL:[NSURL fileURLWithPath:path] error:&copyErr]) {
					dest = path;
				}
			}
			@synchronized(paths) {
				if (dest != nil) {
					paths[i] = dest;
				} else if (failure == nil) {
					failure = copyErr != nil ? copyErr.localizedDescription : @"can't load the item";
				}
			}
			dispatch_group_leave(group);
		}];
	}
	dispatch_group_notify(group, dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
		NSMutableData *d = [NSMutableData data];
		for (NSUInteger i = 0; i < n; i++) {
			if ([paths[i] isKindOfClass:[NSString class]]) {
				const char *s = [(NSString *)paths[i] fileSystemRepresentation];
				[d appendBytes:s length:strlen(s) + 1];
			}
		}
		if (d.length == 0) {
			giodeMediaPicked(NULL, 0, failure != nil ? (char *)failure.UTF8String : (char *)"none of the items could be loaded");
			return;
		}
		giodeMediaPicked((char *)d.bytes, (int)d.length, NULL);
	});
}
@end

// giodePickMedia shows the Photos picker for photos and videos, up to
// limit (0 for no limit). It returns 0 before iOS 14.
int giodePickMedia(CFTypeRef vcRef, int limit, int photos, int videos) {
	if (@available(iOS 14, *)) {
		UIViewController *vc = (__bridge UIViewController *)vcRef;
		dispatch_async(dispatch_get_main_queue(), ^{
			PHPickerConfiguration *cfg = [[PHPickerConfiguration alloc] init];
			NSMutableArray<PHPickerFilter *> *filters = [NSMutableArray array];
			if (photos) {
				[filters addObject:[PHPickerFilter imagesFilter]];
			}
			if (videos) {
				[filters addObject:[PHPickerFilter videosFilter]];
			}
			cfg.filter = [PHPickerFilter anyFilterMatchingSubfilters:filters];
			cfg.selectionLimit = limit;
			// The originals, not converted to JPEG or H.264.
			cfg.preferredAssetRepresentationMode = PHPickerConfigurationAssetRepresentationModeCurrent;
			if (@available(iOS 15, *)) {
				cfg.selection = PHPickerConfigurationSelectionOrdered;
			}
			PHPickerViewController *picker = [[PHPickerViewController alloc] initWithConfiguration:cfg];
			GiodeMediaDelegate *d = [[GiodeMediaDelegate alloc] init];
			mediaDelegate = d;
			picker.delegate = d;
			[vc presentViewController:picker animated:YES completion:nil];
		});
		return 1;
	}
	return 0;
}
