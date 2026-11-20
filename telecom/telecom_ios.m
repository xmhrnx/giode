//go:build ios

#include <stdlib.h>
#include <string.h>
@import Foundation;
@import CallKit;
@import PushKit;
@import AVFoundation;
#include "_cgo_export.h"

// giode/telecom on CallKit, with PushKit VoIP pushes to ring the app
// while it's suspended or not running.

@interface GiodeCalls : NSObject <CXProviderDelegate, PKPushRegistryDelegate>
@property (strong) CXProvider *provider;
@property (strong) CXCallController *controller;
@property (strong) PKPushRegistry *registry;
@property (strong) dispatch_queue_t queue;
@end

static GiodeCalls *calls;

static void event(const char *kind, NSString *a, NSString *b) {
	giodeTelecomEvent((char *)kind, (char *)(a ?: @"").UTF8String, (char *)(b ?: @"").UTF8String);
}

@implementation GiodeCalls
- (void)providerDidReset:(CXProvider *)provider {
	event("reset", nil, nil);
}

- (void)provider:(CXProvider *)provider performAnswerCallAction:(CXAnswerCallAction *)action {
	[self configureAudio];
	[action fulfill];
	event("answer", action.callUUID.UUIDString, nil);
}

- (void)provider:(CXProvider *)provider performEndCallAction:(CXEndCallAction *)action {
	[action fulfill];
	event("end", action.callUUID.UUIDString, nil);
}

- (void)provider:(CXProvider *)provider performStartCallAction:(CXStartCallAction *)action {
	[self configureAudio];
	[action fulfill];
	[provider reportOutgoingCallWithUUID:action.callUUID startedConnectingAtDate:nil];
	event("start", action.callUUID.UUIDString, action.handle.value);
}

- (void)provider:(CXProvider *)provider performSetMutedCallAction:(CXSetMutedCallAction *)action {
	[action fulfill];
	event(action.muted ? "mute" : "unmute", action.callUUID.UUIDString, nil);
}

- (void)provider:(CXProvider *)provider performSetHeldCallAction:(CXSetHeldCallAction *)action {
	[action fulfill];
	event(action.onHold ? "hold" : "unhold", action.callUUID.UUIDString, nil);
}

- (void)provider:(CXProvider *)provider didActivateAudioSession:(AVAudioSession *)session {
	event("audio_on", nil, nil);
}

- (void)provider:(CXProvider *)provider didDeactivateAudioSession:(AVAudioSession *)session {
	event("audio_off", nil, nil);
}

// configureAudio sets up the session CallKit activates: a voice chat.
- (void)configureAudio {
	AVAudioSession *s = [AVAudioSession sharedInstance];
	[s setCategory:AVAudioSessionCategoryPlayAndRecord mode:AVAudioSessionModeVoiceChat
		options:AVAudioSessionCategoryOptionAllowBluetooth error:nil];
}

- (void)pushRegistry:(PKPushRegistry *)registry didUpdatePushCredentials:(PKPushCredentials *)creds forType:(PKPushType)type {
	const unsigned char *b = creds.token.bytes;
	NSMutableString *hex = [NSMutableString string];
	for (NSUInteger i = 0; i < creds.token.length; i++) {
		[hex appendFormat:@"%02x", b[i]];
	}
	event("voip_token", hex, nil);
}

- (void)pushRegistry:(PKPushRegistry *)registry didReceiveIncomingPushWithPayload:(PKPushPayload *)payload
		forType:(PKPushType)type withCompletionHandler:(void (^)(void))completion {
	NSData *json = [NSJSONSerialization dataWithJSONObject:payload.dictionaryPayload options:0 error:nil];
	NSString *s = json ? [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding] : @"{}";
	// iOS ends the app unless the push reports a call before returning.
	char *call = giodeTelecomPush((char *)s.UTF8String);
	NSArray<NSString *> *f = [[NSString stringWithUTF8String:call] componentsSeparatedByString:@"\t"];
	free(call);
	[self reportIncoming:f[0] handle:f[1] name:f[2] video:[f[3] isEqualToString:@"1"] completion:^(NSError *e) {
		completion();
	}];
}

- (void)reportIncoming:(NSString *)uuid handle:(NSString *)handle name:(NSString *)name video:(BOOL)video
		completion:(void (^)(NSError *))done {
	CXCallUpdate *u = [CXCallUpdate new];
	u.remoteHandle = [[CXHandle alloc] initWithType:CXHandleTypeGeneric value:handle];
	u.localizedCallerName = name;
	u.hasVideo = video;
	u.supportsHolding = YES;
	u.supportsGrouping = NO;
	u.supportsUngrouping = NO;
	u.supportsDTMF = NO;
	NSUUID *uid = [[NSUUID alloc] initWithUUIDString:uuid] ?: [NSUUID UUID];
	[self.provider reportNewIncomingCallWithUUID:uid update:u completion:done];
}
@end

