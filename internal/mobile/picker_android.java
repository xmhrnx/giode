package org.giode.mobile;

import android.Manifest;
import android.app.Activity;
import android.app.Fragment;
import android.app.FragmentManager;
import android.content.ContentResolver;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.database.Cursor;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Matrix;
import android.media.ExifInterface;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.provider.MediaStore;
import android.provider.OpenableColumns;
import android.speech.RecognitionListener;
import android.speech.RecognizerIntent;
import android.speech.SpeechRecognizer;
import android.view.View;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

// picker_android runs giode's requests that need an activity result or
// a runtime permission: picking a photo, taking one with the camera,
// and dictation. A headless fragment receives the results.
public final class picker_android {
	// Implemented in Go (mobile_android.go).
	static native void onPhoto(int id, byte[] jpeg, String name, String err);
	static native void onPhotos(int id, byte[][] jpegs, String[] names, String err);
	static native void onSpeech(int id, String text, boolean done, String err);

	private static final String TAG = "giode_picker";

	// Request kinds, by request code.
	private static final Map<Integer, Request> requests = new HashMap<>();

	private static final class Request {
		int maxSide;
		boolean multi;
		int limit;
		Uri output; // the camera's output, in MediaStore
		String lang;
		SpeechRecognizer recognizer;
	}

	public static class PickerFragment extends Fragment {
		@Override
		public void onActivityResult(int code, int result, Intent data) {
			Request r = requests.remove(code);
			if (r == null) {
				return;
			}
			final Context ctx = getActivity().getApplicationContext();
			if (r.multi) {
				pickedMany(ctx, code, r, result, data);
				return;
			}
			Uri uri = r.output != null ? r.output : (data != null ? data.getData() : null);
			if (result != Activity.RESULT_OK || uri == null) {
				if (r.output != null) {
					ctx.getContentResolver().delete(r.output, null, null);
				}
				onPhoto(code, null, null, "");
				return;
			}
			final Uri src = uri;
			final Request req = r;
			new Thread(new Runnable() {
				public void run() {
					try {
						byte[] jpeg = toJPEG(ctx, src, req.maxSide);
						String name = displayName(ctx, src);
						if (req.output != null) {
							ctx.getContentResolver().delete(req.output, null, null);
						}
						onPhoto(code, jpeg, name, "");
					} catch (Exception e) {
						onPhoto(code, null, null, e.toString());
					}
				}
			}).start();
		}

		@Override
		public void onRequestPermissionsResult(int code, String[] perms, int[] results) {
			Request r = requests.get(code);
			if (r == null) {
				return;
			}
			if (results.length == 0 || results[0] != PackageManager.PERMISSION_GRANTED) {
				requests.remove(code);
				onSpeech(code, null, true, "microphone permission denied");
				return;
			}
			listen(getActivity(), code, r);
		}
	}

	private static PickerFragment fragment(Activity activity) {
		FragmentManager fm = activity.getFragmentManager();
		Fragment f = fm.findFragmentByTag(TAG);
		if (f instanceof PickerFragment && f.isAdded()) {
			return (PickerFragment) f;
		}
		PickerFragment p = new PickerFragment();
		fm.beginTransaction().add(p, TAG).commitNow();
		return p;
	}

	// pickPhoto opens the system photo picker (or, with camera, the
	// camera) and reports the photo, re-encoded as JPEG no larger than
	// maxSide pixels on its long side (0 for no limit), to onPhoto. Call
	// on the main thread.
	public static void pickPhoto(View view, int id, int maxSide, boolean camera) throws Exception {
		Activity activity = (Activity) view.getContext();
		Request r = new Request();
		r.maxSide = maxSide;
		Intent intent;
		if (camera) {
			// Without a FileProvider, the camera writes its full-size photo
			// into MediaStore; it's read back and removed from there.
			ContentValues v = new ContentValues();
			v.put(MediaStore.MediaColumns.DISPLAY_NAME, "giode-capture-" + System.currentTimeMillis() + ".jpg");
			v.put(MediaStore.MediaColumns.MIME_TYPE, "image/jpeg");
			r.output = activity.getContentResolver().insert(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, v);
			if (r.output == null) {
				throw new Exception("MediaStore refused to create the capture");
			}
			intent = new Intent(MediaStore.ACTION_IMAGE_CAPTURE);
			intent.putExtra(MediaStore.EXTRA_OUTPUT, r.output);
			intent.addFlags(Intent.FLAG_GRANT_WRITE_URI_PERMISSION);
		} else if (Build.VERSION.SDK_INT >= 33) {
			intent = new Intent(MediaStore.ACTION_PICK_IMAGES);
		} else {
			intent = new Intent(Intent.ACTION_GET_CONTENT);
			intent.setType("image/*");
			intent.addCategory(Intent.CATEGORY_OPENABLE);
		}
		requests.put(id, r);
		fragment(activity).startActivityForResult(intent, id);
	}

