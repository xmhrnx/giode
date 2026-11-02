#import <UIKit/UIKit.h>
#import <PhotosUI/PhotosUI.h>
#import <Photos/Photos.h>
#import <AVFoundation/AVFoundation.h>
#import <Speech/Speech.h>
#include <stdint.h>
#include <objc/runtime.h>
#include "_cgo_export.h"

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

void giodeOpenURL(const char *url) {
	NSURL *u = [NSURL URLWithString:[NSString stringWithUTF8String:url]];
	onMain(^{
		[[UIApplication sharedApplication] openURL:u options:@{} completionHandler:nil];
	});
}

void giodeShare(CFTypeRef vcRef, const char *text, const char *path, const char *subject) {
	UIViewController *vc = (__bridge UIViewController *)vcRef;
	NSMutableArray *items = [NSMutableArray array];
	if (text != NULL) {
		[items addObject:[NSString stringWithUTF8String:text]];
	}
	if (path != NULL) {
		[items addObject:[NSURL fileURLWithPath:[NSString stringWithUTF8String:path]]];
	}
	NSString *subj = subject != NULL ? [NSString stringWithUTF8String:subject] : nil;
	onMain(^{
		UIActivityViewController *a = [[UIActivityViewController alloc] initWithActivityItems:items applicationActivities:nil];
		if (subj.length > 0) {
			[a setValue:subj forKey:@"subject"];
		}
		// On iPad the sheet is a popover and needs an anchor: the
		// middle of the window.
		UIPopoverPresentationController *pop = a.popoverPresentationController;
		if (pop != nil) {
			pop.sourceView = vc.view;
			CGRect b = vc.view.bounds;
			pop.sourceRect = CGRectMake(CGRectGetMidX(b), CGRectGetMidY(b), 1, 1);
			pop.permittedArrowDirections = 0;
		}
		[vc presentViewController:a animated:YES completion:nil];
	});
}

void giodeHaptic(int kind) {
	onMain(^{
		switch (kind) {
		case 0:
			[[[UIImpactFeedbackGenerator alloc] initWithStyle:UIImpactFeedbackStyleLight] impactOccurred];
			break;
		case 1:
			[[[UIImpactFeedbackGenerator alloc] initWithStyle:UIImpactFeedbackStyleMedium] impactOccurred];
			break;
		case 2:
			[[[UIImpactFeedbackGenerator alloc] initWithStyle:UIImpactFeedbackStyleHeavy] impactOccurred];
			break;
		case 3:
			[[[UINotificationFeedbackGenerator alloc] init] notificationOccurred:UINotificationFeedbackTypeSuccess];
			break;
		default:
			[[[UINotificationFeedbackGenerator alloc] init] notificationOccurred:UINotificationFeedbackTypeWarning];
			break;
		}
	});
}

void giodeIdleTimerDisabled(int disabled) {
	onMain(^{
		[UIApplication sharedApplication].idleTimerDisabled = disabled != 0;
	});
}

uintptr_t giodeBeginBackgroundTask(const char *name) {
	__block UIBackgroundTaskIdentifier task = UIBackgroundTaskInvalid;
	NSString *n = [NSString stringWithUTF8String:name];
	task = [[UIApplication sharedApplication] beginBackgroundTaskWithName:n expirationHandler:^{
		// Time's up: iOS suspends the app once the task ends.
		[[UIApplication sharedApplication] endBackgroundTask:task];
		task = UIBackgroundTaskInvalid;
	}];
	return (uintptr_t)task;
}

void giodeEndBackgroundTask(uintptr_t task) {
	if ((UIBackgroundTaskIdentifier)task != UIBackgroundTaskInvalid) {
		[[UIApplication sharedApplication] endBackgroundTask:(UIBackgroundTaskIdentifier)task];
	}
}

void giodeWatchLifecycle(void) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		NSNotificationCenter *nc = [NSNotificationCenter defaultCenter];
		[nc addObserverForName:UIApplicationDidEnterBackgroundNotification object:nil queue:nil usingBlock:^(NSNotification *n) {
			giodeOnBackground();
		}];
		[nc addObserverForName:UIApplicationWillEnterForegroundNotification object:nil queue:nil usingBlock:^(NSNotification *n) {
			giodeOnForeground();
		}];
	});
}

API_AVAILABLE(ios(14))
@interface GiodePhotoDelegate : NSObject <PHPickerViewControllerDelegate>
@property int requestID;
@property int maxSide;
@end

