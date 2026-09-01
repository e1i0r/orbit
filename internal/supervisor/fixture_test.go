package supervisor

import (
	"os"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// fixture is a state root and nothing else. The thread is global — it hangs
// off the root rather than off any task — which is the whole reason this
// package is not internal/task, and it is why these tests need no
// repository, no worktree and no git.
func fixture(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	return s
}

// breakRecord makes the record unreachable from here on.
//
// The open handle is let go of and a directory is put where the file was, so
// the next reader or writer gets the failure a state root on a broken disk
// would give. A directory rather than a permission bit because the bit
// depends on who is running the test, and root would not notice it.
func breakRecord(t *testing.T, s *store.Store) {
	t.Helper()

	if err := s.Close(); err != nil {
		t.Fatalf("let go of the record: %v", err)
	}

	if err := os.RemoveAll(s.DBPath()); err != nil {
		t.Fatalf("remove the record: %v", err)
	}

	if err := os.Mkdir(s.DBPath(), 0o700); err != nil {
		t.Fatalf("put a directory where the record goes: %v", err)
	}
}
