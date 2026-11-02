package org.giode.mobile;

import android.app.Activity;
import android.app.Application;
import android.content.ClipData;
import android.content.ContentResolver;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.view.View;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.util.ArrayList;
import java.io.InputStream;
import java.io.OutputStream;

// mobile_android is giode's Android helper: the system share sheet and
// activity lifecycle callbacks, which Gio doesn't expose.
public final class mobile_android {
	// Implemented in Go (mobile_android.go).
	static native void onBackground();
	static native void onForeground();

	private static boolean watching;

	// watchLifecycle reports the app going to the background (no
	// activity started) and back to the foreground.
	public static synchronized void watchLifecycle(Context ctx) {
		if (watching) {
			return;
		}
		watching = true;
		Application app = (Application) ctx.getApplicationContext();
		app.registerActivityLifecycleCallbacks(new Application.ActivityLifecycleCallbacks() {
			private int started = 0;

			@Override public void onActivityStarted(Activity a) {
				if (started++ == 0) {
					onForeground();
				}
			}

			@Override public void onActivityStopped(Activity a) {
				// The activity may have started before these callbacks
				// were registered, so a stop with no recorded start
				// still means the app left the screen.
				if (started > 0) {
					started--;
				}
				if (started == 0) {
					onBackground();
				}
			}

			@Override public void onActivityCreated(Activity a, Bundle b) {}
			@Override public void onActivityResumed(Activity a) {}
			@Override public void onActivityPaused(Activity a) {}
			@Override public void onActivitySaveInstanceState(Activity a, Bundle b) {}
			@Override public void onActivityDestroyed(Activity a) {}
		});
	}

	// shareText opens the share sheet for text. Call on the main thread.
	public static void shareText(View view, String text, String subject, String title) {
		Intent send = new Intent(Intent.ACTION_SEND);
		send.setType("text/plain");
		send.putExtra(Intent.EXTRA_TEXT, text);
		if (subject != null && !subject.isEmpty()) {
			send.putExtra(Intent.EXTRA_SUBJECT, subject);
		}
		start(view, send, title);
	}

	// shareURI opens the share sheet for content published with
	// publish. Call on the main thread.
	public static void shareURI(View view, String uri, String mime, String title) {
		Uri u = Uri.parse(uri);
		Intent send = new Intent(Intent.ACTION_SEND);
		send.setType(mime);
		send.putExtra(Intent.EXTRA_STREAM, u);
		send.setClipData(ClipData.newRawUri("", u));
		send.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
		start(view, send, title);
	}

	// shareURIs opens the share sheet for several items published with
	// publish, uris separated by newlines. Call on the main thread.
	public static void shareURIs(View view, String uris, String mime, String title) {
		ArrayList<Uri> list = new ArrayList<>();
		ClipData clip = null;
		for (String s : uris.split("\n")) {
			if (s.isEmpty()) {
				continue;
			}
			Uri u = Uri.parse(s);
			list.add(u);
			if (clip == null) {
				clip = ClipData.newRawUri("", u);
			} else {
				clip.addItem(new ClipData.Item(u));
			}
		}
		Intent send = new Intent(Intent.ACTION_SEND_MULTIPLE);
		send.setType(mime);
		send.putParcelableArrayListExtra(Intent.EXTRA_STREAM, list);
		if (clip != null) {
			send.setClipData(clip);
		}
		send.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
		start(view, send, title);
	}

	private static void start(View view, Intent send, String title) {
		Context ctx = view.getContext();
		Intent chooser = Intent.createChooser(send, title == null || title.isEmpty() ? null : title);
		if (!(ctx instanceof Activity)) {
			chooser.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
		}
		ctx.startActivity(chooser);
	}

	// publish copies the file at path into the shared Downloads
	// collection and returns its content URI. Apps built with gogio
	// can't declare a FileProvider, and other apps can't read this
	// app's private files, so MediaStore is the provider that lets a
	// receiving app open the file. Android 10 (API 29) or later. It
	// copies the whole file: call it off the main thread.
	public static String publish(Context ctx, String path, String mime) throws IOException {
		if (Build.VERSION.SDK_INT < 29) {
			throw new IOException("sharing files needs Android 10 or later");
		}
		return copyInto(ctx, "content://media/external/downloads", "Download", path, mime);
	}

	// saveToGallery copies a picture or video into the shared Pictures
	// (or Movies) collection, in an album folder, where the gallery and
	// photo apps show it. Android 10 or later; off the main thread.
	public static String saveToGallery(Context ctx, String path, String mime, String album) throws IOException {
		if (Build.VERSION.SDK_INT < 29) {
			throw new IOException("saving to the gallery needs Android 10 or later");
		}
		boolean video = mime.startsWith("video/");
		String dir = video ? "Movies" : "Pictures";
		if (album != null && !album.isEmpty()) {
			dir += "/" + album;
		}
		return copyInto(ctx, video ? "content://media/external/video/media" : "content://media/external/images/media", dir, path, mime);
	}

	private static String copyInto(Context ctx, String collection, String relPath, String path, String mime) throws IOException {
		File src = new File(path);
		ContentValues v = new ContentValues();
		// MediaStore.MediaColumns names, spelled out: they're newer
		// than the SDK this helper compiles against.
		v.put("_display_name", src.getName());
		v.put("mime_type", mime);
		v.put("relative_path", relPath);
		v.put("is_pending", 1);
		ContentResolver resolver = ctx.getContentResolver();
		Uri uri = resolver.insert(Uri.parse(collection), v);
		if (uri == null) {
			throw new IOException("MediaStore refused to create " + src.getName());
		}
		try {
			InputStream in = new FileInputStream(src);
			try {
				OutputStream out = resolver.openOutputStream(uri);
				if (out == null) {
					throw new IOException("can't write " + uri);
				}
				try {
					byte[] buf = new byte[64 * 1024];
					int n;
					while ((n = in.read(buf)) > 0) {
						out.write(buf, 0, n);
					}
				} finally {
					out.close();
				}
			} finally {
				in.close();
			}
		} catch (IOException e) {
			resolver.delete(uri, null, null);
			throw e;
		}
		ContentValues done = new ContentValues();
		done.put("is_pending", 0);
		resolver.update(uri, done, null, null);
		return uri.toString();
	}
}
