package web

// What the diff and the file answer when there is something to look in.
//
// The rest of this package's tests ask about a task nobody ran: no checkout,
// so every route answers "missing" and nothing reaches git. That is the
// ordinary state of the To Do band and it is worth asking, but it leaves the
// reading itself unasked — which checkout is opened, what git is handed, and
// what comes back. Those are the two things a page cannot work out for
// itself, and they are what this file is about.
//
// A real repository rather than a stand-in, because the reading is git's: a
// fake that answered a canned diff would test the fake. What is faked is the
// port that says where the checkout is, which is the part Orbit owns.

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// there is a worktree port that answers a directory something is at.
type there struct{ path string }

func (th there) Worktree(_, _ string) (string, error) { return th.path, nil }

// aCheckout is a repository with one commit and one change nobody committed,
// and the directory its worktree is in.
//
// The two are the same directory. A linked worktree would be the truer shape,
// and it is not the part under test: what serveDiff asks of the port is a
// path, and what it does with it is run git there.
//
// The change is left uncommitted on purpose. WorktreeDiff marks untracked
// files intent-to-add so that a file an agent wrote and never staged is in
// the diff at all, and a checkout with everything committed would pass that
// test without exercising it.
func aCheckout(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// The developer's own ~/.gitconfig must never reach this: a global
	// diff.external or pager would make the suite answer differently on
	// somebody else's machine.
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")

	gitIn(t, dir, "init", "-q", "-b", "main")
	write(t, dir, "total.go", "package pay\n\nfunc Total(cents int) int { return cents }\n")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-q", "-m", "first")

	write(t, dir, "total.go", "package pay\n\nfunc Total(cents int) int { return cents * 2 }\n")

	return dir
}

// gitIn runs one git command in dir and fails the test rather than carrying
// the error: a repository that did not build is not a finding about the route.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "HOME="+t.TempDir())

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// withCheckout is a task with a checkout somewhere, over a board of one row.
func withCheckout(t *testing.T, id, repoPath, worktree string) *Server {
	t.Helper()

	return server(aBoard{tasks: []view.Task{
		{ID: id, Title: "double the total", Band: view.Running, Repo: "ledger", RepoPath: repoPath},
	}}, there{path: worktree}, "/code", built)
}

// TestTheDiffOfACheckoutThatIsThere.
func TestTheDiffOfACheckoutThatIsThere(t *testing.T) {
	dir := aCheckout(t)

	code, body := ask(t, withCheckout(t, "LED-20", dir, dir), "/api/tasks/LED-20/diff")
	if code != http.StatusOK {
		t.Fatalf("the diff answered %d: %v", code, body)
	}

	if body["missing"] == true {
		t.Errorf("a checkout that is there was answered as missing: %v", body)
	}

	if failed, ok := body["failed"].(string); ok && failed != "" {
		t.Errorf("the diff failed: %s", failed)
	}

	text, ok := body["text"].(string)
	if !ok {
		t.Fatalf("the diff did not come back as text: %v", body)
	}

	if !strings.Contains(text, "total.go") {
		t.Errorf("the diff does not name the file that changed:\n%s", text)
	}

	// Handed over as git wrote it: turning it into hunks is the page's job,
	// and a server that had done it would be a server with two answers.
	if !strings.Contains(text, "@@") {
		t.Errorf("the diff was not handed over as git wrote it:\n%s", text)
	}
}

// TestADiffThatChangedNothingSaysSo.
//
// Empty and missing are different claims and travel as different fields: a
// task that ran and changed nothing is a finding, and a task nobody ran has
// nothing to find.
func TestADiffThatChangedNothingSaysSo(t *testing.T) {
	dir := aCheckout(t)
	gitIn(t, dir, "checkout", "-q", "--", ".")

	code, body := ask(t, withCheckout(t, "LED-21", dir, dir), "/api/tasks/LED-21/diff")
	if code != http.StatusOK {
		t.Fatalf("the diff answered %d: %v", code, body)
	}

	if body["empty"] != true {
		t.Errorf("a diff of nothing did not say it was empty: %v", body)
	}

	if body["missing"] == true {
		t.Errorf("a checkout that is there was answered as missing: %v", body)
	}
}

