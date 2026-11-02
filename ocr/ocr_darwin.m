// SPDX-License-Identifier: Unlicense OR MIT

#import <Foundation/Foundation.h>
#import <Vision/Vision.h>
#import <ImageIO/ImageIO.h>
#include "_cgo_export.h"

// giodeOCR recognises text with Vision on a background queue and answers
// giodeOCRDone with the lines and words as JSON, in the image's pixels
// with a top left origin.
void giodeOCR(int rid, const void *bytes, int n, const char *langs, int fast) {
	NSData *data = [NSData dataWithBytes:bytes length:n];
	NSString *langList = [NSString stringWithUTF8String:langs];
	dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
		@autoreleasepool {
			CGImageSourceRef src = CGImageSourceCreateWithData((__bridge CFDataRef)data, NULL);
			CGImageRef img = src ? CGImageSourceCreateImageAtIndex(src, 0, NULL) : NULL;
			if (src) {
				CFRelease(src);
			}
			if (img == NULL) {
				giodeOCRDone(rid, NULL, "can't decode the image");
				return;
			}
			double w = CGImageGetWidth(img), h = CGImageGetHeight(img);
			VNRecognizeTextRequest *req = [[VNRecognizeTextRequest alloc] init];
			req.recognitionLevel = fast ? VNRequestTextRecognitionLevelFast : VNRequestTextRecognitionLevelAccurate;
			req.usesLanguageCorrection = !fast;
			if (langList.length > 0) {
				req.recognitionLanguages = [langList componentsSeparatedByString:@","];
			}
			VNImageRequestHandler *handler = [[VNImageRequestHandler alloc] initWithCGImage:img options:@{}];
			NSError *err = nil;
			BOOL ok = [handler performRequests:@[ req ] error:&err];
			CGImageRelease(img);
			if (!ok) {
				giodeOCRDone(rid, NULL, (char *)(err.localizedDescription ?: @"recognition failed").UTF8String);
				return;
			}
			// Vision's boxes are normalised with a bottom left origin.
			NSArray *(^box)(CGRect) = ^NSArray *(CGRect r) {
				return @[ @(r.origin.x * w), @((1 - r.origin.y - r.size.height) * h), @((r.origin.x + r.size.width) * w), @((1 - r.origin.y) * h) ];
			};
			NSMutableArray *lines = [NSMutableArray array];
			NSMutableArray *texts = [NSMutableArray array];
			for (VNRecognizedTextObservation *obs in req.results) {
				VNRecognizedText *top = [obs topCandidates:1].firstObject;
				if (top == nil) {
					continue;
				}
				NSString *s = top.string;
				NSMutableArray *words = [NSMutableArray array];
				[s enumerateSubstringsInRange:NSMakeRange(0, s.length) options:NSStringEnumerationByWords usingBlock:^(NSString *word, NSRange r, NSRange enclosing, BOOL *stop) {
					VNRectangleObservation *wb = [top boundingBoxForRange:r error:nil];
					if (wb != nil) {
						[words addObject:@{@"text" : word, @"box" : box(wb.boundingBox)}];
					}
				}];
				[lines addObject:@{@"text" : s, @"confidence" : @(top.confidence), @"box" : box(obs.boundingBox), @"words" : words}];
				[texts addObject:s];
			}
			NSDictionary *res = @{@"text" : [texts componentsJoinedByString:@"\n"], @"lines" : lines};
			NSData *js = [NSJSONSerialization dataWithJSONObject:res options:0 error:&err];
			if (js == nil) {
				giodeOCRDone(rid, NULL, (char *)(err.localizedDescription ?: @"can't encode the result").UTF8String);
				return;
			}
			NSString *str = [[NSString alloc] initWithData:js encoding:NSUTF8StringEncoding];
			giodeOCRDone(rid, (char *)str.UTF8String, NULL);
		}
	});
}
