package icon

import (
	"image"
	"math"
	"testing"
	"testing/fstest"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/xmhrnx/giode/styles"
)

func TestLoadSVG_ViewBoxAndPaths(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
		<path d="M0 0 L24 0 L24 24 Z"/>
	</svg>`
	d, err := LoadSVG([]byte(svg))
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	if x, y, w, h := d.ViewBox(); x != 0 || y != 0 || w != 24 || h != 24 {
		t.Errorf("viewBox = %v,%v,%v,%v, want 0,0,24,24", x, y, w, h)
	}
	if len(d.shapes) != 1 {
		t.Fatalf("shapes = %d, want 1", len(d.shapes))
	}
	if d.shapes[0].fill != "currentColor" || d.shapes[0].stroke != "none" {
		t.Errorf("default paint = fill %q stroke %q", d.shapes[0].fill, d.shapes[0].stroke)
	}
}

func TestLoadSVG_OffsetViewBox(t *testing.T) {
	d, err := LoadSVG([]byte(`<svg viewBox="10 20 100 200"><path d="M0 0 L1 1"/></svg>`))
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	if x, y, w, h := d.ViewBox(); x != 10 || y != 20 || w != 100 || h != 200 {
		t.Errorf("viewBox = %v,%v,%v,%v, want 10,20,100,200", x, y, w, h)
	}
}

func TestLoadSVG_NestedGroupsAndShapes(t *testing.T) {
	svg := `<svg viewBox="0 0 24 24">
		<g>
			<path d="M1 1 L2 2"/>
			<g><rect x="1" y="1" width="4" height="4" rx="1"/></g>
		</g>
		<circle cx="12" cy="12" r="3"/>
		<ellipse cx="5" cy="5" rx="2" ry="1"/>
		<polygon points="0,0 4,0 4,4"/>
		<polyline points="0 0 1 1 2 0"/>
		<line x1="0" y1="0" x2="5" y2="5" stroke="black"/>
		<defs><path d="M9 9 L10 10"/></defs>
		<path d="M3 3 L4 4" display="none"/>
	</svg>`
	d, err := LoadSVG([]byte(svg))
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	if len(d.shapes) != 7 {
		t.Fatalf("shapes = %d, want 7 (defs and display=none skipped)", len(d.shapes))
	}
}

func TestLoadSVG_InheritsStrokeAttributes(t *testing.T) {
	svg := `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
		<path d="M4 4 L20 20"/>
		<g style="stroke: #ff0000; stroke-width: 3"><path d="M4 20 L20 4"/></g>
	</svg>`
	d, err := LoadSVG([]byte(svg))
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	a, b := d.shapes[0], d.shapes[1]
	if a.fill != "none" || a.stroke != "currentColor" || a.strokeWidth != 2 || !a.roundCaps {
		t.Errorf("first shape = %+v", a)
	}
	if b.stroke != "#ff0000" || b.strokeWidth != 3 {
		t.Errorf("style attribute not applied: %+v", b)
	}
}

func TestLoadSVG_Transform(t *testing.T) {
	d, err := LoadSVG([]byte(`<svg viewBox="0 0 24 24"><g transform="translate(10 5) scale(2)"><path d="M1 1 L2 2"/></g></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	got := d.shapes[0].path[0].pts[0]
	if got != f32.Pt(12, 7) {
		t.Errorf("transformed point = %v, want (12,7)", got)
	}
}

func TestLoadSVG_FallsBackToWidthHeight(t *testing.T) {
	d, err := LoadSVG([]byte(`<svg width="32px" height="32px"><path d="M0 0 L1 1"/></svg>`))
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	if _, _, w, h := d.ViewBox(); w != 32 || h != 32 {
		t.Errorf("size = %v,%v, want 32,32", w, h)
	}
}

func TestLoadSVG_Errors(t *testing.T) {
	for name, svg := range map[string]string{
		"no size":   `<svg><path d="M0 0"/></svg>`,
		"malformed": `<svg viewBox="0 0 24 24"><path d="M0 0">`,
		"not svg":   `not xml`,
	} {
		if _, err := LoadSVG([]byte(svg)); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}

func TestMustLoadSVG_PanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustLoadSVG with malformed XML: want panic, got none")
		}
	}()
	MustLoadSVG([]byte(`not xml`))
}

