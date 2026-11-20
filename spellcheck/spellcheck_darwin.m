//go:build darwin

#include <TargetConditionals.h>
#include <stdlib.h>
#include <string.h>
@import Foundation;
#if TARGET_OS_IPHONE
@import UIKit;
#else
@import AppKit;
#endif

// Ranges are UTF-16 offsets, as "start,length;" pairs.

static char *giodeSpellDup(NSString *s) { return strdup(s.UTF8String); }

#if TARGET_OS_IPHONE
// language picks one of UITextChecker's languages for a BCP 47 tag, or
// the user's first.
static NSString *language(const char *tag) {
	NSArray<NSString *> *avail = [UITextChecker availableLanguages];
	NSString *want = tag && *tag ? [NSString stringWithUTF8String:tag] : [NSLocale preferredLanguages].firstObject;
	want = [want stringByReplacingOccurrencesOfString:@"-" withString:@"_"];
	if ([avail containsObject:want]) return want;
	NSString *base = [want componentsSeparatedByString:@"_"].firstObject;
	for (NSString *l in avail) {
		if ([l isEqualToString:base] || [l hasPrefix:[base stringByAppendingString:@"_"]]) return l;
	}
	return nil;
}
#else
static NSString *language(const char *tag) {
	NSSpellChecker *sc = [NSSpellChecker sharedSpellChecker];
	if (!tag || !*tag) return sc.language;
	NSString *want = [[NSString stringWithUTF8String:tag] stringByReplacingOccurrencesOfString:@"-" withString:@"_"];
	NSArray<NSString *> *avail = sc.availableLanguages;
	if ([avail containsObject:want]) return want;
	NSString *base = [want componentsSeparatedByString:@"_"].firstObject;
	for (NSString *l in avail) {
		if ([l isEqualToString:base] || [l hasPrefix:[base stringByAppendingString:@"_"]]) return l;
	}
	return nil;
}
#endif

int giodeSpellAvailable(const char *tag) {
	@autoreleasepool {
		return language(tag) != nil;
	}
}

char *giodeSpellCheck(const char *text, const char *tag) {
	@autoreleasepool {
		NSString *lang = language(tag);
		if (!lang) return giodeSpellDup(@"");
		NSString *s = [NSString stringWithUTF8String:text];
		NSMutableString *out = [NSMutableString string];
		NSUInteger at = 0;
#if TARGET_OS_IPHONE
		UITextChecker *c = [UITextChecker new];
		while (at < s.length) {
			NSRange r = [c rangeOfMisspelledWordInString:s range:NSMakeRange(0, s.length) startingAt:at wrap:NO language:lang];
			if (r.location == NSNotFound) break;
			[out appendFormat:@"%lu,%lu;", (unsigned long)r.location, (unsigned long)r.length];
			at = r.location + r.length;
		}
#else
		NSSpellChecker *sc = [NSSpellChecker sharedSpellChecker];
		while (at < s.length) {
			NSRange r = [sc checkSpellingOfString:s startingAt:at language:lang wrap:NO inSpellDocumentWithTag:0 wordCount:NULL];
			if (r.location == NSNotFound || r.length == 0) break;
			[out appendFormat:@"%lu,%lu;", (unsigned long)r.location, (unsigned long)r.length];
			at = r.location + r.length;
		}
#endif
		return giodeSpellDup(out);
	}
}

char *giodeSpellSuggest(const char *word, const char *tag) {
	@autoreleasepool {
		NSString *lang = language(tag);
		if (!lang) return giodeSpellDup(@"");
		NSString *w = [NSString stringWithUTF8String:word];
#if TARGET_OS_IPHONE
		NSArray<NSString *> *g = [[UITextChecker new] guessesForWordRange:NSMakeRange(0, w.length) inString:w language:lang];
#else
		NSArray<NSString *> *g = [[NSSpellChecker sharedSpellChecker] guessesForWordRange:NSMakeRange(0, w.length) inString:w language:lang inSpellDocumentWithTag:0];
#endif
		return giodeSpellDup([g ?: @[] componentsJoinedByString:@"\n"]);
	}
}

void giodeSpellLearn(const char *word) {
	@autoreleasepool {
		NSString *w = [NSString stringWithUTF8String:word];
#if TARGET_OS_IPHONE
		[UITextChecker learnWord:w];
#else
		[[NSSpellChecker sharedSpellChecker] learnWord:w];
#endif
	}
}
