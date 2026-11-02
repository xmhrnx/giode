// Package ocr recognises text in images on the device: no network, no
// account. It's what a scanner needs for searchable PDFs, copying text
// off a page and searching inside documents.
//
//	res, err := ocr.Recognize(ctx, page, ocr.Options{})
//	fmt.Println(res.Text)
//	for _, l := range res.Lines {
//		for _, w := range l.Words {
//			pdf.InvisibleText(w.Text, w.Box) // a searchable layer
//		}
//	}
//
// iOS and macOS use Apple's Vision framework; Android uses Google's ML
// Kit text recognition, bundled into the app (about 4 MB per CPU
// architecture), so it works offline and without Google Play services.
// ML Kit's bundled model reads Latin scripts; Vision also reads Chinese,
// Japanese, Korean, Cyrillic and more (see Options.Languages). Elsewhere
// Recognize returns giode.ErrNotSupported.
//
// Android builds need the giode CLI, which adds ML Kit to the app from
// this package's android_dependencies.txt.
package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"

	"github.com/xmhrnx/giode"
)

// Options configure recognition.
type Options struct {
	// Languages are BCP 47 tags to read, most likely first: "en-US",
	// "de-DE", "zh-Hans". Empty reads the platform's default (English,
	// and other Latin-script languages). Vision only: ML Kit's bundled
	// model reads Latin scripts whatever is asked.
	Languages []string
	// Fast trades accuracy for speed (Vision's fast level), for live
	// recognition on camera frames.
	Fast bool
}

// Result is the text found in an image.
type Result struct {
	// Text is every line, in reading order, separated by newlines.
	Text  string
	Lines []Line
}

// Line is a line of text.
type Line struct {
	Text string
	// Box is the line's bounds in the image's pixels, top left origin.
	Box image.Rectangle
	// Confidence is from 0 to 1; 0 when the platform doesn't say.
	Confidence float32
	Words      []Word
}

// Word is a word of a line.
type Word struct {
	Text string
	Box  image.Rectangle
}

// The platforms answer in JSON, with boxes as [left, top, right,
// bottom] pixels.
type wireResult struct {
	Text  string     `json:"text"`
	Lines []wireLine `json:"lines"`
}

type wireLine struct {
	Text       string     `json:"text"`
	Confidence float32    `json:"confidence"`
	Box        [4]float64 `json:"box"`
	Words      []wireWord `json:"words"`
}

type wireWord struct {
	Text string     `json:"text"`
	Box  [4]float64 `json:"box"`
}

// Supported reports whether this platform can recognise text.
func Supported() bool { return supported }

// Recognize finds the text in img.
func Recognize(ctx context.Context, img image.Image, opts Options) (Result, error) {
	if !supported {
		return Result{}, giode.ErrNotSupported
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return Result{}, err
	}
	b := img.Bounds()
	res, err := RecognizeData(ctx, buf.Bytes(), opts)
	if err != nil {
		return res, err
	}
	// Boxes are relative to the encoded image, which starts at 0,0.
	if b.Min != (image.Point{}) {
		for i := range res.Lines {
			res.Lines[i].Box = res.Lines[i].Box.Add(b.Min)
			for j := range res.Lines[i].Words {
				res.Lines[i].Words[j].Box = res.Lines[i].Words[j].Box.Add(b.Min)
			}
		}
	}
	return res, nil
}

// RecognizeData finds the text in an encoded image: JPEG or PNG. Boxes
// are in its pixels as stored, ignoring any EXIF orientation.
func RecognizeData(ctx context.Context, data []byte, opts Options) (Result, error) {
	if !supported {
		return Result{}, giode.ErrNotSupported
	}
	js, err := recognize(ctx, data, opts)
	if err != nil {
		return Result{}, err
	}
	return parse(js)
}

// parse reads the platform's JSON answer.
func parse(js []byte) (Result, error) {
	var w wireResult
	if err := json.Unmarshal(js, &w); err != nil {
		return Result{}, err
	}
	rect := func(b [4]float64) image.Rectangle {
		return image.Rect(int(b[0]+0.5), int(b[1]+0.5), int(b[2]+0.5), int(b[3]+0.5))
	}
	res := Result{Text: w.Text}
	for _, wl := range w.Lines {
		l := Line{Text: wl.Text, Box: rect(wl.Box), Confidence: wl.Confidence}
		for _, ww := range wl.Words {
			l.Words = append(l.Words, Word{Text: ww.Text, Box: rect(ww.Box)})
		}
		res.Lines = append(res.Lines, l)
	}
	return res, nil
}
