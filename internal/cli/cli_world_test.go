package cli

// The world's own ports, answered the way the verbs answer through them.
//
// Unread, Learn, Export, Take and Deliver are what every verb reaches the
// machine through; the web and model doors fill the same ports, so one
// exercise here holds for all three. Where a port runs a command (pr),
// the test borrows the fake gh the delivery tests lend.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// mustOpenRepo opens the checkout the test made, failing the test when
// git would not call it one.
func mustOpenRepo(t *testing.T, dir string) repo.Repo {
	t.Helper()

	one, err := repo.Open(dir)
	if err != nil {
		t.Fatalf("open the repository: %v", err)
	}

	return one
}

// worldOf is newWorld over a state root of the test's own, read against
// a workspace with one repository in it. One home for both: the commands
// open their own store from the environment.
func worldOf(t *testing.T) (world, *store.Store, string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	root := t.TempDir()
	repoDir := root + "/payments"

	if err := os.MkdirAll(repoDir, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@orbit.local"},
		{"config", "user.name", "Orbit Tester"},
		{"commit", "-q", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir

		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	s, err := store.New(home)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return newWorld(s, board.NewReader(s, root), words.For("en")), s, repoDir
}

// TestUnreadCountsWhatNobodyLookedAt. Nothing written and nothing read:
// zero, and no error, which is a board rather than a failure.
func TestUnreadCountsWhatNobodyLookedAt(t *testing.T) {
	w, _, root := worldOf(t)

	n, err := w.Unread(root)
	if err != nil {
		t.Fatalf("unread: %v", err)
	}

	if n != 0 {
		t.Errorf("unread counts %d, want nothing unwatched", n)
	}
}

// TestLearningValidatesBeforeWriting. A sentence with nowhere to stand
// is refused; one with a scope is written where it belongs.
func TestLearningValidatesBeforeWriting(t *testing.T) {
	w, _, _ := worldOf(t)

	if err := w.Learn(knowledge.Fact{}); err == nil {
		t.Error("a fact from nowhere was accepted")
	}

	fact := knowledge.Fact{
		Phrase: "amounts are cents",
		Source: knowledge.Human,
		Scope:  knowledge.Scope{Kind: knowledge.General},
	}

	if err := w.Learn(fact); err != nil {
		t.Fatalf("learn: %v", err)
	}
}

// TestExportWritesTheRecordOut. Into a directory that holds nothing,
// which is the one shape export accepts.
func TestExportWritesTheRecordOut(t *testing.T) {
	w, _, _ := worldOf(t)

	into := t.TempDir() + "/out"

	said, err := w.Export(into, "")
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if !strings.Contains(said, "out") {
		t.Errorf("export answered %q", said)
	}
}

// TestTakeNeedsATaskOnTheBoard. The window names a row; a reader naming
// an id nobody wrote gets a refusal naming it back.
func TestTakeNeedsATaskOnTheBoard(t *testing.T) {
	w, _, _ := worldOf(t)

	if _, err := w.Take("ACME-404", ""); err == nil {
		t.Error("taking the keyboard for a task nobody wrote was accepted")
	}
}

// TestDeliverRefusesWhatNothingDeliversBy. The map is pr, merge and
// close-pr; anything else is a name nobody gave a meaning.
func TestDeliverRefusesWhatNothingDeliversBy(t *testing.T) {
	w, s, repoDir := worldOf(t)

	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "PAY-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	// The command opens its own store from the environment, so this
	// world's store is opened the same way to read back what it wrote.
	if _, r, err := openBoth(repoDir); err != nil {
		t.Fatalf("open the workspace: %v", err)
	} else {
		_ = r
	}

	tk, err := task.Load(s, mustOpenRepo(t, repoDir), "PAY-1")
	if err != nil {
		t.Fatalf("load the task: %v", err)
	}

	if _, err := w.Deliver(context.Background(), tk, "launch"); err == nil {
		t.Error("delivering by a name nothing answers to was accepted")
	}
}

// TestDeliverOpensAPullRequest. Through the port, with the fake gh the
// delivery tests lend: pushing is real, GitHub is a shell script, and the
// URL comes back the way a reader would read it.
func TestDeliverOpensAPullRequest(t *testing.T) {
	w, s, _ := worldOf(t)

	dir := withRemote(t, t.TempDir(), "payments")

	if code, _, errOut := run(t, "new", "-repo", dir, "-id", "PAY-1", "make the thing"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	plantWorktree(t, dir, "make the thing")
	fakeGh(t, "echo https://github.test/acme/payments/pull/7")

	tk, err := task.Load(s, mustOpenRepo(t, dir), "PAY-1")
	if err != nil {
		t.Fatalf("load the task: %v", err)
	}

	url, err := w.Deliver(context.Background(), tk, "pr")
	if err != nil {
		t.Fatalf("deliver: %v", err)
	}

	if !strings.Contains(url, "github.test") {
		t.Errorf("deliver answered %q, want the pull request URL", url)
	}
}

// TestServeMCPStopsWhenTheClientGoesAway. Over a closed standard input
// the server has nobody to speak to, and leaving is the whole of what
// there is to do.
func TestServeMCPStopsWhenTheClientGoesAway(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open the null device: %v", err)
	}

	defer devnull.Close()

	old := os.Stdin
	os.Stdin = devnull

	defer func() { os.Stdin = old }()

	done := make(chan error, 1)

	go func() {
		done <- serveMCP(Context{Words: words.For("en")}, "")
	}()

	if err := <-done; err != nil {
		t.Fatalf("serve over a closed connection: %v", err)
	}
}

// TestWebWatchesOneDirectory. Two name nothing coherent, and a missing
// one fails when the board is read: naming is cheap, reading is not.
func TestWebWatchesOneDirectory(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	if code, _, _ := run(t, "web", t.TempDir(), t.TempDir()); code == 0 {
		t.Error("web watching two directories was accepted")
	}

	if code, _, _ := run(t, "web", filepath.Join(t.TempDir(), "nowhere")); code == 0 {
		t.Error("web watching a directory that is not there was accepted")
	}
}
