//go:build darwin

package clipboard

/*
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "clipboard_darwin.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

// Darwin is the production Pasteboard backed by the macOS NSPasteboard.
type Darwin struct{}

// New returns the macOS clipboard.
func New() Darwin { return Darwin{} }

// ChangeCount returns NSPasteboard.changeCount.
func (Darwin) ChangeCount() int {
	return int(C.yoink_change_count())
}

// Read returns the current clipboard text and whether it is concealed/transient.
func (Darwin) Read() (text string, concealed bool, ok bool) {
	var c C.int
	cs := C.yoink_read(&c)
	if cs == nil {
		return "", false, false
	}
	defer C.free(unsafe.Pointer(cs))
	return C.GoString(cs), c != 0, true
}

// Write places text on the clipboard.
func (Darwin) Write(text string) error {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	if C.yoink_write(cs) != 0 {
		return errors.New("clipboard: failed to write to NSPasteboard")
	}
	return nil
}
