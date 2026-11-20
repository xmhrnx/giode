package datagrid

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Source supplies a grid's rows. Value returns row's value in the
// column with key col; nil shows an empty cell, or a placeholder while
// a Loader loads it.
type Source interface {
	Len() int
	Value(row int, col string) any
}

// Keyer gives rows identities, so a selection follows its rows through
// a sort or a reload.
type Keyer interface {
	Key(row int) any
}

// Sorter sorts the rows by keys, in place: the grid calls it when the
// sort changes.
type Sorter interface {
	Sort(keys []SortKey)
}

// Loader loads rows on demand, for a server-side source: Loaded reports
// whether a row is there, and Load asks for rows from to to (exclusive)
// - on screen and a page around them - each frame some are missing.
// Load should start the fetch and return; call the app's Invalidate
// when it's done. Status reports a load in progress or failed, shown
// over the grid.
type Loader interface {
	Loaded(row int) bool
	Load(from, to int)
	Status() (loading bool, err error)
}

// SliceSource is a Source over a slice, sorting and filtering it in
// memory.
type SliceSource[T any] struct {
	all   []T
	rows  []int // indices into all, filtered and sorted
	value func(T, string) any
	keep  func(T) bool
	cols  []ColumnFilter
	sort  []SortKey
}

// NewSliceSource returns a source over rows, with value giving a row's
// value for a column key.
func NewSliceSource[T any](rows []T, value func(row T, col string) any) *SliceSource[T] {
	s := &SliceSource[T]{value: value}
	s.SetRows(rows)
	return s
}

// SetRows replaces the rows, keeping the filter and sort.
func (s *SliceSource[T]) SetRows(rows []T) {
	s.all = rows
	s.refresh()
}

// Filter keeps the rows keep accepts; nil keeps all.
func (s *SliceSource[T]) Filter(keep func(T) bool) {
	s.keep = keep
	s.refresh()
}

// SetFilters keeps the rows whose text (as the grid shows values without
// a Column.Format) contains each filter's text in its column, ignoring
// case, along with Filter's predicate. The grid calls it (Filterer).
func (s *SliceSource[T]) SetFilters(filters []ColumnFilter) {
	s.cols = s.cols[:0]
	for _, f := range filters {
		if f.Text != "" {
			s.cols = append(s.cols, ColumnFilter{Key: f.Key, Text: strings.ToLower(f.Text)})
		}
	}
	s.refresh()
}

// matches reports whether r passes the column filters.
func (s *SliceSource[T]) matches(r T) bool {
	for _, f := range s.cols {
		if !strings.Contains(strings.ToLower(text(s.value(r, f.Key))), f.Text) {
			return false
		}
	}
	return true
}

// Row returns the item shown at row.
func (s *SliceSource[T]) Row(row int) T { return s.all[s.rows[row]] }

func (s *SliceSource[T]) Len() int { return len(s.rows) }

func (s *SliceSource[T]) Value(row int, col string) any {
	return s.value(s.all[s.rows[row]], col)
}

// Key is the row's index in the slice given, so a selection follows
// rows through sorting and filtering.
func (s *SliceSource[T]) Key(row int) any { return s.rows[row] }

func (s *SliceSource[T]) Sort(keys []SortKey) {
	s.sort = append([]SortKey(nil), keys...)
	s.refresh()
}

func (s *SliceSource[T]) refresh() {
	s.rows = s.rows[:0]
	for i, r := range s.all {
		if (s.keep == nil || s.keep(r)) && s.matches(r) {
			s.rows = append(s.rows, i)
		}
	}
	if len(s.sort) == 0 {
		return
	}
	slices.SortStableFunc(s.rows, func(a, b int) int {
		for _, k := range s.sort {
			c := Compare(s.value(s.all[a], k.Key), s.value(s.all[b], k.Key))
			if k.Descending {
				c = -c
			}
			if c != 0 {
				return c
			}
		}
		return 0
	})
}

// Compare orders two cell values: numbers by value, times by time,
// strings case-insensitively with numbers in them by value ("item 2"
// before "item 10"), false before true, nil last.
func Compare(a, b any) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	if x, ok := number(a); ok {
		if y, ok := number(b); ok {
			return cmp.Compare(x, y)
		}
	}
	switch x := a.(type) {
	case time.Time:
		if y, ok := b.(time.Time); ok {
			return x.Compare(y)
		}
	case bool:
		if y, ok := b.(bool); ok {
			switch {
			case x == y:
				return 0
			case !x:
				return -1
			}
			return 1
		}
	}
	return natural(text(a), text(b))
}

func number(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

// natural compares strings case-insensitively, runs of digits by
// value.
func natural(a, b string) int {
	a, b = strings.ToLower(a), strings.ToLower(b)
	for a != "" && b != "" {
		if isDigit(a[0]) && isDigit(b[0]) {
			i, j := digits(a), digits(b)
			na, nb := strings.TrimLeft(a[:i], "0"), strings.TrimLeft(b[:j], "0")
			if c := cmp.Compare(len(na), len(nb)); c != 0 {
				return c
			}
			if c := strings.Compare(na, nb); c != 0 {
				return c
			}
			a, b = a[i:], b[j:]
			continue
		}
		if a[0] != b[0] {
			return cmp.Compare(a[0], b[0])
		}
		a, b = a[1:], b[1:]
	}
	return cmp.Compare(len(a), len(b))
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func digits(s string) int {
	i := 0
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	return i
}

// text is a value's default text.
func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case time.Time:
		if x.IsZero() {
			return ""
		}
		if x.Hour() == 0 && x.Minute() == 0 && x.Second() == 0 {
			return x.Format("2006-01-02")
		}
		return x.Format("2006-01-02 15:04")
	case float32:
		return fmt.Sprintf("%g", x)
	case float64:
		return fmt.Sprintf("%g", x)
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprint(v)
}
