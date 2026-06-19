//go:build darwin

package daemon

import (
	"os"
	"path/filepath"
	"testing"

	keychain "github.com/keybase/go-keychain"

	"github.com/davcevski/yoink/internal/clipboard"
	"github.com/davcevski/yoink/internal/config"
	"github.com/davcevski/yoink/internal/crypto"
	"github.com/davcevski/yoink/internal/keyring"
	"github.com/davcevski/yoink/internal/store"
)

// TestLiveEndToEnd runs the real capture pipeline against the real NSPasteboard
// and Keychain (one process, so no cross-process Keychain prompt) with a temp
// DB. Opt-in:
//
//	YOINK_LIVE_E2E=1 go test ./internal/daemon/
func TestLiveEndToEnd(t *testing.T) {
	if os.Getenv("YOINK_LIVE_E2E") == "" {
		t.Skip("set YOINK_LIVE_E2E=1 to run the live end-to-end test")
	}

	keys := keyring.New()
	t.Cleanup(func() { _ = keychain.DeleteGenericPasswordItem("com.yoink", "encryption-key") })
	key, err := keyring.EnsureKey(keys)
	if err != nil {
		t.Fatalf("EnsureKey: %v", err)
	}
	cipher, err := crypto.New(key)
	if err != nil {
		t.Fatal(err)
	}

	db, err := store.OpenAt(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	pb := clipboard.New()
	prev, _, hadPrev := pb.Read()
	t.Cleanup(func() {
		if hadPrev {
			_ = pb.Write(prev)
		}
	})

	d := New(pb, db, keys, config.Default(), nil)

	const secret = "yoink-e2e-secret-payload"
	if err := pb.Write(secret); err != nil {
		t.Fatalf("clipboard write: %v", err)
	}

	// Drive one capture cycle directly (Run's loop does the same on a tick).
	d.capture()

	clips, err := db.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(clips) != 1 {
		t.Fatalf("captured %d clips, want 1", len(clips))
	}
	if string(clips[0].Content) == secret {
		t.Fatal("clip stored as plaintext")
	}
	pt, err := cipher.Open(clips[0].Content, clips[0].Nonce)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(pt) != secret {
		t.Fatalf("decrypted = %q, want %q", pt, secret)
	}
}
