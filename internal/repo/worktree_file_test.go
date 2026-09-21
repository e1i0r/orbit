package repo

// Where a path inside a worktree actually is.

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestAPathIsResolvedInsideTheWorktreeAndNowhereElse.
//
// The path comes from an engine, which means it comes from a model: it may
// be a relative path, a path with `..` in it, or a path that starts at the
// root of the machine. All three have to land inside the worktree the run
// was given, because the one thing a phase must not be able to do is write
// outside the copy it was handed.
func TestAPathIsResolvedInsideTheWorktreeAndNowhereElse(t *testing.T) {
	wt := t.TempDir()

	root, err := filepath.Abs(wt)
	if err != nil {
		t.Fatalf("resolve the worktree: %v", err)
	}

	for _, one := range []struct {
		why  string
		path string
		want string
	}{
		{"an ordinary path", "internal/db/pr.go", filepath.Join(root, "internal/db/pr.go")},
		{"a path with a dot in it", "./internal/db", filepath.Join(root, "internal/db")},
		{"one that tries to climb out", "../../etc/passwd", filepath.Join(root, "etc/passwd")},
		{"one that starts at the machine's root", "/etc/passwd", filepath.Join(root, "etc/passwd")},
		{"the worktree itself", "", root},
		{"and the worktree said out loud", ".", root},
	} {
		got, err := under(wt, one.path)
		if err != nil {
			t.Errorf("%s (%q) was refused: %v", one.why, one.path, err)
			continue
		}

		if got != one.want {
			t.Errorf("%s (%q) resolves to %q, want %q", one.why, one.path, got, one.want)
		}

		if got != root && !strings.HasPrefix(got, root+string(filepath.Separator)) {
			t.Errorf("%s (%q) resolves outside the worktree: %q", one.why, one.path, got)
		}
	}
}
