#import <UIKit/UIKit.h>
#import <PhotosUI/PhotosUI.h>
#import <AVFoundation/AVFoundation.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <objc/runtime.h>
#include "_cgo_export.h"

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

API_AVAILABLE(ios(14))
@interface GiodePhotoDelegate : NSObject <PHPickerViewControllerDelegate>
@property int requestID;
@property int maxSide;
@end

// pending keeps delegates alive until their picker finishes.
static NSMutableSet *pending;

static NSData *giodeJPEG(UIImage *img, int maxSide) {
	CGSize size = img.size;
	CGFloat longest = MAX(size.width, size.height);
	if (maxSide > 0 && longest > maxSide) {
		CGFloat s = maxSide / longest;
		size = CGSizeMake(floor(size.width * s), floor(size.height * s));
	}
	UIGraphicsImageRendererFormat *f = [UIGraphicsImageRendererFormat defaultFormat];
	f.scale = 1;
	UIGraphicsImageRenderer *r = [[UIGraphicsImageRenderer alloc] initWithSize:size format:f];
	// Drawing applies the image's orientation, so the JPEG is upright.
	UIImage *out = [r imageWithActions:^(UIGraphicsImageRendererContext *ctx) {
		[img drawInRect:CGRectMake(0, 0, size.width, size.height)];
	}];
	return UIImageJPEGRepresentation(out, 0.9);
}

@implementation GiodePhotoDelegate
- (void)picker:(PHPickerViewController *)picker didFinishPicking:(NSArray<PHPickerResult *> *)results API_AVAILABLE(ios(14)) {
	[picker dismissViewControllerAnimated:YES completion:nil];
	int rid = self.requestID;
	int maxSide = self.maxSide;
	[pending removeObject:self];
	if (results.count == 0) {
		giodePhotoDone(rid, NULL, 0, NULL, NULL);
		return;
	}
	NSItemProvider *p = results[0].itemProvider;
	NSString *name = p.suggestedName ?: @"photo";
	if (![p canLoadObjectOfClass:[UIImage class]]) {
		giodePhotoDone(rid, NULL, 0, NULL, (char *)"the picked item isn't an image");
		return;
	}
	// UIImage decodes HEIC, so it never reaches the app.
	[p loadObjectOfClass:[UIImage class] completionHandler:^(id<NSItemProviderReading> obj, NSError *err) {
		if (err != nil || obj == nil) {
			NSString *msg = err != nil ? err.localizedDescription : @"can't load the image";
			giodePhotoDone(rid, NULL, 0, NULL, (char *)msg.UTF8String);
			return;
		}
		NSData *jpeg = giodeJPEG((UIImage *)obj, maxSide);
		giodePhotoDone(rid, (void *)jpeg.bytes, (int)jpeg.length, (char *)name.UTF8String, NULL);
	}];
}
@end

int giodePickPhoto(CFTypeRef vcRef, int rid, int maxSide) {
	if (@available(iOS 14, *)) {
		UIViewController *vc = (__bridge UIViewController *)vcRef;
		onMain(^{
			PHPickerConfiguration *cfg = [[PHPickerConfiguration alloc] init];
			cfg.filter = [PHPickerFilter imagesFilter];
			cfg.selectionLimit = 1;
			PHPickerViewController *picker = [[PHPickerViewController alloc] initWithConfiguration:cfg];
			GiodePhotoDelegate *d = [[GiodePhotoDelegate alloc] init];
			d.requestID = rid;
			d.maxSide = maxSide;
			if (pending == nil) {
				pending = [NSMutableSet set];
			}
			[pending addObject:d];
			picker.delegate = d;
			[vc presentViewController:picker animated:YES completion:nil];
		});
		return 1;
	}
	return 0;
}

// GiodePhotosDelegate collects a multiple pick, in the order picked.
@interface GiodePhotosDelegate : NSObject <PHPickerViewControllerDelegate>
@property(nonatomic) int requestID;
@property(nonatomic) int maxSide;
@end

