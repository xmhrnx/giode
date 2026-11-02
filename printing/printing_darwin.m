// SPDX-License-Identifier: Unlicense OR MIT

#include <TargetConditionals.h>
#import <Foundation/Foundation.h>
#if TARGET_OS_IPHONE
#import <UIKit/UIKit.h>
#else
#import <AppKit/AppKit.h>
#import <PDFKit/PDFKit.h>
#endif
#include "_cgo_export.h"

// giodePrint shows the print panel for a PDF or image file and answers
// giodePrintDone when it closes. view is Gio's view: on iOS the
// UIViewController, on macOS the NSView.
void giodePrint(int rid, CFTypeRef viewRef, const char *path, const char *job, int pdf) {
	NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
	NSString *name = [NSString stringWithUTF8String:job];
	dispatch_async(dispatch_get_main_queue(), ^{
#if TARGET_OS_IPHONE
		if (![UIPrintInteractionController canPrintURL:url]) {
			giodePrintDone(rid, "this file can't be printed");
			return;
		}
		UIPrintInteractionController *pc = [UIPrintInteractionController sharedPrintController];
		UIPrintInfo *info = [UIPrintInfo printInfo];
		info.jobName = name;
		info.outputType = pdf ? UIPrintInfoOutputGeneral : UIPrintInfoOutputPhoto;
		pc.printInfo = info;
		pc.printingItem = url;
		void (^done)(UIPrintInteractionController *, BOOL, NSError *) = ^(UIPrintInteractionController *c, BOOL completed, NSError *err) {
			giodePrintDone(rid, err != nil ? (char *)err.localizedDescription.UTF8String : NULL);
		};
		if (UIDevice.currentDevice.userInterfaceIdiom == UIUserInterfaceIdiomPad) {
			UIViewController *vc = (__bridge UIViewController *)viewRef;
			CGRect b = vc.view.bounds;
			[pc presentFromRect:CGRectMake(CGRectGetMidX(b), CGRectGetMidY(b), 1, 1) inView:vc.view animated:YES completionHandler:done];
		} else {
			[pc presentAnimated:YES completionHandler:done];
		}
#else
		NSView *view = (__bridge NSView *)viewRef;
		NSPrintInfo *info = [[NSPrintInfo sharedPrintInfo] copy];
		info.jobDisposition = NSPrintSpoolJob;
		NSPrintOperation *op = nil;
		if (pdf) {
			PDFDocument *doc = [[PDFDocument alloc] initWithURL:url];
			if (doc == nil) {
				giodePrintDone(rid, "can't open the PDF");
				return;
			}
			op = [doc printOperationForPrintInfo:info scalingMode:kPDFPrintPageScaleToFit autoRotate:YES];
		} else {
			NSImage *img = [[NSImage alloc] initWithContentsOfURL:url];
			if (img == nil) {
				giodePrintDone(rid, "can't open the image");
				return;
			}
			NSImageView *iv = [NSImageView imageViewWithImage:img];
			iv.frame = NSMakeRect(0, 0, img.size.width, img.size.height);
			info.horizontalPagination = NSPrintingPaginationModeFit;
			info.verticalPagination = NSPrintingPaginationModeFit;
			op = [NSPrintOperation printOperationWithView:iv printInfo:info];
		}
		op.jobTitle = name;
		op.showsPrintPanel = YES;
		op.showsProgressPanel = YES;
		NSWindow *win = view.window;
		if (win != nil) {
			[op runOperationModalForWindow:win delegate:nil didRunSelector:NULL contextInfo:NULL];
			// The sheet runs asynchronously; it's queued once shown.
			giodePrintDone(rid, NULL);
		} else {
			[op runOperation];
			giodePrintDone(rid, NULL);
		}
#endif
	});
}
