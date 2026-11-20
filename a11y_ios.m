//go:build ios

@import UIKit;
#include "_cgo_export.h"

// VoiceOver on iOS: Gio's draw view becomes an accessibility container
// whose elements are the tree's nodes a screen reader stops on, in
// reading order. Each element asks Go for its node (as JSON, cached until
// the tree changes). The tree is recorded while VoiceOver or Switch
// Control runs.

@interface GiodeIOSElement : UIAccessibilityElement
@property uintptr_t handle;
@property (weak) UIView *view;
@property int32_t ident;
@property int64_t gen;
@property (strong) NSDictionary *node;
@end

static NSDictionary *axNode(uintptr_t handle, int32_t ident) {
	char *s = giodeAXNode(handle, ident);
	if (s == NULL) {
		return nil;
	}
	NSData *data = [NSData dataWithBytesNoCopy:s length:strlen(s) freeWhenDone:YES];
	id obj = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
	return [obj isKindOfClass:[NSDictionary class]] ? obj : nil;
}

@implementation GiodeIOSElement

- (NSDictionary *)current {
	int64_t gen = giodeAXGeneration();
	if (self.node == nil || self.gen != gen) {
		self.node = axNode(self.handle, self.ident);
		self.gen = gen;
	}
	return self.node;
}

- (BOOL)has:(NSString *)action {
	NSArray *actions = self.current[@"actions"];
	return [actions isKindOfClass:[NSArray class]] && [actions containsObject:action];
}

- (BOOL)perform:(NSString *)action {
	return giodeAXAction(self.handle, self.ident, (char *)action.UTF8String, NULL) != 0;
}

- (NSString *)str:(NSString *)key {
	id v = self.current[key];
	return [v isKindOfClass:[NSString class]] && [v length] > 0 ? v : nil;
}

- (NSString *)role {
	return [self str:@"role"] ?: @"";
}

- (NSString *)accessibilityLabel {
	NSString *label = [self str:@"label"];
	if (label == nil && ([self.role isEqualToString:@"AXStaticText"] || [self.role isEqualToString:@"AXHeading"])) {
		id v = self.current[@"value"];
		label = [v isKindOfClass:[NSString class]] ? v : nil;
	}
	return label;
}

// percent formats a range's value.
- (NSString *)percent {
	double lo = [self.current[@"min"] doubleValue], hi = [self.current[@"max"] doubleValue];
	double v = [self.current[@"value"] doubleValue];
	if (hi <= lo) {
		return nil;
	}
	return [NSNumberFormatter localizedStringFromNumber:@((v - lo) / (hi - lo)) numberStyle:NSNumberFormatterPercentStyle];
}

- (NSString *)accessibilityValue {
	NSString *role = self.role, *sub = [self str:@"subrole"];
	id v = self.current[@"value"];
	NSString *out = nil;
	if ([role isEqualToString:@"AXCheckBox"]) {
		int s = [v intValue];
		if ([sub isEqualToString:@"AXSwitch"]) {
			out = s == 1 ? @"on" : @"off";
		} else {
			out = s == 1 ? @"checked" : (s == 2 ? @"mixed" : @"not checked");
		}
	} else if ([self.current[@"range"] boolValue]) {
		out = [self percent];
	} else if ([role isEqualToString:@"AXTextField"] || [role isEqualToString:@"AXComboBox"] || [role isEqualToString:@"AXPopUpButton"]) {
		out = [v isKindOfClass:[NSString class]] ? v : nil;
	}
	int expanded = [self.current[@"expanded"] intValue];
	if (expanded >= 0) {
		NSString *state = expanded == 1 ? @"expanded" : @"collapsed";
		out = out.length > 0 ? [NSString stringWithFormat:@"%@, %@", out, state] : state;
	}
	if ([self.current[@"invalid"] boolValue]) {
		out = out.length > 0 ? [NSString stringWithFormat:@"%@, invalid", out] : @"invalid";
	}
	return out;
}

- (NSString *)accessibilityHint {
	NSString *help = [self str:@"help"];
	if ([self.current[@"required"] boolValue]) {
		return help != nil ? [NSString stringWithFormat:@"Required. %@", help] : @"Required";
	}
	return help;
}

- (UIAccessibilityTraits)accessibilityTraits {
	NSString *role = self.role;
	UIAccessibilityTraits t = UIAccessibilityTraitNone;
	if ([role isEqualToString:@"AXButton"] || [role isEqualToString:@"AXPopUpButton"] || [role isEqualToString:@"AXMenuItem"] ||
		[role isEqualToString:@"AXCheckBox"] || [role isEqualToString:@"AXRadioButton"] || [role isEqualToString:@"AXComboBox"]) {
		t |= UIAccessibilityTraitButton;
	} else if ([role isEqualToString:@"AXLink"]) {
		t |= UIAccessibilityTraitLink;
	} else if ([role isEqualToString:@"AXImage"]) {
		t |= UIAccessibilityTraitImage;
	} else if ([role isEqualToString:@"AXHeading"]) {
		t |= UIAccessibilityTraitHeader;
	} else if ([role isEqualToString:@"AXSlider"]) {
		t |= UIAccessibilityTraitAdjustable;
	} else if ([role isEqualToString:@"AXProgressIndicator"] || [role isEqualToString:@"AXBusyIndicator"]) {
		t |= UIAccessibilityTraitUpdatesFrequently;
	} else if ([role isEqualToString:@"AXStaticText"]) {
		t |= UIAccessibilityTraitStaticText;
	} else if ([role isEqualToString:@"AXRow"] && [self has:@"press"]) {
		t |= UIAccessibilityTraitButton;
	}
	if (@available(iOS 17.0, *)) {
		if ([role isEqualToString:@"AXCheckBox"]) {
			t |= UIAccessibilityTraitToggleButton;
		}
	}
	if ([self.current[@"selected"] boolValue] || ([role isEqualToString:@"AXRadioButton"] && [self.current[@"value"] intValue] == 1)) {
		t |= UIAccessibilityTraitSelected;
	}
	if ([self.current[@"disabled"] boolValue]) {
		t |= UIAccessibilityTraitNotEnabled;
	}
	return t;
}