@implementation GiodePhotosDelegate
- (void)picker:(PHPickerViewController *)picker didFinishPicking:(NSArray<PHPickerResult *> *)results API_AVAILABLE(ios(14)) {
	[picker dismissViewControllerAnimated:YES completion:nil];
	int rid = self.requestID;
	int maxSide = self.maxSide;
	[pending removeObject:self];
	NSUInteger n = results.count;
	if (n == 0) {
		giodePhotosDone(rid, 0, NULL);
		return;
	}
	NSMutableArray *jpegs = [NSMutableArray arrayWithCapacity:n];
	NSMutableArray *names = [NSMutableArray arrayWithCapacity:n];
	for (NSUInteger i = 0; i < n; i++) {
		[jpegs addObject:[NSNull null]];
		[names addObject:@"photo"];
	}
	dispatch_group_t group = dispatch_group_create();
	__block NSString *failure = nil;
	for (NSUInteger i = 0; i < n; i++) {
		NSItemProvider *p = results[i].itemProvider;
		if (p.suggestedName != nil) {
			names[i] = p.suggestedName;
		}
		if (![p canLoadObjectOfClass:[UIImage class]]) {
			continue;
		}
		dispatch_group_enter(group);
		[p loadObjectOfClass:[UIImage class] completionHandler:^(id<NSItemProviderReading> obj, NSError *err) {
			NSData *jpeg = nil;
			if (obj != nil) {
				jpeg = giodeJPEG((UIImage *)obj, maxSide);
			}
			@synchronized(jpegs) {
				if (jpeg != nil) {
					jpegs[i] = jpeg;
				} else if (failure == nil) {
					failure = err != nil ? err.localizedDescription : @"can't load an image";
				}
			}
			dispatch_group_leave(group);
		}];
	}
	dispatch_group_notify(group, dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
		int count = 0;
		for (NSUInteger i = 0; i < n; i++) {
			if ([jpegs[i] isKindOfClass:[NSData class]]) {
				NSData *d = jpegs[i];
				giodePhotosItem(rid, (void *)d.bytes, (int)d.length, (char *)[names[i] UTF8String]);
				count++;
			}
		}
		giodePhotosDone(rid, count, count == 0 && failure != nil ? (char *)failure.UTF8String : NULL);
	});
}
@end

int giodePickPhotos(CFTypeRef vcRef, int rid, int maxSide, int limit) {
	if (@available(iOS 14, *)) {
		UIViewController *vc = (__bridge UIViewController *)vcRef;
		onMain(^{
			PHPickerConfiguration *cfg = [[PHPickerConfiguration alloc] init];
			cfg.filter = [PHPickerFilter imagesFilter];
			cfg.selectionLimit = limit;
			if (@available(iOS 15, *)) {
				cfg.selection = PHPickerConfigurationSelectionOrdered;
			}
			PHPickerViewController *picker = [[PHPickerViewController alloc] initWithConfiguration:cfg];
			GiodePhotosDelegate *d = [[GiodePhotosDelegate alloc] init];
			d.requestID = rid;
			d.maxSide = maxSide;
			if (pending == nil) {
				pending = [NSMutableSet set];
			}
			[pending addObject:d];
			picker.delegate = d;
			[vc presentViewController:picker animated:YES completion:nil];
		});
		return 1;
	}
	return 0;
}

int giodeHasInfoKey(const char *key) {
	NSString *k = [NSString stringWithUTF8String:key];
	return [[NSBundle mainBundle] objectForInfoDictionaryKey:k] != nil;
}

// Universal links arrive as a browsing-web user activity, which Gio's
// app delegate doesn't handle: giode adds the method and hands the URL
// to Gio's URL path, which reaches Config.OnOpenURL.
extern void gio_onOpenURI(CFTypeRef uri);

static BOOL giodeContinueUserActivity(id self, SEL _cmd, UIApplication *app, NSUserActivity *activity, void (^handler)(NSArray *)) {
	if ([activity.activityType isEqualToString:NSUserActivityTypeBrowsingWeb] && activity.webpageURL != nil) {
		gio_onOpenURI((__bridge CFTypeRef)activity.webpageURL.absoluteString);
		return YES;
	}
	return NO;
}

__attribute__((constructor)) static void giodeInstallUserActivity(void) {
	Class c = objc_getClass("_gioAppDelegate");
	if (c == nil) {
		return;
	}
	SEL sel = @selector(application:continueUserActivity:restorationHandler:);
	struct objc_method_description d = protocol_getMethodDescription(@protocol(UIApplicationDelegate), sel, NO, YES);
	class_addMethod(c, sel, (IMP)giodeContinueUserActivity, d.types);
}

// giodeClipboardImagePNG returns the general pasteboard's image as PNG
// (malloc'ed, length in *n), or NULL when it holds none.
void *giodeClipboardImagePNG(int *n) {
	@autoreleasepool {
		UIPasteboard *pb = [UIPasteboard generalPasteboard];
		if (!pb.hasImages) {
			return NULL;
		}
		NSData *png = [pb dataForPasteboardType:@"public.png"];
		if (png == nil) {
			UIImage *img = pb.image;
			if (img == nil) {
				return NULL;
			}
			png = UIImagePNGRepresentation(img);
		}
		if (png == nil || png.length == 0 || png.length > (64 << 20)) {
			return NULL;
		}
		void *buf = malloc(png.length);
		memcpy(buf, png.bytes, png.length);
		*n = (int)png.length;
		return buf;
	}
}
