package org.giode.screencap;

import android.app.Activity;
import android.app.Fragment;
import android.app.FragmentManager;
import android.content.Context;
import android.content.Intent;
import android.media.projection.MediaProjectionManager;
import android.os.Build;
import android.os.Bundle;
import android.util.DisplayMetrics;
import android.view.View;
import android.view.WindowManager;

import java.nio.ByteBuffer;

// screencap_android runs giode/screencap on MediaProjection: it asks the
// user (Android asks every time), then starts ScreenCaptureService,
// which holds the projection: Android 10 and later only allow one from a
// foreground service of type mediaProjection.
public final class screencap_android {
	// Implemented in Go (screencap_android.go).
	static native void onStarted(int req, String err);
	static native void onFrame(int id, ByteBuffer pixels, int w, int h, int rowStride);
	static native void onStopped(int id);

	private static final String TAG = "giode_screencap";
	private static final int REQUEST = 0x5c4;

	// screenSize returns the display's size in pixels, "w h".
	@SuppressWarnings("deprecation")
	public static String screenSize(Context ctx) {
		WindowManager wm = (WindowManager) ctx.getSystemService(Context.WINDOW_SERVICE);
		DisplayMetrics m = new DisplayMetrics();
		wm.getDefaultDisplay().getRealMetrics(m);
		return m.widthPixels + " " + m.heightPixels;
	}

	// start asks the user to allow capturing the screen and, when they
	// do, captures it at w×h, reporting to onStarted.
	public static void start(final View view, final int req, final int id, final int w, final int h, final int fps) {
		view.post(new Runnable() {
			public void run() {
				Activity activity = (Activity) view.getContext();
				if (Build.VERSION.SDK_INT < 21) {
					onStarted(req, "screen capture needs Android 5");
					return;
				}
				MediaProjectionManager mpm = (MediaProjectionManager) activity.getSystemService(Context.MEDIA_PROJECTION_SERVICE);
				FragmentManager fm = activity.getFragmentManager();
				ConsentFragment f = new ConsentFragment();
				Bundle args = new Bundle();
				args.putInt("req", req);
				args.putInt("id", id);
				args.putInt("w", w);
				args.putInt("h", h);
				args.putInt("fps", fps);
				f.setArguments(args);
				fm.beginTransaction().add(f, TAG + req).commitNow();
				try {
					f.startActivityForResult(mpm.createScreenCaptureIntent(), REQUEST);
				} catch (Exception e) {
					fm.beginTransaction().remove(f).commitAllowingStateLoss();
					onStarted(req, e.toString());
				}
			}
		});
	}

	public static void stop(Context ctx) {
		ScreenCaptureService.stopCapture(ctx);
	}

	public static class ConsentFragment extends Fragment {
		@Override
		public void onActivityResult(int code, int result, Intent data) {
			Bundle a = getArguments();
			int req = a.getInt("req");
			Activity activity = getActivity();
			getFragmentManager().beginTransaction().remove(this).commitAllowingStateLoss();
			if (result != Activity.RESULT_OK || data == null) {
				onStarted(req, "denied");
				return;
			}
			String err = ScreenCaptureService.startCapture(activity, req, a.getInt("id"), result, data,
				a.getInt("w"), a.getInt("h"), a.getInt("fps"));
			if (err != null) {
				onStarted(req, err);
			}
		}
	}
}
