// Command scanner is a small document scanner built on Giode's camera,
// OCR, printing and biometric packages. It's also what CI builds
// for Android and iOS, so those packages' native code compiles there.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/biometric"
	"github.com/xmhrnx/giode/camera"
	"github.com/xmhrnx/giode/components/zoom"
	"github.com/xmhrnx/giode/ocr"
	"github.com/xmhrnx/giode/printing"
)

// page is a scanned or imported page.
type page struct {
	path string
	img  image.Image
}

type scanner struct {
	app        *giode.App
	cam        *camera.Camera
	pages      []page
	status     string
	brightness float64 // of the last analysis frame, 0 to 1
	locked     bool
	viewer     *zoom.Zoom
	buttons    map[string]giode.Element
}

func main() {
	giode.UseGofont()
	giode.SetMode(giode.ModeDark)
	s := &scanner{locked: biometric.Available() != biometric.None, buttons: map[string]giode.Element{}}
	s.app = giode.New(giode.Config{Title: "Scanner", Size: image.Pt(420, 760), Resizable: true})
	s.viewer = zoom.New(func() giode.Element {
		if len(s.pages) == 0 {
			return giode.Text("No pages yet", giode.Styles{Color: "theme.muted"})
		}
		return giode.Image(s.pages[len(s.pages)-1].img, giode.Styles{Width: -1})
	})
	if err := s.app.Run(s.view); err != nil {
		log.Fatal(err)
	}
}

// button returns a button made once per label, so its state persists.
func (s *scanner) button(label string, fn func()) giode.Element {
	b, ok := s.buttons[label]
	if !ok {
		b = giode.Button(label, giode.Styles{Background: "theme.primary", Color: "white", BorderRadius: 8}).OnClick(func() { go fn() })
		s.buttons[label] = b
	}
	return b
}

func (s *scanner) setStatus(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.app.Post(func() { s.status = msg })
}

func (s *scanner) view() giode.Element {
	if s.locked {
		return giode.Box(giode.Styles{Width: -1, Height: -1, Align: giode.AlignCenter, Justify: giode.JustifyCenter, FlexDirection: giode.FlexDirectionColumn, Gap: 12},
			giode.Text("Locked", giode.Styles{FontSize: 22}),
			s.button("Unlock", s.unlock),
			giode.Text(s.status, giode.Styles{Color: "theme.muted"}),
		)
	}
	var top giode.Element
	if s.cam != nil {
		top = s.cam.Preview(camera.PreviewOptions{Fill: true, Overlay: s.drawGuide})
	} else {
		top = s.viewer
	}
	return giode.Box(giode.Styles{Width: -1, Height: -1, FlexDirection: giode.FlexDirectionColumn, Gap: 8, Padding: giode.UniformInset(8)},
		giode.Box(giode.Styles{Width: -1, FlexGrow: 1}, top),
		giode.Text(fmt.Sprintf("%d pages · %s", len(s.pages), s.status), giode.Styles{Color: "theme.muted"}),
		giode.Row(giode.Styles{Gap: 8},
			s.button("Camera", s.toggleCamera),
			s.button("Capture", s.capture),
			s.button("Torch", s.torch),
			s.button("Photos", s.importPhotos),
		),
		giode.Row(giode.Styles{Gap: 8},
			s.button("Text", s.recognize),
			s.button("Share", s.share),
			s.button("Save", s.save),
			s.button("Print", s.print),
		),
	)
}

func (s *scanner) unlock() {
	err := biometric.Authenticate(context.Background(), s.app, biometric.Options{Reason: "Unlock your scans", AllowPasscode: true})
	s.app.Post(func() {
		if err == nil {
			s.locked = false
			s.status = ""
		} else {
			s.status = err.Error()
		}
	})
}

