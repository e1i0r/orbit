package task

// Whether an id is in use, and clearing what an older deletion left behind.

import (
	"fmt"
	"os"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// clearIfDeleted gives up the directory a deleted task left behind, and
// refuses the id when the record says the task is still there.
//
// Two tasks with one name would share a directory and interleave their
// events, so a live id is refused. A dead one is not a task: the reader who
// deleted it sees nothing on the board, and answering "already exists" about
// it is the window contradicting itself.
func clearIfDeleted(s *store.Store, id string) error {
	gone, err := deletedAlready(s, id)
	if err != nil {
		return err
	}

	if !gone {
		return fmt.Errorf("task %q %w", id, ErrExists)
	}

	dir, err := s.TaskDir(id)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clear what %q left behind: %w", id, err)
	}

	return nil
}

// deletedAlready is whether the record's last word on this id is that it was
// deleted.
//
// The last word and not "was it ever": an id written down, deleted and
// written down again is a name in use, and the second life is what the board
// is showing.
func deletedAlready(s *store.Store, id string) (bool, error) {
	d, err := s.Record()
	if err != nil {
		return false, err
	}

	events, err := d.Events(id)
	if err != nil {
		return false, fmt.Errorf("read what the record says about %q: %w", id, err)
	}

	gone := false

	for _, e := range events {
		switch e.Kind {
		case record.TaskCreated:
			gone = false
		case record.TaskDeleted:
			gone = true
		}
	}

	return gone, nil
}
