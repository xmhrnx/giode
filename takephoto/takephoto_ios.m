//go:build ios

// Taking a photo with UIImagePickerController's camera: only apps
// importing this package use the camera.

#import <UIKit/UIKit.h>
#import <AVFoundation/AVFoundation.h>
#include "_cgo_export.h"

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

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

int giodeTakePhotoHasInfoKey(const char *key) {
	NSString *k = [NSString stringWithUTF8String:key];
	return [[NSBundle mainBundle] objectForInfoDictionaryKey:k] != nil;
}

@interface GiodeTakePhotoDelegate : NSObject <UIImagePickerControllerDelegate, UINavigationControllerDelegate>
@property int requestID;
@property int maxSide;
@end

@implementation GiodeTakePhotoDelegate
- (void)imagePickerController:(UIImagePickerController *)picker didFinishPickingMediaWithInfo:(NSDictionary<UIImagePickerControllerInfoKey, id> *)info {
	int rid = self.requestID;
	int maxSide = self.maxSide;
	[pending removeObject:self];
	[picker dismissViewControllerAnimated:YES completion:nil];
	UIImage *img = info[UIImagePickerControllerOriginalImage];
	if (img == nil) {
		giodeTakePhotoDone(rid, NULL, 0, (char *)"the camera returned no image");
		return;
	}
	dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
		NSData *jpeg = giodeJPEG(img, maxSide);
		giodeTakePhotoDone(rid, (void *)jpeg.bytes, (int)jpeg.length, NULL);
	});
}

- (void)imagePickerControllerDidCancel:(UIImagePickerController *)picker {
	int rid = self.requestID;
	[pending removeObject:self];
	[picker dismissViewControllerAnimated:YES completion:nil];
	giodeTakePhotoDone(rid, NULL, 0, NULL);
}
@end

// giodeTakePhoto opens the camera. It returns 1 when the camera opens,
// 0 when the device has none (the simulator), and -1 when the user has
// denied the app camera access.
int giodeTakePhoto(CFTypeRef vcRef, int rid, int maxSide) {
	if (![UIImagePickerController isSourceTypeAvailable:UIImagePickerControllerSourceTypeCamera]) {
		return 0;
	}
	AVAuthorizationStatus st = [AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeVideo];
	if (st == AVAuthorizationStatusDenied || st == AVAuthorizationStatusRestricted) {
		return -1;
	}
	UIViewController *vc = (__bridge UIViewController *)vcRef;
	onMain(^{
		UIImagePickerController *picker = [[UIImagePickerController alloc] init];
		picker.sourceType = UIImagePickerControllerSourceTypeCamera;
		GiodeTakePhotoDelegate *d = [[GiodeTakePhotoDelegate alloc] init];
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
