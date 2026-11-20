// SPDX-License-Identifier: Unlicense OR MIT

package org.giode.notification;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;

// NotificationReceiver takes the buttons and replies handled without
// showing the app. The Go handler runs on a thread of its own, the
// broadcast kept alive until it returns.
public final class NotificationReceiver extends BroadcastReceiver {
	@Override public void onReceive(final Context ctx, final Intent intent) {
		final PendingResult result = goAsync();
		new Thread(new Runnable() {
			@Override public void run() {
				try {
					notification_android.deliver(ctx.getApplicationContext(), intent);
				} finally {
					result.finish();
				}
			}
		}).start();
	}
}
