//go:build !(windows || (darwin && !ios) || (linux && !android))

package systray

// Phones and the web have no tray.
const Supported = false

type Item struct {
	Label, Tooltip                          string
	Checkable, Checked, Disabled, Separator bool
	Items                                   []Item
	OnClick                                 func()
}

func Start(ready func()) {}
func Remove()            {}

func Apply(icon []byte, template bool, title, tooltip string, onTapped func(), items []Item, post func(func())) {
}
