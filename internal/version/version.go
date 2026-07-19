// Package version holds the PieKBS binary version, injected at build time via:
//
//	go build -ldflags "-X github.com/pieteams/piekbs/internal/version.Version=0.4.7"
//
// Defaults to "dev" for local development.
package version

var Version = "dev"
