//go:build unix

package main

import "syscall"

// hardenUmask restricts the process file-creation mask so every file yoink
// creates — including SQLite's WAL/SHM sidecars, which inherit the umask rather
// than history.db's explicit 0600 — is owner-only, and every directory
// owner-only, regardless of the inherited login umask.
func hardenUmask() { syscall.Umask(0o077) }
