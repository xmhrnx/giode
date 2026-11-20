package org.giode.media;

import android.content.Context;
import android.graphics.ImageFormat;
import android.media.AudioAttributes;
import android.media.Image;
import android.media.ImageReader;
import android.media.MediaPlayer;
import android.media.PlaybackParams;
import android.net.Uri;
import android.os.Build;
import android.os.Handler;
import android.os.HandlerThread;
import android.os.Looper;

import java.nio.ByteBuffer;
import java.util.HashMap;
import java.util.Map;

// media_android runs giode/media on MediaPlayer: it plays the sound and
// keeps time, decoding the picture into an ImageReader whose YUV frames
// go to Go.
public final class media_android {
	// Implemented in Go (media_android.go).
	static native void onReady(int id, int durationMs, int w, int h);
	static native void onFrame(int id, ByteBuffer y, ByteBuffer u, ByteBuffer v, int w, int h, int yStride, int uvStride, int uvPixStride);
	static native void onEnded(int id);
	static native void onError(int id, String err);

	private static final class P {
		MediaPlayer mp;
		ImageReader reader;
		HandlerThread thread;
		Handler handler;
		float speed = 1;
		boolean playing;
	}

	private static final Map<Integer, P> players = new HashMap<>();

	private static P get(int id) {
		synchronized (players) {
			return players.get(id);
		}
	}

	// open starts loading src, a path or a URL; it returns "" or an error.
	public static String open(final Context ctx, final int id, final String src, final float volume) {
		final P p = new P();
		p.thread = new HandlerThread("giode-media");
		p.thread.start();
		p.handler = new Handler(p.thread.getLooper());
		synchronized (players) {
			players.put(id, p);
		}
		// MediaPlayer calls back on the looper it's made on.
		p.handler.post(new Runnable() {
			public void run() {
				try {
					p.mp = new MediaPlayer();
					if (Build.VERSION.SDK_INT >= 21) {
						p.mp.setAudioAttributes(new AudioAttributes.Builder()
							.setUsage(AudioAttributes.USAGE_MEDIA)
							.setContentType(AudioAttributes.CONTENT_TYPE_MOVIE).build());
					}
					if (src.contains("://")) {
						p.mp.setDataSource(ctx, Uri.parse(src));
					} else {
						p.mp.setDataSource(src);
					}
					p.mp.setVolume(volume, volume);
					p.mp.setOnPreparedListener(new MediaPlayer.OnPreparedListener() {
						public void onPrepared(MediaPlayer mp) {
							int w = mp.getVideoWidth(), h = mp.getVideoHeight();
							if (w > 0 && h > 0) {
								attachReader(id, p, w, h);
								// Show the first frame.
								mp.seekTo(0);
							}
							onReady(id, mp.getDuration(), w, h);
						}
					});
					p.mp.setOnCompletionListener(new MediaPlayer.OnCompletionListener() {
						public void onCompletion(MediaPlayer mp) {
							p.playing = false;
							onEnded(id);
						}
					});
					p.mp.setOnErrorListener(new MediaPlayer.OnErrorListener() {
						public boolean onError(MediaPlayer mp, int what, int extra) {
							media_android.onError(id, "playback error " + what + "/" + extra);
							return true;
						}
					});
					p.mp.prepareAsync();
				} catch (Exception e) {
					onError(id, e.toString());
				}
			}
		});
		return "";
	}

	private static void attachReader(final int id, P p, int w, int h) {
		p.reader = ImageReader.newInstance(w, h, ImageFormat.YUV_420_888, 3);
		p.reader.setOnImageAvailableListener(new ImageReader.OnImageAvailableListener() {
			public void onImageAvailable(ImageReader r) {
				Image img = r.acquireLatestImage();
				if (img == null) {
					return;
				}
				try {
					Image.Plane[] pl = img.getPlanes();
					onFrame(id, pl[0].getBuffer(), pl[1].getBuffer(), pl[2].getBuffer(), img.getWidth(), img.getHeight(),
						pl[0].getRowStride(), pl[1].getRowStride(), pl[1].getPixelStride());
				} finally {
					img.close();
				}
			}
		}, p.handler);
		p.mp.setSurface(p.reader.getSurface());
	}

	private static void run(int id, final Op op) {
		final P p = get(id);
		if (p == null) {
			return;
		}
		p.handler.post(new Runnable() {
			public void run() {
				if (p.mp == null) {
					return;
				}
				try {
					op.run(p);
				} catch (Exception e) {
					// IllegalStateException before prepared: ignore.
				}
			}
		});
	}

	private interface Op {
		void run(P p);
	}

	public static void play(int id) {
		run(id, new Op() {
			public void run(P p) {
				p.mp.start();
				p.playing = true;
				applySpeed(p);
			}
		});
	}

	public static void pause(int id) {
		run(id, new Op() {
			public void run(P p) {
				p.mp.pause();
				p.playing = false;
			}
		});
	}

	public static void seek(int id, final int ms) {
		run(id, new Op() {
			public void run(P p) {
				if (Build.VERSION.SDK_INT >= 26) {
					p.mp.seekTo(ms, MediaPlayer.SEEK_CLOSEST);
				} else {
					p.mp.seekTo(ms);
				}
			}
		});
	}

	public static void speed(int id, final float s) {
		run(id, new Op() {
			public void run(P p) {
				p.speed = s;
				// Setting a speed on a paused player starts it: wait.
				if (p.playing) {
					applySpeed(p);
				}
			}
		});
	}

	private static void applySpeed(P p) {
		if (Build.VERSION.SDK_INT >= 23) {
			PlaybackParams pp = p.mp.getPlaybackParams();
			p.mp.setPlaybackParams(pp.setSpeed(p.speed));
		}
	}

	public static void volume(int id, final float v) {
		run(id, new Op() {
			public void run(P p) {
				p.mp.setVolume(v, v);
			}
		});
	}

	public static int position(int id) {
		P p = get(id);
		if (p == null || p.mp == null) {
			return 0;
		}
		try {
			return p.mp.getCurrentPosition();
		} catch (Exception e) {
			return 0;
		}
	}

	public static void close(int id) {
		final P p;
		synchronized (players) {
			p = players.remove(id);
		}
		if (p == null) {
			return;
		}
		p.handler.post(new Runnable() {
			public void run() {
				if (p.mp != null) {
					p.mp.release();
				}
				if (p.reader != null) {
					p.reader.close();
				}
				p.thread.quitSafely();
			}
		});
	}
}
