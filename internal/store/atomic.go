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

// WriteIfAbsent puts body at path and answers whether it got there, which
// is no when something already holds that name.
//
// The claim and the content in one step, which is what a file naming who
// holds something has to be. Two processes that both looked, both saw
// nothing and both wrote would both believe they held it; here the
// operating system decides, and it decides once — a hard link fails rather
// than replacing what is there, and it is the only create that both
// refuses an existing name and lands a file with its content already in
// it. A rename would replace the other claim; an O_EXCL open would publish
// an empty file and fill it afterwards, which is the half-written marker
// WriteAtomically exists to avoid.
//
// The temporary lives in the same directory, for the reason it does above:
// a link across filesystems is not a link.
func WriteIfAbsent(path string, body []byte) (written bool, err error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return false, fmt.Errorf("create a temporary file beside %q: %w", path, err)
	}

	tmp := f.Name()

	// The temporary goes either way: on the way out it has either been
	// linked into place, in which case this removes the second name, or
	// it has not, in which case it is litter.
	defer func() { _ = os.Remove(tmp) }() //nolint:errcheck // nothing to tell, and nothing reads it

	if err = writeWhole(f, tmp, body); err != nil {
		return false, err
	}

	if err = os.Link(tmp, path); err != nil {
		if os.IsExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("claim %q: %w", path, err)
	}

	return true, nil
}

// writeWhole is the middle of both writers: mode, bytes, flush, close.
func writeWhole(f *os.File, tmp string, body []byte) error {
	if err := f.Chmod(fileMode); err != nil {
		_ = f.Close() //nolint:errcheck // the error being reported is the one above

		return fmt.Errorf("set the mode of %q: %w", tmp, err)
	}

	if _, err := f.Write(body); err != nil {
		_ = f.Close() //nolint:errcheck // the error being reported is the one above

		return fmt.Errorf("write %q: %w", tmp, err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close() //nolint:errcheck // the error being reported is the one above

		return fmt.Errorf("flush %q: %w", tmp, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close %q: %w", tmp, err)
	}

	return nil
}
