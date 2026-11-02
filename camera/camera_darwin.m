// SPDX-License-Identifier: Unlicense OR MIT

#include <TargetConditionals.h>
#import <Foundation/Foundation.h>
#import <AVFoundation/AVFoundation.h>
#import <CoreVideo/CoreVideo.h>
#import <Accelerate/Accelerate.h>
#if TARGET_OS_IPHONE
#import <UIKit/UIKit.h>
#else
#import <AppKit/AppKit.h>
#endif
#include <stdlib.h>
#include <string.h>
#include "_cgo_export.h"

// giode/camera on AVFoundation: a session with a video data output,
// whose BGRA frames go to Go scaled to the preview size, and a photo
// output for full-resolution pictures. On iOS the frames are rotated to
// the interface orientation, so they arrive upright.

@class GiodeCamera;

@interface GiodePhotoDelegate : NSObject <AVCapturePhotoCaptureDelegate>
@property(nonatomic) int req;
@property(nonatomic, strong) GiodeCamera *camera;
@end

@interface GiodeCamera : NSObject <AVCaptureVideoDataOutputSampleBufferDelegate> {
	// The scaled frame handed to Go, reused frame to frame on the video
	// queue.
	void *scaled;
	size_t scaledLen;
}
@property(nonatomic) int camID;
@property(nonatomic) int maxSide;
@property(nonatomic) BOOL front;
@property(nonatomic, strong) AVCaptureSession *session;
@property(nonatomic, strong) AVCaptureDevice *device;
@property(nonatomic, strong) AVCaptureVideoDataOutput *video;
@property(nonatomic, strong) AVCapturePhotoOutput *photo;
@property(nonatomic, strong) dispatch_queue_t sessionQueue;
@property(nonatomic, strong) dispatch_queue_t videoQueue;
@property(nonatomic, strong) NSMutableSet<GiodePhotoDelegate *> *captures;
@property(nonatomic, strong) id orientationObserver;
@end

static NSMutableDictionary<NSNumber *, GiodeCamera *> *giodeCameras;

static char *giodeErr(NSString *msg) {
	return strdup(msg.UTF8String);
}

int giodeCameraAuthorized(void) {
	switch ([AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeVideo]) {
	case AVAuthorizationStatusAuthorized:
		return 1;
	case AVAuthorizationStatusNotDetermined:
		return -1;
	default:
		return 0;
	}
}

void giodeCameraRequestAccess(int req) {
	[AVCaptureDevice requestAccessForMediaType:AVMediaTypeVideo completionHandler:^(BOOL granted) {
		giodeCameraOnPermission(req, granted ? 1 : 0);
	}];
}

#if TARGET_OS_IPHONE
// The interface orientation, read on the main thread and kept for the
// camera's queues.
static _Atomic(NSInteger) giodeOrientation = UIInterfaceOrientationPortrait;

static void giodeReadOrientation(void) {
	UIInterfaceOrientation o = UIInterfaceOrientationUnknown;
	for (UIScene *s in UIApplication.sharedApplication.connectedScenes) {
		if ([s isKindOfClass:[UIWindowScene class]]) {
			o = ((UIWindowScene *)s).interfaceOrientation;
			break;
		}
	}
	if (o != UIInterfaceOrientationUnknown) {
		giodeOrientation = o;
	}
}

static AVCaptureVideoOrientation giodeVideoOrientation(void) {
	switch ((UIInterfaceOrientation)giodeOrientation) {
	case UIInterfaceOrientationPortraitUpsideDown:
		return AVCaptureVideoOrientationPortraitUpsideDown;
	case UIInterfaceOrientationLandscapeLeft:
		return AVCaptureVideoOrientationLandscapeLeft;
	case UIInterfaceOrientationLandscapeRight:
		return AVCaptureVideoOrientationLandscapeRight;
	default:
		return AVCaptureVideoOrientationPortrait;
	}
}
#endif

@implementation GiodeCamera

- (void)dealloc {
	free(scaled);
}

- (void)applyOrientation {
#if TARGET_OS_IPHONE
	AVCaptureVideoOrientation o = giodeVideoOrientation();
	for (AVCaptureOutput *out in @[ self.video, self.photo ]) {
		AVCaptureConnection *conn = [out connectionWithMediaType:AVMediaTypeVideo];
		if (conn.isVideoOrientationSupported) {
			conn.videoOrientation = o;
		}
		if (out == self.video && conn.isVideoMirroringSupported) {
			conn.automaticallyAdjustsVideoMirroring = NO;
			conn.videoMirrored = self.front;
		}
	}
#endif
}

