package engine

import (
	"path/filepath"
	"strings"
	"testing"
)

// The state root a real Orbit would run under, invented here.
//
// It is a parameter of the test rather than something asked of internal/
// store, because internal/engine imports nothing of Orbit's and the layer
// table keeps it that way. The test does not need the real path: it needs
// any path, and the guarantee that no engine ever names it or anything above
// it.
const (
	syntheticRoot     = "/synthetic/home/.orbit"
	syntheticWorktree = syntheticRoot + "/worktrees/acme/ACME-1"
)

// argvBuilders is every engine in this package that builds a command line.
//
// Fake is absent because it spawns nothing — it is not an omission, it is
// that there is no argv to check. Any engine added later belongs here, and
// the rule below is the one thing it has to pass before it ships.
func argvBuilders() map[string]func(Request) ([]string, error) {
	return map[string]func(Request) ([]string, error){
		"claude": claudeArgs,
	}
}

// requests is the spread of inputs the rule is checked against: a posture
// that asks for everything, one that asks for nothing, and a resumed
// session, each in a worktree that really does sit inside the state root.
func requests() []Request {
	base := Request{
		Prompt: "Phase: implement\nRepository: payments\n\nTask ACME-1:\nRetry on 5xx.\n",
		Model:  "sonnet",
		Dir:    syntheticWorktree,
	}
	full := base
	full.Permissions = []string{PermissionRead, PermissionRepo, PermissionNetwork}
	resumed := full
	resumed.Resume = "9c1f8f2a-4d3b-4a77-9a52-2f0f6f9b5c31"

	return []Request{base, full, resumed}
}

// TestNoEngineArgvNamesTheStateRootOrAnAncestorOfIt is the guard the layout
// has needed since the first plan, and it becomes live now because this is
// the change that introduces the first flag that could carry a directory
// grant.
//
// The engine's working directory sits inside the state root by design: that
// is what makes the record, and the credentials file the design puts beside
// it, reachable by relative path. The layout buys nothing on its own against
// a process running as the same user. The one control that does hold is this
// one — no engine is ever handed a directory permission at or above the
// root. No --add-dir, and no equivalent on any engine added later.
func TestNoEngineArgvNamesTheStateRootOrAnAncestorOfIt(t *testing.T) {
	for name, build := range argvBuilders() {
		for _, req := range requests() {
			got, err := build(req)
			if err != nil {
				t.Fatalf("%s: building the command line failed: %v", name, err)
			}

			for _, arg := range got {
				if arg == "" {
					continue
				}

				if arg == syntheticRoot {
					t.Errorf("%s was handed the state root itself: %v", name, got)
					continue
				}
				// A string prefix catches both shapes at once: an
				// ancestor directory, and a path that stops part way
				// through a segment. Over-strict by a hair, and the hair
				// costs nothing — no legitimate argument of an engine
				// begins the state root's path.
				if strings.HasPrefix(syntheticRoot, arg) {
					t.Errorf("%s was handed %q, which is at or above the state root: %v", name, arg, got)
				}
			}
		}
	}
}

// TestNoEngineArgvNamesAnAncestorDirectoryByName says the same thing the
// other way round, walking up from the root, so that a rule broken by a
// clever bit of quoting still fails on the plain path.
func TestNoEngineArgvNamesAnAncestorDirectoryByName(t *testing.T) {
	var ancestors []string
	for dir := syntheticRoot; ; dir = filepath.Dir(dir) {
		ancestors = append(ancestors, dir)
		if filepath.Dir(dir) == dir {
			break
		}
	}

	for name, build := range argvBuilders() {
		for _, req := range requests() {
			got, err := build(req)
			if err != nil {
				t.Fatalf("%s: building the command line failed: %v", name, err)
			}

			for _, arg := range got {
				for _, up := range ancestors {
					if arg == up {
						t.Errorf("%s was handed %q, an ancestor of the state root: %v", name, arg, got)
					}
				}
			}
		}
	}
}

// TestNoEngineIsHandedADirectoryGrant bans the flag by name as well as by
// effect. --add-dir is claude's way of widening the worktree, and the widest
// thing it could ever be pointed at is the root this test protects; a change
// that adds it for a good local reason should have to delete this test to do
// it.
func TestNoEngineIsHandedADirectoryGrant(t *testing.T) {
	for name, build := range argvBuilders() {
		for _, req := range requests() {
			got, err := build(req)
			if err != nil {
				t.Fatalf("%s: building the command line failed: %v", name, err)
			}

			joined := strings.Join(got, " ")
			for _, forbidden := range []string{"--add-dir", "--addDir"} {
				if strings.Contains(joined, forbidden) {
					t.Errorf("%s carries %s: %v", name, forbidden, got)
				}
			}
		}
	}
}

// TestTheWorktreeIsPassedAsTheWorkingDirectoryNotAsAnArgument records why
// the rule above can be as strict as it is. The engine is told where to work
// by cmd.Dir, so no path needs to appear on the command line at all, and any
// path that does appear is a grant somebody added on purpose.
func TestTheWorktreeIsPassedAsTheWorkingDirectoryNotAsAnArgument(t *testing.T) {
	for name, build := range argvBuilders() {
		for _, req := range requests() {
			got, err := build(req)
			if err != nil {
				t.Fatalf("%s: building the command line failed: %v", name, err)
			}

			for _, arg := range got {
				if arg == req.Dir {
					t.Errorf("%s names the worktree on its command line: %v", name, got)
				}
			}
		}
	}
}
