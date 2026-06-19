// Package store persists encrypted clips in a local SQLite database. Only
// ciphertext, nonces, and keyed hashes are written — never plaintext.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo)

	"github.com/davcevski/yoink/internal/paths"
)

// DefaultDeviceID tags every clip written by v1. The schema carries it so a
// future sync layer can merge clips from multiple devices without migration.
const DefaultDeviceID = "local"

// Clip is one stored clipboard entry. Content is AES-256-GCM ciphertext and
// Nonce its per-clip GCM nonce; Hash is the keyed dedup hash; Bytes is the
// plaintext size (for display).
type Clip struct {
	ID        int64
	Content   []byte
	Nonce     []byte
	Hash      string
	CreatedAt time.Time
	Bytes     int
	DeviceID  string
}

// DB is a handle to the clips database. It is safe for concurrent use; SQLite
// in WAL mode keeps the daemon (writer) and TUI (reader) consistent.
type DB struct {
	sql *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS clips (
    id         INTEGER PRIMARY KEY,
    content    BLOB    NOT NULL,
    nonce      BLOB    NOT NULL,
    hash       TEXT    NOT NULL,
    created_at INTEGER NOT NULL,
    bytes      INTEGER NOT NULL,
    device_id  TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_clips_created_at ON clips(created_at);
CREATE INDEX IF NOT EXISTS idx_clips_hash       ON clips(hash);
`

// Open opens (creating if needed) the database at the default path with
// owner-only permissions and WAL mode.
func Open() (*DB, error) {
	if _, err := paths.EnsureConfigDir(); err != nil {
		return nil, err
	}
	path, err := paths.DBFile()
	if err != nil {
		return nil, err
	}
	return OpenAt(path)
}

// OpenAt opens the database at an explicit path. Used by tests and by Open.
func OpenAt(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One writer at a time avoids "database is locked" under WAL; reads remain
	// concurrent. SQLite serializes writes regardless, so this just makes the
	// pool match reality.
	sqldb.SetMaxOpenConns(1)
	if _, err := sqldb.Exec(schema); err != nil {
		_ = sqldb.Close()
		return nil, err
	}
	if err := chmodIfExists(path); err != nil {
		_ = sqldb.Close()
		return nil, err
	}
	return &DB{sql: sqldb}, nil
}

// Insert writes a clip. CreatedAt and DeviceID default to now / "local" when
// unset.
func (db *DB) Insert(c Clip) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	if c.DeviceID == "" {
		c.DeviceID = DefaultDeviceID
	}
	_, err := db.sql.Exec(
		`INSERT INTO clips (content, nonce, hash, created_at, bytes, device_id) VALUES (?, ?, ?, ?, ?, ?)`,
		c.Content, c.Nonce, c.Hash, c.CreatedAt.Unix(), c.Bytes, c.DeviceID,
	)
	return err
}

// Recent returns the newest clips, most recent first, capped at limit.
func (db *DB) Recent(limit int) ([]Clip, error) {
	rows, err := db.sql.Query(
		`SELECT id, content, nonce, hash, created_at, bytes, device_id
		   FROM clips ORDER BY created_at DESC, id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clips []Clip
	for rows.Next() {
		var c Clip
		var ts int64
		if err := rows.Scan(&c.ID, &c.Content, &c.Nonce, &c.Hash, &ts, &c.Bytes, &c.DeviceID); err != nil {
			return nil, err
		}
		c.CreatedAt = time.Unix(ts, 0)
		clips = append(clips, c)
	}
	return clips, rows.Err()
}

// LatestHash returns the dedup hash of the most recently created clip, or an
// empty string when the database is empty.
func (db *DB) LatestHash() (string, error) {
	var hash string
	err := db.sql.QueryRow(
		`SELECT hash FROM clips ORDER BY created_at DESC, id DESC LIMIT 1`,
	).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return hash, err
}

// Get returns the clip with the given id, or sql.ErrNoRows if absent.
func (db *DB) Get(id int64) (Clip, error) {
	var c Clip
	var ts int64
	err := db.sql.QueryRow(
		`SELECT id, content, nonce, hash, created_at, bytes, device_id FROM clips WHERE id = ?`, id,
	).Scan(&c.ID, &c.Content, &c.Nonce, &c.Hash, &ts, &c.Bytes, &c.DeviceID)
	if err != nil {
		return Clip{}, err
	}
	c.CreatedAt = time.Unix(ts, 0)
	return c, nil
}

// Delete removes the clip with the given id.
func (db *DB) Delete(id int64) error {
	_, err := db.sql.Exec(`DELETE FROM clips WHERE id = ?`, id)
	return err
}

// Prune deletes clips created strictly before the cutoff and returns the number
// removed.
func (db *DB) Prune(before time.Time) (int, error) {
	res, err := db.sql.Exec(`DELETE FROM clips WHERE created_at < ?`, before.Unix())
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// Close closes the underlying database.
func (db *DB) Close() error { return db.sql.Close() }

// chmodIfExists tightens permissions on the database file to owner-only. The
// WAL/SHM sidecar files inherit the umask; we restrict the main file which
// holds the ciphertext at rest.
func chmodIfExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.Chmod(path, paths.FilePerm)
}