// pending keeps delegates alive until their picker finishes.
static NSMutableSet *pending;

static NSData *giodeJPEG(UIImage *img, int maxSide) {
	CGSize size = img.size;
	CGFloat longest = MAX(size.width, size.height);
	if (maxSide > 0 && longest > maxSide) {
		CGFloat s = maxSide / longest;
		size = CGSizeMake(floor(size.width * s), floor(size.height * s));
	}
	UIGraphicsImageRendererFormat *f = [UIGraphicsImageRendererFormat defaultFormat];
	f.scale = 1;
	UIGraphicsImageRenderer *r = [[UIGraphicsImageRenderer alloc] initWithSize:size format:f];
	// Drawing applies the image's orientation, so the JPEG is upright.
	UIImage *out = [r imageWithActions:^(UIGraphicsImageRendererContext *ctx) {
		[img drawInRect:CGRectMake(0, 0, size.width, size.height)];
	}];
	return UIImageJPEGRepresentation(out, 0.9);
}

@implementation GiodePhotoDelegate
- (void)picker:(PHPickerViewController *)picker didFinishPicking:(NSArray<PHPickerResult *> *)results API_AVAILABLE(ios(14)) {
	[picker dismissViewControllerAnimated:YES completion:nil];
	int rid = self.requestID;
	int maxSide = self.maxSide;
	[pending removeObject:self];
	if (results.count == 0) {
		giodePhotoDone(rid, NULL, 0, NULL, NULL);
		return;
	}
	NSItemProvider *p = results[0].itemProvider;
	NSString *name = p.suggestedName ?: @"photo";
	if (![p canLoadObjectOfClass:[UIImage class]]) {
		giodePhotoDone(rid, NULL, 0, NULL, (char *)"the picked item isn't an image");
		return;
	}
	// UIImage decodes HEIC, so it never reaches the app.
	[p loadObjectOfClass:[UIImage class] completionHandler:^(id<NSItemProviderReading> obj, NSError *err) {
		if (err != nil || obj == nil) {
			NSString *msg = err != nil ? err.localizedDescription : @"can't load the image";
			giodePhotoDone(rid, NULL, 0, NULL, (char *)msg.UTF8String);
			return;
		}
		NSData *jpeg = giodeJPEG((UIImage *)obj, maxSide);
		giodePhotoDone(rid, (void *)jpeg.bytes, (int)jpeg.length, (char *)name.UTF8String, NULL);
	}];
}
@end

int giodePickPhoto(CFTypeRef vcRef, int rid, int maxSide) {
	if (@available(iOS 14, *)) {
		UIViewController *vc = (__bridge UIViewController *)vcRef;
		onMain(^{
			PHPickerConfiguration *cfg = [[PHPickerConfiguration alloc] init];
			cfg.filter = [PHPickerFilter imagesFilter];
			cfg.selectionLimit = 1;
			PHPickerViewController *picker = [[PHPickerViewController alloc] initWithConfiguration:cfg];
			GiodePhotoDelegate *d = [[GiodePhotoDelegate alloc] init];
			d.requestID = rid;
			d.maxSide = maxSide;
			if (pending == nil) {
				pending = [NSMutableSet set];
			}
			[pending addObject:d];
			picker.delegate = d;
			[vc presentViewController:picker animated:YES completion:nil];
		});
		return 1;
	}
	return 0;
}

// GiodePhotosDelegate collects a multiple pick, in the order picked.
@interface GiodePhotosDelegate : NSObject <PHPickerViewControllerDelegate>
@property(nonatomic) int requestID;
@property(nonatomic) int maxSide;
@end

