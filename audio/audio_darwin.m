//go:build darwin

#include <TargetConditionals.h>
@import AVFoundation;
@import AudioToolbox;
#if !TARGET_OS_IPHONE
@import CoreAudio;
#endif
#include <stdatomic.h>
#include <stdlib.h>
#include <string.h>
#include "audio_darwin.h"

// A single-producer, single-consumer queue of samples.
typedef struct {
	int16_t *buf;
	int cap;
	_Atomic int r, w; // read and write counts, mod 2*cap
} gaRing;

static void ringInit(gaRing *q, int cap) {
	q->buf = calloc(cap, sizeof(int16_t));
	q->cap = cap;
	atomic_store(&q->r, 0);
	atomic_store(&q->w, 0);
}

static int ringLen(gaRing *q) {
	int n = atomic_load(&q->w) - atomic_load(&q->r);
	if (n < 0) n += 2 * q->cap;
	return n;
}

static int ringPush(gaRing *q, const int16_t *s, int n) {
	int free = q->cap - ringLen(q);
	if (n > free) n = free;
	int w = atomic_load(&q->w);
	for (int i = 0; i < n; i++) {
		q->buf[(w + i) % q->cap] = s[i];
	}
	atomic_store(&q->w, (w + n) % (2 * q->cap));
	return n;
}

static int ringPop(gaRing *q, int16_t *d, int n) {
	int have = ringLen(q);
	if (n > have) n = have;
	int r = atomic_load(&q->r);
	for (int i = 0; i < n; i++) {
		d[i] = q->buf[(r + i) % q->cap];
	}
	atomic_store(&q->r, (r + n) % (2 * q->cap));
	return n;
}

// gaUnit is an audio unit capturing, rendering or both.
typedef struct {
	AudioUnit unit;
	int channels;
	int capture, render;
	gaRing in, out;
	dispatch_semaphore_t ready;
	AudioBufferList *scratch;
	int scratchFrames;
	_Atomic int closed;
} gaUnit;

static OSStatus inputProc(void *ref, AudioUnitRenderActionFlags *flags, const AudioTimeStamp *ts, UInt32 bus, UInt32 frames, AudioBufferList *unused) {
	gaUnit *u = ref;
	if ((int)frames > u->scratchFrames) {
		return noErr; // larger than ever asked for: drop it
	}
	u->scratch->mNumberBuffers = 1;
	u->scratch->mBuffers[0].mNumberChannels = u->channels;
	u->scratch->mBuffers[0].mDataByteSize = frames * 2 * u->channels;
	OSStatus st = AudioUnitRender(u->unit, flags, ts, 1, frames, u->scratch);
	if (st != noErr) {
		return st;
	}
	ringPush(&u->in, u->scratch->mBuffers[0].mData, frames * u->channels);
	dispatch_semaphore_signal(u->ready);
	return noErr;
}

static OSStatus renderProc(void *ref, AudioUnitRenderActionFlags *flags, const AudioTimeStamp *ts, UInt32 bus, UInt32 frames, AudioBufferList *io) {
	gaUnit *u = ref;
	for (UInt32 b = 0; b < io->mNumberBuffers; b++) {
		int16_t *d = io->mBuffers[b].mData;
		int n = io->mBuffers[b].mDataByteSize / 2;
		int got = u->render ? ringPop(&u->out, d, n) : 0;
		if (got < n) {
			memset(d + got, 0, (n - got) * 2);
		}
	}
	return noErr;
}

static void setErr(char **err, const char *what, OSStatus st) {
	char buf[160];
	snprintf(buf, sizeof buf, "audio: %s (%d)", what, (int)st);
	*err = strdup(buf);
}

