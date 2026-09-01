package cli

// Filling the database from the files an older Orbit wrote.
//
// It runs before every command for the reason flatten does: the version that
// has to be typed is the version nobody types.

import (
	"errors"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/migrate"
	"github.com/e1i0r/orbit/internal/store"
)

// migrateRecord copies whatever the logs hold and the database does not, and
// reports whether the command after it may run.
//
// A failure here stops the command, which is the opposite of what flatten
// beside it does. Flatten leaves the old tree where it was, so its worst case
// is a task that did not move; this one fills the record every reader now
// reads, and a record half filled does not look like a file that was missed.
// It looks like a task that never had those events.
//
// The ordering matters more than the reading. Events written into the record
// while the copy is unfinished land in front of the lines still waiting to be
// copied, and the record is read in order — so a command allowed to run over
// a failure here is how a run's history ends up interleaved with itself.
//
// The store is let go of at the end because the command about to run opens
// its own, and the whole point of a handle per process is that there is one.
func migrateRecord(errOut io.Writer) bool {
	s, err := store.Open()
	if err != nil {
		logger.Error("cli/migrate", "%v", err)
		fmt.Fprintf(errOut, "orbit: %v\n", err)

		return false
	}

	out, err := migrate.Run(s)
	if out.Moved() {
		logger.Info("cli/migrate", "copied %s into the database", out)
	}

	err = errors.Join(err, s.Close())
	if err != nil {
		logger.Error("cli/migrate", "%v", err)
		fmt.Fprintf(errOut, "orbit: %v\n", err)

		return false
	}

	return true
}