void giodeTelecomSetup(int video) {
	if (calls) return;
	calls = [GiodeCalls new];
	calls.queue = dispatch_queue_create("giode.telecom", DISPATCH_QUEUE_SERIAL);
	CXProviderConfiguration *cfg = [[CXProviderConfiguration alloc] init];
	cfg.supportsVideo = video != 0;
	cfg.maximumCallGroups = 1;
	cfg.maximumCallsPerCallGroup = 1;
	cfg.supportedHandleTypes = [NSSet setWithObject:@(CXHandleTypeGeneric)];
	cfg.includesCallsInRecents = YES;
	calls.provider = [[CXProvider alloc] initWithConfiguration:cfg];
	[calls.provider setDelegate:calls queue:calls.queue];
	calls.controller = [[CXCallController alloc] initWithQueue:calls.queue];
	calls.registry = [[PKPushRegistry alloc] initWithQueue:calls.queue];
	calls.registry.delegate = calls;
	calls.registry.desiredPushTypes = [NSSet setWithObject:PKPushTypeVoIP];
}

static char *waitError(void (^start)(void (^)(NSError *))) {
	dispatch_semaphore_t sem = dispatch_semaphore_create(0);
	__block NSString *msg = nil;
	start(^(NSError *e) {
		if (e) msg = e.localizedDescription;
		dispatch_semaphore_signal(sem);
	});
	dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, 10 * NSEC_PER_SEC));
	return msg ? strdup(msg.UTF8String) : NULL;
}

char *giodeTelecomIncoming(const char *uuid, const char *handle, const char *name, int video) {
	NSString *u = [NSString stringWithUTF8String:uuid], *h = [NSString stringWithUTF8String:handle], *n = [NSString stringWithUTF8String:name];
	return waitError(^(void (^done)(NSError *)) {
		[calls reportIncoming:u handle:h name:n video:video != 0 completion:done];
	});
}

static char *transaction(CXAction *a) {
	return waitError(^(void (^done)(NSError *)) {
		[calls.controller requestTransaction:[[CXTransaction alloc] initWithAction:a] completion:done];
	});
}

char *giodeTelecomStart(const char *uuid, const char *handle, const char *name, int video) {
	NSUUID *uid = [[NSUUID alloc] initWithUUIDString:[NSString stringWithUTF8String:uuid]];
	CXHandle *h = [[CXHandle alloc] initWithType:CXHandleTypeGeneric value:[NSString stringWithUTF8String:handle]];
	CXStartCallAction *a = [[CXStartCallAction alloc] initWithCallUUID:uid handle:h];
	a.video = video != 0;
	a.contactIdentifier = [NSString stringWithUTF8String:name];
	return transaction(a);
}

void giodeTelecomConnected(const char *uuid) {
	NSUUID *uid = [[NSUUID alloc] initWithUUIDString:[NSString stringWithUTF8String:uuid]];
	[calls.provider reportOutgoingCallWithUUID:uid connectedAtDate:nil];
}

char *giodeTelecomEnd(const char *uuid) {
	NSUUID *uid = [[NSUUID alloc] initWithUUIDString:[NSString stringWithUTF8String:uuid]];
	return transaction([[CXEndCallAction alloc] initWithCallUUID:uid]);
}

void giodeTelecomEnded(const char *uuid, int reason) {
	NSUUID *uid = [[NSUUID alloc] initWithUUIDString:[NSString stringWithUTF8String:uuid]];
	CXCallEndedReason r = CXCallEndedReasonRemoteEnded;
	switch (reason) {
	case 1: r = CXCallEndedReasonFailed; break;
	case 2: r = CXCallEndedReasonUnanswered; break;
	case 3: r = CXCallEndedReasonAnsweredElsewhere; break;
	case 4: r = CXCallEndedReasonDeclinedElsewhere; break;
	}
	[calls.provider reportCallWithUUID:uid endedAtDate:nil reason:r];
}

char *giodeTelecomMute(const char *uuid, int muted) {
	NSUUID *uid = [[NSUUID alloc] initWithUUIDString:[NSString stringWithUTF8String:uuid]];
	return transaction([[CXSetMutedCallAction alloc] initWithCallUUID:uid muted:muted != 0]);
}
