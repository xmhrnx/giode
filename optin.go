package giode

import "fmt"

// notLinked is the error of a feature whose opt-in package (see
// internal/optin) the app doesn't import.
func notLinked(pkg string) error {
	return fmt.Errorf("%w: this needs the github.com/xmhrnx/giode/%s package imported", ErrNotSupported, pkg)
}
