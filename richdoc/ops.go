package richdoc

import (
	"errors"
	"fmt"
)

// OpKind is an operation's type.
type OpKind uint8

const (
	// Replace replaces positions [From, To) of a text block, whose
	// content was Old, with New: inserting (Old empty), deleting (New
	// empty) or formatting (the same text, other marks).
	Replace OpKind = iota
	// Split splits a text block at From; the new block after it takes
	// Attrs.
	Split
	// Join appends the block after Block to it; Attrs are the joined
	// block's and From the length of Block before, for undoing.
	Join
	// InsertBlocks inserts Blocks before Block.
	InsertBlocks
	// RemoveBlocks removes Blocks, starting at Block.
	RemoveBlocks
	// SetBlock replaces Block, which was OldBlock, with NewBlock: a
	// changed kind or attribute, or a table's content.
	SetBlock
)

var opNames = [...]string{"replace", "split", "join", "insertBlocks", "removeBlocks", "setBlock"}

func (k OpKind) String() string {
	if int(k) < len(opNames) {
		return opNames[k]
	}
	return "unknown"
}

// Op is one change to a document. Positions count runes, an atom as one.
type Op struct {
	Kind     OpKind  `json:"kind"`
	Block    int     `json:"block"`
	From     int     `json:"from,omitempty"`
	To       int     `json:"to,omitempty"`
	Old      []Span  `json:"old,omitempty"`
	New      []Span  `json:"new,omitempty"`
	Attrs    *Block  `json:"attrs,omitempty"`
	Blocks   []Block `json:"blocks,omitempty"`
	OldBlock *Block  `json:"oldBlock,omitempty"`
	NewBlock *Block  `json:"newBlock,omitempty"`
}

// IsInsert, IsDelete and IsFormat classify a Replace.
func (o Op) IsInsert() bool { return o.Kind == Replace && o.From == o.To && len(o.New) > 0 }
func (o Op) IsDelete() bool { return o.Kind == Replace && o.From < o.To && len(o.New) == 0 }
func (o Op) IsFormat() bool {
	return o.Kind == Replace && o.From < o.To && spansText(o.Old) == spansText(o.New) && spansLen(o.Old) == spansLen(o.New)
}

// ErrInvalidOp is returned for operations that don't fit the document.
var ErrInvalidOp = errors.New("richdoc: operation doesn't fit the document")

func (d *Doc) textBlock(i int) (*Block, error) {
	if i < 0 || i >= len(d.Blocks) {
		return nil, fmt.Errorf("%w: no block %d", ErrInvalidOp, i)
	}
	b := &d.Blocks[i]
	if !b.Kind.HasText() {
		return nil, fmt.Errorf("%w: block %d is a %s", ErrInvalidOp, i, b.Kind)
	}
	return b, nil
}

