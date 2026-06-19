package store

import (
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := OpenAt(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestInsertAndRecentOrdering(t *testing.T) {
	db := openTestDB(t)
	base := time.Now().Add(-time.Hour)
	for i := 0; i < 3; i++ {
		err := db.Insert(Clip{
			Content:   []byte{byte(i)},
			Nonce:     []byte("nonce"),
			Hash:      string(rune('a' + i)),
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
			Bytes:     i,
		})
		if err != nil {
			t.Fatalf("Insert %d: %v", i, err)
		}
	}

	clips, err := db.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(clips) != 3 {
		t.Fatalf("got %d clips, want 3", len(clips))
	}
	// Newest first.
	if clips[0].Hash != "c" || clips[2].Hash != "a" {
		t.Errorf("unexpected ordering: %q, %q, %q", clips[0].Hash, clips[1].Hash, clips[2].Hash)
	}
}

func TestRecentRespectsLimit(t *testing.T) {
	db := openTestDB(t)
	for i := 0; i < 5; i++ {
		if err := db.Insert(Clip{Content: []byte{1}, Nonce: []byte{1}, Hash: string(rune('a' + i))}); err != nil {
			t.Fatal(err)
		}
	}
	clips, err := db.Recent(2)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(clips) != 2 {
		t.Fatalf("limit not honored: got %d, want 2", len(clips))
	}
}

func TestDefaultsAppliedOnInsert(t *testing.T) {
	db := openTestDB(t)
	if err := db.Insert(Clip{Content: []byte{1}, Nonce: []byte{1}, Hash: "h"}); err != nil {
		t.Fatal(err)
	}
	clips, _ := db.Recent(1)
	if clips[0].DeviceID != DefaultDeviceID {
		t.Errorf("DeviceID = %q, want %q", clips[0].DeviceID, DefaultDeviceID)
	}
	if clips[0].CreatedAt.IsZero() {
		t.Error("CreatedAt not defaulted")
	}
}

func TestLatestHash(t *testing.T) {
	db := openTestDB(t)
	if h, err := db.LatestHash(); err != nil || h != "" {
		t.Fatalf("empty DB LatestHash = %q, %v; want \"\", nil", h, err)
	}
	base := time.Now()
	db.Insert(Clip{Content: []byte{1}, Nonce: []byte{1}, Hash: "old", CreatedAt: base})
	db.Insert(Clip{Content: []byte{2}, Nonce: []byte{2}, Hash: "new", CreatedAt: base.Add(time.Second)})
	h, err := db.LatestHash()
	if err != nil {
		t.Fatalf("LatestHash: %v", err)
	}
	if h != "new" {
		t.Errorf("LatestHash = %q, want \"new\"", h)
	}
}

func TestGet(t *testing.T) {
	db := openTestDB(t)
	db.Insert(Clip{Content: []byte("ct"), Nonce: []byte("n"), Hash: "h", Bytes: 2})
	clips, _ := db.Recent(1)
	id := clips[0].ID

	got, err := db.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != id || string(got.Content) != "ct" || got.Bytes != 2 {
		t.Errorf("Get returned %+v", got)
	}

	if _, err := db.Get(99999); err == nil {
		t.Error("Get(missing) returned nil error")
	}
}

func TestDelete(t *testing.T) {
	db := openTestDB(t)
	db.Insert(Clip{Content: []byte{1}, Nonce: []byte{1}, Hash: "h"})
	clips, _ := db.Recent(1)
	id := clips[0].ID
	if err := db.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	clips, _ = db.Recent(10)
	if len(clips) != 0 {
		t.Errorf("clip not deleted: %d remain", len(clips))
	}
}

func TestPruneRemovesOldClips(t *testing.T) {
	db := openTestDB(t)
	now := time.Now()
	db.Insert(Clip{Content: []byte{1}, Nonce: []byte{1}, Hash: "old", CreatedAt: now.Add(-48 * time.Hour)})
	db.Insert(Clip{Content: []byte{2}, Nonce: []byte{2}, Hash: "fresh", CreatedAt: now})

	cutoff := now.Add(-24 * time.Hour)
	n, err := db.Prune(cutoff)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if n != 1 {
		t.Errorf("pruned %d, want 1", n)
	}
	clips, _ := db.Recent(10)
	if len(clips) != 1 || clips[0].Hash != "fresh" {
		t.Errorf("wrong clip survived prune: %+v", clips)
	}
}
