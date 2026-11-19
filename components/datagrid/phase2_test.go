package datagrid

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles/properties"
)

type task struct {
	Title    string
	Points   float64
	Done     bool
	Priority string
}

func taskGrid(tasks []task) (*DataGrid, *SliceSource[task], *[]string) {
	src := NewSliceSource(tasks, func(t task, col string) any {
		switch col {
		case "title":
			return t.Title
		case "points":
			return t.Points
		case "done":
			return t.Done
		case "priority":
			return t.Priority
		}
		return nil
	})
	var log []string
	g := New(src, []Column{
		{Key: "title", Title: "Title", Width: 200, Editor: EditText, Sortable: true},
		{Key: "points", Title: "Points", Width: 100, Editor: EditNumber, Align: properties.TextAlignEnd, Aggregate: Sum},
		{Key: "done", Title: "Done", Width: 80, Editor: EditCheckbox},
		{Key: "priority", Title: "Priority", Width: 100, Editor: EditSelect, Options: []string{"Low", "High"}},
	})
	g.OnEdit(func(row int, col string, v any) error {
		log = append(log, fmt.Sprintf("%d %s %v", row, col, v))
		i := src.Key(row).(int)
		switch col {
		case "title":
			if v.(string) == "" {
				return errors.New("A title is needed")
			}
			tasks[i].Title = v.(string)
		case "points":
			if v == nil {
				return errors.New("Points are needed")
			}
			tasks[i].Points = v.(float64)
		case "done":
			tasks[i].Done = v.(bool)
		case "priority":
			tasks[i].Priority = v.(string)
		}
		src.SetRows(tasks)
		return nil
	})
	return g, src, &log
}

func sampleTasks() []task {
	return []task{{"Write spec", 3, false, "High"}, {"Review", 2, true, "Low"}, {"Ship", 5, false, "High"}, {"Celebrate", 1, false, "Low"}}
}

func typeText(h *harness, s string) {
	h.r.Queue(key.EditEvent{Text: s, Range: key.Range{Start: 0, End: 1 << 20}})
	h.frame()
}

func TestEditing(t *testing.T) {
	tasks := sampleTasks()
	g, src, log := taskGrid(tasks)
	h := newHarness(g)
	// Double click a title: a field over it.
	h.click(50, rowY(0), 0)
	h.click(50, rowY(0), 0)
	if !g.edit.on || g.edit.in.Text() != "Write spec" {
		t.Fatalf("editing %v, %q", g.edit.on, g.edit.in.Text())
	}
	h.frame()
	g.edit.in.SetText("")
	g.edit.in.SubmitText()
	h.frame()
	if !g.edit.on || g.edit.err == nil || g.edit.err.Error() != "A title is needed" {
		t.Fatalf("validation: on %v, err %v", g.edit.on, g.edit.err)
	}
	g.edit.in.SetText("Write the spec")
	g.edit.in.SubmitText()
	h.frame()
	if g.edit.on || src.Value(0, "title") != "Write the spec" {
		t.Fatalf("commit: on %v, %v", g.edit.on, src.Value(0, "title"))
	}
	// Enter on a number cell edits it; a bad number is refused.
	g.focus = cell{1, 1}
	h.key(key.NameReturn, 0)
	if !g.edit.on || g.edit.in.Text() != "2" {
		t.Fatalf("enter: %v %q", g.edit.on, g.edit.in.Text())
	}
	g.edit.in.SetText("two")
	g.edit.in.SubmitText()
	h.frame()
	if g.edit.err == nil {
		t.Fatal("a word taken for a number")
	}
	g.cancelEdit()
	h.frame()
	// A checkbox flips with a click.
	h.click(330, rowY(2), 0)
	if !tasks[2].Done {
		t.Errorf("checkbox: %v", *log)
	}
	// A select opens its options.
	g.focus = cell{3, 3}
	h.key(key.NameReturn, 0)
	if g.edit.options == nil || !g.edit.options.Opened() {
		t.Fatal("no options menu")
	}
	g.edit.options.Close()
	_ = elements.Text
	_ = f32.Point{}
	_ = pointer.ButtonPrimary
}

func TestGroupingAndAggregates(t *testing.T) {
	g, _, _ := taskGrid(sampleTasks())
	g.GroupBy("priority")
	h := newHarness(g)
	// High: header, 2 rows; Low: header, 2 rows.
	if g.count() != 6 {
		t.Fatalf("view rows %d", g.count())
	}
	if _, grp := g.at(0); grp == nil || grp.value != "High" || len(grp.rows) != 2 {
		t.Fatalf("first group %+v", grp)
	}
	if v := g.aggregate(1, nil); v != 11.0 {
		t.Errorf("footer sum %v", v)
	}
	_, grp := g.at(0)
	if v := g.aggregate(1, grp.rows); v != 8.0 {
		t.Errorf("High's sum %v", v)
	}
	// A click on a group header closes it.
	h.click(50, rowY(0), 0)
	if g.count() != 4 {
		t.Errorf("collapsed: %d rows", g.count())
	}
	// Space opens it again.
	h.key(key.NameSpace, 0)
	if g.count() != 6 {
		t.Errorf("reopened: %d rows", g.count())
	}
	// The footer takes a row at the bottom.
	if g.lay.footerH == 0 {
		t.Error("no footer")
	}
}

func TestColumnsLayout(t *testing.T) {
	g, _, _ := taskGrid(sampleTasks())
	var saved []GridLayout
	g.OnLayoutChange(func(l GridLayout) { saved = append(saved, l) })
	h := newHarness(g)
	// Drag Points' header past Title's middle.
	h.r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(250, 18)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(250, 18)},
	)
	h.frame()
	for _, x := range []float32{200, 120, 60} {
		h.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(x, 18)})
		h.frame()
	}
	h.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(60, 18)})
	h.frame()
	l := g.SaveLayout()
	if l.Columns[0].Key != "points" || l.Columns[1].Key != "title" || len(saved) != 1 {
		t.Fatalf("after dragging: %+v (%d saves)", l.Columns, len(saved))
	}
	if len(g.SortKeys()) != 0 {
		t.Error("a drag sorted")
	}
	// Hide a column, then restore a saved layout.
	g.SetColumnHidden("done", true)
	if len(g.order) != 3 {
		t.Errorf("hidden: %d shown", len(g.order))
	}
	b, _ := json.Marshal(g.SaveLayout())
	g2, _, _ := taskGrid(sampleTasks())
	var back GridLayout
	json.Unmarshal(b, &back)
	g2.RestoreLayout(back)
	if fmt.Sprint(g2.SaveLayout()) != fmt.Sprint(g.SaveLayout()) || len(g2.order) != 3 || g2.cols[g2.order[0]].Key != "points" {
		t.Errorf("restored %+v", g2.SaveLayout())
	}
	// The header menu offers sorting, grouping and the columns.
	g.buildHeaderMenu(1)
	elements.BeginSemantics()
	h.frame()
	_ = elements.EndSemantics()
}
