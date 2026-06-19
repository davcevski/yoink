package clipboard

import "sync"

// Fake is a scriptable in-memory Pasteboard for tests. It models NSPasteboard's
// changeCount semantics: Set bumps the counter the same way a real copy does.
type Fake struct {
	mu        sync.Mutex
	count     int
	text      string
	concealed bool
	hasText   bool
	WriteErr  error // when non-nil, Write returns it
	Written   []string
}

// Set simulates a clipboard write of plain text, bumping ChangeCount.
func (f *Fake) Set(text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
	f.text = text
	f.concealed = false
	f.hasText = true
}

// SetConcealed simulates a concealed/transient copy (e.g. a password manager).
func (f *Fake) SetConcealed(text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
	f.text = text
	f.concealed = true
	f.hasText = true
}

// SetEmpty simulates a non-text clipboard change (image, files, etc.).
func (f *Fake) SetEmpty() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
	f.text = ""
	f.concealed = false
	f.hasText = false
}

// ChangeCount returns the simulated change counter.
func (f *Fake) ChangeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count
}

// Read returns the current simulated clipboard state.
func (f *Fake) Read() (string, bool, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.text, f.concealed, f.hasText
}

// Write records the written text (and bumps the counter, like a real write).
func (f *Fake) Write(text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.WriteErr != nil {
		return f.WriteErr
	}
	f.count++
	f.text = text
	f.hasText = true
	f.Written = append(f.Written, text)
	return nil
}
