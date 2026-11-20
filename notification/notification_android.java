// SPDX-License-Identifier: Unlicense OR MIT

package org.giode.notification;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.RemoteInput;
import android.content.Context;
import android.content.Intent;
import android.os.Build;
import android.os.Bundle;

import org.json.JSONArray;
import org.json.JSONObject;

// notification_android shows giode/notification's notifications. Taps
// and buttons come back through NotificationReceiver (in the background)
// or NotificationActivity (bringing the app up), which start the Go side
// when the app isn't running and hand it the response.
public final class notification_android {
	static final String EXTRA_PAYLOAD = "org.giode.notification.payload";
	static final String EXTRA_ACTION = "org.giode.notification.action";
	static final String EXTRA_ID = "org.giode.notification.id";
	static final String EXTRA_FOREGROUND = "org.giode.notification.foreground";
	static final String REPLY_KEY = "org.giode.notification.reply";
	static final String DEFAULT_CHANNEL = "giode.default";
	static final String SILENT_CHANNEL = "giode.silent";
	static final int NOTIFICATION = 1;

	private static int badge;

	// onResponse hands a response to Go and reports whether the handler
	// showed the notification again (else it's dismissed).
	static native boolean onResponse(String payload, String action, String reply);

	static NotificationManager manager(Context ctx) {
		return (NotificationManager) ctx.getSystemService(Context.NOTIFICATION_SERVICE);
	}

	// startGo loads the app's Go library and runs its main, when it isn't
	// running yet: a button pressed with the app closed starts only its
	// process.
	static void startGo(Context ctx) {
		try {
			Class.forName("org.gioui.Gio").getMethod("init", Context.class).invoke(null, ctx.getApplicationContext());
		} catch (Exception e) {
		}
	}

	public static void defineChannel(Context ctx, String id, String name, String desc, int importance) {
		if (Build.VERSION.SDK_INT < 26) {
			return;
		}
		int imp = NotificationManager.IMPORTANCE_DEFAULT;
		if (importance == 1) {
			imp = NotificationManager.IMPORTANCE_LOW;
		} else if (importance == 2) {
			imp = NotificationManager.IMPORTANCE_HIGH;
		}
		NotificationChannel ch = new NotificationChannel(id, name, imp);
		if (desc != null && !desc.isEmpty()) {
			ch.setDescription(desc);
		}
		ch.setShowBadge(true);
		manager(ctx).createNotificationChannel(ch);
	}

	private static String appName(Context ctx) {
		CharSequence l = ctx.getApplicationInfo().loadLabel(ctx.getPackageManager());
		return l != null ? l.toString() : "Notifications";
	}

	private static int flags(boolean mutable) {
		int f = PendingIntent.FLAG_UPDATE_CURRENT;
		if (Build.VERSION.SDK_INT >= 31) {
			f |= mutable ? PendingIntent.FLAG_MUTABLE : PendingIntent.FLAG_IMMUTABLE;
		} else if (Build.VERSION.SDK_INT >= 23 && !mutable) {
			f |= PendingIntent.FLAG_IMMUTABLE;
		}
		return f;
	}

	private static Intent responseIntent(Context ctx, Class<?> cls, String id, String payload, String action, boolean foreground) {
		Intent i = new Intent(ctx, cls);
		// A distinct action keeps the PendingIntents apart.
		i.setAction("org.giode.notification." + id + "." + action);
		i.putExtra(EXTRA_ID, id);
		i.putExtra(EXTRA_PAYLOAD, payload);
		i.putExtra(EXTRA_ACTION, action);
		i.putExtra(EXTRA_FOREGROUND, foreground);
		return i;
	}