- (void)captureOutput:(AVCaptureOutput *)output didOutputSampleBuffer:(CMSampleBufferRef)sample fromConnection:(AVCaptureConnection *)conn {
	CVPixelBufferRef px = CMSampleBufferGetImageBuffer(sample);
	if (px == NULL) {
		return;
	}
	CVPixelBufferLockBaseAddress(px, kCVPixelBufferLock_ReadOnly);
	size_t w = CVPixelBufferGetWidth(px), h = CVPixelBufferGetHeight(px);
	size_t stride = CVPixelBufferGetBytesPerRow(px);
	void *base = CVPixelBufferGetBaseAddress(px);
	size_t longSide = MAX(w, h);
	if ((int)longSide > self.maxSide && self.maxSide > 0) {
		// Scale down before Go sees it: a full sensor frame is far more
		// than a preview needs.
		size_t sw = w * self.maxSide / longSide, sh = h * self.maxSide / longSide;
		size_t sstride = sw * 4;
		if (scaledLen < sstride * sh) {
			free(scaled);
			scaledLen = sstride * sh;
			scaled = malloc(scaledLen);
		}
		vImage_Buffer src = {base, h, w, stride};
		vImage_Buffer dst = {scaled, sh, sw, sstride};
		vImageScale_ARGB8888(&src, &dst, NULL, kvImageNoFlags);
		giodeCameraOnFrame(self.camID, scaled, (int)sw, (int)sh, (int)sstride);
	} else {
		giodeCameraOnFrame(self.camID, base, (int)w, (int)h, (int)stride);
	}
	CVPixelBufferUnlockBaseAddress(px, kCVPixelBufferLock_ReadOnly);
}

@end

@implementation GiodePhotoDelegate

- (void)captureOutput:(AVCapturePhotoOutput *)output didFinishProcessingPhoto:(AVCapturePhoto *)photo error:(NSError *)error {
	GiodeCamera *cam = self.camera;
	if (error != nil) {
		giodeCameraOnCapture(self.req, NULL, 0, (char *)error.localizedDescription.UTF8String);
	} else {
		NSData *data = [photo fileDataRepresentation];
#if TARGET_OS_IPHONE
		// Turn the pixels upright rather than leave it to the EXIF tag.
		UIImage *img = [UIImage imageWithData:data];
		if (img != nil && img.imageOrientation != UIImageOrientationUp) {
			UIGraphicsImageRendererFormat *f = [UIGraphicsImageRendererFormat formatForTraitCollection:[UITraitCollection traitCollectionWithDisplayScale:1]];
			f.scale = 1;
			UIGraphicsImageRenderer *r = [[UIGraphicsImageRenderer alloc] initWithSize:img.size format:f];
			NSData *upright = [r JPEGDataWithCompressionQuality:0.92 actions:^(UIGraphicsImageRendererContext *ctx) {
				[img drawInRect:CGRectMake(0, 0, img.size.width, img.size.height)];
			}];
			if (upright != nil) {
				data = upright;
			}
		}
#endif
		if (data == nil) {
			giodeCameraOnCapture(self.req, NULL, 0, "no picture data");
		} else {
			giodeCameraOnCapture(self.req, (void *)data.bytes, (int)data.length, NULL);
		}
	}
	dispatch_async(cam.sessionQueue, ^{
		[cam.captures removeObject:self];
	});
}

@end

static GiodeCamera *giodeCamera(int id) {
	@synchronized(giodeCameras) {
		return giodeCameras[@(id)];
	}
}

