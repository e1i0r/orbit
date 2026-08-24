package task

// The review verb, and the brake it feeds.

import (
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// MarkRead writes down that somebody has looked at a task.
//
// One event and no phase: being read is a fact about the task, not about
// anything that ran. It is not refused a second time, because the log is
// append-only and two lines saying the same thing is what an append-only log
// looks like when somebody says it twice — internal/view folds any number of
// them into one Read.
func MarkRead(s *store.Store, t Task) error {
	return emit(s, t, record.Event{Kind: record.TaskRead})
}

// atCap says whether the unread count has reached the cap, and it is the one
// place that decides.
//
// A limit of zero is no cap at all. That is a setting a user chooses and not
// a fact about never having chosen: store.Settings fills in a real number for
// a file that was never written, precisely so the Go zero can keep meaning
// "let everything through".
//
// The count itself is not computed here, and deliberately. internal/view
// folds a record into whether a task is finished and whether it has been
// read, internal/board counts those, and a second fold living in this package
// would be two readers of one format — the drift internal/record/kind.go was
// created to end. Start is handed the number by whoever already holds the
// board.
func atCap(unread, limit int) bool {
	return limit > 0 && unread >= limit
}
