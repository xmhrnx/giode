package org.giode.telecom;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Person;
import android.content.BroadcastReceiver;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.media.AudioAttributes;
import android.media.RingtoneManager;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.telecom.CallAudioState;
import android.telecom.Connection;
import android.telecom.ConnectionRequest;
import android.telecom.DisconnectCause;
import android.telecom.PhoneAccount;
import android.telecom.PhoneAccountHandle;
import android.telecom.TelecomManager;
import android.telecom.VideoProfile;

import java.util.HashMap;
import java.util.Map;

// telecom_android runs giode/telecom on a self-managed ConnectionService:
// Android counts the app's calls as calls (audio focus, Bluetooth and
// car controls, other calls put on hold) and the app shows the
// incoming call itself, as a full-screen call notification.
public final class telecom_android {
	// Implemented in Go (telecom_android.go).
	static native void onEvent(String kind, String a, String b);

	static final String EXTRA_ID = "giode.call.id", EXTRA_NAME = "giode.call.name", EXTRA_HANDLE = "giode.call.handle", EXTRA_VIDEO = "giode.call.video";
	private static final String CHANNEL = "giode.calls";
	private static final String ACTION_ANSWER = "org.giode.telecom.ANSWER", ACTION_DECLINE = "org.giode.telecom.DECLINE";
	private static final int NOTIFICATION_ID = 0x7e1;

	static Context app;
	static PhoneAccountHandle account;
	static final Map<String, CallConnection> connections = new HashMap<>();
	private static boolean receiverRegistered;

	// setup registers the app's phone account; it returns "" or an error.
	public static String setup(Context ctx, boolean video) {
		if (Build.VERSION.SDK_INT < 26) {
			return "self-managed calls need Android 8";
		}
		app = ctx.getApplicationContext();
		try {
			account = new PhoneAccountHandle(new ComponentName(app, GiodeConnectionService.class), "giode");
			CharSequence label = app.getApplicationInfo().loadLabel(app.getPackageManager());
			int caps = PhoneAccount.CAPABILITY_SELF_MANAGED;
			if (video) {
				caps |= PhoneAccount.CAPABILITY_VIDEO_CALLING | PhoneAccount.CAPABILITY_SUPPORTS_VIDEO_CALLING;
			}
			PhoneAccount pa = PhoneAccount.builder(account, label).setCapabilities(caps)
				.addSupportedUriScheme("giode").build();
			telecom().registerPhoneAccount(pa);
		} catch (Exception e) {
			return e.toString();
		}
		registerReceiver();
		return "";
	}

	static TelecomManager telecom() {
		return (TelecomManager) app.getSystemService(Context.TELECOM_SERVICE);
	}

	private static Bundle extras(String id, String handle, String name, boolean video) {
		Bundle b = new Bundle();
		b.putString(EXTRA_ID, id);
		b.putString(EXTRA_HANDLE, handle);
		b.putString(EXTRA_NAME, name);
		b.putBoolean(EXTRA_VIDEO, video);
		return b;
	}

	public static String incoming(String id, String handle, String name, boolean video) {
		try {
			Bundle b = extras(id, handle, name, video);
			b.putParcelable(TelecomManager.EXTRA_INCOMING_CALL_ADDRESS, Uri.fromParts("giode", handle, null));
			if (video) {
				b.putInt(TelecomManager.EXTRA_INCOMING_VIDEO_STATE, VideoProfile.STATE_BIDIRECTIONAL);
			}
			telecom().addNewIncomingCall(account, b);
			return "";
		} catch (Exception e) {
			return e.toString();
		}
	}

	public static String outgoing(String id, String handle, String name, boolean video) {
		try {
			Bundle b = new Bundle();
			b.putParcelable(TelecomManager.EXTRA_PHONE_ACCOUNT_HANDLE, account);
			b.putBundle(TelecomManager.EXTRA_OUTGOING_CALL_EXTRAS, extras(id, handle, name, video));
			if (video) {
				b.putInt(TelecomManager.EXTRA_START_CALL_WITH_VIDEO_STATE, VideoProfile.STATE_BIDIRECTIONAL);
			}
			telecom().placeCall(Uri.fromParts("giode", handle, null), b);
			return "";
		} catch (SecurityException e) {
			return "the app's manifest lacks MANAGE_OWN_CALLS: " + e.getMessage();
		} catch (Exception e) {
			return e.toString();
		}
	}

	private static CallConnection get(String id) {
		synchronized (connections) {
			return connections.get(id);
		}
	}

	public static void connected(String id) {
		CallConnection c = get(id);
		if (c != null) {
			c.setActive();
		}
	}

	// end ends a call from the app; reason -1 is a local hang-up.
	public static void end(String id, int reason) {
		CallConnection c = get(id);
		if (c == null) {
			return;
		}
		int code;
		switch (reason) {
		case 0: code = DisconnectCause.REMOTE; break;
		case 1: code = DisconnectCause.ERROR; break;
		case 2: code = DisconnectCause.MISSED; break;
		case 3: code = DisconnectCause.ANSWERED_ELSEWHERE; break;
		case 4: code = DisconnectCause.REJECTED; break;
		default: code = DisconnectCause.LOCAL;
		}
		c.finish(new DisconnectCause(code), false);
	}

	public static void mute(String id, boolean muted) {
		// Self-managed calls own their audio: report it to the app only.
		CallConnection c = get(id);
		if (c != null) {
			onEvent(muted ? "mute" : "unmute", id, "");
		}
	}

