//go:build darwin

#include <TargetConditionals.h>
#include <stdlib.h>
#include <string.h>
@import Foundation;
@import CoreMedia;
@import CoreVideo;
#if TARGET_OS_IPHONE
@import ReplayKit;
@import UIKit;
#else
@import ScreenCaptureKit;
@import CoreGraphics;
#endif
#include "_cgo_export.h"

static char *giodeErr(NSString *s) { return strdup(s.UTF8String); }

// deliverPixels hands a pixel buffer to Go: BGRA, or the two planes of
// NV12 (what ReplayKit gives).
static void deliverPixels(int ident, CVPixelBufferRef pb) {
	CVPixelBufferLockBaseAddress(pb, kCVPixelBufferLock_ReadOnly);
	OSType f = CVPixelBufferGetPixelFormatType(pb);
	int w = (int)CVPixelBufferGetWidth(pb), h = (int)CVPixelBufferGetHeight(pb);
	if (f == kCVPixelFormatType_32BGRA) {
		giodeScreenBGRA(ident, CVPixelBufferGetBaseAddress(pb), w, h, (int)CVPixelBufferGetBytesPerRow(pb));
	} else if (CVPixelBufferGetPlaneCount(pb) == 2) {
		giodeScreenNV12(ident, CVPixelBufferGetBaseAddressOfPlane(pb, 0), (int)CVPixelBufferGetBytesPerRowOfPlane(pb, 0),
			CVPixelBufferGetBaseAddressOfPlane(pb, 1), (int)CVPixelBufferGetBytesPerRowOfPlane(pb, 1), w, h,
			f == kCVPixelFormatType_420YpCbCr8BiPlanarFullRange);
	}
	CVPixelBufferUnlockBaseAddress(pb, kCVPixelBufferLock_ReadOnly);
}

#if TARGET_OS_IPHONE

char *giodeScreenSources(void) {
	CGRect b = UIScreen.mainScreen.nativeBounds;
	return giodeErr([NSString stringWithFormat:@"screen\tScreen\t0\t0\t%d\t%d\t1\n", (int)b.size.width, (int)b.size.height]);
}

int giodeScreenPermission(void) { return 1; }

char *giodeScreenStart(int ident, unsigned int display, int width, int height, int fps) {
	RPScreenRecorder *r = [RPScreenRecorder sharedRecorder];
	if (!r.available) {
		return giodeErr(@"screen recording isn't available");
	}
	dispatch_semaphore_t done = dispatch_semaphore_create(0);
	__block NSString *msg = nil;
	[r startCaptureWithHandler:^(CMSampleBufferRef sb, RPSampleBufferType type, NSError *err) {
		if (type != RPSampleBufferTypeVideo || err != nil) {
			return;
		}
		CVPixelBufferRef pb = CMSampleBufferGetImageBuffer(sb);
		if (pb) {
			deliverPixels(ident, pb);
		}
	} completionHandler:^(NSError *err) {
		if (err) {
			msg = err.localizedDescription;
		}
		dispatch_semaphore_signal(done);
	}];
	dispatch_semaphore_wait(done, DISPATCH_TIME_FOREVER);
	return msg ? giodeErr(msg) : NULL;
}

void giodeScreenStop(int ident) {
	[[RPScreenRecorder sharedRecorder] stopCaptureWithHandler:nil];
}

#else

char *giodeScreenSources(void) {
	CGDirectDisplayID ids[16];
	uint32_t n = 0;
	CGGetActiveDisplayList(16, ids, &n);
	NSMutableString *out = [NSMutableString string];
	for (uint32_t i = 0; i < n; i++) {
		CGRect b = CGDisplayBounds(ids[i]);
		CGDisplayModeRef m = CGDisplayCopyDisplayMode(ids[i]);
		size_t pw = m ? CGDisplayModeGetPixelWidth(m) : (size_t)b.size.width;
		size_t ph = m ? CGDisplayModeGetPixelHeight(m) : (size_t)b.size.height;
		if (m) CGDisplayModeRelease(m);
		[out appendFormat:@"%u\tDisplay %u\t%d\t%d\t%d\t%d\t%d\n", ids[i], i + 1, (int)b.origin.x, (int)b.origin.y,
			(int)pw, (int)ph, CGDisplayIsMain(ids[i]) ? 1 : 0];
	}
	return giodeErr(out);
}