	// pickPhotos opens the photo picker for up to limit photos (0 for no
	// limit) and reports them to onPhotos, each upright and re-encoded as
	// JPEG no larger than maxSide. Call on the main thread.
	public static void pickPhotos(View view, int id, int maxSide, int limit) throws Exception {
		Activity activity = (Activity) view.getContext();
		Request r = new Request();
		r.maxSide = maxSide;
		r.multi = true;
		Intent intent;
		if (Build.VERSION.SDK_INT >= 33) {
			intent = new Intent(MediaStore.ACTION_PICK_IMAGES);
			int most = MediaStore.getPickImagesMaxLimit();
			intent.putExtra(MediaStore.EXTRA_PICK_IMAGES_MAX, limit > 0 && limit < most ? Math.max(limit, 2) : most);
		} else {
			intent = new Intent(Intent.ACTION_GET_CONTENT);
			intent.setType("image/*");
			intent.addCategory(Intent.CATEGORY_OPENABLE);
			intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true);
		}
		r.limit = limit;
		requests.put(id, r);
		fragment(activity).startActivityForResult(intent, id);
	}

	// pickedMany converts the photos a multiple pick returned, off the
	// main thread.
	static void pickedMany(final Context ctx, final int code, final Request r, int result, Intent data) {
		final ArrayList<Uri> uris = new ArrayList<>();
		if (result == Activity.RESULT_OK && data != null) {
			if (data.getClipData() != null) {
				for (int i = 0; i < data.getClipData().getItemCount(); i++) {
					uris.add(data.getClipData().getItemAt(i).getUri());
				}
			} else if (data.getData() != null) {
				uris.add(data.getData());
			}
		}
		if (r.limit > 0 && uris.size() > r.limit) {
			uris.subList(r.limit, uris.size()).clear();
		}
		if (uris.isEmpty()) {
			onPhotos(code, null, null, "");
			return;
		}
		new Thread(new Runnable() {
			public void run() {
				try {
					byte[][] jpegs = new byte[uris.size()][];
					String[] names = new String[uris.size()];
					for (int i = 0; i < uris.size(); i++) {
						jpegs[i] = toJPEG(ctx, uris.get(i), r.maxSide);
						names[i] = displayName(ctx, uris.get(i));
					}
					onPhotos(code, jpegs, names, "");
				} catch (Throwable e) {
					onPhotos(code, null, null, e.toString());
				}
			}
		}).start();
	}

	// toJPEG decodes the image at uri, applies its EXIF rotation, scales
	// it to maxSide and encodes it as JPEG. HEIC photos are decoded by the
	// platform (Android 9+), so they never reach the app as HEIC.
	static byte[] toJPEG(Context ctx, Uri uri, int maxSide) throws Exception {
		ContentResolver cr = ctx.getContentResolver();
		BitmapFactory.Options bounds = new BitmapFactory.Options();
		bounds.inJustDecodeBounds = true;
		try (InputStream in = cr.openInputStream(uri)) {
			BitmapFactory.decodeStream(in, null, bounds);
		}
		int sample = 1;
		if (maxSide > 0) {
			int longest = Math.max(bounds.outWidth, bounds.outHeight);
			while (longest / (sample * 2) >= maxSide) {
				sample *= 2;
			}
		}
		BitmapFactory.Options opts = new BitmapFactory.Options();
		opts.inSampleSize = sample;
		Bitmap bmp;
		try (InputStream in = cr.openInputStream(uri)) {
			bmp = BitmapFactory.decodeStream(in, null, opts);
		}
		if (bmp == null) {
			throw new Exception("can't decode the image");
		}
		Matrix m = new Matrix();
		if (Build.VERSION.SDK_INT >= 24) {
			try (InputStream in = cr.openInputStream(uri)) {
				int o = new ExifInterface(in).getAttributeInt(ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_NORMAL);
				switch (o) {
				case ExifInterface.ORIENTATION_ROTATE_90:
					m.postRotate(90);
					break;
				case ExifInterface.ORIENTATION_ROTATE_180:
					m.postRotate(180);
					break;
				case ExifInterface.ORIENTATION_ROTATE_270:
					m.postRotate(270);
					break;
				}
			}
		}
		int longest = Math.max(bmp.getWidth(), bmp.getHeight());
		if (maxSide > 0 && longest > maxSide) {
			float s = (float) maxSide / longest;
			m.postScale(s, s);
		}
		if (!m.isIdentity()) {
			bmp = Bitmap.createBitmap(bmp, 0, 0, bmp.getWidth(), bmp.getHeight(), m, true);
		}
		ByteArrayOutputStream out = new ByteArrayOutputStream();
		bmp.compress(Bitmap.CompressFormat.JPEG, 90, out);
		return out.toByteArray();
	}

	static String displayName(Context ctx, Uri uri) {
		try (Cursor c = ctx.getContentResolver().query(uri, new String[]{OpenableColumns.DISPLAY_NAME}, null, null, null)) {
			if (c != null && c.moveToFirst()) {
				return c.getString(0);
			}
		} catch (Exception e) {
			// Fall through to no name.
		}
		return "";
	}

	// dictate starts speech recognition in lang (a BCP 47 tag, or empty
	// for the device language), asking for the microphone permission
	// first if needed. Results go to onSpeech: partial ones with done
	// false, then the final one with done true. The app must import
	// gioui.org/app/permission/microphone. Call on the main thread.
	public static void dictate(View view, int id, String lang) throws Exception {
		Activity activity = (Activity) view.getContext();
		if (!SpeechRecognizer.isRecognitionAvailable(activity)) {
			throw new Exception("speech recognition isn't available on this device");
		}
		Request r = new Request();
		r.lang = lang;
		requests.put(id, r);
		if (Build.VERSION.SDK_INT >= 23 && activity.checkSelfPermission(Manifest.permission.RECORD_AUDIO) != PackageManager.PERMISSION_GRANTED) {
			fragment(activity).requestPermissions(new String[]{Manifest.permission.RECORD_AUDIO}, id);
			return;
		}
		listen(activity, id, r);
	}

	// stopDictation ends recognition early; the final result follows.
	public static void stopDictation(int id) {
		Request r = requests.get(id);
		if (r != null && r.recognizer != null) {
			r.recognizer.stopListening();
		}
	}

	static void listen(Activity activity, final int id, final Request r) {
		final SpeechRecognizer sr = SpeechRecognizer.createSpeechRecognizer(activity);
		r.recognizer = sr;
		sr.setRecognitionListener(new RecognitionListener() {
			private void finish() {
				requests.remove(id);
				sr.destroy();
			}

			@Override public void onPartialResults(Bundle b) {
				String t = first(b);
				if (t != null) {
					onSpeech(id, t, false, "");
				}
			}

			@Override public void onResults(Bundle b) {
				String t = first(b);
				finish();
				onSpeech(id, t == null ? "" : t, true, "");
			}

			@Override public void onError(int error) {
				finish();
				if (error == SpeechRecognizer.ERROR_NO_MATCH || error == SpeechRecognizer.ERROR_SPEECH_TIMEOUT) {
					onSpeech(id, "", true, "");
				} else {
					onSpeech(id, null, true, "speech recognition error " + error);
				}
			}

			@Override public void onReadyForSpeech(Bundle b) {}
			@Override public void onBeginningOfSpeech() {}
			@Override public void onRmsChanged(float v) {}
			@Override public void onBufferReceived(byte[] b) {}
			@Override public void onEndOfSpeech() {}
			@Override public void onEvent(int t, Bundle b) {}
		});
		Intent intent = new Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH);
		intent.putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL, RecognizerIntent.LANGUAGE_MODEL_FREE_FORM);
		intent.putExtra(RecognizerIntent.EXTRA_PARTIAL_RESULTS, true);
		if (r.lang != null && !r.lang.isEmpty()) {
			intent.putExtra(RecognizerIntent.EXTRA_LANGUAGE, r.lang);
		}
		sr.startListening(intent);
	}

	static String first(Bundle b) {
		ArrayList<String> list = b.getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION);
		return list == null || list.isEmpty() ? null : list.get(0);
	}
}
