package org.giode.camera;

import android.Manifest;
import android.app.Activity;
import android.app.Application;
import android.app.Fragment;
import android.app.FragmentManager;
import android.content.Context;
import android.content.pm.PackageManager;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.ImageFormat;
import android.graphics.Matrix;
import android.graphics.Rect;
import android.hardware.camera2.CameraAccessException;
import android.hardware.camera2.CameraCaptureSession;
import android.hardware.camera2.CameraCharacteristics;
import android.hardware.camera2.CameraDevice;
import android.hardware.camera2.CameraManager;
import android.hardware.camera2.CameraMetadata;
import android.hardware.camera2.CaptureRequest;
import android.hardware.camera2.params.MeteringRectangle;
import android.hardware.camera2.params.StreamConfigurationMap;
import android.media.ExifInterface;
import android.media.Image;
import android.media.ImageReader;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.HandlerThread;
import android.util.Size;
import android.view.Surface;
import android.view.View;
import android.view.WindowManager;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.nio.ByteBuffer;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;

// camera_android runs giode/camera on Camera2: a YUV stream for the
// preview and analysis frames, handed to Go as they arrive, and a JPEG
// stream for full-resolution pictures. It closes the camera when the
// activity pauses and reopens it when it resumes.
public final class camera_android {
	// Implemented in Go (camera_android.go).
	static native void onFrame(int id, ByteBuffer y, ByteBuffer u, ByteBuffer v, int w, int h, int yStride, int uvStride, int uvPixStride, int rotation, boolean mirror);
	static native void onCapture(int req, byte[] jpeg, String err);
	static native void onPermission(int req, boolean granted);
	static native void onError(int id, String err);

	private static final String TAG = "giode_camera";
	private static final Map<Integer, Cam> cams = new HashMap<>();

	private static final class Cam {
		int id;
		Context ctx;
		Activity activity;
		boolean front;
		int maxPreview;
		String cameraId;
		CameraCharacteristics chars;
		HandlerThread thread;
		Handler handler;
		CameraDevice device;
		CameraCaptureSession session;
		ImageReader yuv, jpeg;
		boolean torch;
		boolean hasFlash;
		MeteringRectangle[] regions;
		int pendingCapture;
		int captureRotation;
		boolean wanted = true; // open, not closed by Go
		Application.ActivityLifecycleCallbacks lifecycle;
	}

