//go:build darwin && !ios

@import AppKit;
#include "_cgo_export.h"

// The application's menu bar: the app menu (About, Services, Hide, Quit)
// and the app's menus, built from JSON. Items call back with their id.

@interface GiodeMenuTarget : NSObject
@end

@implementation GiodeMenuTarget
- (void)click:(NSMenuItem *)item {
	giodeMenuBarClick((int)item.tag);
}
- (void)quit:(id)sender {
	giodeMenuQuit();
}
@end

static GiodeMenuTarget *menuTarget;

// Gio's key.Modifiers bits.
enum { gioCtrl = 1 << 0, gioCommand = 1 << 1, gioShift = 1 << 2, gioAlt = 1 << 3 };

static NSString *keyEquivalent(NSString *name) {
	static NSDictionary *special;
	if (special == nil) {
		unichar f[12];
		for (int i = 0; i < 12; i++) {
			f[i] = NSF1FunctionKey + i;
		}
		special = @{
			@"⏎": @"\r", @"⎋": @"\x1b", @"Tab": @"\t", @"Space": @" ", @"⌫": @"\x08", @"⌦": @"\x7f",
			@"←": [NSString stringWithFormat:@"%C", (unichar)NSLeftArrowFunctionKey],
			@"→": [NSString stringWithFormat:@"%C", (unichar)NSRightArrowFunctionKey],
			@"↑": [NSString stringWithFormat:@"%C", (unichar)NSUpArrowFunctionKey],
			@"↓": [NSString stringWithFormat:@"%C", (unichar)NSDownArrowFunctionKey],
			@"F1": [NSString stringWithCharacters:&f[0] length:1], @"F2": [NSString stringWithCharacters:&f[1] length:1],
			@"F3": [NSString stringWithCharacters:&f[2] length:1], @"F4": [NSString stringWithCharacters:&f[3] length:1],
			@"F5": [NSString stringWithCharacters:&f[4] length:1], @"F6": [NSString stringWithCharacters:&f[5] length:1],
			@"F7": [NSString stringWithCharacters:&f[6] length:1], @"F8": [NSString stringWithCharacters:&f[7] length:1],
			@"F9": [NSString stringWithCharacters:&f[8] length:1], @"F10": [NSString stringWithCharacters:&f[9] length:1],
			@"F11": [NSString stringWithCharacters:&f[10] length:1], @"F12": [NSString stringWithCharacters:&f[11] length:1],
		};
	}
	NSString *s = special[name];
	return s != nil ? s : name.lowercaseString;
}

static NSMenu *buildMenu(NSString *title, NSArray *items) {
	NSMenu *m = [[NSMenu alloc] initWithTitle:title];
	m.autoenablesItems = NO;
	for (NSDictionary *it in items) {
		if ([it[@"separator"] boolValue]) {
			[m addItem:[NSMenuItem separatorItem]];
			continue;
		}
		NSMenuItem *mi = [[NSMenuItem alloc] initWithTitle:it[@"label"] ?: @"" action:@selector(click:) keyEquivalent:@""];
		mi.target = menuTarget;
		mi.tag = [it[@"id"] integerValue];
		mi.enabled = ![it[@"disabled"] boolValue];
		mi.state = [it[@"checked"] boolValue] ? NSControlStateValueOn : NSControlStateValueOff;
		NSString *key = it[@"key"];
		if (key.length > 0) {
			int mods = [it[@"mods"] intValue];
			NSEventModifierFlags f = 0;
			if (mods & gioCtrl) f |= NSEventModifierFlagControl;
			if (mods & gioCommand) f |= NSEventModifierFlagCommand;
			if (mods & gioShift) f |= NSEventModifierFlagShift;
			if (mods & gioAlt) f |= NSEventModifierFlagOption;
			mi.keyEquivalent = keyEquivalent(key);
			mi.keyEquivalentModifierMask = f;
		}
		NSArray *sub = it[@"items"];
		if (sub.count > 0) {
			mi.action = nil;
			mi.submenu = buildMenu(mi.title, sub);
		}
		[m addItem:mi];
	}
	return m;
}

static NSMenu *appMenu(NSString *name) {
	NSMenu *m = [[NSMenu alloc] initWithTitle:name];
	[m addItemWithTitle:[@"About " stringByAppendingString:name] action:@selector(orderFrontStandardAboutPanel:) keyEquivalent:@""];
	[m addItem:[NSMenuItem separatorItem]];
	NSMenuItem *services = [m addItemWithTitle:@"Services" action:nil keyEquivalent:@""];
	services.submenu = [[NSMenu alloc] initWithTitle:@"Services"];
	NSApp.servicesMenu = services.submenu;
	[m addItem:[NSMenuItem separatorItem]];
	[m addItemWithTitle:[@"Hide " stringByAppendingString:name] action:@selector(hide:) keyEquivalent:@"h"];
	NSMenuItem *others = [m addItemWithTitle:@"Hide Others" action:@selector(hideOtherApplications:) keyEquivalent:@"h"];
	others.keyEquivalentModifierMask = NSEventModifierFlagCommand | NSEventModifierFlagOption;
	[m addItemWithTitle:@"Show All" action:@selector(unhideAllApplications:) keyEquivalent:@""];
	[m addItem:[NSMenuItem separatorItem]];
	NSMenuItem *quit = [m addItemWithTitle:[@"Quit " stringByAppendingString:name] action:@selector(quit:) keyEquivalent:@"q"];
	quit.target = menuTarget;
	return m;
}

// giodeSetMenuBar replaces the main menu with the app menu and the
// menus in specJSON, on the main thread.
void giodeSetMenuBar(const char *specJSON, const char *appName) {
	NSData *js = [NSData dataWithBytes:specJSON length:strlen(specJSON)];
	NSArray *spec = [NSJSONSerialization JSONObjectWithData:js options:0 error:nil];
	NSString *name = [NSString stringWithUTF8String:appName];
	dispatch_async(dispatch_get_main_queue(), ^{
		if (menuTarget == nil) {
			menuTarget = [[GiodeMenuTarget alloc] init];
		}
		NSMenu *bar = [[NSMenu alloc] initWithTitle:@""];
		NSMenuItem *app = [bar addItemWithTitle:name action:nil keyEquivalent:@""];
		app.submenu = appMenu(name);
		for (NSDictionary *menu in spec) {
			NSString *title = menu[@"title"] ?: @"";
			NSMenuItem *top = [bar addItemWithTitle:title action:nil keyEquivalent:@""];
			top.submenu = buildMenu(title, menu[@"items"]);
			if ([title isEqualToString:@"Window"]) {
				NSApp.windowsMenu = top.submenu;
			} else if ([title isEqualToString:@"Help"]) {
				NSApp.helpMenu = top.submenu;
			}
		}
		NSApp.mainMenu = bar;
	});
}
