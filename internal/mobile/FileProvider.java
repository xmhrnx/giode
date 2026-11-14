package org.giode.mobile;

import android.content.ContentProvider;
import android.content.ContentValues;
import android.content.Context;
import android.content.pm.ProviderInfo;
import android.database.Cursor;
import android.database.MatrixCursor;
import android.net.Uri;
import android.os.ParcelFileDescriptor;
import android.webkit.MimeTypeMap;

import java.io.File;
import java.io.FileNotFoundException;

// FileProvider hands the app's files to other apps as content:// URIs,
// read-only, for giode.App.OpenWith: Android refuses file:// URIs
// across apps. It isn't exported: another app reads a file only through
// a URI the app granted it, one file at a time. giode build declares it
// from android_manifest.xml beside this file:
//
//	<provider android:name="org.giode.mobile.FileProvider"
//		android:authorities="<app.id>.giode.files"
//		android:exported="false" android:grantUriPermissions="true"/>
public final class FileProvider extends ContentProvider {
	// uriFor returns the content URI of file, or null when the app
	// doesn't declare the provider (built by an older giode).
	public static Uri uriFor(Context ctx, File file) {
		String authority = ctx.getPackageName() + ".giode.files";
		ProviderInfo info = ctx.getPackageManager().resolveContentProvider(authority, 0);
		if (info == null || !FileProvider.class.getName().equals(info.name)) {
			return null;
		}
		Uri.Builder b = new Uri.Builder().scheme("content").authority(authority).appendPath("f");
		for (String seg : file.getAbsolutePath().split("/")) {
			if (!seg.isEmpty()) {
				b.appendPath(seg);
			}
		}
		return b.build();
	}

	private static File fileOf(Uri uri) throws FileNotFoundException {
		String p = uri.getPath();
		if (p == null || !p.startsWith("/f/")) {
			throw new FileNotFoundException(String.valueOf(uri));
		}
		return new File(p.substring(2));
	}

	static String mimeOf(String name) {
		int dot = name.lastIndexOf('.');
		if (dot >= 0) {
			String t = MimeTypeMap.getSingleton().getMimeTypeFromExtension(name.substring(dot + 1).toLowerCase());
			if (t != null) {
				return t;
			}
		}
		return "application/octet-stream";
	}

	@Override
	public boolean onCreate() {
		return true;
	}

	@Override
	public Cursor query(Uri uri, String[] projection, String selection, String[] selectionArgs, String sortOrder) {
		File f;
		try {
			f = fileOf(uri);
		} catch (FileNotFoundException e) {
			return null;
		}
		if (projection == null) {
			projection = new String[]{"_display_name", "_size"};
		}
		String[] cols = new String[projection.length];
		Object[] vals = new Object[projection.length];
		int n = 0;
		for (String col : projection) {
			// OpenableColumns.DISPLAY_NAME and SIZE.
			if ("_display_name".equals(col)) {
				cols[n] = col;
				vals[n++] = f.getName();
			} else if ("_size".equals(col)) {
				cols[n] = col;
				vals[n++] = f.length();
			}
		}
		String[] c = new String[n];
		Object[] v = new Object[n];
		System.arraycopy(cols, 0, c, 0, n);
		System.arraycopy(vals, 0, v, 0, n);
		MatrixCursor cur = new MatrixCursor(c, 1);
		cur.addRow(v);
		return cur;
	}

	@Override
	public String getType(Uri uri) {
		try {
			return mimeOf(fileOf(uri).getName());
		} catch (FileNotFoundException e) {
			return null;
		}
	}

	@Override
	public ParcelFileDescriptor openFile(Uri uri, String mode) throws FileNotFoundException {
		if (!"r".equals(mode)) {
			throw new SecurityException("giode: files are shared read-only");
		}
		return ParcelFileDescriptor.open(fileOf(uri), ParcelFileDescriptor.MODE_READ_ONLY);
	}

	@Override
	public Uri insert(Uri uri, ContentValues values) {
		throw new UnsupportedOperationException("read-only");
	}

	@Override
	public int delete(Uri uri, String selection, String[] selectionArgs) {
		return 0;
	}

	@Override
	public int update(Uri uri, ContentValues values, String selection, String[] selectionArgs) {
		return 0;
	}
}
