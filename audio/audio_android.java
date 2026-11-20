// SPDX-License-Identifier: Unlicense OR MIT

package org.giode.audio;

import android.Manifest;
import android.app.Activity;
import android.app.Fragment;
import android.app.FragmentManager;
import android.content.Context;
import android.content.pm.PackageManager;
import android.media.AudioAttributes;
import android.media.AudioDeviceInfo;
import android.media.AudioFormat;
import android.media.AudioManager;
import android.media.AudioRecord;
import android.media.AudioTrack;
import android.media.MediaRecorder;
import android.media.audiofx.AcousticEchoCanceler;
import android.media.audiofx.AutomaticGainControl;
import android.media.audiofx.NoiseSuppressor;
import android.os.Build;
import android.os.Bundle;
import android.os.Process;
import android.view.View;

import java.util.HashMap;
import java.util.Map;

// audio_android streams the microphone and the speaker for package
// audio.
public final class audio_android {
	// Implemented in Go.
	static native void onAudio(int id, short[] data, int n);
	static native int onPull(int id, short[] buf);
	static native void onPermission(int req, boolean granted);

	private static final String TAG = "giode_audio";

	private static final class Stream {
		volatile boolean running = true;
		Thread thread;
		AudioRecord record;
		AudioTrack track;
		AcousticEchoCanceler aec;
		NoiseSuppressor ns;
		AutomaticGainControl agc;
		boolean voice;
	}

	private static final Map<Integer, Stream> streams = new HashMap<>();
	private static int voiceStreams;

	public static boolean hasPermission(Context ctx) {
		return Build.VERSION.SDK_INT < 23 || ctx.checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED;
	}

	// requestPermission asks for the microphone, answering onPermission.
	public static void requestPermission(final View view, final int req) {
		view.post(new Runnable() {
			public void run() {
				Activity activity = (Activity) view.getContext();
				if (hasPermission(activity)) {
					onPermission(req, true);
					return;
				}
				FragmentManager fm = activity.getFragmentManager();
				PermissionFragment f = new PermissionFragment();
				Bundle args = new Bundle();
				args.putInt("req", req);
				f.setArguments(args);
				fm.beginTransaction().add(f, TAG + req).commitNow();
				f.requestPermissions(new String[]{Manifest.permission.RECORD_AUDIO}, req & 0xffff);
			}
		});
	}

	public static class PermissionFragment extends Fragment {
		@Override
		public void onRequestPermissionsResult(int code, String[] perms, int[] results) {
			int req = getArguments().getInt("req");
			boolean granted = results.length > 0 && results[0] == PackageManager.PERMISSION_GRANTED;
			getFragmentManager().beginTransaction().remove(this).commitAllowingStateLoss();
			onPermission(req, granted);
		}
	}

	// devices lists devices as "id\tname\tdefault\n" lines.
	public static String devices(Context ctx, boolean input) {
		if (Build.VERSION.SDK_INT < 23) {
			return "";
		}
		AudioManager am = (AudioManager) ctx.getSystemService(Context.AUDIO_SERVICE);
		StringBuilder b = new StringBuilder();
		boolean first = true;
		for (AudioDeviceInfo d : am.getDevices(input ? AudioManager.GET_DEVICES_INPUTS : AudioManager.GET_DEVICES_OUTPUTS)) {
			String name = String.valueOf(d.getProductName());
			String type = typeName(d.getType());
			if (!type.isEmpty()) {
				name = type + (name.isEmpty() || name.equals(android.os.Build.MODEL) ? "" : " (" + name + ")");
			}
			b.append(d.getId()).append('\t').append(name.replace('\t', ' ').replace('\n', ' ')).append('\t').append(first ? "1" : "0").append('\n');
			first = false;
		}
		return b.toString();
	}

