package mcp

// What a --root server may reach.
//
// The root was a search and not a boundary: it scoped every read, and then
// the three tools that change what Orbit knows resolved the caller's
// argument straight off the disk. These tests are the boundary.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestARootIsWhatThisServerMayReachAndNotJustWhereItLooks.
//
// A root scoped every read and no write. The board a --root server folded
// held only what was under that directory, and then orbit_create_task,
// orbit_add_repo and orbit_forget_repo each resolved the caller's argument
// straight off the disk — so a model given a root of ~/work could write a
// task into any checkout on the machine, register any directory, and delete
// the record of a repository it was never able to list.
func TestARootIsWhatThisServerMayReachAndNotJustWhereItLooks(t *testing.T) {
	s, work := newRoot(t)
	inside := gitRepo(t, work, "payments")

	elsewhere := t.TempDir()
	outside := gitRepo(t, elsewhere, "secrets")

	if _, err := s.RegisterRepo(outside.Path); err != nil {
		t.Fatalf("register %q: %v", outside.Path, err)
	}

	sn := Session{Root: work, Version: "test"}

	for _, c := range []struct {
		tool string
		args map[string]any
		why  string
	}{
		{"orbit_create_task", map[string]any{
			"title": "quietly", "repo": outside.Path,
		}, "a task written into a checkout outside the root is work filed against a project nobody asked about"},
		{"orbit_add_repo", map[string]any{
			"path": outside.Path,
		}, "registering a directory outside the root widens the root by using it"},
		{"orbit_forget_repo", map[string]any{
			"repo": outside.Path,
		}, "forgetting deletes the only account of what those runs did, and this session could never list them"},
		{"orbit_inspect_repo", map[string]any{
			"repo": outside.Path,
		}, "a repository this session cannot list is one it cannot describe either"},
	} {
		if why := refused(t, sn, c.tool, c.args); !strings.Contains(why, "outside") {
			t.Errorf("%s did not refuse a path outside the root: %s — %s", c.tool, why, c.why)
		}
	}

	// The root itself still works, or the confinement above cost the
	// feature rather than bounding it.
	got := call(t, sn, "orbit_inspect_repo", map[string]any{"repo": inside.Path})
	if str(t, got["path"]) != inside.Path {
		t.Errorf("orbit_inspect_repo inside the root answered %v, want %q", got["path"], inside.Path)
	}

	if res := sn.Call("orbit_create_task", map[string]any{"title": "properly", "repo": inside.Path}); res.IsError {
		t.Errorf("orbit_create_task inside the root refused: %s", text(t, res))
	}
}

// TestNoRootConfinesNothing. Without a root the scope is the state root's
// own listing — every repository a reader has already used Orbit on — and
// confining to a directory nobody named would break the default case this
// server is started in.
func TestNoRootConfinesNothing(t *testing.T) {
	s, work := newRoot(t)
	elsewhere := gitRepo(t, t.TempDir(), "secrets")

	gitRepo(t, work, "payments")

	if _, err := s.RegisterRepo(elsewhere.Path); err != nil {
		t.Fatalf("register %q: %v", elsewhere.Path, err)
	}

	sn := Session{Version: "test"}

	got := call(t, sn, "orbit_inspect_repo", map[string]any{"repo": elsewhere.Path})
	if str(t, got["path"]) != elsewhere.Path {
		t.Errorf("a session with no root refused %q: %v", elsewhere.Path, got)
	}
}

// TestARootFollowsSymbolicLinks, because on macOS every temporary directory
// is one: a root given as /tmp/x and a repository git answers for at
// /private/tmp/x are the same directory, and a textual comparison would
// refuse the whole workspace this server was started for.
func TestARootFollowsSymbolicLinks(t *testing.T) {
	actual := t.TempDir()
	link := filepath.Join(t.TempDir(), "work")

	if err := os.Symlink(actual, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	r := gitRepo(t, actual, "payments")

	if err := (Session{Root: link}).within(r.Path); err != nil {
		t.Errorf("a repository under the root was refused through a link: %v", err)
	}

	if err := (Session{Root: actual}).within(filepath.Join(link, "payments")); err != nil {
		t.Errorf("the same repository named through the link was refused: %v", err)
	}
}

// TestARepositoryThatIsGoneCanStillBeForgotten. Its checkout is deleted and
// its record is the thing being cleaned up, so EvalSymlinks refuses the path
// outright — and a confinement that needed the path to exist would refuse
// exactly the case orbit_forget_repo is for.
func TestARepositoryThatIsGoneCanStillBeForgotten(t *testing.T) {
	s, work := newRoot(t)
	doomed := gitRepo(t, work, "payments")
	sn := Session{Root: work, Version: "test"}

	if _, err := s.RegisterRepo(doomed.Path); err != nil {
		t.Fatalf("register %q: %v", doomed.Path, err)
	}

	if err := os.RemoveAll(doomed.Path); err != nil {
		t.Fatalf("remove %q: %v", doomed.Path, err)
	}

	if res := sn.Call("orbit_forget_repo", map[string]any{"repo": doomed.Path}); res.IsError {
		t.Errorf("a repository whose checkout is gone could not be forgotten: %s", text(t, res))
	}
}

// TestALinkOutOfTheRootIsStillOutOfIt. The containment is computed on
// resolved paths, so a symbolic link planted inside the root does not make
// what it points at part of the root.
func TestALinkOutOfTheRootIsStillOutOfIt(t *testing.T) {
	work, elsewhere := t.TempDir(), t.TempDir()

	escape := filepath.Join(work, "shortcut")
	if err := os.Symlink(elsewhere, escape); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if err := (Session{Root: work}).within(filepath.Join(escape, "secrets")); err == nil {
		t.Error("a link inside the root walked out of it")
	}
}

// TestAPathThatIsNotThereIsStillPlacedAgainstTheRoot. The two cases above
// meet here: a repository that is gone, named through a link. EvalSymlinks
// refuses the whole path, so resolving only what exists is what puts it back
// under the root — and giving up on a path that will not resolve at all
// would refuse the checkout that was deleted, which is the one case
// orbit_forget_repo exists for.
func TestAPathThatIsNotThereIsStillPlacedAgainstTheRoot(t *testing.T) {
	actual := t.TempDir()
	link := filepath.Join(t.TempDir(), "work")

	if err := os.Symlink(actual, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if err := (Session{Root: actual}).within(filepath.Join(link, "payments", "worktree")); err != nil {
		t.Errorf("a path under the root that is not on disk was refused: %v", err)
	}
}
