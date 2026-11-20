//go:build darwin && !ios

@import AppKit;
#import <ServiceManagement/ServiceManagement.h>
#include <stdlib.h>
#include <string.h>

// giodeLoginItemSet registers the app as a login item (on) or removes
// it, with SMAppService: 1 when done, 0 with err set, -1 before macOS 13.
int giodeLoginItemSet(int on, char **err) {
	if (@available(macOS 13.0, *)) {
		SMAppService *s = SMAppService.mainAppService;
		NSError *e = nil;
		BOOL ok;
		if (on) {
			ok = s.status == SMAppServiceStatusEnabled || [s registerAndReturnError:&e];
		} else {
			ok = s.status == SMAppServiceStatusNotRegistered || [s unregisterAndReturnError:&e];
		}
		if (!ok) {
			*err = strdup((e.localizedDescription ?: @"failed").UTF8String);
			return 0;
		}
		return 1;
	}
	return -1;
}

// giodeLoginItemEnabled reports a login item registered with
// SMAppService that the user hasn't turned off.
int giodeLoginItemEnabled(void) {
	if (@available(macOS 13.0, *)) {
		return SMAppService.mainAppService.status == SMAppServiceStatusEnabled;
	}
	return 0;
}

// giodeWorkspaceOpen opens a URL, or a file or folder (isPath), as
// double-clicking it would.
int giodeWorkspaceOpen(const char *target, int isPath) {
	NSString *s = @(target);
	NSURL *u = isPath ? [NSURL fileURLWithPath:s] : [NSURL URLWithString:s];
	if (u == nil) {
		return 0;
	}
	return [[NSWorkspace sharedWorkspace] openURL:u];
}

// giodeWorkspaceReveal shows path selected in a Finder window.
int giodeWorkspaceReveal(const char *path) {
	NSURL *u = [NSURL fileURLWithPath:@(path)];
	[[NSWorkspace sharedWorkspace] activateFileViewerSelectingURLs:@[u]];
	return 1;
}

// giodeBookmark returns a security-scoped bookmark of path (malloc'd,
// n bytes), or NULL.
void *giodeBookmark(const char *path, int *n) {
	NSError *e = nil;
	NSData *d = [[NSURL fileURLWithPath:@(path)] bookmarkDataWithOptions:NSURLBookmarkCreationWithSecurityScope
	                                         includingResourceValuesForKeys:nil
	                                                          relativeToURL:nil
	                                                                  error:&e];
	if (d == nil) {
		return NULL;
	}
	void *out = malloc(d.length);
	memcpy(out, d.bytes, d.length);
	*n = (int)d.length;
	return out;
}

// giodeResolveBookmark resolves a security-scoped bookmark and starts
// accessing its folder for the rest of the run, returning its path
// (malloc'd), or NULL.
char *giodeResolveBookmark(const void *data, int n, int *stale) {
	BOOL isStale = NO;
	NSError *e = nil;
	NSURL *u = [NSURL URLByResolvingBookmarkData:[NSData dataWithBytes:data length:n]
	                                     options:NSURLBookmarkResolutionWithSecurityScope
	                               relativeToURL:nil
	                         bookmarkDataIsStale:&isStale
	                                       error:&e];
	if (u == nil || ![u startAccessingSecurityScopedResource]) {
		return NULL;
	}
	*stale = isStale;
	return strdup(u.path.fileSystemRepresentation);
}
