// SPDX-License-Identifier: Unlicense OR MIT

#import <Foundation/Foundation.h>
#import <StoreKit/StoreKit.h>
#include <TargetConditionals.h>
#include "_cgo_export.h"

// giode/store on StoreKit's original API, the one Objective-C can call.
// One observer, added at launch, sees every transaction: those answering
// a Buy go to its request, the others to OnUpdate.

@interface GiodeStore : NSObject <SKProductsRequestDelegate, SKPaymentTransactionObserver>
@end

static GiodeStore *store;
// products are the products loaded, by id; requests the product
// requests' ids; buying the Buy requests' ids by product.
static NSMutableDictionary<NSString *, SKProduct *> *products;
static NSMapTable<SKRequest *, NSNumber *> *requests;
static NSMutableDictionary<NSString *, NSNumber *> *buying;
static int restoreID;
static NSMutableArray *restored;

static void result(int rid, id obj, NSString *err) {
	if (err != nil) {
		giodeStoreResult(rid, NULL, (char *)[err UTF8String]);
		return;
	}
	NSData *js = [NSJSONSerialization dataWithJSONObject:obj options:0 error:nil];
	NSString *s = js != nil ? [[NSString alloc] initWithData:js encoding:NSUTF8StringEncoding] : @"{}";
	giodeStoreResult(rid, (char *)[s UTF8String], NULL);
}

static NSString *receipt(void) {
	NSURL *u = [[NSBundle mainBundle] appStoreReceiptURL];
	NSData *d = u != nil ? [NSData dataWithContentsOfURL:u] : nil;
	return d != nil ? [d base64EncodedStringWithOptions:0] : @"";
}

static NSDictionary *purchaseJSON(SKPaymentTransaction *t) {
	SKPaymentTransaction *orig = t.transactionState == SKPaymentTransactionStateRestored && t.originalTransaction != nil ? t.originalTransaction : t;
	NSString *tid = t.transactionIdentifier ?: @"";
	return @{
		@"id": orig.transactionIdentifier ?: tid,
		@"product": t.payment.productIdentifier ?: @"",
		@"token": tid,
		@"time": @((long long)([(t.transactionDate ?: [NSDate date]) timeIntervalSince1970] * 1000)),
		@"pending": @(t.transactionState == SKPaymentTransactionStateDeferred),
		@"receipt": receipt(),
	};
}

static NSString *period(SKProductSubscriptionPeriod *p) API_AVAILABLE(ios(11.2), macos(10.13.2)) {
	NSString *unit = @"D";
	switch (p.unit) {
	case SKProductPeriodUnitDay: unit = @"D"; break;
	case SKProductPeriodUnitWeek: unit = @"W"; break;
	case SKProductPeriodUnitMonth: unit = @"M"; break;
	case SKProductPeriodUnitYear: unit = @"Y"; break;
	}
	return [NSString stringWithFormat:@"P%lu%@", (unsigned long)p.numberOfUnits, unit];
}

static NSDictionary *productJSON(SKProduct *p) {
	NSNumberFormatter *f = [[NSNumberFormatter alloc] init];
	f.numberStyle = NSNumberFormatterCurrencyStyle;
	f.locale = p.priceLocale;
	NSMutableDictionary *o = [@{
		@"id": p.productIdentifier ?: @"",
		@"title": p.localizedTitle ?: @"",
		@"description": p.localizedDescription ?: @"",
		@"price": [f stringFromNumber:p.price] ?: @"",
		@"micros": @([[p.price decimalNumberByMultiplyingByPowerOf10:6] longLongValue]),
		@"currency": [p.priceLocale objectForKey:NSLocaleCurrencyCode] ?: @"",
	} mutableCopy];
	if (@available(iOS 11.2, macOS 10.13.2, *)) {
		if (p.subscriptionPeriod != nil) {
			o[@"subscription"] = @YES;
			o[@"period"] = period(p.subscriptionPeriod);
		}
	}
	return o;
}

@implementation GiodeStore
- (void)productsRequest:(SKProductsRequest *)request didReceiveResponse:(SKProductsResponse *)response {
	NSMutableArray *out = [NSMutableArray array];
	for (SKProduct *p in response.products) {
		products[p.productIdentifier] = p;
		[out addObject:productJSON(p)];
	}
	NSNumber *rid = [requests objectForKey:request];
	[requests removeObjectForKey:request];
	if (rid != nil) {
		result(rid.intValue, out, nil);
	}
}

- (void)request:(SKRequest *)request didFailWithError:(NSError *)error {
	NSNumber *rid = [requests objectForKey:request];
	[requests removeObjectForKey:request];
	if (rid != nil) {
		result(rid.intValue, nil, error.localizedDescription ?: @"request failed");
	}
}