	public static boolean hasPermission(Context ctx) {
		return Build.VERSION.SDK_INT < 23 || ctx.checkSelfPermission(Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED;
	}

	// requestPermission asks for the camera permission and reports the
	// answer to onPermission.
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
				f.requestPermissions(new String[]{Manifest.permission.CAMERA}, req & 0xffff);
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

	// open starts camera id: the back camera, or the front one. It
	// returns "" or an error.
	public static String open(View view, int id, boolean front, int maxPreview) {
		try {
			final Cam c = new Cam();
			c.id = id;
			c.activity = (Activity) view.getContext();
			c.ctx = c.activity.getApplicationContext();
			c.front = front;
			c.maxPreview = maxPreview;
			CameraManager mgr = (CameraManager) c.ctx.getSystemService(Context.CAMERA_SERVICE);
			int want = front ? CameraCharacteristics.LENS_FACING_FRONT : CameraCharacteristics.LENS_FACING_BACK;
			for (String cid : mgr.getCameraIdList()) {
				CameraCharacteristics ch = mgr.getCameraCharacteristics(cid);
				Integer facing = ch.get(CameraCharacteristics.LENS_FACING);
				if (facing != null && facing == want) {
					c.cameraId = cid;
					c.chars = ch;
					break;
				}
			}
			if (c.cameraId == null) {
				return front ? "no front camera" : "no back camera";
			}
			Boolean flash = c.chars.get(CameraCharacteristics.FLASH_INFO_AVAILABLE);
			c.hasFlash = flash != null && flash;
			c.thread = new HandlerThread("giode-camera");
			c.thread.start();
			c.handler = new Handler(c.thread.getLooper());
			synchronized (cams) {
				cams.put(id, c);
			}
			watchLifecycle(c);
			startDevice(c);
			return "";
		} catch (Exception e) {
			return e.toString();
		}
	}

	public static void close(int id) {
		final Cam c;
		synchronized (cams) {
			c = cams.remove(id);
		}
		if (c == null) {
			return;
		}
		c.wanted = false;
		if (c.lifecycle != null) {
			c.activity.getApplication().unregisterActivityLifecycleCallbacks(c.lifecycle);
		}
		c.handler.post(new Runnable() {
			public void run() {
				stopDevice(c);
				c.thread.quitSafely();
			}
		});
	}

	public static boolean hasTorch(int id) {
		Cam c = cam(id);
		return c != null && c.hasFlash;
	}

	public static void torch(int id, final boolean on) {
		final Cam c = cam(id);
		if (c == null) {
			return;
		}
		c.handler.post(new Runnable() {
			public void run() {
				c.torch = on;
				repeat(c);
			}
		});
	}

	// focus focuses at (x, y) in the upright frame, both 0 to 1.
	public static void focus(int id, final float x, final float y) {
		final Cam c = cam(id);
		if (c == null) {
			return;
		}
		c.handler.post(new Runnable() {
			public void run() {
				Rect active = c.chars.get(CameraCharacteristics.SENSOR_INFO_ACTIVE_ARRAY_SIZE);
				if (active == null || c.session == null) {
					return;
				}
				// Back from the upright frame to the sensor's.
				float ux = c.front ? 1 - x : x, uy = y;
				float sx, sy;
				switch (rotation(c)) {
				case 90: sx = uy; sy = 1 - ux; break;
				case 180: sx = 1 - ux; sy = 1 - uy; break;
				case 270: sx = 1 - uy; sy = ux; break;
				default: sx = ux; sy = uy;
				}
				int half = Math.max(active.width(), active.height()) / 20;
				int cx = active.left + (int) (sx * active.width());
				int cy = active.top + (int) (sy * active.height());
				Rect r = new Rect(Math.max(active.left, cx - half), Math.max(active.top, cy - half),
					Math.min(active.right, cx + half), Math.min(active.bottom, cy + half));
				c.regions = new MeteringRectangle[]{new MeteringRectangle(r, MeteringRectangle.METERING_WEIGHT_MAX - 1)};
				try {
					CaptureRequest.Builder b = previewRequest(c);
					b.set(CaptureRequest.CONTROL_AF_MODE, CaptureRequest.CONTROL_AF_MODE_AUTO);
					b.set(CaptureRequest.CONTROL_AF_TRIGGER, CameraMetadata.CONTROL_AF_TRIGGER_START);
					c.session.capture(b.build(), null, c.handler);
					repeat(c);
				} catch (Exception e) {
					onError(c.id, e.toString());
				}
			}
		});
	}

	// capture takes a full-resolution picture and reports it, upright,
	// to onCapture.
	public static void capture(int id, final int req) {
		final Cam c = cam(id);
		if (c == null) {
			onCapture(req, null, "camera closed");
			return;
		}
		c.handler.post(new Runnable() {
			public void run() {
				if (c.session == null) {
					onCapture(req, null, "camera not ready");
					return;
				}
				if (c.pendingCapture != 0) {
					onCapture(req, null, "a capture is already running");
					return;
				}
				try {
					c.pendingCapture = req;
					c.captureRotation = rotation(c);
					CaptureRequest.Builder b = c.device.createCaptureRequest(CameraDevice.TEMPLATE_STILL_CAPTURE);
					b.addTarget(c.jpeg.getSurface());
					b.set(CaptureRequest.JPEG_ORIENTATION, c.captureRotation);
					b.set(CaptureRequest.JPEG_QUALITY, (byte) 92);
					setCommon(c, b);
					c.session.capture(b.build(), null, c.handler);
				} catch (Exception e) {
					c.pendingCapture = 0;
					onCapture(req, null, e.toString());
				}
			}
		});
	}

	private static Cam cam(int id) {
		synchronized (cams) {
			return cams.get(id);
		}
	}

	private static void watchLifecycle(final Cam c) {
		c.lifecycle = new Application.ActivityLifecycleCallbacks() {
			public void onActivityPaused(Activity a) {
				if (a == c.activity) {
					c.handler.post(new Runnable() { public void run() { stopDevice(c); } });
				}
			}
			public void onActivityResumed(Activity a) {
				if (a == c.activity) {
					c.handler.post(new Runnable() {
						public void run() {
							if (c.wanted && c.device == null) {
								startDevice(c);
							}
						}
					});
				}
			}
			public void onActivityCreated(Activity a, Bundle b) {}
			public void onActivityStarted(Activity a) {}
			public void onActivityStopped(Activity a) {}
			public void onActivitySaveInstanceState(Activity a, Bundle b) {}
			public void onActivityDestroyed(Activity a) {}
		};
		c.activity.getApplication().registerActivityLifecycleCallbacks(c.lifecycle);
	}

	private static void startDevice(final Cam c) {
		try {
			StreamConfigurationMap map = c.chars.get(CameraCharacteristics.SCALER_STREAM_CONFIGURATION_MAP);
			Size full = largest(map.getOutputSizes(ImageFormat.JPEG));
			Size prev = previewSize(map.getOutputSizes(ImageFormat.YUV_420_888), full, c.maxPreview);
			c.yuv = ImageReader.newInstance(prev.getWidth(), prev.getHeight(), ImageFormat.YUV_420_888, 3);
			c.yuv.setOnImageAvailableListener(new ImageReader.OnImageAvailableListener() {
				public void onImageAvailable(ImageReader r) {
					Image img = r.acquireLatestImage();
					if (img == null) {
						return;
					}
					try {
						Image.Plane[] p = img.getPlanes();
						onFrame(c.id, p[0].getBuffer(), p[1].getBuffer(), p[2].getBuffer(), img.getWidth(), img.getHeight(),
							p[0].getRowStride(), p[1].getRowStride(), p[1].getPixelStride(), rotation(c), c.front);
					} finally {
						img.close();
					}
				}
			}, c.handler);
			c.jpeg = ImageReader.newInstance(full.getWidth(), full.getHeight(), ImageFormat.JPEG, 2);
			c.jpeg.setOnImageAvailableListener(new ImageReader.OnImageAvailableListener() {
				public void onImageAvailable(ImageReader r) {
					Image img = r.acquireNextImage();
					if (img == null) {
						return;
					}
					final int req = c.pendingCapture;
					final int rot = c.captureRotation;
					c.pendingCapture = 0;
					ByteBuffer buf = img.getPlanes()[0].getBuffer();
					final byte[] data = new byte[buf.remaining()];
					buf.get(data);
					img.close();
					new Thread(new Runnable() {
						public void run() {
							try {
								onCapture(req, upright(data), "");
							} catch (Throwable e) {
								onCapture(req, null, e.toString());
							}
						}
					}).start();
				}
			}, c.handler);
			CameraManager mgr = (CameraManager) c.ctx.getSystemService(Context.CAMERA_SERVICE);
			mgr.openCamera(c.cameraId, new CameraDevice.StateCallback() {
				public void onOpened(CameraDevice d) {
					c.device = d;
					try {
						d.createCaptureSession(Arrays.asList(c.yuv.getSurface(), c.jpeg.getSurface()), new CameraCaptureSession.StateCallback() {
							public void onConfigured(CameraCaptureSession s) {
								c.session = s;
								repeat(c);
							}
							public void onConfigureFailed(CameraCaptureSession s) {
								camera_android.onError(c.id, "camera session configuration failed");
							}
						}, c.handler);
					} catch (Exception e) {
						camera_android.onError(c.id, e.toString());
					}
				}
				public void onDisconnected(CameraDevice d) {
					d.close();
					if (c.device == d) {
						c.device = null;
						c.session = null;
					}
				}
				public void onError(CameraDevice d, int error) {
					d.close();
					if (c.device == d) {
						c.device = null;
						c.session = null;
					}
					camera_android.onError(c.id, "camera error " + error);
				}
			}, c.handler);
		} catch (SecurityException e) {
			onError(c.id, "camera permission denied");
		} catch (Exception e) {
			onError(c.id, e.toString());
		}
	}

	private static void stopDevice(Cam c) {
		if (c.session != null) {
			c.session.close();
			c.session = null;
		}
		if (c.device != null) {
			c.device.close();
			c.device = null;
		}
		if (c.yuv != null) {
			c.yuv.close();
			c.yuv = null;
		}
		if (c.jpeg != null) {
			c.jpeg.close();
			c.jpeg = null;
		}
		if (c.pendingCapture != 0) {
			onCapture(c.pendingCapture, null, "camera closed");
			c.pendingCapture = 0;
		}
	}

	private static CaptureRequest.Builder previewRequest(Cam c) throws CameraAccessException {
		CaptureRequest.Builder b = c.device.createCaptureRequest(CameraDevice.TEMPLATE_PREVIEW);
		b.addTarget(c.yuv.getSurface());
		setCommon(c, b);
		return b;
	}

	private static void setCommon(Cam c, CaptureRequest.Builder b) {
		b.set(CaptureRequest.CONTROL_MODE, CameraMetadata.CONTROL_MODE_AUTO);
		b.set(CaptureRequest.CONTROL_AE_MODE, CaptureRequest.CONTROL_AE_MODE_ON);
		if (c.regions != null) {
			b.set(CaptureRequest.CONTROL_AF_MODE, CaptureRequest.CONTROL_AF_MODE_AUTO);
			b.set(CaptureRequest.CONTROL_AF_REGIONS, c.regions);
			b.set(CaptureRequest.CONTROL_AE_REGIONS, c.regions);
		} else {
			b.set(CaptureRequest.CONTROL_AF_MODE, CaptureRequest.CONTROL_AF_MODE_CONTINUOUS_PICTURE);
		}
		b.set(CaptureRequest.FLASH_MODE, c.torch ? CameraMetadata.FLASH_MODE_TORCH : CameraMetadata.FLASH_MODE_OFF);
	}

	private static void repeat(Cam c) {
		if (c.session == null) {
			return;
		}
		try {
			c.session.setRepeatingRequest(previewRequest(c).build(), null, c.handler);
		} catch (Exception e) {
			onError(c.id, e.toString());
		}
	}

	// rotation is how far to turn the sensor's image clockwise to stand
	// it upright on the display as it's held now.
	private static int rotation(Cam c) {
		Integer so = c.chars.get(CameraCharacteristics.SENSOR_ORIENTATION);
		int sensor = so == null ? 90 : so;
		int display = 0;
		WindowManager wm = (WindowManager) c.ctx.getSystemService(Context.WINDOW_SERVICE);
		if (wm != null) {
			switch (wm.getDefaultDisplay().getRotation()) {
			case Surface.ROTATION_90: display = 90; break;
			case Surface.ROTATION_180: display = 180; break;
			case Surface.ROTATION_270: display = 270; break;
			}
		}
		if (c.front) {
			return (sensor + display) % 360;
		}
		return (sensor - display + 360) % 360;
	}

	private static Size largest(Size[] sizes) {
		Size best = sizes[0];
		for (Size s : sizes) {
			if ((long) s.getWidth() * s.getHeight() > (long) best.getWidth() * best.getHeight()) {
				best = s;
			}
		}
		return best;
	}

	// previewSize picks the largest size with the full picture's aspect
	// ratio and no side over max, else the closest.
	private static Size previewSize(Size[] sizes, Size full, int max) {
		double aspect = (double) full.getWidth() / full.getHeight();
		Size best = null;
		for (Size s : sizes) {
			if (s.getWidth() > max || s.getHeight() > max) {
				continue;
			}
			if (Math.abs((double) s.getWidth() / s.getHeight() - aspect) > 0.02) {
				continue;
			}
			if (best == null || s.getWidth() > best.getWidth()) {
				best = s;
			}
		}
		if (best != null) {
			return best;
		}
		for (Size s : sizes) {
			if (s.getWidth() <= max && s.getHeight() <= max && (best == null || s.getWidth() > best.getWidth())) {
				best = s;
			}
		}
		return best != null ? best : sizes[sizes.length - 1];
	}

	// upright turns a JPEG whose EXIF orientation says it's rotated so
	// its pixels stand upright, as some cameras only tag JPEG_ORIENTATION.
	private static byte[] upright(byte[] jpeg) throws Exception {
		int degrees = 0;
		if (Build.VERSION.SDK_INT >= 24) {
			ExifInterface exif = new ExifInterface(new ByteArrayInputStream(jpeg));
			switch (exif.getAttributeInt(ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_NORMAL)) {
			case ExifInterface.ORIENTATION_ROTATE_90: degrees = 90; break;
			case ExifInterface.ORIENTATION_ROTATE_180: degrees = 180; break;
			case ExifInterface.ORIENTATION_ROTATE_270: degrees = 270; break;
			}
		}
		if (degrees == 0) {
			return jpeg;
		}
		Bitmap src = BitmapFactory.decodeByteArray(jpeg, 0, jpeg.length);
		Matrix m = new Matrix();
		m.postRotate(degrees);
		Bitmap out = Bitmap.createBitmap(src, 0, 0, src.getWidth(), src.getHeight(), m, true);
		src.recycle();
		ByteArrayOutputStream bos = new ByteArrayOutputStream();
		out.compress(Bitmap.CompressFormat.JPEG, 92, bos);
		out.recycle();
		return bos.toByteArray();
	}
}
