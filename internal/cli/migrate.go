package cli

// Filling the database from the files an older Orbit wrote.
//
// It runs before every command for the reason flatten does: the version that
// has to be typed is the version nobody types. Unlike flatten it changes
// nothing anybody reads yet — the files are still what every reader reads —
// so the worst a failure here can do is leave the database behind, and a
// command refusing to run over that would be a command refusing for nothing.

import (
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/migrate"
	"github.com/e1i0r/orbit/internal/store"
)

// migrateRecord copies whatever the logs hold and the database does not.
func migrateRecord(errOut io.Writer) {
	s, err := store.Open()
	if err != nil {
		logger.Error("cli/migrate", "%v", err)
		return
	}

	out, err := migrate.Run(s)
	if out.Moved() {
		logger.Info("cli/migrate", "copied %s into the database", out)
	}

	if err != nil {
		logger.Error("cli/migrate", "%v", err)
		fmt.Fprintf(errOut, "orbit: %v\n", err)
	}
}
