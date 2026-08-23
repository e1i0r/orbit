package repo

import (
	"fmt"
	"os"
)

// mkdir creates a directory and every parent it needs.
//
// 0700 rather than 0755 because in production the only thing this creates is
// the parent of a worktree, which lives inside the state root, and that root
// is private: it holds full checkouts of repositories that are not public.
func mkdir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	return nil
}
