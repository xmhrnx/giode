package giode

import (
	"reflect"
	"regexp"
	"testing"
)

func TestContextActionCLSID(t *testing.T) {
	got := ContextActionCLSID("com-example-send", "send")
	if !regexp.MustCompile(`^\{[0-9A-F]{8}-[0-9A-F]{4}-5[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}\}$`).MatchString(got) {
		t.Errorf("CLSID %s isn't a version 5 UUID in braces", got)
	}
	// The giode CLI's installer derives the same ID: keep the two in
	// step (giode-cli internal/config TestContextActionCLSID).
	if want := "{7D2BADB1-E06C-5098-B45A-CB79C5D1A596}"; got != want {
		t.Errorf("CLSID %s, want %s", got, want)
	}
	if ContextActionCLSID("com-example-send", "other") == got {
		t.Error("two actions share a CLSID")
	}
}

func TestWithoutEmbedding(t *testing.T) {
	if got := withoutEmbedding([]string{"-Embedding", "a", "/embedding"}); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("got %q", got)
	}
}
