// tools.go anchors the six modules in arch.approved as direct requires
// before any package imports them for real. Nothing in this task uses
// these packages — the window that will is task 10 — but an unused
// `require` is exactly what `go mod tidy` prunes, and pruning it here would
// undo the decision this task exists to record. This file is that decision
// made visible to the toolchain: task 10 deletes it once internal/ui
// imports these packages on its own.
package arch

import (
	_ "charm.land/bubbles/v2"
	_ "charm.land/bubbletea/v2"
	_ "charm.land/lipgloss/v2"
	_ "github.com/charmbracelet/colorprofile"
	_ "github.com/charmbracelet/x/ansi"
	_ "github.com/charmbracelet/x/exp/golden"
)
