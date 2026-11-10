//go:build ios

// Dictation, with SFSpeechRecognizer fed from the microphone: only apps
// importing this package link Speech and record from the microphone.

#import <UIKit/UIKit.h>
#import <AVFoundation/AVFoundation.h>
#import <Speech/Speech.h>
#include "_cgo_export.h"

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

int giodeDictateHasInfoKey(const char *key) {
	NSString *k = [NSString stringWithUTF8String:key];
	return [[NSBundle mainBundle] objectForInfoDictionaryKey:k] != nil;
}

@interface GiodeDictation : NSObject
@property int requestID;
@property(strong) SFSpeechRecognizer *recognizer;
@property(strong) AVAudioEngine *engine;
@property(strong) SFSpeechAudioBufferRecognitionRequest *request;
@property(strong) SFSpeechRecognitionTask *task;
@property(strong) NSTimer *timer;
@property(copy) NSString *lastText;
// ended is set once the audio is ended and the final result awaited;
// done once that result is reported.
@property BOOL ended;
@property BOOL done;
@end

@implementation GiodeDictation
@end

// dictations holds the running dictations by id. Only touched on the
// main thread.
static NSMutableDictionary<NSNumber *, GiodeDictation *> *dictations;

// How long a pause ends the dictation, as Android's recognizer does;
// before anything is said, the wait is longer.
static const NSTimeInterval giodeSilence = 2.0;
static const NSTimeInterval giodeFirstWords = 6.0;

static void giodeFinishDictation(GiodeDictation *d, NSString *errMsg) {
	if (d.done) {
		return;
	}
	d.done = YES;
	[d.timer invalidate];
	d.timer = nil;
	if (d.engine.isRunning) {
		[d.engine stop];
	}
	[d.engine.inputNode removeTapOnBus:0];
	[d.request endAudio];
	[d.task cancel];
	[[AVAudioSession sharedInstance] setActive:NO withOptions:AVAudioSessionSetActiveOptionNotifyOthersOnDeactivation error:nil];
	[dictations removeObjectForKey:@(d.requestID)];
	NSString *text = d.lastText ?: @"";
	giodeSpeech(d.requestID, (char *)text.UTF8String, 1, errMsg != nil ? (char *)errMsg.UTF8String : NULL);
}

// giodeEndAudio stops listening; the recognizer then sends its final
// result, or the fallback timer reports what it has.
static void giodeEndAudio(GiodeDictation *d) {
	if (d.ended || d.done) {
		return;
	}
	d.ended = YES;
	[d.engine stop];
	[d.engine.inputNode removeTapOnBus:0];
	[d.request endAudio];
	[d.timer invalidate];
	__weak GiodeDictation *wd = d;
	d.timer = [NSTimer scheduledTimerWithTimeInterval:3.0 repeats:NO block:^(NSTimer *t) {
		GiodeDictation *d = wd;
		if (d != nil) {
			giodeFinishDictation(d, nil);
		}
	}];
}

static void giodeArmSilence(GiodeDictation *d, NSTimeInterval wait) {
	if (d.ended) {
		return;
	}
	[d.timer invalidate];
	__weak GiodeDictation *wd = d;
	d.timer = [NSTimer scheduledTimerWithTimeInterval:wait repeats:NO block:^(NSTimer *t) {
		GiodeDictation *d = wd;
		if (d != nil) {
			giodeEndAudio(d);
		}
	}];
}

static void giodeStartDictation(int rid, NSString *lang) {
	SFSpeechRecognizer *rec = lang.length > 0
		? [[SFSpeechRecognizer alloc] initWithLocale:[NSLocale localeWithLocaleIdentifier:lang]]
		: [[SFSpeechRecognizer alloc] init];
	if (rec == nil || !rec.isAvailable) {
		giodeSpeech(rid, (char *)"", 1, (char *)"speech recognition isn't available for this language");
		return;
	}
	NSError *err = nil;
	AVAudioSession *session = [AVAudioSession sharedInstance];
	if (![session setCategory:AVAudioSessionCategoryRecord mode:AVAudioSessionModeMeasurement options:AVAudioSessionCategoryOptionDuckOthers error:&err] ||
		![session setActive:YES withOptions:AVAudioSessionSetActiveOptionNotifyOthersOnDeactivation error:&err]) {
		giodeSpeech(rid, (char *)"", 1, (char *)err.localizedDescription.UTF8String);
		return;
	}
	GiodeDictation *d = [[GiodeDictation alloc] init];
	d.requestID = rid;
	d.recognizer = rec;
	d.engine = [[AVAudioEngine alloc] init];
	d.request = [[SFSpeechAudioBufferRecognitionRequest alloc] init];
	d.request.shouldReportPartialResults = YES;
	SFSpeechAudioBufferRecognitionRequest *req = d.request;
	AVAudioInputNode *input = d.engine.inputNode;
	[input installTapOnBus:0 bufferSize:1024 format:[input outputFormatForBus:0] block:^(AVAudioPCMBuffer *buf, AVAudioTime *when) {
		[req appendAudioPCMBuffer:buf];
	}];
	[d.engine prepare];
	if (![d.engine startAndReturnError:&err]) {
		[input removeTapOnBus:0];
		[session setActive:NO error:nil];
		giodeSpeech(rid, (char *)"", 1, (char *)err.localizedDescription.UTF8String);
		return;
	}
	if (dictations == nil) {
		dictations = [NSMutableDictionary dictionary];
	}
	dictations[@(rid)] = d;
	__weak GiodeDictation *wd = d;
	d.task = [rec recognitionTaskWithRequest:req resultHandler:^(SFSpeechRecognitionResult *result, NSError *error) {
		dispatch_async(dispatch_get_main_queue(), ^{
			GiodeDictation *d = wd;
			if (d == nil || d.done) {
				return;
			}
			if (result != nil) {
				d.lastText = result.bestTranscription.formattedString;
				if (result.isFinal) {
					giodeFinishDictation(d, nil);
					return;
				}
				giodeSpeech(d.requestID, (char *)d.lastText.UTF8String, 0, NULL);
				giodeArmSilence(d, giodeSilence);
			}
			if (error != nil) {
				// Once the audio has ended, an error (no speech, say)
				// just ends the dictation with what was heard.
				giodeFinishDictation(d, d.ended ? nil : error.localizedDescription);
			}
		});
	}];
	giodeArmSilence(d, giodeFirstWords);
}

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
void giodeDictate(int rid, const char *lang) {
	NSString *l = [NSString stringWithUTF8String:lang];
	[SFSpeechRecognizer requestAuthorization:^(SFSpeechRecognizerAuthorizationStatus st) {
		if (st != SFSpeechRecognizerAuthorizationStatusAuthorized) {
			giodeSpeech(rid, (char *)"", 1, (char *)"speech recognition isn't allowed for this app (Settings > Privacy & Security > Speech Recognition)");
			return;
		}
		// AVAudioApplication replaces this on iOS 17.
		[[AVAudioSession sharedInstance] requestRecordPermission:^(BOOL granted) {
			dispatch_async(dispatch_get_main_queue(), ^{
				if (!granted) {
					giodeSpeech(rid, (char *)"", 1, (char *)"microphone access isn't allowed for this app (Settings > Privacy & Security > Microphone)");
					return;
				}
				giodeStartDictation(rid, l);
			});
		}];
	}];
}
#pragma clang diagnostic pop

void giodeStopDictation(int rid) {
	onMain(^{
		GiodeDictation *d = dictations[@(rid)];
		if (d != nil) {
			giodeEndAudio(d);
		}
	});
}
