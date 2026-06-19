package keyring

// Fake is an in-memory Keyring for tests. The zero value behaves as an empty
// keyring (Get returns ErrNotFound).
type Fake struct {
	Key    []byte
	SetErr error // when non-nil, Set returns it
	GetErr error // when non-nil, Get returns it (takes precedence over Key)
}

// Get returns the stored key or ErrNotFound.
func (f *Fake) Get() ([]byte, error) {
	if f.GetErr != nil {
		return nil, f.GetErr
	}
	if f.Key == nil {
		return nil, ErrNotFound
	}
	return f.Key, nil
}

// Set records the key in memory.
func (f *Fake) Set(key []byte) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.Key = append([]byte(nil), key...)
	return nil
}
