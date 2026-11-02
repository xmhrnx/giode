package ocr

import (
	"image"
	"testing"
)

func TestParse(t *testing.T) {
	js := `{"text":"Hello world\nSecond","lines":[
		{"text":"Hello world","confidence":0.9,"box":[10,20,110.6,40],"words":[
			{"text":"Hello","box":[10,20,55,40]},{"text":"world","box":[60,20,110,40]}]},
		{"text":"Second","confidence":0,"box":[10,50,80,70],"words":[]}]}`
	res, err := parse([]byte(js))
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "Hello world\nSecond" || len(res.Lines) != 2 {
		t.Fatalf("parsed %+v", res)
	}
	l := res.Lines[0]
	if l.Box != image.Rect(10, 20, 111, 40) || l.Confidence != 0.9 || len(l.Words) != 2 || l.Words[1].Box != image.Rect(60, 20, 110, 40) {
		t.Errorf("first line %+v", l)
	}
}