const char *giodeCameraOpen(int id, int front, int maxSide) {
	AVCaptureDevicePosition pos = front ? AVCaptureDevicePositionFront : AVCaptureDevicePositionBack;
	AVCaptureDevice *dev = [AVCaptureDevice defaultDeviceWithDeviceType:AVCaptureDeviceTypeBuiltInWideAngleCamera mediaType:AVMediaTypeVideo position:pos];
	if (dev == nil) {
		dev = [AVCaptureDevice defaultDeviceWithMediaType:AVMediaTypeVideo];
	}
	if (dev == nil) {
		return giodeErr(@"no camera");
	}
	NSError *err = nil;
	AVCaptureDeviceInput *in = [AVCaptureDeviceInput deviceInputWithDevice:dev error:&err];
	if (in == nil) {
		return giodeErr(err.localizedDescription ?: @"can't open the camera");
	}
	GiodeCamera *cam = [GiodeCamera new];
	cam.camID = id;
	cam.maxSide = maxSide;
	cam.front = front != 0;
	cam.device = dev;
	cam.captures = [NSMutableSet set];
	cam.sessionQueue = dispatch_queue_create("giode.camera.session", DISPATCH_QUEUE_SERIAL);
	cam.videoQueue = dispatch_queue_create("giode.camera.video", DISPATCH_QUEUE_SERIAL);
	AVCaptureSession *s = [AVCaptureSession new];
	[s beginConfiguration];
	if ([s canSetSessionPreset:AVCaptureSessionPresetPhoto]) {
		s.sessionPreset = AVCaptureSessionPresetPhoto;
	}
	if (![s canAddInput:in]) {
		return giodeErr(@"can't use the camera");
	}
	[s addInput:in];
	AVCaptureVideoDataOutput *video = [AVCaptureVideoDataOutput new];
	video.videoSettings = @{(id)kCVPixelBufferPixelFormatTypeKey : @(kCVPixelFormatType_32BGRA)};
	video.alwaysDiscardsLateVideoFrames = YES;
	[video setSampleBufferDelegate:cam queue:cam.videoQueue];
	if ([s canAddOutput:video]) {
		[s addOutput:video];
	}
	AVCapturePhotoOutput *photo = [AVCapturePhotoOutput new];
	if ([s canAddOutput:photo]) {
		[s addOutput:photo];
	}
#if TARGET_OS_IPHONE
	if (@available(iOS 16.0, *)) {
		if (dev.activeFormat.supportedMaxPhotoDimensions.count > 0) {
			photo.maxPhotoDimensions = dev.activeFormat.supportedMaxPhotoDimensions.lastObject.CMVideoDimensionsValue;
		}
	} else {
		photo.highResolutionCaptureEnabled = YES;
	}
#endif
	[s commitConfiguration];
	cam.session = s;
	cam.video = video;
	cam.photo = photo;
	@synchronized(giodeCameras) {
		if (giodeCameras == nil) {
			giodeCameras = [NSMutableDictionary dictionary];
		}
		giodeCameras[@(id)] = cam;
	}
#if TARGET_OS_IPHONE
	dispatch_async(dispatch_get_main_queue(), ^{
		[[UIDevice currentDevice] beginGeneratingDeviceOrientationNotifications];
		giodeReadOrientation();
		dispatch_async(cam.sessionQueue, ^{
			[cam applyOrientation];
		});
		cam.orientationObserver = [[NSNotificationCenter defaultCenter] addObserverForName:UIDeviceOrientationDidChangeNotification object:nil queue:[NSOperationQueue mainQueue] usingBlock:^(NSNotification *n) {
			giodeReadOrientation();
			dispatch_async(cam.sessionQueue, ^{
				[cam applyOrientation];
			});
		}];
	});
	[[NSNotificationCenter defaultCenter] addObserverForName:AVCaptureSessionRuntimeErrorNotification object:s queue:nil usingBlock:^(NSNotification *n) {
		NSError *e = n.userInfo[AVCaptureSessionErrorKey];
		giodeCameraOnError(id, (char *)(e.localizedDescription ?: @"camera error").UTF8String);
	}];
#endif
	dispatch_async(cam.sessionQueue, ^{
		[s startRunning];
	});
	return NULL;
}

void giodeCameraClose(int id) {
	GiodeCamera *cam;
	@synchronized(giodeCameras) {
		cam = giodeCameras[@(id)];
		[giodeCameras removeObjectForKey:@(id)];
	}
	if (cam == nil) {
		return;
	}
#if TARGET_OS_IPHONE
	if (cam.orientationObserver != nil) {
		[[NSNotificationCenter defaultCenter] removeObserver:cam.orientationObserver];
		dispatch_async(dispatch_get_main_queue(), ^{
			[[UIDevice currentDevice] endGeneratingDeviceOrientationNotifications];
		});
	}
#endif
	dispatch_async(cam.sessionQueue, ^{
		[cam.video setSampleBufferDelegate:nil queue:nil];
		[cam.session stopRunning];
		if (cam.device.hasTorch && cam.device.torchMode != AVCaptureTorchModeOff && [cam.device lockForConfiguration:nil]) {
			cam.device.torchMode = AVCaptureTorchModeOff;
			[cam.device unlockForConfiguration];
		}
	});
}

