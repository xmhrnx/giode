// SPDX-License-Identifier: Unlicense OR MIT

package org.giode.notification;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;

// NotificationActivity takes taps and foreground buttons: it brings the
// app's task to the front, then hands the response to Go. It has no UI
// of its own.
public final class NotificationActivity extends Activity {
	@Override protected void onCreate(Bundle state) {
		super.onCreate(state);
		final Intent intent = getIntent();
		Intent launch = getPackageManager().getLaunchIntentForPackage(getPackageName());
		if (launch != null) {
			launch.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_RESET_TASK_IF_NEEDED);
			startActivity(launch);
		}
		final android.content.Context app = getApplicationContext();
		new Thread(new Runnable() {
			@Override public void run() {
				notification_android.deliver(app, intent);
			}
		}).start();
		finish();
	}
}