// Apply applies op to d.
func (d *Doc) Apply(op Op) error {
	switch op.Kind {
	case Replace:
		b, err := d.textBlock(op.Block)
		if err != nil {
			return err
		}
		if op.From < 0 || op.To < op.From || op.To > b.Len() {
			return fmt.Errorf("%w: range %d-%d of a block of %d", ErrInvalidOp, op.From, op.To, b.Len())
		}
		b.Spans = replaceSpans(b.Spans, op.From, op.To, op.New)
	case Split:
		b, err := d.textBlock(op.Block)
		if err != nil {
			return err
		}
		n := b.Len()
		if op.From < 0 || op.From > n {
			return fmt.Errorf("%w: split at %d of %d", ErrInvalidOp, op.From, n)
		}
		nb := Block{}
		if op.Attrs != nil {
			nb = op.Attrs.Attrs()
		}
		nb.Spans = sliceSpans(b.Spans, op.From, n)
		b.Spans = sliceSpans(b.Spans, 0, op.From)
		d.Blocks = append(d.Blocks, Block{})
		copy(d.Blocks[op.Block+2:], d.Blocks[op.Block+1:])
		d.Blocks[op.Block+1] = nb
	case Join:
		b, err := d.textBlock(op.Block)
		if err != nil {
			return err
		}
		next, err := d.textBlock(op.Block + 1)
		if err != nil {
			return err
		}
		b.Spans = normalize(append(cloneSpans(b.Spans), cloneSpans(next.Spans)...))
		d.Blocks = append(d.Blocks[:op.Block+1], d.Blocks[op.Block+2:]...)
	case InsertBlocks:
		if op.Block < 0 || op.Block > len(d.Blocks) {
			return fmt.Errorf("%w: insert at %d", ErrInvalidOp, op.Block)
		}
		ins := make([]Block, len(op.Blocks))
		for i, b := range op.Blocks {
			ins[i] = b.Clone()
		}
		d.Blocks = append(d.Blocks[:op.Block], append(ins, d.Blocks[op.Block:]...)...)
	case RemoveBlocks:
		if op.Block < 0 || op.Block+len(op.Blocks) > len(d.Blocks) || len(op.Blocks) >= len(d.Blocks) {
			return fmt.Errorf("%w: remove %d of %d blocks at %d", ErrInvalidOp, len(op.Blocks), len(d.Blocks), op.Block)
		}
		d.Blocks = append(d.Blocks[:op.Block], d.Blocks[op.Block+len(op.Blocks):]...)
	case SetBlock:
		if op.Block < 0 || op.Block >= len(d.Blocks) || op.NewBlock == nil {
			return fmt.Errorf("%w: set block %d", ErrInvalidOp, op.Block)
		}
		d.Blocks[op.Block] = op.NewBlock.Clone()
	default:
		return fmt.Errorf("%w: kind %d", ErrInvalidOp, op.Kind)
	}
	return nil
}

// Invert returns the operation undoing op.
func (o Op) Invert() Op {
	switch o.Kind {
	case Replace:
		return Op{Kind: Replace, Block: o.Block, From: o.From, To: o.From + spansLen(o.New), Old: o.New, New: o.Old}
	case Split:
		return Op{Kind: Join, Block: o.Block, From: o.From, Attrs: o.Attrs}
	case Join:
		return Op{Kind: Split, Block: o.Block, From: o.From, Attrs: o.Attrs}
	case InsertBlocks:
		return Op{Kind: RemoveBlocks, Block: o.Block, Blocks: o.Blocks}
	case RemoveBlocks:
		return Op{Kind: InsertBlocks, Block: o.Block, Blocks: o.Blocks}
	case SetBlock:
		return Op{Kind: SetBlock, Block: o.Block, OldBlock: o.NewBlock, NewBlock: o.OldBlock}
	}
	return o
}

// Pos is a position in a document: a block and an offset in it.
type Pos struct {
	Block  int `json:"block"`
	Offset int `json:"offset"`
}

// Less reports whether p comes before q.
func (p Pos) Less(q Pos) bool {
	return p.Block < q.Block || p.Block == q.Block && p.Offset < q.Offset
}

// Map returns where p is after op, as a cursor that stays after text
// inserted at it when after is set and before it otherwise. Collaborative
// editing maps the local selection through remote operations this way.
func (o Op) Map(p Pos, after bool) Pos {
	switch o.Kind {
	case Replace:
		if p.Block != o.Block {
			return p
		}
		added := spansLen(o.New)
		switch {
		case p.Offset < o.From || p.Offset == o.From && !after && o.From == o.To:
		case p.Offset >= o.To && (p.Offset > o.To || o.From != o.To || after):
			p.Offset += added - (o.To - o.From)
		default: // inside the replaced range
			p.Offset = o.From + added
			if !after && o.From != o.To {
				p.Offset = o.From
			}
		}
	case Split:
		if p.Block > o.Block {
			p.Block++
		} else if p.Block == o.Block && (p.Offset > o.From || p.Offset == o.From && after) {
			p.Block++
			p.Offset -= o.From
		}
	case Join:
		if p.Block == o.Block+1 {
			p.Block = o.Block
			p.Offset += o.From
		} else if p.Block > o.Block+1 {
			p.Block--
		}
	case InsertBlocks:
		if p.Block >= o.Block {
			p.Block += len(o.Blocks)
		}
	case RemoveBlocks:
		n := len(o.Blocks)
		switch {
		case p.Block >= o.Block+n:
			p.Block -= n
		case p.Block >= o.Block:
			p = Pos{Block: o.Block, Offset: 0}
		}
	}
	return p
}
