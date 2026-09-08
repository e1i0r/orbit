// Package ui is the browser half of Orbit, built and carried inside the
// binary.
//
// The Go file lives beside the sources it embeds rather than beside the
// server that serves them, because go:embed cannot reach outside its own
// directory and copying the build output into internal/web would leave two
// of it. dist/ is committed for the reason site/ is: the alternative is a
// build step that has to run before `go build` on every machine, and a Go
// project that needs npm to compile is a Go project that does not.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var built embed.FS

// Files is what was built, rooted so that a request for "/" finds
// index.html.
func Files() (fs.FS, error) {
	return fs.Sub(built, "dist")
}
