// tools.go anchors the modules in arch.approved that no package imports for
// real yet.
//
// It started as all six, written before internal/ui existed. Five of them —
// bubbletea, bubbles, lipgloss, colorprofile and x/ansi — are now imported
// by the window and its tests, so they hold their own place in go.mod and
// their blank imports are gone. x/exp/golden is the one left: the golden
// helper in internal/ui compares against filenames a test function's own
// name could not produce, so it does not use that package's RequireEqual,
// and an approved module nothing imports is exactly what `go mod tidy`
// prunes. `make check` runs `go mod tidy -diff`, so the prune would fail the
// build rather than pass quietly. This import is the decision to keep it
// made visible to the toolchain.
package arch

import (
	_ "github.com/charmbracelet/x/exp/golden"
)