	private static String typeName(int t) {
		switch (t) {
		case AudioDeviceInfo.TYPE_BUILTIN_MIC: return "Microphone";
		case AudioDeviceInfo.TYPE_BUILTIN_SPEAKER: return "Speaker";
		case AudioDeviceInfo.TYPE_BUILTIN_EARPIECE: return "Earpiece";
		case AudioDeviceInfo.TYPE_WIRED_HEADSET: return "Headset";
		case AudioDeviceInfo.TYPE_WIRED_HEADPHONES: return "Headphones";
		case AudioDeviceInfo.TYPE_BLUETOOTH_SCO: return "Bluetooth headset";
		case AudioDeviceInfo.TYPE_BLUETOOTH_A2DP: return "Bluetooth";
		case AudioDeviceInfo.TYPE_USB_DEVICE:
		case AudioDeviceInfo.TYPE_USB_HEADSET: return "USB";
		}
		return "";
	}

	private static AudioDeviceInfo device(Context ctx, int id, boolean input) {
		if (id == 0 || Build.VERSION.SDK_INT < 23) {
			return null;
		}
		AudioManager am = (AudioManager) ctx.getSystemService(Context.AUDIO_SERVICE);
		for (AudioDeviceInfo d : am.getDevices(input ? AudioManager.GET_DEVICES_INPUTS : AudioManager.GET_DEVICES_OUTPUTS)) {
			if (d.getId() == id) {
				return d;
			}
		}
		return null;
	}

	// voiceMode puts the phone in call mode while voice streams run,
	// and back after.
	private static void voiceMode(Context ctx, boolean on, boolean speaker) {
		AudioManager am = (AudioManager) ctx.getSystemService(Context.AUDIO_SERVICE);
		if (on) {
			if (voiceStreams++ == 0) {
				am.setMode(AudioManager.MODE_IN_COMMUNICATION);
			}
			setSpeaker(am, speaker);
		} else if (--voiceStreams <= 0) {
			voiceStreams = 0;
			if (Build.VERSION.SDK_INT >= 31) {
				am.clearCommunicationDevice();
			} else {
				am.setSpeakerphoneOn(false);
			}
			am.setMode(AudioManager.MODE_NORMAL);
		}
	}

	@SuppressWarnings("deprecation")
	private static void setSpeaker(AudioManager am, boolean on) {
		if (Build.VERSION.SDK_INT >= 31) {
			for (AudioDeviceInfo d : am.getAvailableCommunicationDevices()) {
				if (on ? d.getType() == AudioDeviceInfo.TYPE_BUILTIN_SPEAKER : d.getType() == AudioDeviceInfo.TYPE_BUILTIN_EARPIECE) {
					am.setCommunicationDevice(d);
					return;
				}
			}
			return;
		}
		am.setSpeakerphoneOn(on);
	}

	// openInput starts recording into onAudio; it returns "" or an error.
	public static String openInput(final Context ctx, final int id, int rate, int channels, boolean voice, int deviceId, final int frame) {
		try {
			int ch = channels == 2 ? AudioFormat.CHANNEL_IN_STEREO : AudioFormat.CHANNEL_IN_MONO;
			int min = AudioRecord.getMinBufferSize(rate, ch, AudioFormat.ENCODING_PCM_16BIT);
			if (min <= 0) {
				return "unsupported format " + rate + " Hz";
			}
			int source = voice ? MediaRecorder.AudioSource.VOICE_COMMUNICATION : MediaRecorder.AudioSource.MIC;
			final Stream s = new Stream();
			s.voice = voice;
			s.record = new AudioRecord(source, rate, ch, AudioFormat.ENCODING_PCM_16BIT, Math.max(min, frame * 2 * 4));
			if (s.record.getState() != AudioRecord.STATE_INITIALIZED) {
				s.record.release();
				return "the microphone couldn't be opened";
			}
			AudioDeviceInfo dev = device(ctx, deviceId, true);
			if (dev != null && Build.VERSION.SDK_INT >= 23) {
				s.record.setPreferredDevice(dev);
			}
			if (voice) {
				int session = s.record.getAudioSessionId();
				if (AcousticEchoCanceler.isAvailable()) {
					s.aec = AcousticEchoCanceler.create(session);
					if (s.aec != null) s.aec.setEnabled(true);
				}
				if (NoiseSuppressor.isAvailable()) {
					s.ns = NoiseSuppressor.create(session);
					if (s.ns != null) s.ns.setEnabled(true);
				}
				if (AutomaticGainControl.isAvailable()) {
					s.agc = AutomaticGainControl.create(session);
					if (s.agc != null) s.agc.setEnabled(true);
				}
				voiceMode(ctx, true, false);
			}
			s.record.startRecording();
			s.thread = new Thread(new Runnable() {
				public void run() {
					Process.setThreadPriority(Process.THREAD_PRIORITY_URGENT_AUDIO);
					short[] buf = new short[frame];
					while (s.running) {
						int n = s.record.read(buf, 0, buf.length);
						if (n < 0) {
							break;
						}
						if (n > 0) {
							onAudio(id, buf, n);
						}
					}
				}
			}, TAG + "-in");
			synchronized (streams) {
				streams.put(id, s);
			}
			s.thread.start();
			return "";
		} catch (Exception e) {
			return String.valueOf(e);
		}
	}

