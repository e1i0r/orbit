package board

import (
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
// It is the whole of the polling design: the row the record has been read up
// to, and the events read so far.
type taskState struct {
	// repo is the repository the task is filed under: the first one it
	// joined that this board can see. It is where the checkout, the diff
	// and the editor are opened from, and it is one of repos.
	repo *repoState
	// repos is every repository the task is worked in, oldest join first,
	// including any this window is not opened over. What the row says about
	// where the work went has to be what the record says, not what this
	// directory happens to hold.
	repos []RepoInfo
	id    string
	// at is the row of the record this task has been read up to, and it is
	// the one thing a Reader remembers. It is an optimisation rather than a
	// second copy of the truth: throw the Reader away and the next one
	// reaches the same board by reading from row zero.
	//
	// It is kept per task and not only per reader because a task the window
	// has just found starts behind the others, and because an event handed
	// to a task once must not be handed to it twice — the catch-up read and
	// the stream can overlap, and this is what makes that harmless.
	at int64
	// events is everything read so far, oldest first. view.Fold is a
	// function of the whole log and holds nothing between calls, so the
	// delta is appended here and the fold re-run over the total — which
	// costs a walk of a slice already in memory, and never a second query.
	events []record.Event
	task   view.Task
	// band is the band at the previous refresh, and it is what Entered is a
	// crossing of. A task the reader has not folded yet has the zero Band,
	// view.ToDo, so a task that arrives already needing you does cross.
	band view.Band
	// err is the last read's verdict, kept so that a task whose history
	// could not be read keeps saying so.
	err error
	// seen says a refresh has folded this task at least once. It is what
	// makes a task with nothing recorded — written down and never run —
	// fold once rather than never.
	seen bool
}
