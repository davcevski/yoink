//go:build !unix

package main

// hardenUmask is a no-op off unix. yoink targets macOS; this stub only keeps
// cross-platform vet/build green (mirroring the keyring/clipboard stubs).
func hardenUmask() {}
