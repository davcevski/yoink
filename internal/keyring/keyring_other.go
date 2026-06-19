//go:build !darwin

package keyring

import "errors"

// errUnsupported keeps the module buildable on non-darwin platforms (CI lint,
// cross-platform `go vet`) while making clear yoink only runs on macOS.
var errUnsupported = errors.New("keyring: only supported on macOS")

// Keychain is a non-functional placeholder on non-darwin platforms.
type Keychain struct{}

// New returns the placeholder provider.
func New() Keychain { return Keychain{} }

// Get always reports the platform is unsupported.
func (Keychain) Get() ([]byte, error) { return nil, errUnsupported }

// Set always reports the platform is unsupported.
func (Keychain) Set([]byte) error { return errUnsupported }
