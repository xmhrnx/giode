//go:build ios

// Adding pictures and videos to the photo library: only apps importing
// this package use PHPhotoLibrary.

#import <Foundation/Foundation.h>
#import <Photos/Photos.h>
#include "_cgo_export.h"

int giodeGalleryHasInfoKey(const char *key) {
	NSString *k = [NSString stringWithUTF8String:key];
	return [[NSBundle mainBundle] objectForInfoDictionaryKey:k] != nil;
}

// giodeGallerySave adds a picture or video to the photo library, asking
// for add-only access first, and answers giodeGallerySaved.
void giodeGallerySave(int rid, const char *path, int video) {
	NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
	void (^save)(void) = ^{
		[[PHPhotoLibrary sharedPhotoLibrary] performChanges:^{
			if (video) {
				[PHAssetChangeRequest creationRequestForAssetFromVideoAtFileURL:url];
			} else {
				[PHAssetChangeRequest creationRequestForAssetFromImageAtFileURL:url];
			}
		} completionHandler:^(BOOL ok, NSError *err) {
			giodeGallerySaved(rid, ok ? NULL : (char *)(err.localizedDescription ?: @"can't save to Photos").UTF8String);
		}];
	};
	if (@available(iOS 14, *)) {
		[PHPhotoLibrary requestAuthorizationForAccessLevel:PHAccessLevelAddOnly handler:^(PHAuthorizationStatus st) {
			if (st == PHAuthorizationStatusAuthorized || st == PHAuthorizationStatusLimited) {
				save();
			} else {
				giodeGallerySaved(rid, "adding to Photos isn't allowed for this app (Settings > Privacy & Security > Photos)");
			}
		}];
	} else {
		save();
	}
}
