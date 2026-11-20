package form

import (
	"errors"
	"image"
	"regexp"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/fonts"
	"github.com/xmhrnx/giode/styles"
)

func TestRules(t *testing.T) {
	for _, tc := range []struct {
		r    Rule
		v    string
		fail bool
	}{
		{Required(), "", true}, {Required(), "  ", true}, {Required(), "x", false},
		{MinLength(3), "ab", true}, {MinLength(3), "", false}, {MinLength(3), "abc", false},
		{MaxLength(2), "abc", true},
		{Email(), "ada@example.com", false}, {Email(), "ada@", true}, {Email(), "Ada <ada@x.io>", true}, {Email(), "ada@localhost", true},
		{Pattern(regexp.MustCompile(`^\d{5}$`), "5 digits"), "1234", true},
		{Number(1, 10), "11", true}, {Number(1, 10), "x", true}, {Number(1, 10), "2.5", false},
		{Matches(func() string { return "pw" }, "Doesn't match"), "pw", false},
		{Check(func(s string) error { return errors.New("taken") }), "bob", true},
	} {
		if got := tc.r(tc.v) != ""; got != tc.fail {
			t.Errorf("rule on %q: failed %v, want %v", tc.v, got, tc.fail)
		}
	}
	if !isRequired(Required("Enter a name")) || isRequired(MinLength(2)) {
		t.Error("required detection")
	}
}

func TestForm(t *testing.T) {
	fonts.UseGofont()
	name, email := input.New("Name"), input.New("Email")
	email.SetText("ada@example.com")
	agree := "no"
	f := New()
	f.Input("name", "Name", name, Required("Enter your name"), MinLength(2))
	f.Input("email", "Email", email, Required(), Email())
	f.Value("agree", "Terms", func() string { return agree }, func(s string) { agree = s },
		func(v string) string {
			if v != "yes" {
				return "Accept the terms"
			}
			return ""
		})
	var submitted map[string]string
	f.OnSubmit(func(v map[string]string) { submitted = v })

	if f.Dirty() || !f.Field("name").Required() || f.Field("agree").Required() {
		t.Fatal("initial state")
	}
	frame := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Constraints: layout.Constraints{Max: image.Pt(400, 600)}, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		elements.BeginFrame()
		elements.Stack(styles.Styles{}, f.ErrorSummary(), f.Row("name", name), f.Row("email", email)).Layout(gtx)
		elements.RunAfterLayout()
		elements.EndFrame()
	}
	frame()
	// Errors stay hidden until a submit attempt.
	if len(f.Errors()) != 0 {
		t.Fatal("errors before submit")
	}
	f.Submit()
	frame()
	if submitted != nil {
		t.Fatal("submitted an invalid form")
	}
	errs := f.Errors()
	if len(errs) != 2 || errs[0].Message != "Enter your name" || errs[1].Name != "agree" {
		t.Fatalf("errors %+v", errs)
	}
	// Fixing a field clears its error as it's typed.
	name.SetText("A")
	frame()
	if f.Field("name").Error() != "At least 2 characters" {
		t.Errorf("live: %q", f.Field("name").Error())
	}
	name.SetText("Ada")
	frame()
	if f.Field("name").Error() != "" {
		t.Error("error stayed after the fix")
	}
	agree = "yes"
	f.Submit()
	if submitted == nil || submitted["name"] != "Ada" || submitted["agree"] != "yes" {
		t.Fatalf("submitted %v", submitted)
	}
	if !f.Dirty() || strings.Join(f.DirtyFields(), ",") != "name,agree" {
		t.Errorf("dirty %v", f.DirtyFields())
	}
	f.MarkClean()
	if f.Dirty() {
		t.Error("dirty after MarkClean")
	}
	name.SetText("Grace")
	f.Reset()
	if name.Text() != "Ada" || f.Dirty() || len(f.Errors()) != 0 {
		t.Errorf("reset: %q dirty %v", name.Text(), f.Dirty())
	}
}
