package cli

// Keeping the derived index up with the record, which is one call: the
// projector reads the tail of every log that grew since the last time and
// folds it. On a tree nobody has touched it reads nothing.

import (
	"io"

	"github.com/e1i0r/orbit/internal/index"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
)

// project folds whatever the record has gained into the index.
//
// It runs before the command rather than after it so that a reader is asking
// about the record as it was when it was asked, and it is here rather than
// inside each command for the same reason the log is opened here: a write
// path every caller has to remember is a write path somebody forgets.
//
// Nothing it can go wrong at is worth refusing to run a command over. The
// index holds nothing the record does not, so a fold that failed costs a
// question answered from an index that is behind — and the command that was
// typed reads the record, as it always has.
func project(errOut io.Writer) {
	s, err := store.Open()
	if err != nil {
		logger.Error("cli/index", "%v", err)
		return
	}

	x, err := index.Open(s.IndexPath())
	if err != nil {
		logger.Error("cli/index", "%v", err)
		return
	}

	defer func() {
		if err := x.Close(); err != nil {
			logger.Error("cli/index", "%v", err)
		}
	}()

	folded, err := index.Build(x, s)
	if folded > 0 {
		logger.Info("cli/index", "folded %d event(s) into the index", folded)
	}

	if err != nil {
		logger.Error("cli/index", "%v", err)
	}
}
