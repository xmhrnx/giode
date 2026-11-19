package datagrid

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles/properties"
)

type harness struct {
	r   input.Router
	g   *DataGrid
	now time.Time
}

func newHarness(g *DataGrid) *harness {
	fonts.UseGofont()
	h := &harness{g: g, now: time.Unix(1000, 0)}
	h.frame()
	h.frame()
	return h
}

func (h *harness) frame() {
	elements.BeginFrame()
	defer elements.EndFrame()
	h.now = h.now.Add(16 * time.Millisecond)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Now: h.now, Source: h.r.Source(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 400))}
	elements.SetWindowSize(image.Pt(600, 400))
	h.g.Layout(gtx)
	elements.DrainContextMenus(gtx)
	h.r.Frame(&ops)
}

func (h *harness) click(x, y float32, mods key.Modifiers) {
	p := f32.Pt(x, y)
	t := time.Duration(h.now.UnixNano())
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: p, Time: t},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: p, Modifiers: mods, Time: t},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: p, Modifiers: mods, Time: t},
	)
	h.frame()
	h.frame()
}

func (h *harness) key(name key.Name, mods key.Modifiers) {
	h.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
	h.frame()
}

// rowY is the middle of row's line on screen (header 36, rows 32).
func rowY(row int) float32 { return 36 + float32(row)*32 + 16 }

func testGrid(n int) (*DataGrid, *SliceSource[deal]) {
	src := NewSliceSource(deals(n), dealValue)
	g := New(src, []Column{
		{Key: "id", Title: "#", Width: 80, Sortable: true},
		{Key: "name", Title: "Deal", Width: 200, Sortable: true},
		{Key: "value", Title: "Value", Width: 100, Sortable: true, Align: properties.TextAlignEnd},
	})
	return g, src
}

func TestSelection(t *testing.T) {
	g, _ := testGrid(50)
	var activated []int
	g.OnActivate(func(r int) { activated = append(activated, r) })
	h := newHarness(g)
	h.click(100, rowY(2), 0)
	if fmt.Sprint(g.SelectedRows()) != "[2]" {
		t.Fatalf("click: %v", g.SelectedRows())
	}
	h.click(100, rowY(5), key.ModShift)
	if fmt.Sprint(g.SelectedRows()) != "[2 3 4 5]" {
		t.Fatalf("shift-click: %v", g.SelectedRows())
	}
	h.click(100, rowY(3), key.ModShortcut)
	if fmt.Sprint(g.SelectedRows()) != "[2 4 5]" {
		t.Fatalf("ctrl-click: %v", g.SelectedRows())
	}
	// A double click activates.
	h.click(100, rowY(7), 0)
	h.click(100, rowY(7), 0)
	if fmt.Sprint(activated) != "[7]" {
		t.Errorf("double click: %v", activated)
	}
	// Keys: down moves the selection, shift+down extends it, Enter
	// activates, Ctrl+A selects all.
	h.key(key.NameDownArrow, 0)
	if fmt.Sprint(g.SelectedRows()) != "[8]" {
		t.Fatalf("down: %v", g.SelectedRows())
	}
	h.key(key.NameDownArrow, key.ModShift)
	h.key(key.NameDownArrow, key.ModShift)
	if fmt.Sprint(g.SelectedRows()) != "[8 9 10]" {
		t.Fatalf("shift+down: %v", g.SelectedRows())
	}
	h.key(key.NameReturn, 0)
	if fmt.Sprint(activated) != "[7 10]" {
		t.Errorf("enter: %v", activated)
	}
	h.key("A", key.ModShortcut)
	if !g.AllSelected() || g.SelectedCount() != 50 {
		t.Errorf("ctrl+A: %d", g.SelectedCount())
	}
	// End with Ctrl goes to the last row and scrolls to it.
	h.key(key.NameEnd, key.ModShortcut)
	h.frame()
	if r, _ := g.Focused(); r != 49 || g.lay.last != 50 {
		t.Errorf("ctrl+end: focus %d, last row on screen %d", r, g.lay.last)
	}
}

