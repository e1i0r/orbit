package board

import (
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/view"
)

// repoState is one repository the board knows about.
//
// path is git's top level, which is both what repo.Discover answers and what
// task.Create filed this repository's record under. The store hashes that
// path into the directory name everything under repos/ is reached by, so the
// two have to be the same string: a path that had passed through a symlink
// unresolved would look up a directory that does not exist and lose every
// task in it. Both sides go through repo.Open, so both sides resolve.
type repoState struct {
	path string
	name string
}

// taskState is what the Reader remembers about one task between refreshes.
// It is the whole of the polling design: an offset, the size the last stat
// saw, and the events read so far.
type taskState struct {
	repo *repoState
	id   string
	path string // the task's events.jsonl
	// offset is the next byte to read, and it is only ever what
	// record.ReadFrom answered — never arithmetic of this package's own.
	// That matters because ReadFrom advances the offset past a complete,
	// newline-terminated line and no further: a torn final line is a write
	// in flight rather than damage, and holding the offset before it is
	// what makes the next read pick the line up once it lands.
	offset int64
	// size is the file size the last stat saw. An unchanged size is the
	// cheap answer this design is built on: nothing was appended, so there
	// is nothing to open, read or parse.
	size int64
	// modTime is the modification time seen at the last stat.
	modTime time.Time
	// events is everything read so far, oldest first. view.Fold is a
	// function of the whole log and holds nothing between calls, so the
	// delta is appended here and the fold re-run over the total — which
	// costs a walk of a slice already in memory, and never a re-read of the
	// file. That is the trade the offsets buy: the I/O and the JSON are
	// what polling has to avoid, not the switch statement.
	events []record.Event
	task   view.Task
	// band is the band at the previous refresh, and it is what Entered is a
	// crossing of. A task the reader has not folded yet has the zero Band,
	// view.ToDo, so a task that arrives already needing you does cross.
	band view.Band
	// err is the last read's verdict, kept so that a log nobody can read
	// keeps saying so on the refreshes that skip it for being unchanged.
	err error
	// seen says a refresh has folded this task at least once. It is what
	// makes a task with an empty log — written down and never run — fold
	// once rather than never.
	seen bool
}
