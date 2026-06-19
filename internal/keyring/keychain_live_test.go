//go:build darwin

package keyring

import (
	"bytes"
	"os"
	"testing"

	keychain "github.com/keybase/go-keychain"

	"github.com/davcevski/yoink/internal/crypto"
)

// TestLiveKeychain exercises the real macOS login Keychain. Opt-in, since it
// touches the user's keychain:
//
//	YOINK_LIVE_KEYCHAIN=1 go test ./internal/keyring/
//
// It writes a throwaway key under the yoink service and deletes it afterward.
func TestLiveKeychain(t *testing.T) {
	if os.Getenv("YOINK_LIVE_KEYCHAIN") == "" {
		t.Skip("set YOINK_LIVE_KEYCHAIN=1 to run the live Keychain test")
	}

	kc := New()
	t.Cleanup(func() { _ = keychain.DeleteGenericPasswordItem(service, account) })

	key, err := crypto.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	if err := kc.Set(key); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := kc.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, key) {
		t.Fatal("round-tripped key does not match")
	}

	// Set again to confirm the update-in-place path works.
	key2, _ := crypto.NewKey()
	if err := kc.Set(key2); err != nil {
		t.Fatalf("Set (update): %v", err)
	}
	got2, _ := kc.Get()
	if !bytes.Equal(got2, key2) {
		t.Fatal("update-in-place did not replace the key")
	}
}
