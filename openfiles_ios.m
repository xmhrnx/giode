//go:build ios

#import <UIKit/UIKit.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#include "_cgo_export.h"

// answerDocuments hands Go the files' paths, each ending in a NUL.
static void answerDocuments(NSArray<NSURL *> *urls) {
	if (urls.count == 0) {
		giodeDocumentsPicked(NULL, 0);
		return;
	}
	NSMutableData *d = [NSMutableData data];
	for (NSURL *u in urls) {
		const char *p = u.fileSystemRepresentation;
		[d appendBytes:p length:strlen(p) + 1];
	}
	giodeDocumentsPicked((char *)d.bytes, (int)d.length);
}

@interface GiodeDocumentsDelegate : NSObject <UIDocumentPickerDelegate>
@end

@implementation GiodeDocumentsDelegate
- (void)documentPicker:(UIDocumentPickerViewController *)controller didPickDocumentsAtURLs:(NSArray<NSURL *> *)urls {
	answerDocuments(urls);
}

- (void)documentPickerWasCancelled:(UIDocumentPickerViewController *)controller {
	giodeDocumentsPicked(NULL, 0);
}
@end

// The picker holds its delegate weakly; one picker shows at a time.
static GiodeDocumentsDelegate *documentsDelegate;

// giodeOpenDocuments shows the document picker for files with exts
// (one per line; any file when empty), copied into the app's temporary
// directory, several when multiple is set. It returns 0 before iOS 14.
int giodeOpenDocuments(CFTypeRef vcRef, char *exts, int multiple) {
	if (@available(iOS 14, *)) {
		UIViewController *vc = (__bridge UIViewController *)vcRef;
		NSString *list = [NSString stringWithUTF8String:exts];
		BOOL many = multiple != 0;
		dispatch_async(dispatch_get_main_queue(), ^{
			NSMutableArray<UTType *> *types = [NSMutableArray array];
			for (NSString *e in [list componentsSeparatedByString:@"\n"]) {
				if (e.length == 0) {
					continue;
				}
				UTType *t = [UTType typeWithFilenameExtension:e];
				if (t != nil) {
					[types addObject:t];
				}
			}
			if (types.count == 0) {
				[types addObject:UTTypeItem];
			}
			UIDocumentPickerViewController *picker = [[UIDocumentPickerViewController alloc] initForOpeningContentTypes:types asCopy:YES];
			picker.allowsMultipleSelection = many;
			documentsDelegate = [[GiodeDocumentsDelegate alloc] init];
			picker.delegate = documentsDelegate;
			[vc presentViewController:picker animated:YES completion:nil];
		});
		return 1;
	}
	return 0;
}
