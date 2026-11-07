package org.giode.mobile;

import android.app.Activity;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.media.AudioAttributes;
import android.media.AudioFormat;
import android.media.AudioManager;
import android.media.AudioTrack;
import android.net.Uri;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;
import android.view.View;
import android.view.WindowManager;

import java.security.KeyStore;
import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;

// services_android holds giode's small Android services: opening URLs,
// haptic feedback, keeping the screen on, and secure storage backed by
// the Android Keystore.
public final class services_android {
	// openURL opens url in the app that handles it, usually the
	// browser. Call on the main thread.
	public static void openURL(View view, String url) {
		Context ctx = view.getContext();
		Intent intent = new Intent(Intent.ACTION_VIEW, Uri.parse(url));
		if (!(ctx instanceof Activity)) {
			intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
		}
		ctx.startActivity(intent);
	}

	// haptic performs a HapticFeedbackConstants feedback on view. Call on
	// the main thread.
	public static boolean haptic(View view, int constant) {
		if (view.performHapticFeedback(constant)) {
			return true;
		}
		// CONFIRM and REJECT need Android 11; fall back to a plain tap.
		return view.performHapticFeedback(android.view.HapticFeedbackConstants.VIRTUAL_KEY);
	}

	// keepScreenOn keeps the screen from dimming while the activity is
	// shown. Call on the main thread.
	public static void keepScreenOn(View view, boolean on) {
		Context ctx = view.getContext();
		if (!(ctx instanceof Activity)) {
			return;
		}
		if (on) {
			((Activity) ctx).getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
		} else {
			((Activity) ctx).getWindow().clearFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
		}
	}

	private static final String KEY_ALIAS = "giode.securestore";
	private static final String PREFS = "giode_securestore";

	// key returns the store's AES key, created on first use. It lives
	// in the Android Keystore: it can't be read out of the device, so
	// values copied off it (in a backup, say) can't be decrypted.
	private static SecretKey key() throws Exception {
		KeyStore ks = KeyStore.getInstance("AndroidKeyStore");
		ks.load(null);
		if (ks.containsAlias(KEY_ALIAS)) {
			return ((KeyStore.SecretKeyEntry) ks.getEntry(KEY_ALIAS, null)).getSecretKey();
		}
		KeyGenerator gen = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore");
		gen.init(new KeyGenParameterSpec.Builder(KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT)
				.setBlockModes(KeyProperties.BLOCK_MODE_GCM)
				.setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
				.setKeySize(256)
				.build());
		return gen.generateKey();
	}

	private static void requireKeystore() throws Exception {
		if (Build.VERSION.SDK_INT < 23) {
			throw new Exception("secure storage needs Android 6.0 or later");
		}
	}

	// secureSet encrypts value with the Keystore key and stores it under
	// name.
	public static void secureSet(Context ctx, String name, byte[] value) throws Exception {
		requireKeystore();
		Cipher c = Cipher.getInstance("AES/GCM/NoPadding");
		c.init(Cipher.ENCRYPT_MODE, key());
		byte[] iv = c.getIV();
		byte[] sealed = c.doFinal(value);
		byte[] out = new byte[1 + iv.length + sealed.length];
		out[0] = (byte) iv.length;
		System.arraycopy(iv, 0, out, 1, iv.length);
		System.arraycopy(sealed, 0, out, 1 + iv.length, sealed.length);
		prefs(ctx).edit().putString(name, Base64.encodeToString(out, Base64.NO_WRAP)).commit();
	}

	// secureGet returns the value stored under name, or null.
	public static byte[] secureGet(Context ctx, String name) throws Exception {
		requireKeystore();
		String s = prefs(ctx).getString(name, null);
		if (s == null) {
			return null;
		}
		byte[] in = Base64.decode(s, Base64.NO_WRAP);
		int n = in[0];
		Cipher c = Cipher.getInstance("AES/GCM/NoPadding");
		c.init(Cipher.DECRYPT_MODE, key(), new GCMParameterSpec(128, in, 1, n));
		return c.doFinal(in, 1 + n, in.length - 1 - n);
	}

	// secureDelete removes the value stored under name.
	public static void secureDelete(Context ctx, String name) {
		prefs(ctx).edit().remove(name).commit();
	}

	// playSound plays 16-bit PCM once through a static AudioTrack, as a
	// notification sound, and releases the track when it has played.
	// Any thread.
	public static void playSound(Context ctx, byte[] pcm, int rate, int channels) {
		int mask = channels == 2 ? AudioFormat.CHANNEL_OUT_STEREO : AudioFormat.CHANNEL_OUT_MONO;
		final AudioTrack track;
		if (Build.VERSION.SDK_INT >= 21) {
			track = new AudioTrack(
				new AudioAttributes.Builder()
					.setUsage(AudioAttributes.USAGE_NOTIFICATION_EVENT)
					.setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
					.build(),
				new AudioFormat.Builder()
					.setEncoding(AudioFormat.ENCODING_PCM_16BIT)
					.setSampleRate(rate)
					.setChannelMask(mask)
					.build(),
				pcm.length, AudioTrack.MODE_STATIC, AudioManager.AUDIO_SESSION_ID_GENERATE);
		} else {
			track = new AudioTrack(AudioManager.STREAM_NOTIFICATION, rate, mask,
				AudioFormat.ENCODING_PCM_16BIT, pcm.length, AudioTrack.MODE_STATIC);
		}
		if (track.getState() != AudioTrack.STATE_INITIALIZED) {
			track.release();
			throw new IllegalStateException("AudioTrack not initialized");
		}
		track.write(pcm, 0, pcm.length);
		int frames = pcm.length / (2 * channels);
		track.setNotificationMarkerPosition(frames);
		track.setPlaybackPositionUpdateListener(new AudioTrack.OnPlaybackPositionUpdateListener() {
			@Override
			public void onMarkerReached(AudioTrack t) {
				t.release();
			}

			@Override
			public void onPeriodicNotification(AudioTrack t) {
			}
		}, new Handler(Looper.getMainLooper()));
		track.play();
	}

	private static SharedPreferences prefs(Context ctx) {
		return ctx.getApplicationContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
	}
}
