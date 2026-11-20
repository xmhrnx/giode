package richdoc

import "time"

// History is a document's undo and redo stacks. Typing and deleting in a
// row, within a second and without moving the caret elsewhere, undo
// together; an input rule undoes on its own first, leaving what was
// typed.
type History struct {
	// Limit caps the undo steps kept; 0 keeps 500.
	Limit int

	undo, redo []entry
	last       time.Time
	now        func() time.Time
}

type entry struct {
	ops           []Op
	before, after Selection
	label         string
}

func (h *History) clock() time.Time {
	if h.now != nil {
		return h.now()
	}
	return time.Now()
}

// Record adds a transaction to the history and clears the redo stack.
func (h *History) Record(tx *Tx) {
	if tx == nil || len(tx.Ops) == 0 {
		return
	}
	h.redo = nil
	now := h.clock()
	ops, before, after := tx.Ops, tx.Before, tx.After
	var rule *entry
	if tx.rule >= 0 && tx.rule < len(ops) {
		rule = &entry{ops: ops[tx.rule:], before: tx.mid, after: after, label: "rule"}
		ops, after = ops[:tx.rule], tx.mid
	}
	if len(ops) > 0 {
		e := entry{ops: ops, before: before, after: after, label: tx.Label}
		if n := len(h.undo); n > 0 && (e.label == "type" || e.label == "delete") {
			prev := &h.undo[n-1]
			if prev.label == e.label && prev.after == e.before && now.Sub(h.last) < time.Second {
				prev.ops = append(prev.ops, e.ops...)
				prev.after = e.after
				e.ops = nil
			}
		}
		if e.ops != nil {
			h.undo = append(h.undo, e)
		}
	}
	if rule != nil {
		h.undo = append(h.undo, *rule)
	}
	h.last = now
	if limit := h.Limit; limit > 0 && len(h.undo) > limit || limit == 0 && len(h.undo) > 500 {
		if limit == 0 {
			limit = 500
		}
		h.undo = h.undo[len(h.undo)-limit:]
	}
}

// Break ends the current group: the next typing undoes separately.
func (h *History) Break() { h.last = time.Time{} }

// CanUndo and CanRedo report whether there's something to undo or redo.
func (h *History) CanUndo() bool { return len(h.undo) > 0 }
func (h *History) CanRedo() bool { return len(h.redo) > 0 }

// Undo reverts the last step on s and returns the operations it applied,
// for syncing; nil when there's nothing to undo.
func (h *History) Undo(s *State) *Tx {
	if len(h.undo) == 0 {
		return nil
	}
	e := h.undo[len(h.undo)-1]
	h.undo = h.undo[:len(h.undo)-1]
	inv := make([]Op, len(e.ops))
	for i, op := range e.ops {
		inv[len(e.ops)-1-i] = op.Invert()
	}
	for _, op := range inv {
		if err := s.Doc.Apply(op); err != nil {
			// The document changed under the history (a collaborator's
			// edit): it no longer applies.
			h.undo, h.redo = nil, nil
			return nil
		}
	}
	s.Select(e.before)
	h.redo = append(h.redo, e)
	h.last = time.Time{}
	return &Tx{Ops: inv, Before: e.after, After: s.Sel, Label: "undo", rule: -1}
}

// Redo reapplies the last undone step.
func (h *History) Redo(s *State) *Tx {
	if len(h.redo) == 0 {
		return nil
	}
	e := h.redo[len(h.redo)-1]
	h.redo = h.redo[:len(h.redo)-1]
	for _, op := range e.ops {
		if err := s.Doc.Apply(op); err != nil {
			h.undo, h.redo = nil, nil
			return nil
		}
	}
	s.Select(e.after)
	h.undo = append(h.undo, e)
	h.last = time.Time{}
	return &Tx{Ops: e.ops, Before: e.before, After: s.Sel, Label: "redo", rule: -1}
}

// Clear forgets all steps.
func (h *History) Clear() { h.undo, h.redo = nil, nil }