@implementation GiodePhotosDelegate
- (void)picker:(PHPickerViewController *)picker didFinishPicking:(NSArray<PHPickerResult *> *)results API_AVAILABLE(ios(14)) {
	[picker dismissViewControllerAnimated:YES completion:nil];
	int rid = self.requestID;
	int maxSide = self.maxSide;
	[pending removeObject:self];
	NSUInteger n = results.count;
	if (n == 0) {
		giodePhotosDone(rid, 0, NULL);
		return;
	}
	NSMutableArray *jpegs = [NSMutableArray arrayWithCapacity:n];
	NSMutableArray *names = [NSMutableArray arrayWithCapacity:n];
	for (NSUInteger i = 0; i < n; i++) {
		[jpegs addObject:[NSNull null]];
		[names addObject:@"photo"];
	}
	dispatch_group_t group = dispatch_group_create();
	__block NSString *failure = nil;
	for (NSUInteger i = 0; i < n; i++) {
		NSItemProvider *p = results[i].itemProvider;
		if (p.suggestedName != nil) {
			names[i] = p.suggestedName;
		}
		if (![p canLoadObjectOfClass:[UIImage class]]) {
			continue;
		}
		dispatch_group_enter(group);
		[p loadObjectOfClass:[UIImage class] completionHandler:^(id<NSItemProviderReading> obj, NSError *err) {
			NSData *jpeg = nil;
			if (obj != nil) {
				jpeg = giodeJPEG((UIImage *)obj, maxSide);
			}
			@synchronized(jpegs) {
				if (jpeg != nil) {
					jpegs[i] = jpeg;
				} else if (failure == nil) {
					failure = err != nil ? err.localizedDescription : @"can't load an image";
				}
			}
			dispatch_group_leave(group);
		}];
	}
	dispatch_group_notify(group, dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
		int count = 0;
		for (NSUInteger i = 0; i < n; i++) {
			if ([jpegs[i] isKindOfClass:[NSData class]]) {
				NSData *d = jpegs[i];
				giodePhotosItem(rid, (void *)d.bytes, (int)d.length, (char *)[names[i] UTF8String]);
				count++;
			}
		}
		giodePhotosDone(rid, count, count == 0 && failure != nil ? (char *)failure.UTF8String : NULL);
	});
}
@end

int giodePickPhotos(CFTypeRef vcRef, int rid, int maxSide, int limit) {
	if (@available(iOS 14, *)) {
		UIViewController *vc = (__bridge UIViewController *)vcRef;
		onMain(^{
			PHPickerConfiguration *cfg = [[PHPickerConfiguration alloc] init];
			cfg.filter = [PHPickerFilter imagesFilter];
			cfg.selectionLimit = limit;
			if (@available(iOS 15, *)) {
				cfg.selection = PHPickerConfigurationSelectionOrdered;
			}
			PHPickerViewController *picker = [[PHPickerViewController alloc] initWithConfiguration:cfg];
			GiodePhotosDelegate *d = [[GiodePhotosDelegate alloc] init];
			d.requestID = rid;
			d.maxSide = maxSide;
			if (pending == nil) {
				pending = [NSMutableSet set];
			}
			[pending addObject:d];
			picker.delegate = d;
			[vc presentViewController:picker animated:YES completion:nil];
		});
		return 1;
	}
	return 0;
}

// giodeShareFiles opens the share sheet for several files, paths
// separated by newlines.
void giodeShareFiles(CFTypeRef vcRef, const char *paths) {
	UIViewController *vc = (__bridge UIViewController *)vcRef;
	NSMutableArray *items = [NSMutableArray array];
	for (NSString *p in [[NSString stringWithUTF8String:paths] componentsSeparatedByString:@"\n"]) {
		if (p.length > 0) {
			[items addObject:[NSURL fileURLWithPath:p]];
		}
	}
	onMain(^{
		UIActivityViewController *a = [[UIActivityViewController alloc] initWithActivityItems:items applicationActivities:nil];
		UIPopoverPresentationController *pop = a.popoverPresentationController;
		if (pop != nil) {
			pop.sourceView = vc.view;
			CGRect b = vc.view.bounds;
			pop.sourceRect = CGRectMake(CGRectGetMidX(b), CGRectGetMidY(b), 1, 1);
			pop.permittedArrowDirections = 0;
		}
		[vc presentViewController:a animated:YES completion:nil];
	});
}

// giodeSaveToPhotos adds a picture or video to the photo library, asking
// for add-only access first, and answers giodeSavedToPhotos.
void giodeSaveToPhotos(int rid, const char *path, int video) {
	NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
	void (^save)(void) = ^{
		[[PHPhotoLibrary sharedPhotoLibrary] performChanges:^{
			if (video) {
				[PHAssetChangeRequest creationRequestForAssetFromVideoAtFileURL:url];
			} else {
				[PHAssetChangeRequest creationRequestForAssetFromImageAtFileURL:url];
			}
		} completionHandler:^(BOOL ok, NSError *err) {
			giodeSavedToPhotos(rid, ok ? NULL : (char *)(err.localizedDescription ?: @"can't save to Photos").UTF8String);
		}];
	};
	if (@available(iOS 14, *)) {
		[PHPhotoLibrary requestAuthorizationForAccessLevel:PHAccessLevelAddOnly handler:^(PHAuthorizationStatus st) {
			if (st == PHAuthorizationStatusAuthorized || st == PHAuthorizationStatusLimited) {
				save();
			} else {
				giodeSavedToPhotos(rid, "adding to Photos isn't allowed for this app (Settings > Privacy & Security > Photos)");
			}
		}];
	} else {
		save();
	}
}

