//go:build !windows

package queue

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
)

// lockQueue holds the queue while it is read and a run is started from it,
// so that two starts cannot both see the last free slot. flock, because the
// kernel lets go of it when the process holding it dies, and a lock left
// behind by a crash would stop every run on the machine.
func lockQueue(s *store.Store) (func(), error) {
	return lockFile(filepath.Join(s.Root(), "queue.lock"), true)
}

// lockFile takes an exclusive lock on path, waiting for it or not, and
// answers how to let it go. errBusy is a lock somebody else holds, when
// not waiting.
func lockFile(path string, wait bool) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open the queue's lock %q: %w", path, err)
	}

	how := syscall.LOCK_EX
	if !wait {
		how |= syscall.LOCK_NB
	}

	if err := syscall.Flock(int(f.Fd()), how); err != nil {
		closeErr := f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, errors.Join(errBusy, closeErr)
		}

		return nil, errors.Join(fmt.Errorf("take the queue's lock %q: %w", path, err), closeErr)
	}

	// Closing the file is what lets go of the lock.
	return func() {
		if err := f.Close(); err != nil {
			logger.Warn("queue", "let go of the queue's lock %q: %v", path, err)
		}
	}, nil
}
