package giode

import "testing"

func TestShareFilesMIME(t *testing.T) {
	for _, tc := range []struct {
		paths []string
		want  string
	}{
		{[]string{"a.jpg", "b.jpg"}, "image/jpeg"},
		{[]string{"a.jpg", "b.png"}, "image/*"},
		{[]string{"a.jpg", "b.pdf"}, "*/*"},
	} {
		if got := (ShareFiles{Paths: tc.paths}).mimeType(); got != tc.want {
			t.Errorf("%v: %s, want %s", tc.paths, got, tc.want)
		}
	}
}