	private static void registerReceiver() {
		if (receiverRegistered) {
			return;
		}
		receiverRegistered = true;
		IntentFilter f = new IntentFilter();
		f.addAction(ACTION_ANSWER);
		f.addAction(ACTION_DECLINE);
		BroadcastReceiver r = new BroadcastReceiver() {
			public void onReceive(Context ctx, Intent i) {
				CallConnection c = get(i.getStringExtra(EXTRA_ID));
				if (c == null) {
					return;
				}
				if (ACTION_ANSWER.equals(i.getAction())) {
					c.onAnswer();
					// Bring the app up to show the call.
					Intent launch = app.getPackageManager().getLaunchIntentForPackage(app.getPackageName());
					if (launch != null) {
						launch.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_REORDER_TO_FRONT);
						app.startActivity(launch);
					}
				} else {
					c.onReject();
				}
			}
		};
		if (Build.VERSION.SDK_INT >= 33) {
			app.registerReceiver(r, f, Context.RECEIVER_NOT_EXPORTED);
		} else {
			app.registerReceiver(r, f);
		}
	}

	private static PendingIntent action(String action, String id, int code) {
		Intent i = new Intent(action).setPackage(app.getPackageName()).putExtra(EXTRA_ID, id);
		return PendingIntent.getBroadcast(app, code, i, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
	}

	// ring shows the incoming call: a full-screen notification over
	// the lock screen, a heads-up one otherwise.
	@SuppressWarnings("deprecation")
	static void ring(CallConnection c) {
		NotificationManager nm = (NotificationManager) app.getSystemService(Context.NOTIFICATION_SERVICE);
		if (nm.getNotificationChannel(CHANNEL) == null) {
			NotificationChannel ch = new NotificationChannel(CHANNEL, "Incoming calls", NotificationManager.IMPORTANCE_HIGH);
			ch.setSound(RingtoneManager.getDefaultUri(RingtoneManager.TYPE_RINGTONE),
				new AudioAttributes.Builder().setUsage(AudioAttributes.USAGE_NOTIFICATION_RINGTONE).build());
			nm.createNotificationChannel(ch);
		}
		Intent launch = app.getPackageManager().getLaunchIntentForPackage(app.getPackageName());
		PendingIntent full = PendingIntent.getActivity(app, 0, launch, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
		PendingIntent answer = action(ACTION_ANSWER, c.id, 1), decline = action(ACTION_DECLINE, c.id, 2);
		Notification.Builder b = new Notification.Builder(app, CHANNEL)
			.setSmallIcon(android.R.drawable.sym_call_incoming)
			.setContentTitle(c.name)
			.setContentText(c.video ? "Incoming video call" : "Incoming call")
			.setCategory(Notification.CATEGORY_CALL)
			.setOngoing(true)
			.setFullScreenIntent(full, true);
		if (Build.VERSION.SDK_INT >= 31) {
			Person caller = new Person.Builder().setName(c.name).setImportant(true).build();
			b.setStyle(Notification.CallStyle.forIncomingCall(caller, decline, answer));
		} else {
			b.addAction(new Notification.Action.Builder(null, "Decline", decline).build());
			b.addAction(new Notification.Action.Builder(null, "Answer", answer).build());
		}
		nm.notify(NOTIFICATION_ID, b.build());
	}

	static void stopRinging() {
		NotificationManager nm = (NotificationManager) app.getSystemService(Context.NOTIFICATION_SERVICE);
		nm.cancel(NOTIFICATION_ID);
	}

	// CallConnection is one call.
	static final class CallConnection extends Connection {
		final String id, name;
		final boolean video;
		private boolean done;

		CallConnection(String id, String handle, String name, boolean video) {
			this.id = id;
			this.name = name == null || name.isEmpty() ? handle : name;
			this.video = video;
			setConnectionProperties(PROPERTY_SELF_MANAGED);
			setConnectionCapabilities(CAPABILITY_MUTE | CAPABILITY_HOLD | CAPABILITY_SUPPORT_HOLD);
			setAudioModeIsVoip(true);
			setAddress(Uri.fromParts("giode", handle, null), TelecomManager.PRESENTATION_ALLOWED);
			setCallerDisplayName(this.name, TelecomManager.PRESENTATION_ALLOWED);
			if (video) {
				setVideoState(VideoProfile.STATE_BIDIRECTIONAL);
			}
			synchronized (connections) {
				connections.put(id, this);
			}
		}

		@Override
		public void onShowIncomingCallUi() {
			ring(this);
		}

		@Override
		public void onAnswer() {
			stopRinging();
			setActive();
			onEvent("answer", id, "");
			onEvent("audio_on", "", "");
		}

		@Override
		public void onAnswer(int videoState) {
			onAnswer();
		}

		@Override
		public void onReject() {
			finish(new DisconnectCause(DisconnectCause.REJECTED), true);
		}

		@Override
		public void onDisconnect() {
			finish(new DisconnectCause(DisconnectCause.LOCAL), true);
		}

		@Override
		public void onAbort() {
			finish(new DisconnectCause(DisconnectCause.CANCELED), true);
		}

		@Override
		public void onHold() {
			setOnHold();
			onEvent("hold", id, "");
		}

		@Override
		public void onUnhold() {
			setActive();
			onEvent("unhold", id, "");
		}

		@Override
		@SuppressWarnings("deprecation")
		public void onCallAudioStateChanged(CallAudioState s) {
			onEvent(s.isMuted() ? "mute" : "unmute", id, "");
		}

		// finish ends the call; byUser reports it to the app.
		void finish(DisconnectCause cause, boolean byUser) {
			if (done) {
				return;
			}
			done = true;
			stopRinging();
			setDisconnected(cause);
			destroy();
			synchronized (connections) {
				connections.remove(id);
			}
			if (byUser) {
				onEvent("end", id, "");
			}
			onEvent("audio_off", "", "");
		}
	}
}
