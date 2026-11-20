#import <UIKit/UIKit.h>
#include <stdlib.h>
#include <string.h>

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
