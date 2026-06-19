package keyring

import (
	"bytes"
	"errors"
	"testing"

	"github.com/davcevski/yoink/internal/crypto"
)

func TestEnsureKeyGeneratesWhenAbsent(t *testing.T) {
	f := &Fake{}
	key, err := EnsureKey(f)
	if err != nil {
		t.Fatalf("EnsureKey: %v", err)
	}
	if len(key) != crypto.KeySize {
		t.Fatalf("generated key len = %d, want %d", len(key), crypto.KeySize)
	}
	if !bytes.Equal(key, f.Key) {
		t.Fatal("EnsureKey did not persist the generated key")
	}
}

func TestEnsureKeyReturnsExisting(t *testing.T) {
	existing := bytes.Repeat([]byte("k"), crypto.KeySize)
	f := &Fake{Key: existing}
	key, err := EnsureKey(f)
	if err != nil {
		t.Fatalf("EnsureKey: %v", err)
	}
	if !bytes.Equal(key, existing) {
		t.Fatal("EnsureKey overwrote an existing key")
	}
}

func TestEnsureKeyPropagatesUnexpectedGetError(t *testing.T) {
	boom := errors.New("keychain locked")
	f := &Fake{GetErr: boom}
	if _, err := EnsureKey(f); !errors.Is(err, boom) {
		t.Fatalf("EnsureKey err = %v, want %v", err, boom)
	}
}

func TestFakeGetReturnsNotFoundWhenEmpty(t *testing.T) {
	f := &Fake{}
	if _, err := f.Get(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get err = %v, want ErrNotFound", err)
	}
}
