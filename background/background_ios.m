// SPDX-License-Identifier: Unlicense OR MIT

//go:build ios

@import Foundation;
@import BackgroundTasks;
#include <string.h>
#include "_cgo_export.h"

// The app's tasks are registered with BGTaskScheduler as the process
// starts, before launching ends as the scheduler requires: every
// identifier Info.plist permits (giode.toml's app.background_tasks).
// A launch runs the task's Go function, which answers
// giodeBackgroundDone; a periodic task is submitted again then.

static NSMutableDictionary<NSNumber *, BGTask *> *running;
static int nextRun;

static NSString *optionsKey(NSString *ident) {
	return [@"giode.background." stringByAppendingString:ident];
}

static BOOL submit(NSString *ident, NSDictionary *o, NSError **err) {
	BGTaskRequest *req;
	if ([o[@"long"] boolValue]) {
		BGProcessingTaskRequest *p = [[BGProcessingTaskRequest alloc] initWithIdentifier:ident];
		p.requiresNetworkConnectivity = [o[@"network"] boolValue];
		p.requiresExternalPower = [o[@"charging"] boolValue];
		req = p;
	} else {
		req = [[BGAppRefreshTaskRequest alloc] initWithIdentifier:ident];
	}
	double delay = [o[@"delay"] doubleValue];
	if (delay > 0) {
		req.earliestBeginDate = [NSDate dateWithTimeIntervalSinceNow:delay];
	}
	return [[BGTaskScheduler sharedScheduler] submitTaskRequest:req error:err];
}

__attribute__((constructor)) static void giodeBackgroundRegister(void) {
	NSArray *ids = [[NSBundle mainBundle] objectForInfoDictionaryKey:@"BGTaskSchedulerPermittedIdentifiers"];
	if (![ids isKindOfClass:[NSArray class]]) {
		return;
	}
	running = [NSMutableDictionary dictionary];
	NSString *prefix = [[NSBundle mainBundle].bundleIdentifier stringByAppendingString:@"."];
	for (NSString *ident in ids) {
		if (![ident isKindOfClass:[NSString class]] || ![ident hasPrefix:prefix]) {
			continue;
		}
		NSString *name = [ident substringFromIndex:prefix.length];
		[[BGTaskScheduler sharedScheduler] registerForTaskWithIdentifier:ident usingQueue:nil launchHandler:^(BGTask *task) {
			int rid;
			@synchronized (running) {
				rid = ++nextRun;
				running[@(rid)] = task;
			}
			task.expirationHandler = ^{
				giodeBackgroundExpire(rid);
			};
			giodeBackgroundRun(rid, (char *)name.UTF8String);
		}];
	}
}

void giodeBackgroundDone(int rid, int ok) {
	BGTask *task;
	@synchronized (running) {
		task = running[@(rid)];
		[running removeObjectForKey:@(rid)];
	}
	if (task == nil) {
		return;
	}
	NSDictionary *o = [[NSUserDefaults standardUserDefaults] dictionaryForKey:optionsKey(task.identifier)];
	double interval = [o[@"interval"] doubleValue];
	if (interval > 0) {
		NSMutableDictionary *next = [o mutableCopy];
		next[@"delay"] = @(interval);
		submit(task.identifier, next, nil);
	}
	[task setTaskCompletedWithSuccess:ok != 0];
}

// giodeBackgroundSchedule submits a task's first request and keeps its
// options for the next ones; it returns an error message or NULL.
char *giodeBackgroundSchedule(const char *name, double interval, double delay, int network, int charging, int isLong) {
	NSString *ident = [NSString stringWithFormat:@"%@.%s", [NSBundle mainBundle].bundleIdentifier, name];
	NSDictionary *o = @{@"interval": @(interval), @"delay": @(delay), @"network": @(network != 0), @"charging": @(charging != 0), @"long": @(isLong != 0)};
	[[NSUserDefaults standardUserDefaults] setObject:o forKey:optionsKey(ident)];
	NSError *err = nil;
	if (!submit(ident, o, &err)) {
		NSString *msg = err.localizedDescription ?: @"BGTaskScheduler refused the request";
		if (err.code == BGTaskSchedulerErrorCodeNotPermitted) {
			msg = [NSString stringWithFormat:@"%@ isn't permitted: add \"%s\" to app.background_tasks in giode.toml", ident, name];
		}
		return strdup(msg.UTF8String);
	}
	return NULL;
}

void giodeBackgroundCancel(const char *name) {
	NSString *ident = [NSString stringWithFormat:@"%@.%s", [NSBundle mainBundle].bundleIdentifier, name];
	[[NSUserDefaults standardUserDefaults] removeObjectForKey:optionsKey(ident)];
	[[BGTaskScheduler sharedScheduler] cancelTaskRequestWithIdentifier:ident];
}
