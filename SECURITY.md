# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

- Preferred: open a private report via GitHub's
  [**Report a vulnerability**](https://github.com/davcevski/yoink/security/advisories/new)
  (Security tab → Advisories).

You'll get an acknowledgement as soon as possible, and a fix or mitigation will
be coordinated before any public disclosure.

## Threat model

yoink stores clipboard history, which routinely contains passwords, tokens, and
keys. The history is **encrypted at rest** to protect against:

- a **stolen laptop**, and
- **exfiltration of `history.db`** (copying the database file off the machine).

### What protects your data

- **AES-256-GCM** encrypts every clip before it is written. The database holds
  only ciphertext + a per-clip random nonce, so a copied `history.db` never
  reveals clip **contents** without the key. It does keep per-clip metadata in
  the clear — timestamp, plaintext byte length, and a keyed dedup hash — so a
  copy still discloses *when*, *how big*, and *which clips repeat*, just not what
  they say.
- The **256-bit key lives in the macOS login Keychain**, never in the config
  file or the database. The daemon and TUI read it at runtime.
- **Deduplication uses a keyed HMAC-SHA256**, so duplicate detection never
  stores or compares plaintext.
- **Secrets are skipped:** clips flagged concealed/transient on the pasteboard
  (used by password managers) never touch the disk.
- The daemon **pauses capture** rather than writing plaintext if the key is
  missing or unreadable.
- Decrypted plaintext exists only in process memory for a TUI session and is
  never written back to disk.
- `history.db`, its SQLite `-wal`/`-shm` sidecars, and the daemon log are all
  created owner-only (`0600`); the config directory is `0700`. yoink sets a
  `0077` umask at startup so nothing it writes is group/world-readable, and
  provisions the log file itself so launchd cannot create it world-readable.

### Out of scope

- A **live memory dump** of an unlocked, running machine.
- Full-disk protection — that's **FileVault's** job (yoink's "Layer 1"
  baseline). `yoink install` warns when FileVault is off.
- Image/file/rich clipboard contents (v1 captures **text only**).
