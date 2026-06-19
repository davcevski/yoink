//go:build !darwin

package clipboard

import "errors"

var errUnsupported = errors.New("clipboard: only supported on macOS")

// Darwin is a non-functional placeholder on non-darwin platforms, present so the
// module builds for CI lint and cross-platform vet.
type Darwin struct{}

// New returns the placeholder clipboard.
func New() Darwin { return Darwin{} }

// ChangeCount always returns 0.
func (Darwin) ChangeCount() int { return 0 }

// Read always reports no content.
func (Darwin) Read() (string, bool, bool) { return "", false, false }

// Write always reports the platform is unsupported.
func (Darwin) Write(string) error { return errUnsupported }
