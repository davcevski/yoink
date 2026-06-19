// Package buildinfo exposes release metadata injected at build time through
// -ldflags. The zero values below are used for `go run` / `go build` without
// linker flags so the binary always reports something sensible.
package buildinfo

// These variables are overwritten at link time by GoReleaser, e.g.
//
//	-X github.com/davcevski/yoink/internal/buildinfo.Version=v1.2.3
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String renders a single-line, human-readable build identifier.
func String() string {
	return Version + " (" + Commit + ", built " + Date + ")"
}
