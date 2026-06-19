// Package clipboard wraps the macOS NSPasteboard. The daemon polls ChangeCount
// (a free integer read) and only reads contents when it moves; the TUI writes a
// selected clip back. The interface is intentionally tiny so it is cheap to
// fake in tests and the native binding stays a thin boundary.
package clipboard

// Pasteboard is the minimal clipboard surface yoink needs.
type Pasteboard interface {
	// ChangeCount returns NSPasteboard.changeCount, which the OS increments on
	// every clipboard write. Cheap enough to poll frequently.
	ChangeCount() int

	// Read returns the current clipboard text. concealed is true when the
	// content is marked transient/concealed (e.g. by a password manager). ok is
	// false when there is no plain-text content to capture.
	Read() (text string, concealed bool, ok bool)

	// Write places text on the clipboard.
	Write(text string) error
}
