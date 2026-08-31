package store

// The move from repos/<key>/tasks/<id> to tasks/<id>, run once against a
// state root written by an older Orbit.
//
// It copies and verifies, and deletes nothing at all. The old tree stays
// exactly where it is: a record is the only account of what a run did, and
// the way to be sure a migration did not eat one is to still have it. What
// that costs is disk — task directories are text — and what it buys is that
// a version of Orbit from before this change still reads its own tree.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Flatten copies every task filed under a repository up to tasks/, and
// answers the ids it copied.
//
// A task already at the top level is left alone, which is what makes this
// safe to call on every start: the second run finds everything in place and
// copies nothing. The one case it will not decide is two repositories that
// each hold a task of the same name — one flat tree cannot hold both, and
// choosing which one wins is not a choice a migration gets to make quietly.
// It reports that pair and carries on with the rest.
func (s *Store) Flatten() ([]string, error) {
	repos, err := s.Repos()
	if err != nil {
		return nil, err
	}

	var (
		moved  []string
		failed []error
	)

	for _, r := range repos {
		ids, err := s.filedTasks(r.Key)
		if err != nil {
			failed = append(failed, err)
			continue
		}

		for _, id := range ids {
			done, err := s.flattenTask(r, id)
			if err != nil {
				failed = append(failed, err)
				continue
			}

			if done {
				moved = append(moved, id)
			}
		}
	}

	return moved, errors.Join(failed...)
}

// filedTasks lists the tasks still sitting under one repository's directory.
func (s *Store) filedTasks(key string) ([]string, error) {
	dir := filepath.Join(s.root, "repos", key, "tasks")

	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read %q: %w", dir, err)
	}

	var ids []string

	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}

	return ids, nil
}

// flattenTask copies one task up, and says whether it had to.
func (s *Store) flattenTask(r RepoRef, id string) (bool, error) {
	src := filepath.Join(s.root, "repos", r.Key, "tasks", id)

	dst, err := s.TaskDir(id)
	if err != nil {
		return false, fmt.Errorf("%q under %q: %w", id, r.Path, err)
	}

	if _, err := os.Stat(dst); err == nil {
		return false, s.alreadyThere(id, r, src, dst)
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("read %q: %w", dst, err)
	}

	if err := copyTree(src, dst); err != nil {
		return false, err
	}

	if err := s.joinRepo(id, r.Path); err != nil {
		return false, err
	}

	return true, nil
}

// alreadyThere tells a task that has been copied already from two tasks of
// the same name in two repositories. The first is the ordinary second run
// and is nothing; the second is a name collision, and it names both
// directories because whoever reads it is the one who has to rename one.
func (s *Store) alreadyThere(id string, r RepoRef, src, dst string) error {
	joined, err := s.TaskRepos(id)
	if err != nil {
		return err
	}

	for _, p := range joined {
		if p == r.Path {
			return nil
		}
	}

	return fmt.Errorf("two tasks are named %q — %q and %q — and one flat tree holds one of them: rename one of the two, and %q is the one orbit did not move", id, dst, src, src)
}

// copyTree copies a directory, reading every file back to check it arrived.
//
// The check is the whole point of doing this by hand rather than with a
// rename: a rename is atomic and leaves nothing behind to compare against,
// and this leaves the original in place precisely so there is something to
// compare against.
func copyTree(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read %q: %w", src, err)
	}

	if err := os.MkdirAll(dst, dirMode); err != nil {
		return fmt.Errorf("create %q: %w", dst, err)
	}

	for _, e := range entries {
		from, to := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())

		if e.IsDir() {
			if err := copyTree(from, to); err != nil {
				return err
			}

			continue
		}

		if err := copyFile(from, to); err != nil {
			return err
		}
	}

	return nil
}

// copyFile writes one file and reads it back before calling it copied.
func copyFile(from, to string) error {
	body, err := os.ReadFile(from)
	if err != nil {
		return fmt.Errorf("read %q: %w", from, err)
	}

	if err := os.WriteFile(to, body, fileMode); err != nil {
		return fmt.Errorf("write %q: %w", to, err)
	}

	back, err := os.ReadFile(to)
	if err != nil {
		return fmt.Errorf("read back %q: %w", to, err)
	}

	if !bytes.Equal(body, back) {
		return fmt.Errorf("%q came back from %q with %d bytes of %d", from, to, len(back), len(body))
	}

	return nil
}