func TestCompilePath_CompactNumbers(t *testing.T) {
	c := compilePath("M1.5.5L10-5l.5.5")
	want := []f32.Point{{X: 1.5, Y: .5}, {X: 10, Y: -5}, {X: 10.5, Y: -4.5}}
	if len(c) != 3 {
		t.Fatalf("segments = %d, want 3", len(c))
	}
	for i, w := range want {
		if c[i].pts[0] != w {
			t.Errorf("segment %d = %v, want %v", i, c[i].pts[0], w)
		}
	}
}

func TestCompilePath_ArcEndsAtTarget(t *testing.T) {
	// A half circle of radius 10 from (0,10) to (20,10), with packed
	// flags.
	c := compilePath("M0 10a10 10 0 0120 0")
	last := c[len(c)-1]
	if last.op != 'C' {
		t.Fatalf("last op = %c, want C (arcs become cubics)", last.op)
	}
	if end := last.pts[2]; end != f32.Pt(20, 10) {
		t.Errorf("arc ends at %v, want (20,10)", end)
	}
	// The arc's apex (sweep flag 1, clockwise in screen space) is above
	// the chord at y=0.
	minY := float32(math.MaxFloat32)
	for _, s := range c {
		for _, p := range s.pts {
			if s.op == 'C' {
				minY = min(minY, p.Y)
			}
		}
	}
	if minY > 1 {
		t.Errorf("arc control points never rise above the chord (min y %v)", minY)
	}
}

func TestCompilePath_ImplicitLinetoAfterMove(t *testing.T) {
	c := compilePath("M0 0 10 0 10 10z")
	ops := ""
	for _, s := range c {
		ops += string(s.op)
	}
	if ops != "MLLZ" {
		t.Errorf("ops = %q, want MLLZ", ops)
	}
}

func TestRegistry_FSAndNamed(t *testing.T) {
	fsys := fstest.MapFS{
		"icons/home.svg":         {Data: []byte(`<svg viewBox="0 0 24 24"><path d="M0 0 L1 1"/></svg>`)},
		"icons/brand/github.svg": {Data: []byte(`<svg viewBox="0 0 24 24"><path d="M0 0 L1 1"/></svg>`)},
		"icons/readme.txt":       {Data: []byte("ignored")},
		"icons/broken.svg":       {Data: []byte("<svg>")},
	}
	r := NewRegistry()
	err := r.RegisterFS(fsys, "icons")
	if err == nil {
		t.Error("RegisterFS with a broken file: want error")
	}
	names := r.Names()
	if len(names) != 2 || names[0] != "brand/github" || names[1] != "home" {
		t.Fatalf("names = %v", names)
	}

	var missing []string
	r.OnMissing = func(name string) { missing = append(missing, name) }
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(100, 100)}}
	if dims := r.Named("home").Layout(gtx); dims.Size != image.Pt(24, 24) {
		t.Errorf("icon size = %v, want 24x24", dims.Size)
	}
	if dims := r.Named("nope", styles.Styles{Width: 16}).Layout(gtx); dims.Size != image.Pt(16, 16) {
		t.Errorf("missing icon size = %v, want 16x16 placeholder", dims.Size)
	}
	r.Named("nope")
	if len(missing) != 1 {
		t.Errorf("OnMissing calls = %v, want one", missing)
	}
}

func TestIconVG_InvalidDataDoesNotPanic(t *testing.T) {
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(100, 100)}}
	IconVG([]byte("garbage")).Layout(gtx)
	if _, err := LoadIconVG([]byte("garbage")); err == nil {
		t.Error("LoadIconVG with garbage: want error")
	}
}

func TestSVG_Renders(t *testing.T) {
	gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(100, 100)}}
	dims := SVG("M0 0 L24 0 L24 24 Z", styles.Styles{Width: 32}).Layout(gtx)
	if dims.Size != image.Pt(32, 32) {
		t.Errorf("size = %v, want 32x32 (single axis makes it square)", dims.Size)
	}
}
