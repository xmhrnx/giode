package org.giode.printing;

import android.app.Activity;
import android.content.Context;
import android.os.Bundle;
import android.os.CancellationSignal;
import android.os.ParcelFileDescriptor;
import android.print.PageRange;
import android.print.PrintAttributes;
import android.print.PrintDocumentAdapter;
import android.print.PrintDocumentInfo;
import android.print.PrintManager;
import android.view.View;

import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;

// printing_android prints a PDF file through PrintManager, reporting to
// onDone when the print dialog closes.
public final class printing_android {
	static native void onDone(int id, String err);

	public static void print(final View view, final int id, final String path, final String job) {
		view.post(new Runnable() {
			public void run() {
				try {
					Activity activity = (Activity) view.getContext();
					PrintManager pm = (PrintManager) activity.getSystemService(Context.PRINT_SERVICE);
					if (pm == null) {
						onDone(id, "this device can't print");
						return;
					}
					pm.print(job, new PrintDocumentAdapter() {
						@Override
						public void onLayout(PrintAttributes oldAttrs, PrintAttributes newAttrs, CancellationSignal cancel, LayoutResultCallback cb, Bundle extras) {
							if (cancel.isCanceled()) {
								cb.onLayoutCancelled();
								return;
							}
							PrintDocumentInfo info = new PrintDocumentInfo.Builder(job + ".pdf")
								.setContentType(PrintDocumentInfo.CONTENT_TYPE_DOCUMENT)
								.build();
							cb.onLayoutFinished(info, !newAttrs.equals(oldAttrs));
						}

						@Override
						public void onWrite(PageRange[] pages, ParcelFileDescriptor dest, CancellationSignal cancel, WriteResultCallback cb) {
							try {
								InputStream in = new FileInputStream(path);
								OutputStream out = new FileOutputStream(dest.getFileDescriptor());
								try {
									byte[] buf = new byte[64 * 1024];
									int n;
									while ((n = in.read(buf)) > 0) {
										if (cancel.isCanceled()) {
											cb.onWriteCancelled();
											return;
										}
										out.write(buf, 0, n);
									}
								} finally {
									in.close();
									out.close();
								}
								cb.onWriteFinished(new PageRange[]{PageRange.ALL_PAGES});
							} catch (Exception e) {
								cb.onWriteFailed(e.toString());
							}
						}

						@Override
						public void onFinish() {
							onDone(id, "");
						}
					}, null);
				} catch (Throwable e) {
					onDone(id, e.toString());
				}
			}
		});
	}
}