// giodeScreenPermission reports whether the app may record the screen,
// asking (which opens System Settings) when it may not.
int giodeScreenPermission(void) {
	if (CGPreflightScreenCaptureAccess()) {
		return 1;
	}
	return CGRequestScreenCaptureAccess() ? 1 : 0;
}

API_AVAILABLE(macos(12.3))
@interface GiodeScreenOutput : NSObject <SCStreamOutput, SCStreamDelegate>
@property int ident;
@property (strong) SCStream *stream;
@end

@implementation GiodeScreenOutput
- (void)stream:(SCStream *)stream didOutputSampleBuffer:(CMSampleBufferRef)sb ofType:(SCStreamOutputType)type {
	if (type != SCStreamOutputTypeScreen) {
		return;
	}
	CFArrayRef atts = CMSampleBufferGetSampleAttachmentsArray(sb, false);
	if (atts && CFArrayGetCount(atts) > 0) {
		NSDictionary *a = (__bridge NSDictionary *)CFArrayGetValueAtIndex(atts, 0);
		NSNumber *st = a[SCStreamFrameInfoStatus];
		if (st && st.integerValue != SCFrameStatusComplete) {
			return; // idle or blank: no new pixels
		}
	}
	CVPixelBufferRef pb = CMSampleBufferGetImageBuffer(sb);
	if (pb) {
		deliverPixels(self.ident, pb);
	}
}
- (void)stream:(SCStream *)stream didStopWithError:(NSError *)error {
	giodeScreenStopped(self.ident);
}
@end

static NSMutableDictionary *outputs;

char *giodeScreenStart(int ident, unsigned int display, int width, int height, int fps) {
	if (@available(macOS 12.3, *)) {
		dispatch_semaphore_t done = dispatch_semaphore_create(0);
		__block NSString *msg = nil;
		[SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent *content, NSError *err) {
			if (err || !content) {
				msg = err ? err.localizedDescription : @"no shareable content";
				dispatch_semaphore_signal(done);
				return;
			}
			SCDisplay *d = nil;
			for (SCDisplay *x in content.displays) {
				if (x.displayID == display) {
					d = x;
				}
			}
			if (!d) {
				d = content.displays.firstObject;
			}
			if (!d) {
				msg = @"no display";
				dispatch_semaphore_signal(done);
				return;
			}
			SCContentFilter *filter = [[SCContentFilter alloc] initWithDisplay:d excludingWindows:@[]];
			SCStreamConfiguration *cfg = [[SCStreamConfiguration alloc] init];
			cfg.width = width;
			cfg.height = height;
			cfg.minimumFrameInterval = CMTimeMake(1, fps);
			cfg.pixelFormat = kCVPixelFormatType_32BGRA;
			cfg.showsCursor = YES;
			GiodeScreenOutput *out = [GiodeScreenOutput new];
			out.ident = ident;
			SCStream *s = [[SCStream alloc] initWithFilter:filter configuration:cfg delegate:out];
			out.stream = s;
			NSError *e = nil;
			dispatch_queue_t q = dispatch_queue_create("giode.screencap", DISPATCH_QUEUE_SERIAL);
			if (![s addStreamOutput:out type:SCStreamOutputTypeScreen sampleHandlerQueue:q error:&e]) {
				msg = e.localizedDescription;
				dispatch_semaphore_signal(done);
				return;
			}
			[s startCaptureWithCompletionHandler:^(NSError *se) {
				if (se) {
					msg = se.localizedDescription;
				} else {
					@synchronized ([GiodeScreenOutput class]) {
						if (!outputs) outputs = [NSMutableDictionary dictionary];
						outputs[@(ident)] = out;
					}
				}
				dispatch_semaphore_signal(done);
			}];
		}];
		dispatch_semaphore_wait(done, DISPATCH_TIME_FOREVER);
		return msg ? giodeErr(msg) : NULL;
	}
	return giodeErr(@"screen capture needs macOS 12.3");
}

void giodeScreenStop(int ident) {
	if (@available(macOS 12.3, *)) {
		GiodeScreenOutput *out;
		@synchronized ([GiodeScreenOutput class]) {
			out = outputs[@(ident)];
			[outputs removeObjectForKey:@(ident)];
		}
		if (out) {
			dispatch_semaphore_t done = dispatch_semaphore_create(0);
			[out.stream stopCaptureWithCompletionHandler:^(NSError *e) {
				dispatch_semaphore_signal(done);
			}];
			dispatch_semaphore_wait(done, dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC));
		}
	}
}
#endif