- (CGRect)accessibilityFrame {
	NSDictionary *n = self.current;
	UIView *view = self.view;
	if (n == nil || view == nil) {
		return CGRectZero;
	}
	CGFloat scale = view.contentScaleFactor;
	if (scale <= 0) {
		scale = 1;
	}
	CGRect r = CGRectMake([n[@"x"] doubleValue] / scale, [n[@"y"] doubleValue] / scale, [n[@"w"] doubleValue] / scale, [n[@"h"] doubleValue] / scale);
	return UIAccessibilityConvertFrameToScreenCoordinates(r, view);
}

- (BOOL)accessibilityActivate {
	if ([self has:@"press"]) {
		return [self perform:@"press"];
	}
	if ([self has:@"focus"]) {
		return [self perform:@"focus"];
	}
	return NO;
}

- (void)accessibilityIncrement {
	[self perform:@"increment"];
}

- (void)accessibilityDecrement {
	[self perform:@"decrement"];
}

@end

// GiodeIOSView is a view controller's draw view and its elements.
@interface GiodeIOSView : NSObject
@property uintptr_t handle;
@property (weak) UIView *view;
@property (strong) NSMutableDictionary<NSNumber *, GiodeIOSElement *> *elements;
@end

@implementation GiodeIOSView
@end

static NSMutableDictionary<NSNumber *, GiodeIOSView *> *axViews;

static BOOL readerRunning(void) {
	return UIAccessibilityIsVoiceOverRunning() || UIAccessibilityIsSwitchControlRunning();
}

// axRefresh sets the view's elements in reading order.
static void axRefresh(GiodeIOSView *ax) {
	UIView *view = ax.view;
	if (view == nil) {
		return;
	}
	char *s = giodeAXOrder(ax.handle);
	NSData *data = [NSData dataWithBytesNoCopy:s length:strlen(s) freeWhenDone:YES];
	NSArray *order = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
	NSMutableArray *els = [NSMutableArray array];
	NSMutableDictionary *keep = [NSMutableDictionary dictionary];
	for (NSNumber *k in order) {
		GiodeIOSElement *e = ax.elements[k];
		if (e == nil) {
			e = [[GiodeIOSElement alloc] initWithAccessibilityContainer:view];
			e.handle = ax.handle;
			e.view = view;
			e.ident = k.intValue;
		}
		keep[k] = e;
		[els addObject:e];
	}
	ax.elements = keep;
	view.isAccessibilityElement = NO;
	view.accessibilityElements = els;
}

void giodeAXInstall(uintptr_t handle) {
	dispatch_async(dispatch_get_main_queue(), ^{
		UIViewController *vc = (__bridge UIViewController *)(void *)handle;
		UIView *view = nil;
		for (UIView *v in vc.view.subviews) {
			if ([NSStringFromClass([v class]) isEqualToString:@"GioView"]) {
				view = v;
				break;
			}
		}
		if (view == nil) {
			return;
		}
		if (axViews == nil) {
			axViews = [NSMutableDictionary dictionary];
		}
		GiodeIOSView *ax = axViews[@(handle)];
		if (ax == nil) {
			ax = [[GiodeIOSView alloc] init];
			ax.handle = handle;
			ax.elements = [NSMutableDictionary dictionary];
			axViews[@(handle)] = ax;
			void (^status)(NSNotification *) = ^(NSNotification *n) {
				if (readerRunning()) {
					giodeAXWanted(handle);
				}
			};
			NSNotificationCenter *nc = [NSNotificationCenter defaultCenter];
			[nc addObserverForName:UIAccessibilityVoiceOverStatusDidChangeNotification object:nil queue:nil usingBlock:status];
			[nc addObserverForName:UIAccessibilitySwitchControlStatusDidChangeNotification object:nil queue:nil usingBlock:status];
		}
		ax.view = view;
		if (readerRunning()) {
			giodeAXWanted(handle);
		}
		axRefresh(ax);
	});
}

// giodeAXChanged tells VoiceOver what changed: a new layout refreshes the
// elements, keyboard focus moving (focus > 0) moves VoiceOver's cursor
// there, and announce is read. It frees changed and announce.
void giodeAXChanged(uintptr_t handle, int32_t focus, int32_t *changed, int n, char *announce, int assertive, int layout) {
	free(changed);
	NSString *text = announce != NULL ? [NSString stringWithUTF8String:announce] : nil;
	free(announce);
	dispatch_async(dispatch_get_main_queue(), ^{
		GiodeIOSView *ax = axViews[@(handle)];
		if (ax == nil) {
			return;
		}
		if (layout) {
			axRefresh(ax);
		}
		id target = focus > 0 ? ax.elements[@(focus)] : nil;
		if (layout || target != nil) {
			UIAccessibilityPostNotification(UIAccessibilityLayoutChangedNotification, target);
		}
		if (text != nil) {
			if (assertive) {
				UIAccessibilityPostNotification(UIAccessibilityAnnouncementNotification, text);
			} else {
				NSAttributedString *queued = [[NSAttributedString alloc] initWithString:text attributes:@{UIAccessibilitySpeechAttributeQueueAnnouncement: @YES}];
				UIAccessibilityPostNotification(UIAccessibilityAnnouncementNotification, queued);
			}
		}
	});
}