// TestWhitespaceIsLeftOutWhenAsked.
//
// A reformatting run that touched two hundred files buries the one line
// somebody has to read, and ?space=ignore is the only way past it.
//
// The change here is spacing and nothing else — two spaces where the commit
// had one — because that is the only case where ignoring whitespace has
// anything to ignore. A line that moved and was also reindented still counts
// as moved, and git prints it as the worktree holds it either way: what -w
// decides is which lines changed, not how they are written out.
func TestWhitespaceIsLeftOutWhenAsked(t *testing.T) {
	dir := aCheckout(t)
	write(t, dir, "total.go", "package pay\n\nfunc  Total(cents int) int { return cents }\n")

	_, plain := ask(t, withCheckout(t, "LED-22", dir, dir), "/api/tasks/LED-22/diff")
	_, quiet := ask(t, withCheckout(t, "LED-22", dir, dir), "/api/tasks/LED-22/diff?space=ignore")

	if plain["empty"] == true {
		t.Error("a change of spacing did not reach the plain diff at all")
	}

	before, ok := plain["text"].(string)
	if !ok {
		t.Fatalf("the plain diff did not come back as text: %v", plain)
	}

	if !strings.Contains(before, "func  Total") {
		t.Errorf("the plain diff does not carry the spacing that changed:\n%s", before)
	}

	if quiet["empty"] != true {
		t.Errorf("a diff whose only change was spacing still had something in it: %v", quiet)
	}
}

// TestAFileOfTheWorktreeWhole.
//
// What opens a hunk out: git writes three lines of context and the reader
// who wants to see above them is asking about the file, not the change.
func TestAFileOfTheWorktreeWhole(t *testing.T) {
	dir := aCheckout(t)

	code, body := ask(t, withCheckout(t, "LED-23", dir, dir), "/api/tasks/LED-23/file?path=total.go")
	if code != http.StatusOK {
		t.Fatalf("the file answered %d: %v", code, body)
	}

	text, ok := body["text"].(string)
	if !ok {
		t.Fatalf("the file did not come back as text: %v", body)
	}

	if !strings.Contains(text, "package pay") {
		t.Errorf("the file came back without its own contents: %v", body)
	}

	// The whole file and not the three lines around the hunk.
	if strings.Contains(text, "@@") {
		t.Errorf("the file came back as a diff:\n%s", text)
	}
}

// TestAFileThatIsNotThereIsSaid.
func TestAFileThatIsNotThereIsSaid(t *testing.T) {
	dir := aCheckout(t)

	code, body := ask(t, withCheckout(t, "LED-24", dir, dir), "/api/tasks/LED-24/file?path=nothing.go")
	if code != http.StatusOK {
		t.Fatalf("the file answered %d: %v", code, body)
	}

	if body["missing"] != true {
		t.Errorf("a file that is not there did not say so: %v", body)
	}
}

// TestARepositoryThatCannotBeOpenedIsAFailure.
//
// Not the same claim as a missing checkout. A task with no worktree is the
// ordinary state of the To Do band; a task whose repository is not a
// repository is a thing that broke, and answering it as "missing" would hide
// it behind a state the reader has no reason to act on.
func TestARepositoryThatCannotBeOpenedIsAFailure(t *testing.T) {
	worktree := t.TempDir()
	notARepo := t.TempDir()

	code, body := ask(t, withCheckout(t, "LED-25", notARepo, worktree), "/api/tasks/LED-25/diff")
	if code != http.StatusInternalServerError {
		t.Fatalf("a repository that is not one answered %d: %v", code, body)
	}

	if body == nil {
		t.Error("the failure said nothing about itself")
	}
}

// TestAWorktreeThatCannotBeFoundIsAFailure.
func TestAWorktreeThatCannotBeFoundIsAFailure(t *testing.T) {
	dir := aCheckout(t)

	s := server(aBoard{tasks: []view.Task{
		{ID: "LED-26", Title: "no state root", Band: view.Running, Repo: "ledger", RepoPath: dir},
	}}, nowhere{path: ""}, "/code", built)

	code, _ := ask(t, s, "/api/tasks/LED-26/diff")
	if code != http.StatusInternalServerError {
		t.Errorf("a worktree that could not be found answered %d", code)
	}
}
