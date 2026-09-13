package ui

// The reading itself: which checkout is opened, what git is handed, and what
// comes back.
//
// impact_test.go asks what the pane draws once a reading is in it. This asks
// where the reading comes from — the one part of the pane that leaves the
// process, and the one part that got it wrong once: without the base branch
// it asked git what was uncommitted, and a task that had committed its work
// (which is every task that finished) had nothing uncommitted. The pane said
// "this change touched no files" beside a diff of nineteen, and the
// disagreement sent it back to git on every poll.
//
// A real repository rather than a stand-in, because the reading is git's: a
// fake that answered a canned Impact would test the fake. What is faked is
// the port that says where the checkout is, which is the part Orbit owns.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// aHistory is a repository with a couple of commits and one change nobody
// committed, and the directory its worktree is in.
//
// The change is left uncommitted on purpose. A task that committed its work
// is the case this reading got wrong, and a checkout with nothing in it would
// pass every assertion here without reaching the history at all: Impact
// returns early on no changes, before it reads a single commit.
func aHistory(t *testing.T) string {
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

	gitThere(t, dir, "init", "-q", "-b", "main")
	put(t, dir, "pricing.go", "package pay\n\nfunc Price(cents int) int { return cents }\n")
	gitThere(t, dir, "add", ".")
	gitThere(t, dir, "commit", "-q", "-m", "pricing")

	// A second commit that touches both files, so the history has something
	// to say about what usually comes along with the first.
	put(t, dir, "invoice.go", "package pay\n\nfunc Invoice() int { return Price(1) }\n")
	gitThere(t, dir, "add", ".")
	gitThere(t, dir, "commit", "-q", "-m", "invoice follows pricing")

	put(t, dir, "pricing.go", "package pay\n\nfunc Price(cents int) int { return cents * 2 }\n")

	return dir
}

