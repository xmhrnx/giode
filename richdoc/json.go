package richdoc

import "encoding/json"

// jsonBlock is a block's JSON: {"type": "heading", "level": 2,
// "content": [{"text": "Hi", "marks": {"bold": true}}]}.
type jsonBlock struct {
	Type    string     `json:"type"`
	Level   int        `json:"level,omitempty"`
	Checked bool       `json:"checked,omitempty"`
	Lang    string     `json:"lang,omitempty"`
	Content []Span     `json:"content,omitempty"`
	Rows    [][][]Span `json:"rows,omitempty"`
	Src     string     `json:"src,omitempty"`
	Alt     string     `json:"alt,omitempty"`
}

// MarshalJSON writes the block with its kind by name.
func (b Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonBlock{Type: b.Kind.String(), Level: b.Level, Checked: b.Checked, Lang: b.Lang,
		Content: b.Spans, Rows: b.Cells, Src: b.Src, Alt: b.Alt})
}

// UnmarshalJSON reads a block written by MarshalJSON.
func (b *Block) UnmarshalJSON(data []byte) error {
	var j jsonBlock
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*b = Block{Kind: ParseKind(j.Type), Level: j.Level, Checked: j.Checked, Lang: j.Lang,
		Spans: normalize(j.Content), Cells: j.Rows, Src: j.Src, Alt: j.Alt}
	return nil
}

// MarshalJSON writes {"blocks": [...]}.
func (d *Doc) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks []Block `json:"blocks"`
	}{d.Blocks})
}

// UnmarshalJSON reads a document written by MarshalJSON.
func (d *Doc) UnmarshalJSON(data []byte) error {
	var j struct {
		Blocks []Block `json:"blocks"`
	}
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	d.Blocks = j.Blocks
	if len(d.Blocks) == 0 {
		d.Blocks = []Block{{}}
	}
	return nil
}

// MarshalJSON leaves out empty marks.
func (s Span) MarshalJSON() ([]byte, error) {
	type span struct {
		Text  string `json:"text,omitempty"`
		Marks *Marks `json:"marks,omitempty"`
		Atom  *Atom  `json:"atom,omitempty"`
	}
	j := span{Text: s.Text, Atom: s.Atom}
	if s.Marks != (Marks{}) {
		j.Marks = &s.Marks
	}
	return json.Marshal(j)
}