int giodeHasInfoKey(const char *key) {
	NSString *k = [NSString stringWithUTF8String:key];
	return [[NSBundle mainBundle] objectForInfoDictionaryKey:k] != nil;
}

@interface GiodeCameraDelegate : NSObject <UIImagePickerControllerDelegate, UINavigationControllerDelegate>
@property int requestID;
@property int maxSide;
@end

@implementation GiodeCameraDelegate
- (void)imagePickerController:(UIImagePickerController *)picker didFinishPickingMediaWithInfo:(NSDictionary<UIImagePickerControllerInfoKey, id> *)info {
	int rid = self.requestID;
	int maxSide = self.maxSide;
	[pending removeObject:self];
	[picker dismissViewControllerAnimated:YES completion:nil];
	UIImage *img = info[UIImagePickerControllerOriginalImage];
	if (img == nil) {
		giodePhotoDone(rid, NULL, 0, NULL, (char *)"the camera returned no image");
		return;
	}
	dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
		NSData *jpeg = giodeJPEG(img, maxSide);
		giodePhotoDone(rid, (void *)jpeg.bytes, (int)jpeg.length, (char *)"photo", NULL);
	});
}

- (void)imagePickerControllerDidCancel:(UIImagePickerController *)picker {
	int rid = self.requestID;
	[pending removeObject:self];
	[picker dismissViewControllerAnimated:YES completion:nil];
	giodePhotoDone(rid, NULL, 0, NULL, NULL);
}
@end

// giodePickCamera opens the camera. It returns 1 when the camera opens,
// 0 when the device has none (the simulator), and -1 when the user has
// denied the app camera access.
int giodePickCamera(CFTypeRef vcRef, int rid, int maxSide) {
	if (![UIImagePickerController isSourceTypeAvailable:UIImagePickerControllerSourceTypeCamera]) {
		return 0;
	}
	AVAuthorizationStatus st = [AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeVideo];
	if (st == AVAuthorizationStatusDenied || st == AVAuthorizationStatusRestricted) {
		return -1;
	}
	UIViewController *vc = (__bridge UIViewController *)vcRef;
	onMain(^{
		UIImagePickerController *picker = [[UIImagePickerController alloc] init];
		picker.sourceType = UIImagePickerControllerSourceTypeCamera;
		GiodeCameraDelegate *d = [[GiodeCameraDelegate alloc] init];
		d.requestID = rid;
		d.maxSide = maxSide;
		if (pending == nil) {
			pending = [NSMutableSet set];
		}
		[pending addObject:d];
		picker.delegate = d;
		[vc presentViewController:picker animated:YES completion:nil];
	});
	return 1;
}

// Dictation, with SFSpeechRecognizer fed from the microphone.

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

// Universal links arrive as a browsing-web user activity, which Gio's
// app delegate doesn't handle: giode adds the method and hands the URL
// to Gio's URL path, which reaches Config.OnOpenURL.
extern void gio_onOpenURI(CFTypeRef uri);

static BOOL giodeContinueUserActivity(id self, SEL _cmd, UIApplication *app, NSUserActivity *activity, void (^handler)(NSArray *)) {
	if ([activity.activityType isEqualToString:NSUserActivityTypeBrowsingWeb] && activity.webpageURL != nil) {
		gio_onOpenURI((__bridge CFTypeRef)activity.webpageURL.absoluteString);
		return YES;
	}
	return NO;
}

__attribute__((constructor)) static void giodeInstallUserActivity(void) {
	Class c = objc_getClass("_gioAppDelegate");
	if (c == nil) {
		return;
	}
	SEL sel = @selector(application:continueUserActivity:restorationHandler:);
	struct objc_method_description d = protocol_getMethodDescription(@protocol(UIApplicationDelegate), sel, NO, YES);
	class_addMethod(c, sel, (IMP)giodeContinueUserActivity, d.types);
}