#if !TARGET_OS_IPHONE
// deviceForUID returns the device with a UID, or 0.
static AudioDeviceID deviceForUID(const char *uid) {
	AudioObjectPropertyAddress a = {kAudioHardwarePropertyDevices, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
	UInt32 size = 0;
	if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &a, 0, NULL, &size) != noErr) {
		return 0;
	}
	int n = size / sizeof(AudioDeviceID);
	AudioDeviceID *ids = malloc(size);
	AudioDeviceID found = 0;
	if (AudioObjectGetPropertyData(kAudioObjectSystemObject, &a, 0, NULL, &size, ids) == noErr) {
		for (int i = 0; i < n && !found; i++) {
			CFStringRef s = NULL;
			UInt32 ss = sizeof s;
			AudioObjectPropertyAddress ua = {kAudioDevicePropertyDeviceUID, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
			if (AudioObjectGetPropertyData(ids[i], &ua, 0, NULL, &ss, &s) == noErr && s) {
				char buf[512];
				if (CFStringGetCString(s, buf, sizeof buf, kCFStringEncodingUTF8) && strcmp(buf, uid) == 0) {
					found = ids[i];
				}
				CFRelease(s);
			}
		}
	}
	free(ids);
	return found;
}
#endif

static void configureSession(int capture, int voice, int speaker) {
#if TARGET_OS_IPHONE
	AVAudioSession *s = [AVAudioSession sharedInstance];
	AVAudioSessionCategory cat = AVAudioSessionCategoryPlayback;
	AVAudioSessionCategoryOptions opts = 0;
	AVAudioSessionMode mode = AVAudioSessionModeDefault;
	if (capture || voice) {
		cat = AVAudioSessionCategoryPlayAndRecord;
		opts = AVAudioSessionCategoryOptionAllowBluetooth | AVAudioSessionCategoryOptionAllowBluetoothA2DP;
		if (!voice || speaker) {
			opts |= AVAudioSessionCategoryOptionDefaultToSpeaker;
		}
	}
	if (voice) {
		mode = AVAudioSessionModeVoiceChat;
	}
	[s setCategory:cat mode:mode options:opts error:nil];
	[s setActive:YES error:nil];
#endif
}

