#import <AVFoundation/AVFoundation.h>

// giodePlayers holds the sounds still playing: AVAudioPlayer stops one
// that's released.
static NSMutableArray<AVAudioPlayer *> *giodePlayers;

int giodePlaySound(const void *data, int n) {
	NSData *d = [NSData dataWithBytes:data length:n];
	NSError *err = nil;
	AVAudioPlayer *p = [[AVAudioPlayer alloc] initWithData:d error:&err];
	if (p == nil) {
		return 0;
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		static dispatch_once_t once;
		dispatch_once(&once, ^{
			// Ambient: mixed with music, silent in silent mode, like a UI
			// sound.
			AVAudioSession *session = [AVAudioSession sharedInstance];
			if (session.category != AVAudioSessionCategoryPlayAndRecord && session.category != AVAudioSessionCategoryPlayback) {
				[session setCategory:AVAudioSessionCategoryAmbient error:nil];
			}
		});
		if (giodePlayers == nil) {
			giodePlayers = [NSMutableArray array];
		}
		NSMutableArray *done = [NSMutableArray array];
		for (AVAudioPlayer *q in giodePlayers) {
			if (!q.isPlaying) {
				[done addObject:q];
			}
		}
		[giodePlayers removeObjectsInArray:done];
		[giodePlayers addObject:p];
		[p play];
	});
	return 1;
}
