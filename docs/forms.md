# Forms

The `form` package validates and tracks a form's fields: rules per field, errors shown after
the user leaves a field or tries to submit, a summary of what's wrong, required markers, and
whether anything changed since the form was loaded or saved.

```go
import "github.com/xmhrnx/giode/form"

name, email := giode.Input.Text("Your name"), giode.Input.Text("you@example.com")
f := form.New()
f.Input("name", "Name", name, form.Required("Enter your name"), form.MaxLength(80))
f.Input("email", "Email", email, form.Required(), form.Email())
f.Value("plan", "Plan", func() string { return plan }, func(s string) { plan = s }, form.Required()) // any component
f.OnSubmit(func(v map[string]string) { save(v); f.MarkClean() })

giode.Stack(st,
	f.ErrorSummary(),
	f.Row("name", name),
	f.Row("email", email),
	giode.Button("Save").OnClick(f.Submit),
)
```

`Input` binds a text field: it's marked required for screen readers, shows its error in its
border, and is checked when the user leaves it. `Value` binds anything through a getter and a
setter. `Row` lays out a field's label (with a red `*` when required), the element and its
error under it; `form.Label` is the label alone.

Errors show once the user has left a field or tried to submit, and then follow every edit, so
an error goes as soon as it's fixed. `Submit` validates everything, focuses the first wrong
field and calls the `OnSubmit` handler only when all are valid. `ErrorSummary` lists the
problems after a failed attempt, each a link to its field, announced by screen readers.

| Rule | |
|---|---|
| `Required(msg...)` | not empty or spaces |
| `MinLength(n)`, `MaxLength(n)` | characters |
| `Email()` | an email address |
| `Pattern(re, msg)` | matches a regular expression |
| `Number(lo, hi)` | a number in range |
| `Matches(get, msg)` | equals another value, as a password confirmation |
| `Check(fn)` | any check returning an error |

Rules other than `Required` let an empty value pass. `Dirty`, `DirtyFields`, `MarkClean` and
`Reset` track and undo changes: ask before leaving with unsaved changes, or enable Save only
when something changed. `Values` and `Errors` read the form.
