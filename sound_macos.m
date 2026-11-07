//go:build darwin && !ios

#import <AppKit/AppKit.h>

// giodePlaying holds the sounds still playing: NSSound stops one that's
// released.
static NSMutableArray<NSSound *> *giodePlaying;

int giodePlaySound(const void *data, int n) {
	NSData *d = [NSData dataWithBytes:data length:n];
	NSSound *s = [[NSSound alloc] initWithData:d];
	if (s == nil) {
		return 0;
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		if (giodePlaying == nil) {
			giodePlaying = [NSMutableArray array];
		}
		NSMutableArray *done = [NSMutableArray array];
		for (NSSound *p in giodePlaying) {
			if (!p.isPlaying) {
				[done addObject:p];
			}
		}
		[giodePlaying removeObjectsInArray:done];
		[giodePlaying addObject:s];
		[s play];
	});
	return 1;
}