void *giodeAudioOpen(int capture, int render, int voice, int speaker, double rate, int channels, const char *device, char **err) {
	configureSession(capture, voice, speaker);
	AudioComponentDescription desc = {0};
	desc.componentType = kAudioUnitType_Output;
	desc.componentManufacturer = kAudioUnitManufacturer_Apple;
#if TARGET_OS_IPHONE
	desc.componentSubType = voice ? kAudioUnitSubType_VoiceProcessingIO : kAudioUnitSubType_RemoteIO;
#else
	desc.componentSubType = voice ? kAudioUnitSubType_VoiceProcessingIO : kAudioUnitSubType_HALOutput;
#endif
	AudioComponent comp = AudioComponentFindNext(NULL, &desc);
	if (!comp) {
		setErr(err, "no audio unit", 0);
		return NULL;
	}
	gaUnit *u = calloc(1, sizeof(gaUnit));
	u->channels = channels;
	u->capture = capture;
	u->render = render;
	int cap = (int)(rate * 0.5) * channels; // half a second
	ringInit(&u->in, cap);
	ringInit(&u->out, cap);
	u->ready = dispatch_semaphore_create(0);
	u->scratchFrames = 8192;
	u->scratch = calloc(1, sizeof(AudioBufferList));
	u->scratch->mBuffers[0].mData = calloc(u->scratchFrames * channels, 2);
	OSStatus st = AudioComponentInstanceNew(comp, &u->unit);
	if (st != noErr) {
		setErr(err, "creating the audio unit", st);
		return NULL;
	}
	UInt32 on = 1, off = 0;
	int voiceUnit = voice != 0;
	// A voice unit always runs both ways; others only what's asked.
	if (capture || voiceUnit) {
		st = AudioUnitSetProperty(u->unit, kAudioOutputUnitProperty_EnableIO, kAudioUnitScope_Input, 1, &on, sizeof on);
		if (st != noErr) {
			setErr(err, "enabling input", st);
			AudioComponentInstanceDispose(u->unit);
			return NULL;
		}
	}
	if (!render && !voiceUnit) {
		AudioUnitSetProperty(u->unit, kAudioOutputUnitProperty_EnableIO, kAudioUnitScope_Output, 0, &off, sizeof off);
	}
#if !TARGET_OS_IPHONE
	if (device && device[0]) {
		AudioDeviceID dev = deviceForUID(device);
		if (dev) {
			AudioUnitSetProperty(u->unit, kAudioOutputUnitProperty_CurrentDevice, kAudioUnitScope_Global, 0, &dev, sizeof dev);
		}
	} else if (capture && !voiceUnit) {
		// The HAL unit starts on the default output device; capture
		// from the default input.
		AudioDeviceID dev = 0;
		UInt32 size = sizeof dev;
		AudioObjectPropertyAddress a = {kAudioHardwarePropertyDefaultInputDevice, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
		if (AudioObjectGetPropertyData(kAudioObjectSystemObject, &a, 0, NULL, &size, &dev) == noErr) {
			AudioUnitSetProperty(u->unit, kAudioOutputUnitProperty_CurrentDevice, kAudioUnitScope_Global, 0, &dev, sizeof dev);
		}
	}
#endif
	AudioStreamBasicDescription f = {0};
	f.mSampleRate = rate;
	f.mFormatID = kAudioFormatLinearPCM;
	f.mFormatFlags = kAudioFormatFlagIsSignedInteger | kAudioFormatFlagIsPacked;
	f.mBytesPerPacket = 2 * channels;
	f.mFramesPerPacket = 1;
	f.mBytesPerFrame = 2 * channels;
	f.mChannelsPerFrame = channels;
	f.mBitsPerChannel = 16;
	if (capture || voiceUnit) {
		st = AudioUnitSetProperty(u->unit, kAudioUnitProperty_StreamFormat, kAudioUnitScope_Output, 1, &f, sizeof f);
		if (st != noErr) {
			setErr(err, "setting the input format", st);
			AudioComponentInstanceDispose(u->unit);
			return NULL;
		}
		AURenderCallbackStruct cb = {inputProc, u};
		AudioUnitSetProperty(u->unit, kAudioOutputUnitProperty_SetInputCallback, kAudioUnitScope_Global, 1, &cb, sizeof cb);
	}
	if (render || voiceUnit) {
		st = AudioUnitSetProperty(u->unit, kAudioUnitProperty_StreamFormat, kAudioUnitScope_Input, 0, &f, sizeof f);
		if (st != noErr) {
			setErr(err, "setting the output format", st);
			AudioComponentInstanceDispose(u->unit);
			return NULL;
		}
		AURenderCallbackStruct cb = {renderProc, u};
		AudioUnitSetProperty(u->unit, kAudioUnitProperty_SetRenderCallback, kAudioUnitScope_Input, 0, &cb, sizeof cb);
	}
	if ((st = AudioUnitInitialize(u->unit)) != noErr) {
		setErr(err, "initializing the audio unit", st);
		AudioComponentInstanceDispose(u->unit);
		return NULL;
	}
	if ((st = AudioOutputUnitStart(u->unit)) != noErr) {
		setErr(err, "starting the audio unit", st);
		AudioUnitUninitialize(u->unit);
		AudioComponentInstanceDispose(u->unit);
		return NULL;
	}
	return u;
}

void giodeAudioSetRender(void *ref, int on) {
	((gaUnit *)ref)->render = on;
}

void giodeAudioClose(void *ref) {
	gaUnit *u = ref;
	atomic_store(&u->closed, 1);
	AudioOutputUnitStop(u->unit);
	AudioUnitUninitialize(u->unit);
	AudioComponentInstanceDispose(u->unit);
	dispatch_semaphore_signal(u->ready);
	// The unit stopped calling back; the memory may go.
	free(u->in.buf);
	free(u->out.buf);
	free(u->scratch->mBuffers[0].mData);
	free(u->scratch);
	free(u);
}

// giodeAudioRead waits up to ms milliseconds for captured samples and
// copies up to n of them; -1 once closed.
int giodeAudioRead(void *ref, int16_t *dst, int n, int ms) {
	gaUnit *u = ref;
	if (ringLen(&u->in) == 0) {
		dispatch_semaphore_wait(u->ready, dispatch_time(DISPATCH_TIME_NOW, (int64_t)ms * NSEC_PER_MSEC));
	}
	if (atomic_load(&u->closed)) {
		return -1;
	}
	return ringPop(&u->in, dst, n);
}

// giodeAudioWrite queues samples to play, returning how many fit.
int giodeAudioWrite(void *ref, const int16_t *src, int n) {
	return ringPush(&((gaUnit *)ref)->out, src, n);
}

// giodeAudioPermission asks for the microphone and waits for the
// answer: 1 allowed, 0 not.
int giodeAudioPermission(void) {
	AVAuthorizationStatus st = [AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeAudio];
	if (st == AVAuthorizationStatusAuthorized) {
		return 1;
	}
	if (st != AVAuthorizationStatusNotDetermined) {
		return 0;
	}
	dispatch_semaphore_t done = dispatch_semaphore_create(0);
	__block int ok = 0;
	[AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL granted) {
		ok = granted;
		dispatch_semaphore_signal(done);
	}];
	dispatch_semaphore_wait(done, DISPATCH_TIME_FOREVER);
	return ok;
}

