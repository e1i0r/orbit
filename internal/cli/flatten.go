package cli

// The one-time move of a state root written by an older Orbit, where a task
// lived under the repository it was written against.
//
// It runs before every command rather than as a command of its own because
// the version that has to be typed is the version somebody does not type:
// the first thing they run after upgrading is `orbit top`, and a window that
// lists nothing because the tree moved is a bug report, not a prompt. On a
// root that has already moved it reads two directories and does nothing.

import (
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
)

// flatten copies whatever is still filed under a repository up to tasks/.
//
// Nothing it can find is worth refusing to run a command over: the old tree
// is left exactly where it was, so the worst case is a command that reads a
// task that did not move. What it did is written to the log, and what needs
// a person — two repositories holding a task of the same name — is printed
// as well. That line is wiped a moment later when the command is the window,
// which is the same trade the log itself makes; the log keeps it.
func flatten(errOut io.Writer) {
	s, err := store.Open()
	if err != nil {
		logger.Error("cli/flatten", "%v", err)
		return
	}

	moved, err := s.Flatten()
	if len(moved) > 0 {
		logger.Info("cli/flatten", "moved %d task(s) to the flat tree: %s", len(moved), strings.Join(moved, ", "))
	}

	if err != nil {
		logger.Error("cli/flatten", "%v", err)
		fmt.Fprintf(errOut, "orbit: %v\n", err)
	}
}
