// Package form validates and tracks a form's fields: rules per field
// (required, lengths, email, patterns, matching another field),
// errors shown after the user leaves a field or tries to submit, a
// summary of what's wrong, required markers, and whether anything
// changed since it was loaded or saved.
//
//	f := form.New()
//	f.Input("email", "Email", emailInput, form.Required(), form.Email())
//	f.Input("password", "Password", pwInput, form.Required(), form.MinLength(8))
//	f.OnSubmit(func(v map[string]string) { save(v) })
//
//	giode.Stack(st,
//		f.ErrorSummary(),
//		f.Row("email", emailInput),
//		f.Row("password", pwInput),
//		giode.Button("Save").OnClick(f.Submit),
//	)
package form

import (
	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// Rule checks a value and returns why it's wrong, or "".
type Rule func(value string) string

// Form is a set of fields.
type Form struct {
	fields    []*Field
	onSubmit  func(map[string]string)
	attempted bool
}

// New returns an empty form.
func New() *Form { return &Form{} }

// Field is one value of a form.
type Field struct {
	Name, Label string

	form     *Form
	get      func() string
	set      func(string)
	focus    func()
	show     func(msg string)
	rules    []Rule
	required bool
	initial  string
	last     string
	touched  bool
	err      string
}

// Value adds a field read by get and written by set (for a checkbox, a
// select, a date): any component. set may be nil when Reset needn't
// restore it.
func (f *Form) Value(name, label string, get func() string, set func(string), rules ...Rule) *Field {
	fd := &Field{Name: name, Label: label, form: f, get: get, set: set, rules: rules, initial: get()}
	fd.last = fd.initial
	for _, r := range rules {
		if isRequired(r) {
			fd.required = true
		}
	}
	f.fields = append(f.fields, fd)
	return fd
}

// Input adds a text field: it's marked required for screen readers,
// shows its error in its border and is checked when the user leaves
// it.
func (f *Form) Input(name, label string, in *input.Input, rules ...Rule) *Field {
	fd := f.Value(name, label, in.Text, in.SetText, rules...)
	fd.focus = in.Focus
	fd.show = func(msg string) { in.Invalid(msg) }
	in.Required(fd.required)
	in.Label(label)
	in.OnBlur(func() {
		fd.touched = true
		fd.Validate()
	})
	return fd
}

// Focus sets how to focus the field, for the error summary's links.
func (fd *Field) Focus(fn func()) *Field { fd.focus = fn; return fd }

// Field returns the field named name, or nil.
func (f *Form) Field(name string) *Field {
	for _, fd := range f.fields {
		if fd.Name == name {
			return fd
		}
	}
	return nil
}

// Validate checks the field and shows its error; it reports whether
// it's valid.
func (fd *Field) Validate() bool {
	v := fd.get()
	fd.err = ""
	for _, r := range fd.rules {
		if msg := r(v); msg != "" {
			fd.err = msg
			break
		}
	}
	if fd.show != nil {
		fd.show(fd.err)
	}
	return fd.err == ""
}

// Error is the field's error, once shown: after the user left the
// field, or tried to submit.
func (fd *Field) Error() string { return fd.err }

// Required reports whether the field must be filled.
func (fd *Field) Required() bool { return fd.required }

// Dirty reports whether the value changed since it was loaded or
// saved.
func (fd *Field) Dirty() bool { return fd.get() != fd.initial }

// refresh rechecks a field whose error shows as the user edits it, so
// an error goes away as soon as it's fixed.
func (fd *Field) refresh() {
	v := fd.get()
	if v == fd.last {
		return
	}
	fd.last = v
	if fd.touched || fd.form.attempted {
		fd.Validate()
	}
}

// Validate checks every field, shows their errors and focuses the
// first wrong one; it reports whether all are valid.
func (f *Form) Validate() bool {
	f.attempted = true
	ok := true
	var first *Field
	for _, fd := range f.fields {
		if !fd.Validate() {
			ok = false
			if first == nil {
				first = fd
			}
		}
	}
	if first != nil && first.focus != nil {
		first.focus()
	}
	return ok
}

// OnSubmit sets what Submit does with the values once they're valid.
func (f *Form) OnSubmit(fn func(values map[string]string)) *Form {
	f.onSubmit = fn
	return f
}

// Submit validates the form and, when it's valid, calls the OnSubmit
// handler.
func (f *Form) Submit() {
	if f.Validate() && f.onSubmit != nil {
		f.onSubmit(f.Values())
	}
}

// Values are the fields' values by name.
func (f *Form) Values() map[string]string {
	out := make(map[string]string, len(f.fields))
	for _, fd := range f.fields {
		out[fd.Name] = fd.get()
	}
	return out
}

// Dirty reports whether any field changed since the form was loaded
// or saved: to ask before leaving with unsaved changes, or to enable
// Save.
func (f *Form) Dirty() bool {
	for _, fd := range f.fields {
		if fd.Dirty() {
			return true
		}
	}
	return false
}

// DirtyFields are the names of the changed fields.
func (f *Form) DirtyFields() []string {
	var out []string
	for _, fd := range f.fields {
		if fd.Dirty() {
			out = append(out, fd.Name)
		}
	}
	return out
}

// MarkClean takes the current values as the saved ones.
func (f *Form) MarkClean() {
	for _, fd := range f.fields {
		fd.initial = fd.get()
	}
}

// Reset puts back the saved values and clears the errors.
func (f *Form) Reset() {
	f.attempted = false
	for _, fd := range f.fields {
		if fd.set != nil {
			fd.set(fd.initial)
		}
		fd.last, fd.touched, fd.err = fd.get(), false, ""
		if fd.show != nil {
			fd.show("")
		}
	}
}

// FieldError is a field's error, for a custom summary.
type FieldError struct{ Name, Label, Message string }

// Errors are the errors showing, in the fields' order.
func (f *Form) Errors() []FieldError {
	var out []FieldError
	for _, fd := range f.fields {
		if fd.err != "" {
			out = append(out, FieldError{fd.Name, fd.Label, fd.err})
		}
	}
	return out
}

// Row lays out a field: its label (with a required marker), the
// element and its error under it.
func (f *Form) Row(name string, el elements.Element) elements.Element {
	fd := f.Field(name)
	if fd == nil {
		return el
	}
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		fd.refresh()
		kids := []elements.Element{Label(fd.Label, fd.required), el}
		if fd.err != "" {
			kids = append(kids, elements.Accessible(elements.Text(fd.err, styles.Styles{Color: "theme.danger", FontSize: 13}),
				func(n *elements.SemanticNode) { n.Live = elements.LivePolite }))
		}
		return elements.Stack(styles.Styles{Gap: 4, Width: -1}, kids...).Layout(gtx)
	})
}