// drawGuide frames the middle of the picture, brighter as the page gets
// brighter: a stand-in for a page outline from edge detection.
func (s *scanner) drawGuide(gtx layout.Context, toView func(f32.Point) f32.Point) {
	tl, br := toView(f32.Pt(0.1, 0.1)), toView(f32.Pt(0.9, 0.9))
	r := image.Rect(int(tl.X), int(tl.Y), int(br.X), int(br.Y))
	a := uint8(80 + 175*s.brightness)
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: a},
		clip.Stroke{Path: clip.Rect(r).Path(), Width: float32(gtx.Dp(3))}.Op())
}

func (s *scanner) toggleCamera() {
	if s.cam != nil {
		cam := s.cam
		s.app.Post(func() { s.cam = nil })
		cam.Close()
		return
	}
	cam, err := camera.Open(context.Background(), s.app, camera.Options{
		OnFrame: func(f camera.Frame) {
			var sum int
			for _, v := range f.Image.Pix {
				sum += int(v)
			}
			b := float64(sum) / float64(len(f.Image.Pix)) / 255
			s.app.Post(func() { s.brightness = b })
		},
	})
	if err != nil {
		s.setStatus("camera: %v", err)
		return
	}
	s.app.Post(func() { s.cam = cam })
}

func (s *scanner) capture() {
	cam := s.cam
	if cam == nil {
		s.setStatus("open the camera first")
		return
	}
	data, err := cam.Capture(context.Background())
	if err != nil {
		s.setStatus("capture: %v", err)
		return
	}
	s.addJPEG(data, "")
}

func (s *scanner) torch() {
	if cam := s.cam; cam != nil && cam.HasTorch() {
		if err := cam.Torch(!cam.TorchOn()); err != nil {
			s.setStatus("torch: %v", err)
		}
	}
}

// addJPEG saves a page and shows it.
func (s *scanner) addJPEG(data []byte, name string) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		s.setStatus("decode: %v", err)
		return
	}
	if name == "" {
		name = fmt.Sprintf("page-%d.jpg", time.Now().UnixNano())
	}
	dir := filepath.Join(os.TempDir(), "giode-scanner")
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, filepath.Base(name))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		s.setStatus("save: %v", err)
		return
	}
	s.app.Post(func() {
		s.pages = append(s.pages, page{path: path, img: img})
		s.status = "added " + filepath.Base(path)
	})
}

func (s *scanner) importPhotos() {
	files, err := s.app.PickPhotos(context.Background(), giode.PhotoOptions{MaxSide: 2400}, 0)
	if err != nil {
		s.setStatus("photos: %v", err)
		return
	}
	for _, f := range files {
		data, err := io.ReadAll(f)
		f.Close()
		if err == nil {
			s.addJPEG(data, f.Name)
		}
	}
}

func (s *scanner) recognize() {
	if len(s.pages) == 0 {
		return
	}
	res, err := ocr.Recognize(context.Background(), s.pages[len(s.pages)-1].img, ocr.Options{})
	if err != nil {
		s.setStatus("text: %v", err)
		return
	}
	s.app.SetClipboard(res.Text)
	s.setStatus("copied %d lines of text", len(res.Lines))
}

func (s *scanner) paths() []string {
	var ps []string
	for _, p := range s.pages {
		ps = append(ps, p.path)
	}
	return ps
}

func (s *scanner) share() {
	if err := s.app.Share(context.Background(), giode.ShareFiles{Paths: s.paths(), Title: "Scans"}); err != nil {
		s.setStatus("share: %v", err)
	}
}

func (s *scanner) save() {
	for _, p := range s.paths() {
		if err := s.app.SaveToPhotos(context.Background(), p, "Scanner"); err != nil {
			s.setStatus("save: %v", err)
			return
		}
	}
	s.setStatus("saved to Photos")
}

func (s *scanner) print() {
	if len(s.pages) == 0 {
		return
	}
	target := s.pages[len(s.pages)-1].path
	if err := printing.Print(context.Background(), s.app, target, printing.Options{JobName: "Scan"}); err != nil {
		s.setStatus("print: %v", err)
	}
}
