package ui

// tree_test.go is the map pane's reading of a repository, asked for against a
// real checkout on disk.
//
// treeOf is the one place this package turns a task into git commands, and it
// holds three different answers for what look like the same failure: the port
// could not say where the checkout is, the checkout is not there, and git
// could not read it. Only the middle one is an ordinary state — a task whose
// worktree was cleaned up — and telling it apart from the other two is the
// whole reason the os.Stat sits between the port and the git call.
//
// The repository fixture is gitrepo_test.go's, and the port is a stub: where
// a checkout lives is internal/store's answer, and this package may not name
// that.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/view"
)

// treeReader is a Reader that says where the checkout is, or that it cannot.
type treeReader struct {
	path string
	err  error
}

func (r *treeReader) Refresh() (board.Board, board.Changed, error) {
	return board.Board{}, board.Changed{}, nil
}
func (r *treeReader) Rescan() error                             { return nil }
func (r *treeReader) Log(string, string) ([]view.Entry, error)  { return nil, nil }
func (r *treeReader) Worktree(string, string) (string, error)   { return r.path, r.err }
func (r *treeReader) Files(string, string) ([]view.File, error) { return nil, nil }

func (r *treeReader) FileText(string, string, string) (view.FileText, error) {
	return view.FileText{}, nil
}
func (r *treeReader) SupervisorLog() ([]view.SupervisorLine, error) { return nil, nil }

// readTree runs the command treeOf hands back and returns what came in.
func readTree(t *testing.T, r Reader, repoPath, id string) treeMsg {
	t.Helper()

	msg, ok := treeOf(r, view.Task{ID: id, Repo: "retry", RepoPath: repoPath})().(treeMsg)
	if !ok {
		t.Fatal("treeOf's command did not raise a treeMsg")
	}

	if msg.id != id {
		t.Errorf("the reading came back for %q, asked for %q", msg.id, id)
	}

	return msg
}

// TestATreeOfACheckoutThatIsThere.
func TestATreeOfACheckoutThatIsThere(t *testing.T) {
	repoPath := gitRepo(t)
	tree := worktreeOf(t, repoPath, "ACME-2")

	msg := readTree(t, &treeReader{path: tree}, repoPath, "ACME-2")

	if msg.err != nil {
		t.Fatalf("reading a checkout that is there failed: %v", msg.err)
	}

	if msg.missing {
		t.Error("a checkout that is there was answered as missing")
	}

	// The tree names what the repository holds, and it is verb.Grow's shape
	// rather than a list of paths: the browser draws the same tree as
	// hexagons, and a second builder here is how the two would come to
	// disagree about what is in a repository.
	if !holds(msg.tree, "retry.go") {
		t.Errorf("the tree does not hold the file the repository has:\n%+v", msg.tree)
	}
}

// holds is whether a cell or anything under it is named this. The tree is a
// directory lattice and the file may sit several cells down, so the question
// is asked recursively rather than of the root's own row.
func holds(c verb.Cell, name string) bool {
	if c.Name == name {
		return true
	}

	for _, inside := range c.Cells {
		if holds(inside, name) {
			return true
		}
	}

	return false
}

// TestACheckoutThatIsNotThereIsMissingAndNotAnError.
//
// This is the assertion the os.Stat exists for. Asking git first would answer
// `chdir …: no such file or directory`, which is true and is not something a
// reader can act on; "missing" is a state the pane can draw.
func TestACheckoutThatIsNotThereIsMissingAndNotAnError(t *testing.T) {
	repoPath := gitRepo(t)
	gone := filepath.Join(t.TempDir(), "taken-away")

	msg := readTree(t, &treeReader{path: gone}, repoPath, "ACME-3")

	if !msg.missing {
		t.Error("a checkout that is not there was not answered as missing")
	}

	if msg.err != nil {
		t.Errorf("missing came back as an error too: %v", msg.err)
	}
}

// TestAPortThatCannotSayWhereIsAnError.
//
// Not the same claim as missing: the store could not answer at all, and there
// is nothing to look at.
func TestAPortThatCannotSayWhereIsAnError(t *testing.T) {
	msg := readTree(t, &treeReader{err: errors.New("no such task")}, "/code", "ACME-4")

	if msg.err == nil {
		t.Fatal("a port that could not answer did not say so")
	}

	if msg.missing {
		t.Error("a port failure was answered as a missing checkout")
	}
}

// TestADirectoryThatIsNotARepositoryIsAnError.
//
// The third answer, and the one that has to stay an error: the directory is
// there, so it is not missing, and git cannot read it, so there is no tree.
func TestADirectoryThatIsNotARepositoryIsAnError(t *testing.T) {
	repoPath := gitRepo(t)
	notARepo := t.TempDir()

	if err := os.WriteFile(filepath.Join(notARepo, "x.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msg := readTree(t, &treeReader{path: notARepo}, repoPath, "ACME-5")

	if msg.err == nil {
		t.Fatal("a directory that is not a repository was read as a tree")
	}

	if msg.missing {
		t.Error("a directory that is there was answered as missing")
	}
}