// Label is a field label, with a required marker.
func Label(text string, required bool) elements.Element {
	spans := []elements.Span{elements.SpanText(text, styles.Styles{Color: "theme.text", FontWeight: properties.FontWeightSemiBold})}
	if required {
		spans = append(spans, elements.SpanText(" *", styles.Styles{Color: "theme.danger"}))
	}
	return elements.RichText(styles.Styles{FontSize: 14}, spans...)
}

// ErrorSummary lists what's wrong after a submit attempt, each a link
// to its field; it's empty otherwise. Screen readers announce it.
func (f *Form) ErrorSummary() elements.Element {
	return elements.Raw(func(gtx layout.Context) layout.Dimensions {
		for _, fd := range f.fields {
			fd.refresh()
		}
		errs := f.Errors()
		if !f.attempted || len(errs) == 0 {
			return layout.Dimensions{}
		}
		title := "Fix 1 problem:"
		if len(errs) > 1 {
			title = "Fix " + itoa(len(errs)) + " problems:"
		}
		kids := []elements.Element{elements.Text(title, styles.Styles{Color: "theme.danger", FontWeight: properties.FontWeightSemiBold})}
		for _, e := range errs {
			fd := f.Field(e.Name)
			kids = append(kids, elements.NewPressable(func(elements.PressState) elements.Element {
				return elements.Text("• "+e.Label+": "+e.Message, styles.Styles{Color: "theme.danger", TextDecoration: properties.TextDecorationUnderline})
			}).OnClick(func(elements.ClickEvent) {
				if fd.focus != nil {
					fd.focus()
				}
			}))
		}
		box := elements.Box(styles.Styles{FlexDirection: properties.FlexDirectionColumn, Gap: 4, Width: -1,
			Padding: properties.UniformInset(12), BorderRadius: 8, BorderWidth: 1, BorderColor: "theme.danger"}, kids...)
		return elements.AccessibleGroup(elements.SemanticNode{Kind: elements.SemanticAlert, Label: title}, box).Layout(gtx)
	})
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
