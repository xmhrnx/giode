package org.giode.ocr;

import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Rect;

import com.google.android.gms.tasks.OnCompleteListener;
import com.google.android.gms.tasks.Task;
import com.google.mlkit.vision.common.InputImage;
import com.google.mlkit.vision.text.Text;
import com.google.mlkit.vision.text.TextRecognition;
import com.google.mlkit.vision.text.TextRecognizer;
import com.google.mlkit.vision.text.latin.TextRecognizerOptions;

import org.json.JSONArray;
import org.json.JSONObject;

// ocr_android recognises text with ML Kit and answers onResult with the
// lines and words as JSON (see ocr.go's wire types).
public final class ocr_android {
	// Implemented in Go (ocr_android.go).
	static native void onResult(int id, String json, String err);

	private static TextRecognizer recognizer;

	private static synchronized TextRecognizer recognizer() {
		if (recognizer == null) {
			recognizer = TextRecognition.getClient(TextRecognizerOptions.DEFAULT_OPTIONS);
		}
		return recognizer;
	}

	public static void recognize(final int id, byte[] data) {
		final Bitmap bmp = BitmapFactory.decodeByteArray(data, 0, data.length);
		if (bmp == null) {
			onResult(id, null, "can't decode the image");
			return;
		}
		Task<Text> task;
		try {
			task = recognizer().process(InputImage.fromBitmap(bmp, 0));
		} catch (Throwable e) {
			bmp.recycle();
			onResult(id, null, e.toString());
			return;
		}
		task.addOnCompleteListener(new OnCompleteListener<Text>() {
			public void onComplete(Task<Text> t) {
				bmp.recycle();
				if (!t.isSuccessful()) {
					Exception e = t.getException();
					onResult(id, null, e != null ? e.toString() : "recognition failed");
					return;
				}
				try {
					onResult(id, toJSON(t.getResult()), "");
				} catch (Throwable e) {
					onResult(id, null, e.toString());
				}
			}
		});
	}

	private static String toJSON(Text text) throws Exception {
		JSONObject res = new JSONObject();
		JSONArray lines = new JSONArray();
		StringBuilder all = new StringBuilder();
		for (Text.TextBlock b : text.getTextBlocks()) {
			for (Text.Line l : b.getLines()) {
				JSONObject line = new JSONObject();
				line.put("text", l.getText());
				line.put("box", box(l.getBoundingBox()));
				float conf = 0;
				try {
					conf = l.getConfidence();
				} catch (Throwable e) {
					// Older ML Kit versions don't report it.
				}
				line.put("confidence", (double) conf);
				JSONArray words = new JSONArray();
				for (Text.Element e : l.getElements()) {
					JSONObject w = new JSONObject();
					w.put("text", e.getText());
					w.put("box", box(e.getBoundingBox()));
					words.put(w);
				}
				line.put("words", words);
				lines.put(line);
				if (all.length() > 0) {
					all.append('\n');
				}
				all.append(l.getText());
			}
		}
		res.put("text", all.toString());
		res.put("lines", lines);
		return res.toString();
	}

	private static JSONArray box(Rect r) throws Exception {
		JSONArray a = new JSONArray();
		if (r == null) {
			r = new Rect();
		}
		a.put(r.left);
		a.put(r.top);
		a.put(r.right);
		a.put(r.bottom);
		return a;
	}
}
