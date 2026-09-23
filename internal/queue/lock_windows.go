//go:build windows

package queue

import "github.com/e1i0r/orbit/internal/store"

// lockQueue is nothing here: Windows has no flock, and two starts racing
// for the last slot can both take it. The limit is kept loosely rather
// than not at all.
func lockQueue(_ *store.Store) (func(), error) { return func() {}, nil }

// lockFile always succeeds here, for the reason lockQueue does.
func lockFile(_ string, _ bool) (func(), error) { return func() {}, nil }

// processAlive cannot be asked here; a pid written down is taken at its word.
func processAlive(pid int) bool { return pid > 0 }
