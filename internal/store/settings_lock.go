package store

// The lock a settings change takes, and how it is given back.

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/logger"
)

// lockPatience is how long a change waits for another one to finish, and
// lockStale is when a lock file is taken for the leftover of a process that
// died holding it.
//
// A change holds the lock for one read and one write of a file measured in
// hundreds of bytes, so two seconds is thousands of turns and a minute is
// nobody's. The alternative to breaking a stale lock is a settings file that
// no longer accepts changes because something was killed at the wrong
// instant, and the person it happens to has no way to know what to delete.
const (
	lockPatience = 2 * time.Second
	lockStale    = time.Minute
)

// lockSettings takes the settings lock and answers how to give it back.
//
// A file created with O_EXCL is the lock, because a file system promising
// that exactly one of two creations succeeds is the one thing every one of
// them promises across processes, without a library and without a syscall
// this program would have to write twice for two operating systems.
func (s *Store) lockSettings() (func() error, error) {
	path := s.settingsPath() + lockSuffix
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return nil, fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}

	for waited := time.Duration(0); ; waited += lockPoll {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
		if err == nil {
			token, stampErr := stamp(f)
			if stampErr != nil {
				return nil, errors.Join(stampErr, f.Close(), os.Remove(path))
			}

			return release(f, path, token), nil
		}

		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("take the settings lock %q: %w", path, err)
		}

		if broken, err := breakStale(path); err != nil {
			return nil, err
		} else if broken {
			continue
		}

		if waited >= lockPatience {
			return nil, fmt.Errorf("another orbit has been changing the settings for %v; if none is, delete %q", lockPatience, path)
		}

		time.Sleep(lockPoll)
	}
}

// lockPoll is how often the wait looks again. It is short enough that an
// ordinary change is not noticeably delayed by having waited for one.
const lockPoll = 10 * time.Millisecond

// lockSuffix is what the settings lock is called, beside the file it guards
// rather than in a directory of its own: whoever is told to delete it finds
// it in the listing they are already looking at.
const lockSuffix = ".lock"

// stamp writes an identity into a lock just taken.
//
// A random token and not a pid: breakStale says below why a pid is the wrong
// evidence on a state root shared between machines, and this needs no process
// table — it only has to differ from whatever is written next.
func stamp(f *os.File) (string, error) {
	var raw [16]byte

	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("stamp the settings lock: %w", err)
	}

	token := hex.EncodeToString(raw[:])
	if _, err := f.WriteString(token); err != nil {
		return "", fmt.Errorf("stamp the settings lock: %w", err)
	}

	return token, nil
}

// holds is whether the lock file still carries this token.
func holds(path, token string) (bool, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("read the settings lock %q: %w", path, err)
	}

	return strings.TrimSpace(string(body)) == token, nil
}

// release gives the settings lock back.
//
// A lock that is already gone is not a failure of the change that held it:
// another process breaking it as stale removes the file, and answering the
// resulting ENOENT made `orbit set` report a failure for a setting that had
// been written — and the window, reading that failure, snap the switch back
// to what it had just stopped being.
//
// The token is checked before the file is removed, so a lock broken as stale
// and taken by somebody else is left with whoever holds it now rather than
// pulled out from under them.
func release(f *os.File, path, token string) func() error {
	return func() error {
		closeErr := f.Close()

		held, err := holds(path, token)
		if err != nil || !held {
			return errors.Join(closeErr, err)
		}

		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.Join(closeErr, fmt.Errorf("give back the settings lock %q: %w", path, err))
		}

		return closeErr
	}
}

// breakStale removes a lock nobody is holding, and answers whether it did.
//
// Age is the only evidence available. A pid in the file would be better on
// one machine and worse across a state root on a shared disk, where the
// number belongs to somebody else's process table.
func breakStale(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		// It went away between the failed create and this look, which is
		// the holder finishing. The next turn of the loop takes it.
		return false, nil //nolint:nilerr // a lock that is gone is not a fault
	}

	if time.Since(info.ModTime()) < lockStale {
		return false, nil
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("remove the stale settings lock %q: %w", path, err)
	}
	// Breaking it is the right thing to do and no reason to do it quietly.
	// A lock older than a minute is the footprint of a process that died
	// between taking it and giving it back, and that is worth knowing about
	// on its own — the change that follows will succeed and say nothing.
	logger.Warn("store/settings", "broke a settings lock left behind %s ago: something died holding it",
		time.Since(info.ModTime()).Round(time.Second))

	return true, nil
}
