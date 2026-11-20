package datagrid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gioui.org/io/key"

	"github.com/xmhrnx/giode/styles/properties"
)

type contact struct {
	name, city string
	deals      int
}

func contacts() *SliceSource[contact] {
	return NewSliceSource([]contact{
		{"Ada Lovelace", "London", 3},
		{"Grace Hopper", "New York", 5},
		{"Alan Turing", "London, UK", 1},
		{"Edsger \"EWD\" Dijkstra", "Austin", 2},
	}, func(c contact, col string) any {
		switch col {
		case "name":
			return c.name
		case "city":
			return c.city
		case "deals":
			return c.deals
		}
		return nil
	})
}

func contactCols() []Column {
	return []Column{
		{Key: "name", Title: "Name", Filterable: true},
		{Key: "city", Title: "City", Filterable: true},
		{Key: "deals", Title: "Deals", Align: properties.TextAlignEnd},
	}
}

func TestExportCSVAndTSV(t *testing.T) {
	g := New(contacts(), contactCols())
	var b bytes.Buffer
	if err := g.Export(&b, CSV); err != nil {
		t.Fatal(err)
	}
	want := "Name,City,Deals\nAda Lovelace,London,3\nGrace Hopper,New York,5\nAlan Turing,\"London, UK\",1\n\"Edsger \"\"EWD\"\" Dijkstra\",Austin,2\n"
	if b.String() != want {
		t.Errorf("CSV:\n%s\nwant:\n%s", b.String(), want)
	}
	b.Reset()
	g.ExportRows(&b, TSV, []int{1}, false)
	if b.String() != "Grace Hopper\tNew York\t5\n" {
		t.Errorf("TSV row: %q", b.String())
	}
	g.Select(2)
	if got := g.CopyCSV(); got != "Alan Turing,\"London, UK\",1" {
		t.Errorf("CopyCSV = %q", got)
	}
}

func TestFilterRowFiltersSliceSource(t *testing.T) {
	src := contacts()
	var got [][]ColumnFilter
	g := New(src, contactCols()).FilterRow(true).OnFilter(func(f []ColumnFilter) { got = append(got, f) })
	h := newHarness(g)
	in := g.filterIn["city"]
	if in == nil {
		t.Fatal("no filter field for the city column")
	}
	if g.filterIn["deals"] != nil {
		t.Error("a field for a column that isn't Filterable")
	}
	in.Focus()
	h.frame()
	h.frame()
	h.r.Queue(key.EditEvent{Text: "lond"})
	h.frame()
	if src.Len() != 4 {
		t.Fatalf("filtered before the typing pause: %d rows", src.Len())
	}
	for range 20 {
		h.frame()
	}
	if src.Len() != 2 || len(got) != 1 || got[0][0] != (ColumnFilter{Key: "city", Text: "lond"}) {
		t.Fatalf("rows %d, OnFilter %v", src.Len(), got)
	}
	// Saved and restored with the layout.
	l := g.SaveLayout()
	js, _ := json.Marshal(l)
	if !strings.Contains(string(js), `"filters":[{"key":"city","text":"lond"}]`) {
		t.Errorf("layout JSON %s", js)
	}
	g.ClearFilters()
	if src.Len() != 4 || in.Text() != "" {
		t.Errorf("after ClearFilters: %d rows, field %q", src.Len(), in.Text())
	}
	g.RestoreLayout(l)
	if src.Len() != 2 || in.Text() != "lond" {
		t.Errorf("after RestoreLayout: %d rows, field %q", src.Len(), in.Text())
	}
	// Two filters combine.
	g.SetFilter("name", "ada")
	if src.Len() != 1 || src.Row(0).name != "Ada Lovelace" {
		t.Errorf("two filters: %d rows", src.Len())
	}
}

// A source that filters on a server gets the filters.
type serverSource struct {
	n       int
	filters []ColumnFilter
}

func (s *serverSource) Len() int                      { return s.n }
func (s *serverSource) Value(row int, col string) any { return fmt.Sprint(col, row) }
func (s *serverSource) SetFilters(f []ColumnFilter)   { s.filters = f; s.n = 3 }

func TestFiltererGetsFilters(t *testing.T) {
	src := &serverSource{n: 100}
	g := New(src, contactCols())
	h := newHarness(g)
	g.SetFilter("name", "x")
	h.frame()
	if len(src.filters) != 1 || src.filters[0].Text != "x" || g.count() != 3 {
		t.Errorf("filters %v, rows %d", src.filters, g.count())
	}
}
