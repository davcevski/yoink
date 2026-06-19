// Package crypto provides yoink's at-rest encryption: AES-256-GCM for clip
// contents and a keyed HMAC-SHA256 for plaintext-free deduplication. It uses
// only the Go standard library — no third-party crypto.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// KeySize is the required key length in bytes (256-bit).
const KeySize = 32

// ErrKeySize is returned by New when the key is not exactly KeySize bytes.
var ErrKeySize = fmt.Errorf("crypto: key must be %d bytes", KeySize)

// Cipher encrypts and authenticates clip contents and derives keyed dedup
// hashes. It is safe for concurrent use: the underlying AEAD is stateless and
// each Seal generates its own random nonce.
type Cipher struct {
	aead cipher.AEAD
	key  []byte // retained only for HMAC keying
}

// New returns a Cipher bound to key, which must be exactly KeySize bytes.
func New(key []byte) (*Cipher, error) {
	if len(key) != KeySize {
		return nil, ErrKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	dup := make([]byte, len(key))
	copy(dup, key)
	return &Cipher{aead: aead, key: dup}, nil
}

// NewKey returns a freshly generated, cryptographically random 256-bit key.
func NewKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// Seal encrypts plaintext, returning the ciphertext and the per-call random
// nonce. The nonce must be stored alongside the ciphertext and passed back to
// Open. A fresh nonce is generated for every call, so identical plaintexts
// still produce distinct ciphertexts.
func (c *Cipher) Seal(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = c.aead.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Open authenticates and decrypts ciphertext using nonce. It returns an error
// if the data has been tampered with or the nonce is the wrong size.
func (c *Cipher) Open(ciphertext, nonce []byte) ([]byte, error) {
	if len(nonce) != c.aead.NonceSize() {
		return nil, errors.New("crypto: invalid nonce size")
	}
	return c.aead.Open(nil, nonce, ciphertext, nil)
}

// Hash returns the hex-encoded HMAC-SHA256 of plaintext keyed with the cipher's
// key. It lets the daemon detect duplicate clips without storing or comparing
// plaintext, and is stable across process restarts for a given key.
func (c *Cipher) Hash(plaintext []byte) string {
	mac := hmac.New(sha256.New, c.key)
	mac.Write(plaintext)
	return hex.EncodeToString(mac.Sum(nil))
}

// Zero overwrites the cipher's retained key material. Best-effort hygiene for
// shutdown; the Cipher must not be used afterwards.
func (c *Cipher) Zero() {
	for i := range c.key {
		c.key[i] = 0
	}
}
