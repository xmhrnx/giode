//go:build darwin

#include <TargetConditionals.h>
#include <mach/mach_time.h>
#include <stdlib.h>
#include <string.h>
@import Foundation;
@import AVFoundation;
@import CoreMedia;
@import CoreVideo;
#include "_cgo_export.h"

// giode/media on AVPlayer: the player plays the sound and keeps time; a
// video output hands each new frame, as BGRA, to Go.

@interface GiodeMediaPlayer : NSObject
@property int ident;
@property (strong) AVPlayer *player;
@property (strong) AVPlayerItem *item;
@property (strong) AVPlayerItemVideoOutput *output;
@property (strong) dispatch_source_t timer;
@property float rate;
@property BOOL reported;
@end

static void *statusContext = &statusContext;

@implementation GiodeMediaPlayer
- (void)observeValueForKeyPath:(NSString *)keyPath ofObject:(id)object change:(NSDictionary *)change context:(void *)context {
	if (context != statusContext) {
		[super observeValueForKeyPath:keyPath ofObject:object change:change context:context];
		return;
	}
	if (self.item.status == AVPlayerItemStatusReadyToPlay && !self.reported) {
		self.reported = YES;
		CMTime d = self.item.duration;
		double secs = CMTIME_IS_NUMERIC(d) ? CMTimeGetSeconds(d) : 0;
		CGSize sz = self.item.presentationSize;
		giodeMediaReady(self.ident, secs, (int)sz.width, (int)sz.height);
	} else if (self.item.status == AVPlayerItemStatusFailed) {
		NSString *msg = self.item.error.localizedDescription ?: @"playback failed";
		giodeMediaFailed(self.ident, (char *)msg.UTF8String);
	}
}

- (void)ended:(NSNotification *)n {
	giodeMediaEnded(self.ident);
}

- (void)failedToEnd:(NSNotification *)n {
	NSError *e = n.userInfo[AVPlayerItemFailedToPlayToEndTimeErrorKey];
	NSString *msg = e.localizedDescription ?: @"playback failed";
	giodeMediaFailed(self.ident, (char *)msg.UTF8String);
}

- (void)pullFrame {
	CMTime t = [self.output itemTimeForMachAbsoluteTime:mach_absolute_time()];
	if (![self.output hasNewPixelBufferForItemTime:t]) {
		return;
	}
	CVPixelBufferRef pb = [self.output copyPixelBufferForItemTime:t itemTimeForDisplay:NULL];
	if (!pb) {
		return;
	}
	CVPixelBufferLockBaseAddress(pb, kCVPixelBufferLock_ReadOnly);
	giodeMediaFrame(self.ident, CVPixelBufferGetBaseAddress(pb), (int)CVPixelBufferGetWidth(pb),
		(int)CVPixelBufferGetHeight(pb), (int)CVPixelBufferGetBytesPerRow(pb));
	CVPixelBufferUnlockBaseAddress(pb, kCVPixelBufferLock_ReadOnly);
	CVPixelBufferRelease(pb);
}
@end

static NSMutableDictionary<NSNumber *, GiodeMediaPlayer *> *players;

static GiodeMediaPlayer *lookup(int ident) {
	@synchronized ([GiodeMediaPlayer class]) {
		return players[@(ident)];
	}
}

