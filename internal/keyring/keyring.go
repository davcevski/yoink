// Package keyring stores yoink's 256-bit encryption key in the macOS login
// Keychain. The key never lives in the config file or the database.
package keyring

import (
	"errors"

	"github.com/davcevski/yoink/internal/crypto"
)

// ErrNotFound indicates the encryption key has not yet been stored.
var ErrNotFound = errors.New("keyring: encryption key not found")

// Keyring is the minimal surface the rest of yoink needs: read the key and
// write it. Kept deliberately small so a fake is trivial and the abstraction
// stays strong.
type Keyring interface {
	Get() ([]byte, error)
	Set(key []byte) error
}

// EnsureKey returns the existing key, generating and storing a fresh one if
// none exists yet. This is the create-if-absent path used by `yoink install`
// and the TUI; the daemon must never call it (it stays read-only and pauses
// when the key is missing).
func EnsureKey(k Keyring) ([]byte, error) {
	key, err := k.Get()
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	fresh, err := crypto.NewKey()
	if err != nil {
		return nil, err
	}
	if err := k.Set(fresh); err != nil {
		// A concurrent creator may have won the race; prefer whatever is now
		// stored so both processes converge on a single key.
		if existing, getErr := k.Get(); getErr == nil {
			return existing, nil
		}
		return nil, err
	}
	return fresh, nil
}
