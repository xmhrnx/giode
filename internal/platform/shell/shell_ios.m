#import <UIKit/UIKit.h>

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

void giodeOpenURL(const char *url) {
	NSURL *u = [NSURL URLWithString:[NSString stringWithUTF8String:url]];
	onMain(^{
		[[UIApplication sharedApplication] openURL:u options:@{} completionHandler:nil];
	});
}

void giodeShare(CFTypeRef vcRef, const char *text, const char *path, const char *subject) {
	UIViewController *vc = (__bridge UIViewController *)vcRef;
	NSMutableArray *items = [NSMutableArray array];
	if (text != NULL) {
		[items addObject:[NSString stringWithUTF8String:text]];
	}
	if (path != NULL) {
		[items addObject:[NSURL fileURLWithPath:[NSString stringWithUTF8String:path]]];
	}
	NSString *subj = subject != NULL ? [NSString stringWithUTF8String:subject] : nil;
	onMain(^{
		UIActivityViewController *a = [[UIActivityViewController alloc] initWithActivityItems:items applicationActivities:nil];
		if (subj.length > 0) {
			[a setValue:subj forKey:@"subject"];
		}
		// On iPad the sheet is a popover and needs an anchor: the
		// middle of the window.
		UIPopoverPresentationController *pop = a.popoverPresentationController;
		if (pop != nil) {
			pop.sourceView = vc.view;
			CGRect b = vc.view.bounds;
			pop.sourceRect = CGRectMake(CGRectGetMidX(b), CGRectGetMidY(b), 1, 1);
			pop.permittedArrowDirections = 0;
		}
		[vc presentViewController:a animated:YES completion:nil];
	});
}

// giodeShareFiles opens the share sheet for several files, paths
// separated by newlines.
void giodeShareFiles(CFTypeRef vcRef, const char *paths) {
	UIViewController *vc = (__bridge UIViewController *)vcRef;
	NSMutableArray *items = [NSMutableArray array];
	for (NSString *p in [[NSString stringWithUTF8String:paths] componentsSeparatedByString:@"\n"]) {
		if (p.length > 0) {
			[items addObject:[NSURL fileURLWithPath:p]];
		}
	}
	onMain(^{
		UIActivityViewController *a = [[UIActivityViewController alloc] initWithActivityItems:items applicationActivities:nil];
		UIPopoverPresentationController *pop = a.popoverPresentationController;
		if (pop != nil) {
			pop.sourceView = vc.view;
			CGRect b = vc.view.bounds;
			pop.sourceRect = CGRectMake(CGRectGetMidX(b), CGRectGetMidY(b), 1, 1);
			pop.permittedArrowDirections = 0;
		}
		[vc presentViewController:a animated:YES completion:nil];
	});
}

// GiodeOpenIn keeps a document interaction controller alive while it
// shows, and gives its Quick Look preview a view controller.
@interface GiodeOpenIn : NSObject <UIDocumentInteractionControllerDelegate>
@property(nonatomic, strong) UIDocumentInteractionController *doc;
@property(nonatomic, weak) UIViewController *vc;
@end

static GiodeOpenIn *openIn;

@implementation GiodeOpenIn
- (UIViewController *)documentInteractionControllerViewControllerForPreview:(UIDocumentInteractionController *)c {
	return self.vc;
}
- (void)documentInteractionControllerDidDismissOpenInMenu:(UIDocumentInteractionController *)c {
	if (openIn == self) {
		openIn = nil;
	}
}
- (void)documentInteractionControllerDidEndPreview:(UIDocumentInteractionController *)c {
	if (openIn == self) {
		openIn = nil;
	}
}
@end

// giodeOpenIn shows the "Open in…" menu of the apps that open the file
// at path or, when none does, its Quick Look preview.
void giodeOpenIn(CFTypeRef vcRef, const char *path) {
	UIViewController *vc = (__bridge UIViewController *)vcRef;
	NSURL *u = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
	onMain(^{
		GiodeOpenIn *o = [[GiodeOpenIn alloc] init];
		o.vc = vc;
		o.doc = [UIDocumentInteractionController interactionControllerWithURL:u];
		o.doc.delegate = o;
		openIn = o;
		CGRect b = vc.view.bounds;
		CGRect anchor = CGRectMake(CGRectGetMidX(b), CGRectGetMidY(b), 1, 1);
		if (![o.doc presentOpenInMenuFromRect:anchor inView:vc.view animated:YES]) {
			if (![o.doc presentPreviewAnimated:YES]) {
				openIn = nil;
			}
		}
	});
}