	// show posts a notification and returns null, "off" when the app's
	// notifications are off, or an error.
	@SuppressWarnings("deprecation")
	public static String show(Context ctx, String id, String title, String body, String thread, String channel, int importance,
			String actionsJSON, String payload, int count, boolean silent) {
		try {
			NotificationManager nm = manager(ctx);
			if (Build.VERSION.SDK_INT >= 24 && !nm.areNotificationsEnabled()) {
				return "off";
			}
			Notification.Builder b;
			if (Build.VERSION.SDK_INT >= 26) {
				if (silent) {
					// Android 8 and later set sounds per channel.
					channel = SILENT_CHANNEL;
					if (nm.getNotificationChannel(channel) == null) {
						defineChannel(ctx, channel, appName(ctx), "", 1);
					}
				} else if (channel.isEmpty()) {
					channel = DEFAULT_CHANNEL;
					if (nm.getNotificationChannel(channel) == null) {
						defineChannel(ctx, channel, appName(ctx), "", 0);
					}
				}
				b = new Notification.Builder(ctx, channel);
			} else {
				b = new Notification.Builder(ctx);
				b.setPriority(importance == 1 ? Notification.PRIORITY_LOW : importance == 2 ? Notification.PRIORITY_HIGH : Notification.PRIORITY_DEFAULT);
				if (!silent && importance != 1) {
					b.setDefaults(Notification.DEFAULT_ALL);
				}
			}
			int icon = ctx.getApplicationInfo().icon;
			b.setSmallIcon(icon != 0 ? icon : android.R.drawable.ic_dialog_info);
			b.setContentTitle(title);
			b.setContentText(body);
			b.setStyle(new Notification.BigTextStyle().bigText(body));
			b.setAutoCancel(true);
			b.setShowWhen(true);
			if (silent && Build.VERSION.SDK_INT < 26) {
				b.setSound(null);
				b.setVibrate(null);
			}
			if (count <= 0) {
				count = badge;
			}
			if (count > 0) {
				b.setNumber(count);
			}
			if (!thread.isEmpty()) {
				b.setGroup(thread);
			}
			int code = id.hashCode();
			b.setContentIntent(PendingIntent.getActivity(ctx, code,
				responseIntent(ctx, NotificationActivity.class, id, payload, "", true), flags(false)));
			JSONArray actions = new JSONArray(actionsJSON);
			for (int k = 0; k < actions.length(); k++) {
				JSONObject a = actions.getJSONObject(k);
				String aid = a.getString("id");
				boolean reply = a.optBoolean("reply") && Build.VERSION.SDK_INT >= 24;
				boolean fg = a.optBoolean("foreground");
				PendingIntent pi;
				if (fg && !reply) {
					pi = PendingIntent.getActivity(ctx, code + 1 + k,
						responseIntent(ctx, NotificationActivity.class, id, payload, aid, true), flags(false));
				} else {
					pi = PendingIntent.getBroadcast(ctx, code + 1 + k,
						responseIntent(ctx, NotificationReceiver.class, id, payload, aid, false), flags(reply));
				}
				Notification.Action.Builder ab = new Notification.Action.Builder(0, a.getString("title"), pi);
				if (reply) {
					ab.addRemoteInput(new RemoteInput.Builder(REPLY_KEY).setLabel(a.optString("placeholder", a.getString("title"))).build());
				}
				b.addAction(ab.build());
			}
			nm.notify(id, NOTIFICATION, b.build());
			if (!thread.isEmpty() && Build.VERSION.SDK_INT >= 24) {
				// The group's summary, which Android shows when it bundles
				// the thread.
				Notification.Builder s = Build.VERSION.SDK_INT >= 26 ? new Notification.Builder(ctx, channel) : new Notification.Builder(ctx);
				s.setSmallIcon(icon != 0 ? icon : android.R.drawable.ic_dialog_info);
				s.setContentTitle(title);
				s.setGroup(thread);
				s.setGroupSummary(true);
				s.setAutoCancel(true);
				if (Build.VERSION.SDK_INT >= 26) {
					s.setGroupAlertBehavior(Notification.GROUP_ALERT_CHILDREN);
				}
				nm.notify("giode.thread." + thread, NOTIFICATION, s.build());
			}
			return null;
		} catch (Exception e) {
			return e.toString();
		}
	}

	public static void remove(Context ctx, String id) {
		manager(ctx).cancel(id, NOTIFICATION);
	}

	public static void removeAll(Context ctx) {
		manager(ctx).cancelAll();
	}

	public static void setBadge(int n) {
		badge = n;
	}

	// deliver hands a response to Go, starting it first if needed, and
	// dismisses the notification unless the handler showed it again.
	static void deliver(Context ctx, Intent intent) {
		String id = intent.getStringExtra(EXTRA_ID);
		String payload = intent.getStringExtra(EXTRA_PAYLOAD);
		String action = intent.getStringExtra(EXTRA_ACTION);
		if (id == null || payload == null) {
			return;
		}
		String reply = "";
		if (Build.VERSION.SDK_INT >= 20) {
			Bundle in = RemoteInput.getResultsFromIntent(intent);
			if (in != null) {
				CharSequence t = in.getCharSequence(REPLY_KEY);
				if (t != null) {
					reply = t.toString();
				}
			}
		}
		startGo(ctx);
		boolean keep = onResponse(payload, action != null ? action : "", reply);
		if (!keep) {
			remove(ctx, id);
		}
	}
}