void giodeCameraCapture(int id, int req) {
	GiodeCamera *cam = giodeCamera(id);
	if (cam == nil) {
		giodeCameraOnCapture(req, NULL, 0, "camera closed");
		return;
	}
	dispatch_async(cam.sessionQueue, ^{
		AVCapturePhotoSettings *settings;
		if ([cam.photo.availablePhotoCodecTypes containsObject:AVVideoCodecTypeJPEG]) {
			settings = [AVCapturePhotoSettings photoSettingsWithFormat:@{AVVideoCodecKey : AVVideoCodecTypeJPEG}];
		} else {
			settings = [AVCapturePhotoSettings photoSettings];
		}
#if TARGET_OS_IPHONE
		if (@available(iOS 16.0, *)) {
			settings.maxPhotoDimensions = cam.photo.maxPhotoDimensions;
		} else {
			settings.highResolutionPhotoEnabled = YES;
		}
		// Keep the torch's light rather than fire the flash.
		settings.flashMode = AVCaptureFlashModeOff;
#endif
		GiodePhotoDelegate *d = [GiodePhotoDelegate new];
		d.req = req;
		d.camera = cam;
		[cam.captures addObject:d];
		[cam.photo capturePhotoWithSettings:settings delegate:d];
	});
}

int giodeCameraHasTorch(int id) {
	GiodeCamera *cam = giodeCamera(id);
	return cam != nil && cam.device.hasTorch && [cam.device isTorchModeSupported:AVCaptureTorchModeOn];
}

const char *giodeCameraTorch(int id, int on) {
	GiodeCamera *cam = giodeCamera(id);
	if (cam == nil) {
		return giodeErr(@"camera closed");
	}
	AVCaptureDevice *dev = cam.device;
	if (!dev.hasTorch) {
		return giodeErr(@"this camera has no light");
	}
	NSError *err = nil;
	if (![dev lockForConfiguration:&err]) {
		return giodeErr(err.localizedDescription ?: @"can't configure the camera");
	}
	dev.torchMode = on ? AVCaptureTorchModeOn : AVCaptureTorchModeOff;
	[dev unlockForConfiguration];
	return NULL;
}

const char *giodeCameraFocus(int id, float x, float y) {
	GiodeCamera *cam = giodeCamera(id);
	if (cam == nil) {
		return giodeErr(@"camera closed");
	}
	// Points of interest are in the sensor's own frame: landscape, with
	// the home button (or its place) on the right.
	CGPoint p = CGPointMake(x, y);
#if TARGET_OS_IPHONE
	if (cam.front) {
		x = 1 - x;
	}
	switch ((UIInterfaceOrientation)giodeOrientation) {
	case UIInterfaceOrientationLandscapeRight:
		p = CGPointMake(x, y);
		break;
	case UIInterfaceOrientationLandscapeLeft:
		p = CGPointMake(1 - x, 1 - y);
		break;
	case UIInterfaceOrientationPortraitUpsideDown:
		p = CGPointMake(1 - y, x);
		break;
	default:
		p = CGPointMake(y, 1 - x);
	}
#endif
	AVCaptureDevice *dev = cam.device;
	NSError *err = nil;
	if (![dev lockForConfiguration:&err]) {
		return giodeErr(err.localizedDescription ?: @"can't configure the camera");
	}
	if (dev.isFocusPointOfInterestSupported && [dev isFocusModeSupported:AVCaptureFocusModeAutoFocus]) {
		dev.focusPointOfInterest = p;
		dev.focusMode = AVCaptureFocusModeAutoFocus;
	}
	if (dev.isExposurePointOfInterestSupported && [dev isExposureModeSupported:AVCaptureExposureModeContinuousAutoExposure]) {
		dev.exposurePointOfInterest = p;
		dev.exposureMode = AVCaptureExposureModeContinuousAutoExposure;
	}
	[dev unlockForConfiguration];
	return NULL;
}