char *giodeMediaOpen(int ident, const char *src, int muted) {
	NSString *s = [NSString stringWithUTF8String:src];
	NSURL *url = [s containsString:@"://"] ? [NSURL URLWithString:s] : [NSURL fileURLWithPath:s];
	if (!url) {
		return strdup("not a valid URL");
	}
#if TARGET_OS_IPHONE
	// Play through the silent switch, unless a call holds the session.
	AVAudioSession *session = [AVAudioSession sharedInstance];
	if (![session.category isEqualToString:AVAudioSessionCategoryPlayAndRecord]) {
		[session setCategory:AVAudioSessionCategoryPlayback error:nil];
		[session setActive:YES error:nil];
	}
#endif
	GiodeMediaPlayer *p = [GiodeMediaPlayer new];
	p.ident = ident;
	p.rate = 1;
	p.item = [AVPlayerItem playerItemWithURL:url];
	// Keep the pitch of speech at other speeds.
	p.item.audioTimePitchAlgorithm = AVAudioTimePitchAlgorithmSpectral;
	NSDictionary *attrs = @{(id)kCVPixelBufferPixelFormatTypeKey: @(kCVPixelFormatType_32BGRA)};
	p.output = [[AVPlayerItemVideoOutput alloc] initWithPixelBufferAttributes:attrs];
	[p.item addOutput:p.output];
	p.player = [AVPlayer playerWithPlayerItem:p.item];
	p.player.actionAtItemEnd = AVPlayerActionAtItemEndPause;
	p.player.muted = muted != 0;
	[p.item addObserver:p forKeyPath:@"status" options:NSKeyValueObservingOptionNew context:statusContext];
	NSNotificationCenter *nc = [NSNotificationCenter defaultCenter];
	[nc addObserver:p selector:@selector(ended:) name:AVPlayerItemDidPlayToEndTimeNotification object:p.item];
	[nc addObserver:p selector:@selector(failedToEnd:) name:AVPlayerItemFailedToPlayToEndTimeNotification object:p.item];
	dispatch_queue_t q = dispatch_queue_create("giode.media", DISPATCH_QUEUE_SERIAL);
	p.timer = dispatch_source_create(DISPATCH_SOURCE_TYPE_TIMER, 0, 0, q);
	dispatch_source_set_timer(p.timer, DISPATCH_TIME_NOW, NSEC_PER_SEC / 120, NSEC_PER_MSEC);
	__weak GiodeMediaPlayer *weak = p;
	dispatch_source_set_event_handler(p.timer, ^{
		[weak pullFrame];
	});
	dispatch_resume(p.timer);
	@synchronized ([GiodeMediaPlayer class]) {
		if (!players) players = [NSMutableDictionary dictionary];
		players[@(ident)] = p;
	}
	return NULL;
}

void giodeMediaPlay(int ident) {
	GiodeMediaPlayer *p = lookup(ident);
	[p.player playImmediatelyAtRate:p.rate];
}

void giodeMediaPause(int ident) {
	[lookup(ident).player pause];
}

void giodeMediaSeek(int ident, double secs) {
	CMTime t = CMTimeMakeWithSeconds(secs, 600);
	[lookup(ident).player seekToTime:t toleranceBefore:kCMTimeZero toleranceAfter:kCMTimeZero];
}

void giodeMediaRate(int ident, double rate) {
	GiodeMediaPlayer *p = lookup(ident);
	p.rate = rate;
	if (p.player.rate != 0) {
		p.player.rate = rate;
	}
}

void giodeMediaVolume(int ident, double v) {
	GiodeMediaPlayer *p = lookup(ident);
	p.player.volume = v;
	p.player.muted = v == 0;
}

double giodeMediaPosition(int ident) {
	GiodeMediaPlayer *p = lookup(ident);
	if (!p) return 0;
	CMTime t = p.player.currentTime;
	return CMTIME_IS_NUMERIC(t) ? CMTimeGetSeconds(t) : 0;
}

void giodeMediaClose(int ident) {
	GiodeMediaPlayer *p;
	@synchronized ([GiodeMediaPlayer class]) {
		p = players[@(ident)];
		[players removeObjectForKey:@(ident)];
	}
	if (!p) return;
	dispatch_source_cancel(p.timer);
	[p.player pause];
	[[NSNotificationCenter defaultCenter] removeObserver:p];
	[p.item removeObserver:p forKeyPath:@"status" context:statusContext];
	[p.player replaceCurrentItemWithPlayerItem:nil];
}
