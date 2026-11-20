package richdoc

// Table edits. Each replaces the table block with a SetBlock operation.

// ReplaceSpans returns spans with positions [from, to) replaced by ins,
// for editing text outside a document's blocks, such as a table cell.
func ReplaceSpans(spans []Span, from, to int, ins []Span) []Span {
	n := spansLen(spans)
	from, to = max(0, min(from, n)), max(0, min(to, n))
	if to < from {
		from, to = to, from
	}
	return replaceSpans(spans, from, to, ins)
}

// SliceSpans returns the spans covering positions [from, to).
func SliceSpans(spans []Span, from, to int) []Span { return sliceSpans(spans, from, to) }

// SpansLen is the length of spans in positions.
func SpansLen(spans []Span) int { return spansLen(spans) }

// SpansText is spans' plain text.
func SpansText(spans []Span) string { return spansText(spans) }

func (s *State) table(i int) (*Block, bool) {
	if i < 0 || i >= len(s.Doc.Blocks) || s.Doc.Blocks[i].Kind != Table {
		return nil, false
	}
	return &s.Doc.Blocks[i], true
}

// SetCell sets a table cell's content.
func (s *State) SetCell(block, row, col int, spans []Span) *Tx {
	t, ok := s.table(block)
	if !ok || row < 0 || row >= len(t.Cells) || col < 0 || col >= len(t.Cells[row]) {
		return nil
	}
	s.begin("cell")
	nb := t.Clone()
	nb.Cells[row][col] = normalize(cloneSpans(spans))
	s.setBlock(block, nb)
	return s.commit()
}

// InsertRow adds an empty row to a table before row at (its row count
// to append).
func (s *State) InsertRow(block, at int) *Tx {
	t, ok := s.table(block)
	if !ok {
		return nil
	}
	cols := 0
	for _, r := range t.Cells {
		cols = max(cols, len(r))
	}
	at = max(0, min(at, len(t.Cells)))
	s.begin("table")
	nb := t.Clone()
	nb.Cells = append(nb.Cells[:at], append([][][]Span{make([][]Span, max(cols, 1))}, nb.Cells[at:]...)...)
	s.setBlock(block, nb)
	return s.commit()
}

// InsertColumn adds an empty column before column at.
func (s *State) InsertColumn(block, at int) *Tx {
	t, ok := s.table(block)
	if !ok {
		return nil
	}
	s.begin("table")
	nb := t.Clone()
	for r := range nb.Cells {
		a := max(0, min(at, len(nb.Cells[r])))
		nb.Cells[r] = append(nb.Cells[r][:a], append([][]Span{nil}, nb.Cells[r][a:]...)...)
	}
	s.setBlock(block, nb)
	return s.commit()
}

// DeleteRow removes a row; the last row removes the table.
func (s *State) DeleteRow(block, row int) *Tx {
	t, ok := s.table(block)
	if !ok || row < 0 || row >= len(t.Cells) {
		return nil
	}
	if len(t.Cells) == 1 {
		return s.DeleteBlock(block)
	}
	s.begin("table")
	nb := t.Clone()
	nb.Cells = append(nb.Cells[:row], nb.Cells[row+1:]...)
	s.setBlock(block, nb)
	return s.commit()
}

// DeleteColumn removes a column; the last column removes the table.
func (s *State) DeleteColumn(block, col int) *Tx {
	t, ok := s.table(block)
	if !ok {
		return nil
	}
	cols := 0
	for _, r := range t.Cells {
		cols = max(cols, len(r))
	}
	if cols <= 1 {
		return s.DeleteBlock(block)
	}
	s.begin("table")
	nb := t.Clone()
	for r := range nb.Cells {
		if col < len(nb.Cells[r]) {
			nb.Cells[r] = append(nb.Cells[r][:col], nb.Cells[r][col+1:]...)
		}
	}
	s.setBlock(block, nb)
	return s.commit()
}

// DeleteBlock removes block i.
func (s *State) DeleteBlock(i int) *Tx {
	if i < 0 || i >= len(s.Doc.Blocks) {
		return nil
	}
	s.begin("delete")
	s.Sel = Caret(s.removeBlocks(i, 1))
	return s.commit()
}
