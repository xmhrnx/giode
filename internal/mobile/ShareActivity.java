package org.giode.mobile;

import android.app.Activity;
import android.content.ClipData;
import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;
import android.os.ParcelFileDescriptor;
import java.util.ArrayList;
import java.util.List;

// ShareActivity receives what other apps share with this one (ACTION_SEND
// and ACTION_SEND_MULTIPLE), the share target behind
// giode.Config.OnShareReceived. It opens the shared files at once, while
// the sender's read permission lasts, hands them to the app as open file
// descriptors, brings the app to the front and finishes, never showing
// itself. giode build declares it with app.share_target:
//
//	<activity android:name="org.giode.mobile.ShareActivity" android:exported="true"
//		android:excludeFromRecents="true" android:noHistory="true" android:taskAffinity=""
//		android:theme="@android:style/Theme.Translucent.NoTitleBar">
//		<intent-filter>
//			<action android:name="android.intent.action.SEND"/>
//			<category android:name="android.intent.category.DEFAULT"/>
//			<data android:mimeType="*/*"/>
//		</intent-filter>
//		... the same for android.intent.action.SEND_MULTIPLE
//	</activity>
public final class ShareActivity extends Activity {
	// Shared is one share: text, subject and files.
	static final class Shared {
		String text = "", subject = "";
		final List<Integer> fds = new ArrayList<>();
		final List<String> names = new ArrayList<>();
		final List<String> mimes = new ArrayList<>();
		final List<Long> sizes = new ArrayList<>();
	}

	private static final List<Shared> pending = new ArrayList<>();
	// listening is set once the app's Go code takes shares, so onShared
	// (in the app's native library) can be called.
	private static boolean listening;

	static native void onShared(String text, String subject, int[] fds, String[] names, String[] mimes, long[] sizes);

	@Override protected void onCreate(Bundle state) {
		super.onCreate(state);
		try {
			Shared s = collect(getIntent());
			if (s != null) {
				synchronized (pending) {
					pending.add(s);
				}
				flush();
			}
			Intent launch = getPackageManager().getLaunchIntentForPackage(getPackageName());
			if (launch != null) {
				launch.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
				startActivity(launch);
			}
		} finally {
			finish();
		}
	}

	// listen delivers the shares received so far, and later ones as they
	// arrive, to onShared. Call on the main thread.
	public static void listen() {
		listening = true;
		flush();
	}

	private static void flush() {
		if (!listening) {
			return;
		}
		List<Shared> todo;
		synchronized (pending) {
			todo = new ArrayList<>(pending);
			pending.clear();
		}
		for (Shared s : todo) {
			int n = s.fds.size();
			int[] fds = new int[n];
			long[] sizes = new long[n];
			for (int i = 0; i < n; i++) {
				fds[i] = s.fds.get(i);
				sizes[i] = s.sizes.get(i);
			}
			onShared(s.text, s.subject, fds, s.names.toArray(new String[0]), s.mimes.toArray(new String[0]), sizes);
		}
	}

	private Shared collect(Intent in) {
		if (in == null) {
			return null;
		}
		String action = in.getAction();
		if (!Intent.ACTION_SEND.equals(action) && !Intent.ACTION_SEND_MULTIPLE.equals(action)) {
			return null;
		}
		Shared s = new Shared();
		CharSequence text = in.getCharSequenceExtra(Intent.EXTRA_TEXT);
		if (text != null) {
			s.text = text.toString();
		}
		String subject = in.getStringExtra(Intent.EXTRA_SUBJECT);
		if (subject != null) {
			s.subject = subject;
		}
		List<Uri> uris = new ArrayList<>();
		if (Intent.ACTION_SEND_MULTIPLE.equals(action)) {
			ArrayList<Uri> list = in.getParcelableArrayListExtra(Intent.EXTRA_STREAM);
			if (list != null) {
				uris.addAll(list);
			}
		} else {
			Uri u = in.getParcelableExtra(Intent.EXTRA_STREAM);
			if (u != null) {
				uris.add(u);
			}
		}
		// Some apps put the files only in the clip data.
		ClipData clip = in.getClipData();
		if (uris.isEmpty() && clip != null) {
			for (int i = 0; i < clip.getItemCount(); i++) {
				Uri u = clip.getItemAt(i).getUri();
				if (u != null) {
					uris.add(u);
				}
			}
		}
		for (Uri u : uris) {
			try {
				ParcelFileDescriptor pfd = getContentResolver().openFileDescriptor(u, "r");
				if (pfd == null) {
					continue;
				}
				long size = pfd.getStatSize();
				String name = picker_android.displayName(this, u);
				if (name.isEmpty()) {
					name = u.getLastPathSegment() != null ? u.getLastPathSegment() : "";
				}
				String mime = getContentResolver().getType(u);
				s.fds.add(pfd.detachFd());
				s.names.add(name);
				s.mimes.add(mime != null ? mime : "");
				s.sizes.add(size);
			} catch (Exception e) {
				// Skip a file that can't be opened.
			}
		}
		if (s.text.isEmpty() && s.subject.isEmpty() && s.fds.isEmpty()) {
			return null;
		}
		return s;
	}
}
