package com.google.mlkit.vision.text;

import com.google.android.gms.tasks.Task;
import com.google.mlkit.vision.common.InputImage;

public interface TextRecognizer extends java.io.Closeable {
	Task<Text> process(InputImage image);
	void close();
}
