package crypto

import (
	"bytes"
	"testing"
)

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()
	key, err := NewKey()
	if err != nil {
		t.Fatalf("NewKey: %v", err)
	}
	c, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNewRejectsBadKeySize(t *testing.T) {
	for _, n := range []int{0, 1, 16, 31, 33, 64} {
		if _, err := New(make([]byte, n)); err != ErrKeySize {
			t.Errorf("New(len=%d): got %v, want ErrKeySize", n, err)
		}
	}
}

func TestSealOpenRoundTrip(t *testing.T) {
	c := newTestCipher(t)
	cases := [][]byte{
		nil,
		[]byte(""),
		[]byte("hello"),
		bytes.Repeat([]byte("x"), 100_000),
	}
	for _, pt := range cases {
		ct, nonce, err := c.Seal(pt)
		if err != nil {
			t.Fatalf("Seal: %v", err)
		}
		got, err := c.Open(ct, nonce)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if !bytes.Equal(got, pt) {
			t.Errorf("round-trip mismatch: got %q want %q", got, pt)
		}
	}
}

func TestOpenRejectsTamperedCiphertext(t *testing.T) {
	c := newTestCipher(t)
	ct, nonce, err := c.Seal([]byte("secret token"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	ct[0] ^= 0xFF
	if _, err := c.Open(ct, nonce); err == nil {
		t.Fatal("Open accepted tampered ciphertext, want auth error")
	}
}

func TestOpenRejectsWrongNonceSize(t *testing.T) {
	c := newTestCipher(t)
	ct, _, err := c.Seal([]byte("data"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := c.Open(ct, []byte("short")); err == nil {
		t.Fatal("Open accepted wrong-size nonce")
	}
}

func TestSealUsesUniqueNonces(t *testing.T) {
	c := newTestCipher(t)
	pt := []byte("same content")
	_, n1, _ := c.Seal(pt)
	ct2, n2, _ := c.Seal(pt)
	if bytes.Equal(n1, n2) {
		t.Error("nonces repeated across Seal calls")
	}
	// Identical plaintext must still produce different ciphertext.
	ct1, _, _ := c.Seal(pt)
	if bytes.Equal(ct1, ct2) {
		t.Error("identical plaintext produced identical ciphertext")
	}
}

func TestHashIsDeterministicAndKeyed(t *testing.T) {
	key, _ := NewKey()
	c1, _ := New(key)
	c2, _ := New(key)
	pt := []byte("dedup me")

	if c1.Hash(pt) != c2.Hash(pt) {
		t.Error("Hash not deterministic for same key + plaintext")
	}
	if c1.Hash(pt) == c1.Hash([]byte("other")) {
		t.Error("Hash collided for different plaintext")
	}

	other := newTestCipher(t)
	if c1.Hash(pt) == other.Hash(pt) {
		t.Error("Hash not keyed: same plaintext hashed equally under different keys")
	}
}

func TestZeroClearsKey(t *testing.T) {
	c := newTestCipher(t)
	c.Zero()
	for i, b := range c.key {
		if b != 0 {
			t.Fatalf("key byte %d not zeroed: %d", i, b)
		}
	}
}
