package base

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// debugf receives layout diagnostics; nil discards them.
var debugf atomic.Pointer[func(format string, v ...any)]

// SetDebugLogger sets the function layout diagnostics go to, such as
// the warning about FlexGrow in an unbounded container; nil turns them
// off. The app sets it from Config.Logger at LogLevelDebug; component
// authors don't need to.
func SetDebugLogger(fn func(format string, v ...any)) {
	if fn == nil {
		debugf.Store(nil)
		return
	}
	debugf.Store(&fn)
}

var (
	warnedMu sync.Mutex
	warned   = map[string]bool{}
)

// WarnOnce logs a diagnostic the first time a given message comes up.
func WarnOnce(format string, v ...any) {
	fn := debugf.Load()
	if fn == nil {
		return
	}
	msg := fmt.Sprintf(format, v...)
	warnedMu.Lock()
	seen := warned[msg]
	warned[msg] = true
	warnedMu.Unlock()
	if !seen {
		(*fn)("%s", msg)
	}
}

// Describer names an element for diagnostics: its kind, and the first
// text inside it, which is usually enough to find it in the code.
// Elements implement it so Describe can stay independent of the
// concrete type.
type Describer interface {
	Describe() string
}

// Describe names an element for diagnostics: its kind and the first
// text inside it, which is usually enough to find it in the code.
func Describe(el Element) string {
	if d, ok := el.(Describer); ok {
		return d.Describe()
	}
	return fmt.Sprintf("%T", el)
}

// TextInliner reports the text content of a text element, for
// diagnostics that name a container by the first text inside it.
type TextInliner interface {
	InlineText() string
}
