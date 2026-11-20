// Package combobox provides searchable selection inputs: Combobox
// picks one option from a list or an asynchronous source as the user
// types, MultiSelect picks several as removable chips (a tag input when
// it may create new ones), and List is their keyboard-driven option
// panel, also used by the command palette.
package combobox

import (
	"context"

	"github.com/xmhrnx/giode/elements"
)

// Option is a choice.
type Option struct {
	// Value identifies it; Label is shown, Value when empty.
	Value, Label string
	// Detail is shown dimmed beside the label: a shortcut, an email.
	Detail string
	// Group puts it under a heading. Options of a group should be
	// together.
	Group string
	// Icon is shown before the label.
	Icon elements.Element
	// Keywords are also searched.
	Keywords []string
	Disabled bool

	matched []int
}

func (o Option) text() string {
	if o.Label != "" {
		return o.Label
	}
	return o.Value
}

// Source fetches the options matching query, for options that live on
// a server. ctx is canceled when the query changes before it returns.
type Source func(ctx context.Context, query string) ([]Option, error)
