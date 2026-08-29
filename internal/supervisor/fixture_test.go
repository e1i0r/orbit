package supervisor

import (
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
