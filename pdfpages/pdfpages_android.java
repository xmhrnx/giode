package org.giode.pdfpages;

import android.graphics.Bitmap;
import android.graphics.Color;
import android.graphics.pdf.PdfRenderer;
import android.os.ParcelFileDescriptor;

import java.io.File;
import java.nio.ByteBuffer;
import java.util.HashMap;
import java.util.Map;

// pdfpages_android renders PDF pages with PdfRenderer, which keeps one
// page open at a time; Go serialises the calls per document.
public final class pdfpages_android {
	private static final Map<Integer, PdfRenderer> docs = new HashMap<>();
	private static int next;

	// open returns the document's id.
	public static synchronized int open(String path) throws Exception {
		ParcelFileDescriptor fd = ParcelFileDescriptor.open(new File(path), ParcelFileDescriptor.MODE_READ_ONLY);
		PdfRenderer r = new PdfRenderer(fd);
		next++;
		docs.put(next, r);
		return next;
	}

	public static int pageCount(int id) throws Exception {
		return doc(id).getPageCount();
	}

	// pageSize returns the page's width (high 16 bits) and height (low
	// 16 bits) in points.
	public static int pageSize(int id, int page) throws Exception {
		PdfRenderer r = doc(id);
		synchronized (r) {
			PdfRenderer.Page p = r.openPage(page);
			try {
				return (Math.min(p.getWidth(), 0x7fff) << 16) | Math.min(p.getHeight(), 0xffff);
			} finally {
				p.close();
			}
		}
	}

	// render returns the page as RGBA bytes, w by h pixels.
	public static byte[] render(int id, int page, int w, int h) throws Exception {
		PdfRenderer r = doc(id);
		synchronized (r) {
			PdfRenderer.Page p = r.openPage(page);
			Bitmap bmp = Bitmap.createBitmap(w, h, Bitmap.Config.ARGB_8888);
			try {
				bmp.eraseColor(Color.WHITE);
				p.render(bmp, null, null, PdfRenderer.Page.RENDER_MODE_FOR_DISPLAY);
				ByteBuffer buf = ByteBuffer.allocate(w * h * 4);
				bmp.copyPixelsToBuffer(buf);
				return buf.array();
			} finally {
				bmp.recycle();
				p.close();
			}
		}
	}

	public static synchronized void close(int id) {
		PdfRenderer r = docs.remove(id);
		if (r != null) {
			r.close();
		}
	}

	private static synchronized PdfRenderer doc(int id) throws Exception {
		PdfRenderer r = docs.get(id);
		if (r == null) {
			throw new Exception("document closed");
		}
		return r;
	}
}
