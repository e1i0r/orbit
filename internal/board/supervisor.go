package board

import (
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// SupervisorLog reads the global supervisor thread and folds it into the
// lines a reader is shown.
//
// It is a function and not only a method because the thread is one file
// under the state root, and a caller that wants it wants no board at all:
// internal/cli and the MCP server were each building a Reader with an empty
// root to reach the method. NewReader's own doc argues that a Reader with
// no root is a constructor called wrong, and it is — repo.Discover resolves
// "" to the process's working directory, so one Refresh on either of those
// would have walked wherever orbit happened to be started from.
func SupervisorLog(s *store.Store) ([]view.SupervisorLine, error) {
	if s == nil {
		return nil, nil
	}

	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	// A thread nobody has written to yet is an empty thread and not a fault,
	// which is what every reader sees on their first day. It used to take a
	// check for a missing file to say so; a table with no rows says it on
	// its own.
	events, err := d.Messages()
	if err != nil {
		return nil, err
	}

	return view.SupervisorThread(events), nil
}

// SupervisorLog is the same thread, for a caller that has a Reader in hand.
func (r *Reader) SupervisorLog() ([]view.SupervisorLine, error) {
	if r == nil {
		return nil, nil
	}

	return SupervisorLog(r.store)
}