// giodeAudioDevices lists devices as "uid\tname\tdefault\n" lines; the
// caller frees the result.
char *giodeAudioDevices(int input) {
	NSMutableString *out = [NSMutableString string];
#if TARGET_OS_IPHONE
	AVAudioSession *s = [AVAudioSession sharedInstance];
	if (input) {
		NSString *cur = s.currentRoute.inputs.firstObject.UID;
		for (AVAudioSessionPortDescription *p in s.availableInputs) {
			[out appendFormat:@"%@\t%@\t%d\n", p.UID, p.portName, [p.UID isEqualToString:cur]];
		}
	} else {
		for (AVAudioSessionPortDescription *p in s.currentRoute.outputs) {
			[out appendFormat:@"%@\t%@\t1\n", p.UID, p.portName];
		}
	}
#else
	AudioObjectPropertyAddress a = {kAudioHardwarePropertyDevices, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
	UInt32 size = 0;
	AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &a, 0, NULL, &size);
	int n = size / sizeof(AudioDeviceID);
	AudioDeviceID *ids = malloc(size > 0 ? size : 1);
	AudioObjectGetPropertyData(kAudioObjectSystemObject, &a, 0, NULL, &size, ids);
	AudioDeviceID def = 0;
	UInt32 ds = sizeof def;
	AudioObjectPropertyAddress da = {input ? kAudioHardwarePropertyDefaultInputDevice : kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
	AudioObjectGetPropertyData(kAudioObjectSystemObject, &da, 0, NULL, &ds, &def);
	for (int i = 0; i < n; i++) {
		// Only devices with channels in the direction asked.
		AudioObjectPropertyAddress ca = {kAudioDevicePropertyStreamConfiguration,
			input ? kAudioDevicePropertyScopeInput : kAudioDevicePropertyScopeOutput, kAudioObjectPropertyElementMain};
		UInt32 cs = 0;
		if (AudioObjectGetPropertyDataSize(ids[i], &ca, 0, NULL, &cs) != noErr || cs == 0) {
			continue;
		}
		AudioBufferList *bl = malloc(cs);
		int chans = 0;
		if (AudioObjectGetPropertyData(ids[i], &ca, 0, NULL, &cs, bl) == noErr) {
			for (UInt32 b = 0; b < bl->mNumberBuffers; b++) {
				chans += bl->mBuffers[b].mNumberChannels;
			}
		}
		free(bl);
		if (chans == 0) {
			continue;
		}
		CFStringRef uid = NULL, name = NULL;
		UInt32 ss = sizeof uid;
		AudioObjectPropertyAddress ua = {kAudioDevicePropertyDeviceUID, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
		AudioObjectPropertyAddress na = {kAudioObjectPropertyName, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
		AudioObjectGetPropertyData(ids[i], &ua, 0, NULL, &ss, &uid);
		ss = sizeof name;
		AudioObjectGetPropertyData(ids[i], &na, 0, NULL, &ss, &name);
		if (uid) {
			[out appendFormat:@"%@\t%@\t%d\n", (__bridge NSString *)uid, name ? (__bridge NSString *)name : @"", ids[i] == def];
		}
		if (uid) CFRelease(uid);
		if (name) CFRelease(name);
	}
	free(ids);
#endif
	return strdup(out.UTF8String);
}

// giodeAudioPreferInput chooses the input on iOS.
void giodeAudioPreferInput(const char *uid) {
#if TARGET_OS_IPHONE
	AVAudioSession *s = [AVAudioSession sharedInstance];
	NSString *want = @(uid);
	for (AVAudioSessionPortDescription *p in s.availableInputs) {
		if ([p.UID isEqualToString:want]) {
			[s setPreferredInput:p error:nil];
		}
	}
#endif
}
