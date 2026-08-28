package store

// Putting one small file on disk in one step, for the two files under the
// state root that are read while they are being written.

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomically puts body at path in one step: a temporary file beside it,
// flushed, then renamed over the top.
//
// os.WriteFile truncates first and writes second, and everything between
// those two is a file that exists and is shorter than it should be. What
// that costs depends on who reads it. A settings.json caught there will not
// parse, and a process killed at that point — or a machine that loses power —
// leaves exactly the file keepUnreadable was written to cope with, so this is
// the other half of the same fix: one stops the loss, this stops the damage
// that causes it. A run marker caught there names no pid, and its reader is
// deliberately unforgiving about that (see task/alive.go): the board reads
// every marker twice a second, and a run refuses to start over a claim it
// cannot rule out.
//
// The temporary is made in the same directory rather than in the system's
// temp dir, because a rename is atomic only within one filesystem; across
// two it is a copy, which is the thing being avoided.
func WriteAtomically(path string, body []byte) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create a temporary file beside %q: %w", path, err)
	}
	tmp := f.Name()
	// Every path that does not end in the rename takes the temporary with
	// it, so a write that failed leaves the directory as it found it rather
	// than littered with half-written files.
	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()
	if err = f.Chmod(fileMode); err != nil {
		return fmt.Errorf("set the mode of %q: %w", tmp, err)
	}
	if _, err = f.Write(body); err != nil {
		return fmt.Errorf("write %q: %w", tmp, err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("flush %q: %w", tmp, err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("close %q: %w", tmp, err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace %q with %q: %w", path, tmp, err)
	}
	return nil
}
