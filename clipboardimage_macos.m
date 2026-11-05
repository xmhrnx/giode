//go:build darwin && !ios

@import AppKit;
#include <stdlib.h>
#include <string.h>

// giodeClipboardImagePNG returns the general pasteboard's image as PNG
// (malloc'ed, length in *n): PNG when it holds one, else TIFF or any
// image NSImage reads, converted. NULL when it holds no image.
void *giodeClipboardImagePNG(int *n) {
	@autoreleasepool {
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		NSData *png = [pb dataForType:NSPasteboardTypePNG];
		if (png == nil) {
			NSData *tiff = [pb dataForType:NSPasteboardTypeTIFF];
			if (tiff == nil) {
				NSArray *imgs = [pb readObjectsForClasses:@[[NSImage class]] options:nil];
				if (imgs.count > 0) {
					tiff = [(NSImage *)imgs[0] TIFFRepresentation];
				}
			}
			if (tiff == nil) {
				return NULL;
			}
			NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiff];
			png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
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
