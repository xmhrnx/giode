//go:build ios

@import UIKit;

int giodeIncreaseContrast(void) {
	return UIAccessibilityDarkerSystemColorsEnabled() ? 1 : 0;
}

int giodeBoldText(void) {
	return UIAccessibilityIsBoldTextEnabled() ? 1 : 0;
}

// giodeTextScale is Dynamic Type's size over the default (Large), as
// body text grows: 17 points at Large.
double giodeTextScale(void) {
	__block UIContentSizeCategory cat = nil;
	void (^read)(void) = ^{
		cat = UIApplication.sharedApplication.preferredContentSizeCategory;
	};
	if ([NSThread isMainThread]) {
		read();
	} else {
		dispatch_sync(dispatch_get_main_queue(), read);
	}
	NSDictionary<UIContentSizeCategory, NSNumber *> *body = @{
		UIContentSizeCategoryExtraSmall: @14,
		UIContentSizeCategorySmall: @15,
		UIContentSizeCategoryMedium: @16,
		UIContentSizeCategoryLarge: @17,
		UIContentSizeCategoryExtraLarge: @19,
		UIContentSizeCategoryExtraExtraLarge: @21,
		UIContentSizeCategoryExtraExtraExtraLarge: @23,
		UIContentSizeCategoryAccessibilityMedium: @28,
		UIContentSizeCategoryAccessibilityLarge: @33,
		UIContentSizeCategoryAccessibilityExtraLarge: @40,
		UIContentSizeCategoryAccessibilityExtraExtraLarge: @47,
		UIContentSizeCategoryAccessibilityExtraExtraExtraLarge: @53,
	};
	NSNumber *pt = cat != nil ? body[cat] : nil;
	return pt != nil ? pt.doubleValue / 17.0 : 0;
}
