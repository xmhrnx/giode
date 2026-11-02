package com.google.mlkit.vision.text;

import android.graphics.Rect;
import java.util.List;

public class Text {
	public String getText() { throw new UnsupportedOperationException(); }
	public List<TextBlock> getTextBlocks() { throw new UnsupportedOperationException(); }

	public static class TextBlock {
		public String getText() { throw new UnsupportedOperationException(); }
		public List<Line> getLines() { throw new UnsupportedOperationException(); }
	}

	public static class Line {
		public String getText() { throw new UnsupportedOperationException(); }
		public Rect getBoundingBox() { throw new UnsupportedOperationException(); }
		public float getConfidence() { throw new UnsupportedOperationException(); }
		public List<Element> getElements() { throw new UnsupportedOperationException(); }
	}

	public static class Element {
		public String getText() { throw new UnsupportedOperationException(); }
		public Rect getBoundingBox() { throw new UnsupportedOperationException(); }
	}
}
