package giode

import (
	"strings"
)

// snapshotNode is a node as reported to clients.
type snapshotNode struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Label    string `json:"label,omitempty"`
	Value    string `json:"value,omitempty"`
	Bounds   [4]int `json:"bounds"`
	Disabled bool   `json:"disabled,omitempty"`
	Actions  string `json:"actions,omitempty"`
	Scroll   string `json:"scroll,omitempty"`
}

// Interfaces stateful components implement for automation.
type (
	clicker     interface{ Click() }
	textChanger interface{ ChangeText(string) }
	textSetter  interface{ SetText(string) }
	submitter   interface{ SubmitText() }
	toggler     interface {
		Checked() bool
		Toggle()
	}
	valueChanger  interface{ Change(float32) }
	optionChooser interface {
		Options() []string
		Choose(int)
	}
)

func actionsOf(target any) string {
	var acts []string
	if _, ok := target.(clicker); ok {
		acts = append(acts, "click")
	}
	_, tc := target.(textChanger)
	_, ts := target.(textSetter)
	if tc || ts {
		acts = append(acts, "type")
	}
	if _, ok := target.(toggler); ok {
		acts = append(acts, "set_value(bool)")
	}
	if _, ok := target.(valueChanger); ok {
		acts = append(acts, "set_value(0-1)")
	}
	if _, ok := target.(optionChooser); ok {
		acts = append(acts, "set_value(option)")
	}
	return strings.Join(acts, ",")
}
