package daemon

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davcevski/yoink/internal/clipboard"
	"github.com/davcevski/yoink/internal/config"
	"github.com/davcevski/yoink/internal/crypto"
	"github.com/davcevski/yoink/internal/keyring"
	"github.com/davcevski/yoink/internal/store"
)

func testKeyring(t *testing.T) *keyring.Fake {
	t.Helper()
	key, err := crypto.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	return &keyring.Fake{Key: key}
}

func testStore(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.OpenAt(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newDaemon(t *testing.T, clip clipboard.Pasteboard, st Store, keys keyReader) *Daemon {
	t.Helper()
	cfg := config.Default()
	return New(clip, st, keys, cfg, nil)
}

func TestCaptureStoresNewClip(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	d := newDaemon(t, clip, db, testKeyring(t))

	clip.Set("hello world")
	d.capture()

	clips, err := db.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(clips) != 1 {
		t.Fatalf("got %d clips, want 1", len(clips))
	}
	if clips[0].Bytes != len("hello world") {
		t.Errorf("Bytes = %d, want %d", clips[0].Bytes, len("hello world"))
	}
	// Content must be ciphertext, not plaintext.
	if strings.Contains(string(clips[0].Content), "hello world") {
		t.Error("plaintext leaked into stored content")
	}
}

func TestCaptureSkipsWhenNothingCopied(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	d := newDaemon(t, clip, db, testKeyring(t))

	d.capture() // changeCount still 0
	if clips, _ := db.Recent(10); len(clips) != 0 {
		t.Fatalf("captured with no change: %d clips", len(clips))
	}
}

func TestCaptureSkipsConcealed(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	d := newDaemon(t, clip, db, testKeyring(t))

	clip.SetConcealed("hunter2")
	d.capture()
	if clips, _ := db.Recent(10); len(clips) != 0 {
		t.Fatalf("captured a concealed clip: %d clips", len(clips))
	}
}

func TestCaptureSkipsNonText(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	d := newDaemon(t, clip, db, testKeyring(t))

	clip.SetEmpty()
	d.capture()
	if clips, _ := db.Recent(10); len(clips) != 0 {
		t.Fatalf("captured a non-text clip: %d clips", len(clips))
	}
}

func TestCaptureSkipsOversize(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	cfg := config.Default()
	cfg.MaxClipBytes = 8
	d := New(clip, db, testKeyring(t), cfg, nil)

	clip.Set("this is definitely longer than eight bytes")
	d.capture()
	if clips, _ := db.Recent(10); len(clips) != 0 {
		t.Fatalf("captured an oversize clip: %d clips", len(clips))
	}
}

func TestCaptureDedupsConsecutiveIdentical(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	d := newDaemon(t, clip, db, testKeyring(t))

	clip.Set("same")
	d.capture()
	clip.Set("same") // bumps changeCount but identical content
	d.capture()

	if clips, _ := db.Recent(10); len(clips) != 1 {
		t.Fatalf("dedup failed: got %d clips, want 1", len(clips))
	}
}

func TestCapturePausesWhenKeyMissing(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	d := newDaemon(t, clip, db, &keyring.Fake{}) // empty keyring → ErrNotFound

	clip.Set("secret while key absent")
	d.capture()

	if clips, _ := db.Recent(10); len(clips) != 0 {
		t.Fatalf("wrote a clip with no key available: %d clips", len(clips))
	}
	if !d.keyMissing {
		t.Error("daemon did not record the missing-key pause")
	}
}

func TestCaptureResumesWhenKeyAppears(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	keys := &keyring.Fake{}
	d := newDaemon(t, clip, db, keys)

	clip.Set("first")
	d.capture() // paused, no key

	key, _ := crypto.NewKey()
	keys.Key = key
	clip.Set("second")
	d.capture() // key now present

	clips, _ := db.Recent(10)
	if len(clips) != 1 {
		t.Fatalf("did not resume after key appeared: %d clips", len(clips))
	}
	if d.keyMissing {
		t.Error("daemon still flagged as key-missing after resume")
	}
}

func TestPruneRemovesExpired(t *testing.T) {
	db := testStore(t)
	cfg := config.Default()
	cfg.RetentionDays = 1
	d := New(&clipboard.Fake{}, db, testKeyring(t), cfg, nil)

	db.Insert(store.Clip{Content: []byte{1}, Nonce: []byte{1}, Hash: "old", CreatedAt: time.Now().Add(-48 * time.Hour)})
	db.Insert(store.Clip{Content: []byte{2}, Nonce: []byte{2}, Hash: "fresh", CreatedAt: time.Now()})

	d.prune()
	clips, _ := db.Recent(10)
	if len(clips) != 1 || clips[0].Hash != "fresh" {
		t.Fatalf("prune did not remove expired clip: %+v", clips)
	}
}

func TestRunCapturesUntilCancelled(t *testing.T) {
	clip := &clipboard.Fake{}
	db := testStore(t)
	cfg := config.Default()
	cfg.PollIntervalMS = 5
	d := New(clip, db, testKeyring(t), cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()

	clip.Set("captured by the loop")

	deadline := time.After(2 * time.Second)
	for {
		if clips, _ := db.Recent(10); len(clips) == 1 {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("clip was not captured by the running loop")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}
