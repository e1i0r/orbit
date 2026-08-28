package board

import (
	"errors"
	"os"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/view"
)

// SupervisorLog reads the global supervisor thread and folds it into viewable lines.
func (r *Reader) SupervisorLog() ([]view.SupervisorLine, error) {
	if r == nil || r.store == nil {
		return nil, nil
	}
	events, err := record.Read(r.store.SupervisorLogPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return view.SupervisorThread(events), nil
}
