// Package ui is the browser half of Orbit, built and carried inside the
// binary.
//
// The Go file lives beside the sources it embeds rather than beside the
// server that serves them, because go:embed cannot reach outside its own
// directory and copying the build output into internal/web would leave two
// of it.
//
// dist/ is not committed. It is 365 kB of minified bundle that changes on
// every edit to the web, and a repository whose diffs are mostly rebuilt
// JavaScript is one whose diffs nobody reads. `make ui` writes it, and CI
// writes it before it builds, tests or releases anything.
//
// What *is* committed is dist/.keep, so that this embed has a directory to
// find: a Go project that will not compile without npm is a Go project that
// does not build, and go:embed refuses a pattern that matches nothing. A
// binary built without `make ui` therefore has no window, and says so —
// see Files below and internal/web/page.go.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var built embed.FS

// Files is what was built, rooted so that a request for "/" finds
// index.html.
//
// A binary compiled without `make ui` carries only the .keep file, and this
// answers the empty window rather than an error: there is nothing wrong with
// such a binary — everything but `orbit web` works — and the one command
// that needs the window says so itself, in a sentence that names what to
// run.
func Files() (fs.FS, error) {
	return fs.Sub(built, "dist")
}

// Built reports whether this binary carries a window at all.
func Built() bool {
	_, err := fs.Stat(built, "dist/index.html")

	return err == nil
}