	// openOutput starts playing what onPull returns.
	public static String openOutput(final Context ctx, final int id, int rate, int channels, boolean voice, boolean speaker, int deviceId) {
		try {
			int ch = channels == 2 ? AudioFormat.CHANNEL_OUT_STEREO : AudioFormat.CHANNEL_OUT_MONO;
			int min = AudioTrack.getMinBufferSize(rate, ch, AudioFormat.ENCODING_PCM_16BIT);
			if (min <= 0) {
				return "unsupported format " + rate + " Hz";
			}
			AudioAttributes attrs = new AudioAttributes.Builder()
				.setUsage(voice ? AudioAttributes.USAGE_VOICE_COMMUNICATION : AudioAttributes.USAGE_MEDIA)
				.setContentType(voice ? AudioAttributes.CONTENT_TYPE_SPEECH : AudioAttributes.CONTENT_TYPE_MUSIC)
				.build();
			AudioFormat fmt = new AudioFormat.Builder().setSampleRate(rate).setChannelMask(ch).setEncoding(AudioFormat.ENCODING_PCM_16BIT).build();
			final Stream s = new Stream();
			s.voice = voice;
			s.track = new AudioTrack(attrs, fmt, Math.max(min, rate / 25 * 2 * channels), AudioTrack.MODE_STREAM, AudioManager.AUDIO_SESSION_ID_GENERATE);
			AudioDeviceInfo dev = device(ctx, deviceId, false);
			if (dev != null && Build.VERSION.SDK_INT >= 23) {
				s.track.setPreferredDevice(dev);
			}
			if (voice) {
				voiceMode(ctx, true, speaker);
			}
			s.track.play();
			final int chunk = rate / 100 * channels;
			s.thread = new Thread(new Runnable() {
				public void run() {
					Process.setThreadPriority(Process.THREAD_PRIORITY_URGENT_AUDIO);
					short[] buf = new short[chunk];
					while (s.running) {
						int n = onPull(id, buf);
						if (n <= 0) {
							java.util.Arrays.fill(buf, (short) 0);
							n = buf.length;
						}
						if (s.track.write(buf, 0, n) < 0) {
							break;
						}
					}
				}
			}, TAG + "-out");
			synchronized (streams) {
				streams.put(id, s);
			}
			s.thread.start();
			return "";
		} catch (Exception e) {
			return String.valueOf(e);
		}
	}

	public static void close(Context ctx, int id) {
		Stream s;
		synchronized (streams) {
			s = streams.remove(id);
		}
		if (s == null) {
			return;
		}
		s.running = false;
		try {
			if (s.record != null) s.record.stop();
			if (s.track != null) s.track.pause();
			s.thread.join(500);
		} catch (Exception e) {
		}
		if (s.aec != null) s.aec.release();
		if (s.ns != null) s.ns.release();
		if (s.agc != null) s.agc.release();
		if (s.record != null) s.record.release();
		if (s.track != null) s.track.release();
		if (s.voice) {
			voiceMode(ctx, false, false);
		}
	}
}
