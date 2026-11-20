package form

import (
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Required needs a value other than spaces; msg replaces "Required".
func Required(msg ...string) Rule {
	m := "Required"
	if len(msg) > 0 {
		m = msg[0]
	}
	return func(v string) string {
		if v == requiredProbe {
			return requiredProbe
		}
		if strings.TrimSpace(v) == "" {
			return m
		}
		return ""
	}
}

// requiredProbe is a value only Required's rules answer with itself,
// so a field knows it's required.
const requiredProbe = "\x00giode/form.required"

func isRequired(r Rule) bool { return r(requiredProbe) == requiredProbe }

// optional skips a rule for an empty value: only Required checks
// those.
func optional(check func(string) string) Rule {
	return func(v string) string {
		if strings.TrimSpace(v) == "" {
			return ""
		}
		return check(v)
	}
}

// MinLength needs at least n characters.
func MinLength(n int) Rule {
	return optional(func(v string) string {
		if utf8.RuneCountInString(v) < n {
			return fmt.Sprintf("At least %d characters", n)
		}
		return ""
	})
}

// MaxLength allows at most n characters.
func MaxLength(n int) Rule {
	return optional(func(v string) string {
		if utf8.RuneCountInString(v) > n {
			return fmt.Sprintf("At most %d characters", n)
		}
		return ""
	})
}

// Email needs an email address.
func Email() Rule {
	return optional(func(v string) string {
		a, err := mail.ParseAddress(strings.TrimSpace(v))
		if err != nil || a.Name != "" || !strings.Contains(a.Address[strings.LastIndex(a.Address, "@")+1:], ".") {
			return "Enter an email address"
		}
		return ""
	})
}

// Pattern needs the value to match re; msg says what's expected.
func Pattern(re *regexp.Regexp, msg string) Rule {
	return optional(func(v string) string {
		if !re.MatchString(v) {
			return msg
		}
		return ""
	})
}

// Number needs a number between lo and hi.
func Number(lo, hi float64) Rule {
	return optional(func(v string) string {
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return "Enter a number"
		}
		if n < lo || n > hi {
			return fmt.Sprintf("Between %s and %s", strconv.FormatFloat(lo, 'f', -1, 64), strconv.FormatFloat(hi, 'f', -1, 64))
		}
		return ""
	})
}

// Matches needs the same value as other, as a password confirmation
// does.
func Matches(other func() string, msg string) Rule {
	return func(v string) string {
		if v != other() {
			return msg
		}
		return ""
	}
}

// Check makes a rule from a function returning an error.
func Check(fn func(string) error) Rule {
	return optional(func(v string) string {
		if err := fn(v); err != nil {
			return err.Error()
		}
		return ""
	})
}