// gitThere runs one git command in dir and fails the test rather than
// carrying the error: a repository that did not build is not a finding about
// the reading.
func gitThere(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "HOME="+t.TempDir())

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func put(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// weighedOut is the reading an impactOf came back with.
func weighedOut(t *testing.T, cmd tea.Cmd) impactMsg {
	t.Helper()

	got := cmd()

	msg, ok := got.(impactMsg)
	if !ok {
		t.Fatalf("the reading came back as %T, not an impactMsg", got)
	}

	return msg
}

// TestAWorktreeThatCannotBeFoundIsCarriedBack.
//
// The port's refusal has to travel, not be swallowed: a pane that dropped it
// would say "nothing read yet" forever, and forever is what it did while the
// checkout was on a volume that had gone away.
func TestAWorktreeThatCannotBeFoundIsCarriedBack(t *testing.T) {
	gone := errors.New("the worktree is on a volume that is not mounted")

	cmd := impactOf(&fakeReader{treeErr: gone}, view.Task{ID: "ACME-1", RepoPath: "/w/api"})

	msg := weighedOut(t, cmd)
	if msg.id != "ACME-1" {
		t.Errorf("the reading came back about %q", msg.id)
	}

	if !errors.Is(msg.err, gone) {
		t.Errorf("the port's refusal was not carried back: %v", msg.err)
	}
}

// TestTheHistoryIsReadWithTheBaseBranchInHand.
//
// The regression this whole command exists to keep from coming back. A task
// with changes has to be weighed against what the history says, and the only
// way to ask is with the repository's base branch — without it the question
// becomes "what is uncommitted", which a finished task answers with nothing.
func TestTheHistoryIsReadWithTheBaseBranchInHand(t *testing.T) {
	dir := aHistory(t)

	cmd := impactOf(&fakeReader{worktree: dir}, view.Task{
		ID: "ACME-2", Repo: "ledger", RepoPath: dir,
	})

	msg := weighedOut(t, cmd)
	if msg.err != nil {
		t.Fatalf("the reading was refused: %v", msg.err)
	}

	if msg.id != "ACME-2" {
		t.Errorf("the reading came back about %q", msg.id)
	}

	// The file that was written and never staged is in the answer. It is
	// invisible to git diff without the intent-to-add WorktreeDiff does,
	// and it is the most interesting file there is.
	if !strings.Contains(strings.Join(msg.impact.Changed, " "), "pricing.go") {
		t.Errorf("the reading does not carry the file that changed: %v", msg.impact.Changed)
	}

	// And the history behind it was actually read, which is the part that
	// only happens when the base branch came along.
	if msg.impact.Commits == 0 {
		t.Error("the reading counted no commits: the history was never read")
	}
}

// TestARepositoryThatCannotBeOpenedIsCarriedBackToo.
//
// A checkout that is there and a repository that is not one are different
// refusals and both have to reach the pane: the pane draws them differently,
// and folding them would tell a reader there is no coupling here when what
// happened is that git said no.
func TestARepositoryThatCannotBeOpenedIsCarriedBackToo(t *testing.T) {
	notARepo := t.TempDir()

	cmd := impactOf(&fakeReader{worktree: notARepo}, view.Task{
		ID: "ACME-3", Repo: "ledger", RepoPath: notARepo,
	})

	msg := weighedOut(t, cmd)
	if msg.err == nil {
		t.Errorf("a repository that is not one was read anyway: %v", msg.impact)
	}

	if msg.id != "ACME-3" {
		t.Errorf("the refusal came back about %q", msg.id)
	}
}

// TestNothingChangedIsAnAnswerAndNotARefusal.
//
// Empty and failed are different facts. A task that ran and touched nothing
// is worth saying out loud; answering it with an error would put a red pane
// beside a task that did exactly what it was asked.
func TestNothingChangedIsAnAnswerAndNotARefusal(t *testing.T) {
	dir := aHistory(t)
	gitThere(t, dir, "checkout", "-q", "--", ".")

	cmd := impactOf(&fakeReader{worktree: dir}, view.Task{
		ID: "ACME-4", Repo: "ledger", RepoPath: dir,
	})

	msg := weighedOut(t, cmd)
	if msg.err != nil {
		t.Fatalf("a task that changed nothing was refused: %v", msg.err)
	}

	if len(msg.impact.Changed) != 0 {
		t.Errorf("a worktree with nothing in it said it changed %v", msg.impact.Changed)
	}
}

// TestTheReadingIsAskedForOnce.
//
// askImpact is the guard on the expensive end: five hundred commits of git
// log, on a screen that redraws ten times a second. Already read and already
// out are both reasons not to ask again, and a pane that asked on every tick
// spent its life loading rather than being read.
func TestTheReadingIsAskedForOnce(t *testing.T) {
	dir := aHistory(t)

	m, _ := testModel(t, 100, 30)
	m.opts.Reader = &fakeReader{worktree: dir}
	m.detail = "ACME-5"
	m.board.Tasks = []view.Task{{ID: "ACME-5", RepoPath: dir}}

	_, cmd := m.askImpact()
	if cmd == nil {
		t.Fatal("the first look did not ask for the reading")
	}

	// Out, so the second look does not ask again.
	m.weigh.reachAsking = true
	if _, again := m.askImpact(); again != nil {
		t.Error("a reading that is already out was asked for a second time")
	}

	// Come back, so the third does not either.
	m.weigh.reachAsking, m.weigh.reachKnown = false, true
	if _, again := m.askImpact(); again != nil {
		t.Error("a reading that already came back was asked for again")
	}
}

// TestATaskWithNoRepositoryIsNotAskedAbout.
//
// A task that has not been given a repository has nothing to weigh, and
// asking git about an empty path is a subprocess that can only refuse.
func TestATaskWithNoRepositoryIsNotAskedAbout(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Reader = &fakeReader{worktree: "/w/api"}
	m.detail = "ACME-6"
	m.board.Tasks = []view.Task{{ID: "ACME-6"}}

	if _, cmd := m.askImpact(); cmd != nil {
		t.Error("a task with no repository was asked about anyway")
	}
}

// TestTheReadingLandsOnTheTaskOnScreen, and says it is no longer out.
func TestTheReadingLandsOnTheTaskOnScreen(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.detail = "ACME-7"

	got := m.tookImpact(impactMsg{
		id:     "ACME-7",
		impact: repo.Impact{Changed: []string{"pricing.go"}, Commits: 400},
	})

	if !got.weigh.reachKnown {
		t.Error("the reading did not land")
	}

	if got.weigh.reachAsking {
		t.Error("the pane still says it is reading")
	}

	if got.weigh.reach.Commits != 400 {
		t.Errorf("the reading that landed counted %d commits", got.weigh.reach.Commits)
	}
}

// TestAMovedAwayFromTaskForgetsWhatItRead.
//
// The next task's pane would otherwise open wearing the last task's answer,
// and a reader who trusts it is weighing a change they are not looking at.
func TestAMovedAwayFromTaskForgetsWhatItRead(t *testing.T) {
	m := reading(t, repo.Impact{Changed: []string{"pricing.go"}, Commits: 400})
	m.weigh.reread = true

	got := m.forgetImpact()

	if got.weigh.reachKnown || got.weigh.reachAsking {
		t.Error("the reading survived the move")
	}

	if len(got.weigh.reach.Changed) != 0 || got.weigh.reachErr != nil {
		t.Errorf("what it read survived the move: %v %v", got.weigh.reach, got.weigh.reachErr)
	}

	if got.weigh.reread {
		t.Error("the second-reading flag survived, so the next task would never reread")
	}
}
