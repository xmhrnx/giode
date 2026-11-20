package org.giode.screencap;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.Service;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.content.pm.ServiceInfo;
import android.graphics.PixelFormat;
import android.hardware.display.DisplayManager;
import android.hardware.display.VirtualDisplay;
import android.media.Image;
import android.media.ImageReader;
import android.media.projection.MediaProjection;
import android.media.projection.MediaProjectionManager;
import android.os.Build;
import android.os.Handler;
import android.os.HandlerThread;
import android.os.IBinder;
import android.os.SystemClock;

// ScreenCaptureService holds the MediaProjection while the screen is
// captured, showing an ongoing notification. The package's
// android_manifest.xml declares it; the Giode CLI merges that into the
// app's manifest.
public final class ScreenCaptureService extends Service {
	private static final String CHANNEL = "giode.screencap";
	private static final int NOTIFICATION_ID = 0x5c4;

	private static final Object lock = new Object();
	// The pending start, handed from startCapture to onStartCommand.
	private static Intent pendingData;
	private static int pendingReq, pendingID, pendingResult, pendingW, pendingH, pendingFPS;

	private MediaProjection projection;
	private VirtualDisplay display;
	private ImageReader reader;
	private HandlerThread thread;
	private int id;
	private long lastFrame, interval;

	// startCapture starts the service, which starts capturing. It
	// returns an error, or null when the service reports to onStarted.
	static String startCapture(Context ctx, int req, int id, int result, Intent data, int w, int h, int fps) {
		try {
			ctx.getPackageManager().getServiceInfo(new ComponentName(ctx, ScreenCaptureService.class), 0);
		} catch (PackageManager.NameNotFoundException e) {
			return "the app's manifest doesn't declare org.giode.screencap.ScreenCaptureService (build with the Giode CLI)";
		}
		synchronized (lock) {
			pendingReq = req;
			pendingID = id;
			pendingResult = result;
			pendingData = data;
			pendingW = w;
			pendingH = h;
			pendingFPS = fps;
		}
		Intent i = new Intent(ctx, ScreenCaptureService.class);
		try {
			if (Build.VERSION.SDK_INT >= 26) {
				ctx.startForegroundService(i);
			} else {
				ctx.startService(i);
			}
		} catch (Exception e) {
			return e.toString();
		}
		return null;
	}

	static void stopCapture(Context ctx) {
		ctx.stopService(new Intent(ctx, ScreenCaptureService.class));
	}

	@Override
	public int onStartCommand(Intent intent, int flags, int startId) {
		Notification n = notification();
		if (Build.VERSION.SDK_INT >= 29) {
			startForeground(NOTIFICATION_ID, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION);
		} else {
			startForeground(NOTIFICATION_ID, n);
		}
		int req, w, h, fps, result;
		Intent data;
		synchronized (lock) {
			req = pendingReq;
			id = pendingID;
			result = pendingResult;
			data = pendingData;
			w = pendingW;
			h = pendingH;
			fps = pendingFPS;
			pendingData = null;
		}
		if (data == null) {
			stopSelf();
			return START_NOT_STICKY;
		}
		try {
			begin(result, data, w, h, fps);
			screencap_android.onStarted(req, "");
		} catch (Exception e) {
			end();
			stopSelf();
			screencap_android.onStarted(req, e.toString());
		}
		return START_NOT_STICKY;
	}

	private void begin(int result, Intent data, int w, int h, int fps) {
		MediaProjectionManager mpm = (MediaProjectionManager) getSystemService(Context.MEDIA_PROJECTION_SERVICE);
		projection = mpm.getMediaProjection(result, data);
		if (projection == null) {
			throw new IllegalStateException("no media projection");
		}
		thread = new HandlerThread("giode-screencap");
		thread.start();
		Handler handler = new Handler(thread.getLooper());
		interval = 1000 / Math.max(fps, 1);
		// Android 14 requires a callback before the virtual display.
		projection.registerCallback(new MediaProjection.Callback() {
			@Override
			public void onStop() {
				screencap_android.onStopped(id);
				stopSelf();
			}
		}, handler);
		reader = ImageReader.newInstance(w, h, PixelFormat.RGBA_8888, 2);
		reader.setOnImageAvailableListener(new ImageReader.OnImageAvailableListener() {
			public void onImageAvailable(ImageReader r) {
				Image img = r.acquireLatestImage();
				if (img == null) {
					return;
				}
				try {
					long now = SystemClock.uptimeMillis();
					if (now - lastFrame < interval) {
						return;
					}
					lastFrame = now;
					Image.Plane p = img.getPlanes()[0];
					screencap_android.onFrame(id, p.getBuffer(), img.getWidth(), img.getHeight(), p.getRowStride());
				} finally {
					img.close();
				}
			}
		}, handler);
		int dpi = getResources().getDisplayMetrics().densityDpi;
		display = projection.createVirtualDisplay("giode-screencap", w, h, dpi,
			DisplayManager.VIRTUAL_DISPLAY_FLAG_AUTO_MIRROR, reader.getSurface(), null, handler);
	}

	private void end() {
		if (display != null) {
			display.release();
			display = null;
		}
		if (projection != null) {
			projection.stop();
			projection = null;
		}
		if (reader != null) {
			reader.close();
			reader = null;
		}
		if (thread != null) {
			thread.quitSafely();
			thread = null;
		}
	}

	@Override
	public void onDestroy() {
		end();
		super.onDestroy();
	}

	@Override
	public IBinder onBind(Intent intent) {
		return null;
	}

	@SuppressWarnings("deprecation")
	private Notification notification() {
		CharSequence label = getApplicationInfo().loadLabel(getPackageManager());
		Notification.Builder b;
		if (Build.VERSION.SDK_INT >= 26) {
			NotificationManager nm = (NotificationManager) getSystemService(Context.NOTIFICATION_SERVICE);
			if (nm.getNotificationChannel(CHANNEL) == null) {
				nm.createNotificationChannel(new NotificationChannel(CHANNEL, "Screen sharing", NotificationManager.IMPORTANCE_LOW));
			}
			b = new Notification.Builder(this, CHANNEL);
		} else {
			b = new Notification.Builder(this);
		}
		return b.setSmallIcon(android.R.drawable.ic_menu_share)
			.setContentTitle(label)
			.setContentText("Sharing the screen")
			.setOngoing(true)
			.build();
	}
}
