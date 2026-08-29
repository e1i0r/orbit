package board

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/e1i0r/orbit/internal/view"
)

// Files is what one task's own directory holds right now.
//
// It is read from disk on every ask rather than cached beside the events:
// the record grows by appending and can be followed by offset, and a
// directory cannot — a file appears, and nothing in the events says so.
//
// A task that has not run yet has no directory, and that is an empty listing
// rather than a failure: it is the same answer a task whose run left nothing
// gets, and it is the true one.
//
// The listing is in name order because os.ReadDir answers in name order and
// filtering keeps it. That order is what the pane draws, and a listing that
// came back arranged differently between two polls of the same second would
// move the row a reader is reaching for.
func (r *Reader) Files(repoPath, id string) ([]view.File, error) {
	dir, err := r.store.TaskDir(repoPath, id)
	if err != nil {
		return nil, fmt.Errorf("locate the directory of task %s: %w", id, err)
	}

	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read the directory of task %s: %w", id, err)
	}

	out := make([]view.File, 0, len(entries))

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		// A file that has gone between the listing and the measure is a file
		// the run removed while this was reading, and the true listing is
		// the one without it. Any other failure is the caller's to see.
		info, err := e.Info()
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("measure %s of task %s: %w", e.Name(), id, err)
		}

		out = append(out, view.File{Name: e.Name(), Size: info.Size()})
	}

	return out, nil
}
