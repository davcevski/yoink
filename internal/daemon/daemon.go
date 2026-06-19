// Package daemon runs yoink's always-on capture loop: it polls the clipboard,
// encrypts new text clips, and prunes expired ones. Lifecycle is orchestrated
// with a context and tickers (channels); all mutable runtime state is confined
// to the single capture goroutine, so no mutex is needed.
package daemon

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"github.com/davcevski/yoink/internal/clipboard"
	"github.com/davcevski/yoink/internal/config"
	"github.com/davcevski/yoink/internal/crypto"
	"github.com/davcevski/yoink/internal/store"
)

// Store is the subset of the clip store the daemon writes through. Defined here,
// at the consumer, so the abstraction stays narrow.
type Store interface {
	Insert(store.Clip) error
	LatestHash() (string, error)
	Prune(before time.Time) (int, error)
}

// keyReader reads the encryption key. The daemon never generates a key; if it
// is missing the daemon pauses capture rather than writing plaintext.
type keyReader interface {
	Get() ([]byte, error)
}

// pruneInterval is how often expired clips are removed after the startup prune.
const pruneInterval = time.Hour

// Daemon captures clipboard changes into the store.
type Daemon struct {
	clip   clipboard.Pasteboard
	store  Store
	keys   keyReader
	cfg    config.Config
	logger *log.Logger

	// Runtime state, owned exclusively by the capture goroutine.
	lastSeen   int
	cipher     *crypto.Cipher
	keyMissing bool
}

// New constructs a Daemon. A nil logger discards output.
func New(clip clipboard.Pasteboard, st Store, keys keyReader, cfg config.Config, logger *log.Logger) *Daemon {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Daemon{clip: clip, store: st, keys: keys, cfg: cfg, logger: logger}
}

// Run drives the poll and prune loops until ctx is cancelled. It returns nil on
// graceful shutdown.
func (d *Daemon) Run(ctx context.Context) error {
	d.logger.Printf("yoink daemon started (poll=%dms retention=%dd max_clip=%dB)",
		d.cfg.PollIntervalMS, d.cfg.RetentionDays, d.cfg.MaxClipBytes)

	// lastSeen starts at zero so the first poll evaluates whatever is already on
	// the clipboard (e.g. something copied while the daemon was restarting).
	// Keyed-hash dedup against the latest stored clip prevents re-storing
	// unchanged content across KeepAlive restarts.
	d.prune() // prune once at startup

	poll := time.NewTicker(time.Duration(d.cfg.PollIntervalMS) * time.Millisecond)
	defer poll.Stop()
	pruneTick := time.NewTicker(pruneInterval)
	defer pruneTick.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Println("yoink daemon stopped")
			return nil
		case <-poll.C:
			d.capture()
		case <-pruneTick.C:
			d.prune()
		}
	}
}

// capture inspects the clipboard once and stores a new clip if warranted. It
// implements the design's capture pseudocode.
func (d *Daemon) capture() {
	n := d.clip.ChangeCount()
	if n == d.lastSeen {
		return // nothing copied
	}
	d.lastSeen = n

	text, concealed, ok := d.clip.Read()
	if !ok || concealed || strings.TrimSpace(text) == "" {
		return // non-text, concealed/transient secret, or whitespace-only
	}
	if len(text) > d.cfg.MaxClipBytes {
		return // too big to keep
	}

	cipher, err := d.loadCipher()
	if err != nil {
		if !d.keyMissing {
			d.logger.Printf("encryption key unavailable, pausing capture: %v", err)
			d.keyMissing = true
		}
		return // never write plaintext
	}
	if d.keyMissing {
		d.logger.Println("encryption key available, resuming capture")
		d.keyMissing = false
	}

	plaintext := []byte(text)
	hash := cipher.Hash(plaintext)

	latest, err := d.store.LatestHash()
	if err != nil {
		d.logger.Printf("dedup check failed: %v", err)
		return
	}
	if hash == latest {
		return // consecutive duplicate copy
	}

	ciphertext, nonce, err := cipher.Seal(plaintext)
	if err != nil {
		d.logger.Printf("encrypt failed: %v", err)
		return
	}
	if err := d.store.Insert(store.Clip{
		Content:   ciphertext,
		Nonce:     nonce,
		Hash:      hash,
		CreatedAt: time.Now(),
		Bytes:     len(plaintext),
	}); err != nil {
		d.logger.Printf("store insert failed: %v", err)
	}
}

// loadCipher lazily builds (and caches) the cipher from the stored key so the
// daemon resumes automatically once a missing key appears.
func (d *Daemon) loadCipher() (*crypto.Cipher, error) {
	if d.cipher != nil {
		return d.cipher, nil
	}
	key, err := d.keys.Get()
	if err != nil {
		return nil, err
	}
	c, err := crypto.New(key)
	if err != nil {
		return nil, err
	}
	d.cipher = c
	return c, nil
}

// prune removes clips older than the configured retention window.
func (d *Daemon) prune() {
	cutoff := time.Now().AddDate(0, 0, -d.cfg.RetentionDays)
	n, err := d.store.Prune(cutoff)
	if err != nil {
		d.logger.Printf("prune failed: %v", err)
		return
	}
	if n > 0 {
		d.logger.Printf("pruned %d expired clip(s)", n)
	}
}