func TestSortByHeader(t *testing.T) {
	g, src := testGrid(30)
	var sorts [][]SortKey
	g.OnSort(func(k []SortKey) { sorts = append(sorts, k) })
	h := newHarness(g)
	g.Select(0)         // deal 1
	h.click(300, 18, 0) // Value header
	if src.Len() != 30 || fmt.Sprint(g.SortKeys()) != "[{value false}]" {
		t.Fatalf("sorted %v", g.SortKeys())
	}
	for i := 1; i < src.Len(); i++ {
		if src.Value(i-1, "value").(float64) > src.Value(i, "value").(float64) {
			t.Fatal("not ascending")
		}
	}
	// The selection follows its row.
	if rows := g.SelectedRows(); len(rows) != 1 || src.Value(rows[0], "id") != 1 {
		t.Errorf("selection after sort: %v", rows)
	}
	h.click(300, 18, 0)
	h.click(100, 18, key.ModShift) // add Deal
	if fmt.Sprint(g.SortKeys()) != "[{value true} {name false}]" {
		t.Errorf("descending, then a second level: %v", g.SortKeys())
	}
	// Value is descending: a plain click clears the sort.
	h.click(300, 18, 0)
	if len(g.SortKeys()) != 0 || len(sorts) != 4 {
		t.Errorf("cleared: %v after %d sorts", g.SortKeys(), len(sorts))
	}
}

func TestCheckboxesAndCopy(t *testing.T) {
	g, _ := testGrid(5)
	g.Checkboxes(true)
	h := newHarness(g)
	h.click(20, rowY(1), 0)
	h.click(20, rowY(3), 0)
	if fmt.Sprint(g.SelectedRows()) != "[1 3]" {
		t.Fatalf("checkboxes: %v", g.SelectedRows())
	}
	lines := strings.Split(g.CopyText(), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "2\tDeal 2 with") {
		t.Errorf("copy: %q", lines)
	}
	h.click(20, 18, 0) // the header's: all
	if g.SelectedCount() != 5 {
		t.Errorf("select all: %d", g.SelectedCount())
	}
	h.click(20, 18, 0)
	if g.SelectedCount() != 0 {
		t.Errorf("none: %d", g.SelectedCount())
	}
}

func TestResizeColumn(t *testing.T) {
	g, _ := testGrid(5)
	h := newHarness(g)
	// The # column's right edge is at 80.
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(80, 18)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(80, 18)},
	)
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(140, 18)})
	h.frame()
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(140, 18)})
	h.frame()
	if w := g.Columns()[0].Width; w != 140 {
		t.Errorf("resized to %d", w)
	}
	if len(g.SortKeys()) != 0 {
		t.Error("resizing sorted")
	}
}

func TestCompare(t *testing.T) {
	for _, c := range []struct {
		a, b any
		want int
	}{
		{2, 10, -1}, {2.5, 2, 1}, {"item 2", "item 10", -1}, {"Apple", "banana", -1},
		{false, true, -1}, {nil, "a", 1}, {time.Unix(1, 0), time.Unix(2, 0), -1},
	} {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%v, %v) = %d", c.a, c.b, got)
		}
	}
}

// A frame of a 100 000 row grid lays out only what's on screen.
func BenchmarkScroll100k(b *testing.B) {
	g, _ := testGrid(100000)
	h := newHarness(g)
	b.ResetTimer()
	for i := range b.N {
		g.offY = float32((i * 997) % (100000 * 32))
		h.frame()
	}
}

// Scrolling smoothly: a few px a frame, most cells seen before.
func BenchmarkScrollSmooth100k(b *testing.B) {
	g, _ := testGrid(100000)
	h := newHarness(g)
	b.ResetTimer()
	for i := range b.N {
		g.offY = float32((i * 7) % (100000 * 32))
		h.frame()
	}
}