- (void)paymentQueue:(SKPaymentQueue *)queue updatedTransactions:(NSArray<SKPaymentTransaction *> *)transactions {
	for (SKPaymentTransaction *t in transactions) {
		NSString *pid = t.payment.productIdentifier;
		NSNumber *rid = pid != nil ? buying[pid] : nil;
		switch (t.transactionState) {
		case SKPaymentTransactionStatePurchasing:
			continue;
		case SKPaymentTransactionStateFailed: {
			[queue finishTransaction:t];
			if (rid != nil) {
				[buying removeObjectForKey:pid];
				BOOL canceled = [t.error.domain isEqualToString:SKErrorDomain] && t.error.code == SKErrorPaymentCancelled;
				result(rid.intValue, nil, canceled ? @"canceled" : (t.error.localizedDescription ?: @"purchase failed"));
			}
			continue;
		}
		case SKPaymentTransactionStateRestored:
			if (restoreID != 0) {
				[restored addObject:purchaseJSON(t)];
				[queue finishTransaction:t];
				continue;
			}
			break;
		default:
			break;
		}
		NSDictionary *js = purchaseJSON(t);
		if (rid != nil) {
			[buying removeObjectForKey:pid];
			result(rid.intValue, js, nil);
		} else {
			NSData *d = [NSJSONSerialization dataWithJSONObject:js options:0 error:nil];
			giodeStoreUpdate((char *)[[[NSString alloc] initWithData:d encoding:NSUTF8StringEncoding] UTF8String]);
		}
	}
}

- (void)paymentQueueRestoreCompletedTransactionsFinished:(SKPaymentQueue *)queue {
	int rid = restoreID;
	NSArray *out = [restored copy];
	restoreID = 0;
	[restored removeAllObjects];
	if (rid != 0) {
		result(rid, out, nil);
	}
}

- (void)paymentQueue:(SKPaymentQueue *)queue restoreCompletedTransactionsFailedWithError:(NSError *)error {
	int rid = restoreID;
	restoreID = 0;
	[restored removeAllObjects];
	if (rid != 0) {
		BOOL canceled = [error.domain isEqualToString:SKErrorDomain] && error.code == SKErrorPaymentCancelled;
		result(rid, nil, canceled ? @"canceled" : (error.localizedDescription ?: @"restore failed"));
	}
}
@end

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) {
		b();
	} else {
		dispatch_async(dispatch_get_main_queue(), b);
	}
}

__attribute__((constructor)) static void giodeStoreInit(void) {
	store = [[GiodeStore alloc] init];
	products = [NSMutableDictionary dictionary];
	requests = [NSMapTable strongToStrongObjectsMapTable];
	buying = [NSMutableDictionary dictionary];
	restored = [NSMutableArray array];
	// Transactions left from earlier launches arrive once the observer
	// is added.
	dispatch_async(dispatch_get_main_queue(), ^{
		[[SKPaymentQueue defaultQueue] addTransactionObserver:store];
	});
}

void giodeStoreProducts(int rid, const char *ids) {
	NSData *d = [[NSString stringWithUTF8String:ids] dataUsingEncoding:NSUTF8StringEncoding];
	NSArray *list = [NSJSONSerialization JSONObjectWithData:d options:0 error:nil];
	onMain(^{
		SKProductsRequest *r = [[SKProductsRequest alloc] initWithProductIdentifiers:[NSSet setWithArray:list ?: @[]]];
		r.delegate = store;
		[requests setObject:@(rid) forKey:r];
		[r start];
	});
}

void giodeStoreBuy(int rid, const char *product) {
	NSString *pid = [NSString stringWithUTF8String:product];
	onMain(^{
		SKProduct *p = products[pid];
		if (p == nil) {
			result(rid, nil, [NSString stringWithFormat:@"product %@ wasn't loaded with Products", pid]);
			return;
		}
		if (![SKPaymentQueue canMakePayments]) {
			result(rid, nil, @"purchases are disabled on this device");
			return;
		}
		if (buying[pid] != nil) {
			result(rid, nil, @"a purchase of this product is in progress");
			return;
		}
		buying[pid] = @(rid);
		[[SKPaymentQueue defaultQueue] addPayment:[SKPayment paymentWithProduct:p]];
	});
}

void giodeStoreFinish(int rid, const char *token) {
	NSString *tid = [NSString stringWithUTF8String:token];
	onMain(^{
		SKPaymentQueue *q = [SKPaymentQueue defaultQueue];
		for (SKPaymentTransaction *t in q.transactions) {
			if ([t.transactionIdentifier isEqualToString:tid] && t.transactionState != SKPaymentTransactionStatePurchasing) {
				[q finishTransaction:t];
			}
		}
		result(rid, @{}, nil);
	});
}

void giodeStoreRestore(int rid) {
	onMain(^{
		if (restoreID != 0) {
			result(rid, nil, @"a restore is in progress");
			return;
		}
		restoreID = rid;
		[restored removeAllObjects];
		[[SKPaymentQueue defaultQueue] restoreCompletedTransactions];
	});
}

// giodeStoreBillingRequired reports an App Store install: every iOS app,
// and a Mac app with an App Store receipt (Developer ID apps have none).
int giodeStoreBillingRequired(void) {
#if TARGET_OS_IPHONE
	return 1;
#else
	NSURL *u = [[NSBundle mainBundle] appStoreReceiptURL];
	return u != nil && [[NSFileManager defaultManager] fileExistsAtPath:u.path];
#endif
}
