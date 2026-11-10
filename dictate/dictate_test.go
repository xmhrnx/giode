package dictate

import (
	"testing"

	"github.com/xmhrnx/giode/internal/optin"
)

// Importing the package turns the feature on.
func TestLinked(t *testing.T) {
	if !optin.DictateLinked {
		t.Error("not linked")
	}
}
