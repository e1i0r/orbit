package verb

// Reading what a task changed, off a real checkout.
//
// The diff, the tree, what it reaches and the checks on both sides are
// settled against a worktree git answers in — which is why these tests
// make one rather than pointing at a directory.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
)

// checkedOut is a task with a checkout on disk: a real worktree of the
// repository, so checkout() finds work to read and git answers in it.
func checkedOut(t *testing.T, w *testWorld, id string) repo.Repo {
	t.Helper()

	r := w.gitRepo(t, "acme")

	w.wrote(t, id, r.Path, "change the webhook")

	dir, err := w.store.WorktreeDir(r.Path, id)
	if err != nil {
		t.Fatalf("worktree dir: %v", err)
	}

	add := exec.Command("git", "worktree", "add", dir, "-b", "orbit/"+id)
	add.Dir = r.Path

	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v (%s)", err, out)
	}

	if err := os.WriteFile(filepath.Join(dir, "done.txt"), []byte("did it\n"), 0o600); err != nil {
		t.Fatalf("write the work: %v", err)
	}

	return r
}

// TestReadingWhatATaskChanged. The diff, the tree, what it reaches, and
// the checks on both sides — each read off a real checkout.
func TestReadingWhatATaskChanged(t *testing.T) {
	w := worldOf(t)
	r := checkedOut(t, w, "ACME-8")
	at := In{Task: "ACME-8", Repo: r.Path, By: "operator"}

	// The diff is git's own, whole: what a reader asked for is the change,
	// and a verb that read it and answered nothing reads the same as one
	// that found nothing.
	diff := mustAsk(t, w, "task diff", at)
	for _, want := range []string{"diff --git", "done.txt", "+did it"} {
		if !strings.Contains(diff.Said, want) {
			t.Errorf("the diff answered %q, want it to carry %q", diff.Said, want)
		}
	}

	tree := mustAsk(t, w, "task tree", at)
	if !strings.Contains(tree.Said, "done.txt") {
		t.Errorf("the tree answered %q, which names no file", tree.Said)
	}

	impact := mustAsk(t, w, "task impact", at)
	if !strings.Contains(impact.Said, "1 files changed") {
		t.Errorf("what it reaches answered %q", impact.Said)
	}

	// The flow Orbit ships has no checks, so there is nothing to run on
	// either side — which is an answer and not a refusal.
	compared := mustAsk(t, w, "task compare", at)
	if !strings.Contains(compared.Said, "no checks") {
		t.Errorf("comparing answered %q", compared.Said)
	}

	// The task's own flow, then the one Orbit ships, with its phases
	// numbered from one the way a reader counts them.
	flow := mustAsk(t, w, "task flow", at)
	for _, want := range []string{"ACME-8: task", " 1  implement", " 2  review"} {
		if !strings.Contains(flow.Said, want) {
			t.Errorf("the flow answered %q, want it to carry %q", flow.Said, want)
		}
	}
}

// TestACheckoutWithNothingInItSaysSo, which is a different answer from a
// task that never had a checkout at all: one has been worked in and changed
// nothing, the other has not been started.
func TestACheckoutWithNothingInItSaysSo(t *testing.T) {
	w := worldOf(t)
	r := checkedOut(t, w, "ACME-10")

	dir, err := w.store.WorktreeDir(r.Path, "ACME-10")
	if err != nil {
		t.Fatalf("worktree dir: %v", err)
	}

	if err := os.Remove(filepath.Join(dir, "done.txt")); err != nil {
		t.Fatalf("take the work back out: %v", err)
	}

	out := mustAsk(t, w, "task diff", In{Task: "ACME-10", Repo: r.Path, By: "operator"})
	if !strings.Contains(out.Said, "ACME-10 has changed nothing yet") {
		t.Errorf("a checkout with nothing in it answered %q", out.Said)
	}
}

// TestReadingWithoutACheckout. A task that never ran has nothing to read,
// and that is the true answer rather than a failure.
func TestReadingWithoutACheckout(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-9", r.Path, "never started")

	for _, name := range []string{"task diff", "task tree", "task impact"} {
		out := mustAsk(t, w, name, In{Task: "ACME-9", By: "operator"})
		if !strings.Contains(out.Said, "ACME-9") {
			t.Errorf("%s answered %q, which names no task", name, out.Said)
		}
	}

	// Comparing is running, not reading: with no checkout there is
	// nothing to run anything in, and that is a refusal.
	if err := refuseErr(t, w, "task compare", In{Task: "ACME-9", By: "operator"}); !strings.Contains(err.Error(), "ACME-9") {
		t.Errorf("compare refused with %q, which names no task", err)
	}
}

// TestJoiningNeedsANameItKnows. Joining the second checkout of a workspace
// answers the directory to work in.
func TestJoiningNeedsANameItKnows(t *testing.T) {
	w := worldOf(t)
	ws := t.TempDir()

	a := w.gitRepoIn(t, ws, "acme")
	b := w.gitRepoIn(t, ws, "ledger")

	w.wrote(t, "ACME-10", a.Path, "reach into both")

	_ = b

	out := mustAsk(t, w, "task join", In{
		Task: "ACME-10", Repo: a.Path, Args: map[string]string{"name": "ledger"}, By: "operator",
	})
	if !strings.Contains(out.Said, "ledger") {
		t.Errorf("join answered %q", out.Said)
	}
}

// TestComparingBothSidesOfAChange. With a flow that checks one true
// thing, the same check runs on the base and on the work, and the reader
// is told what differs.
func TestComparingBothSidesOfAChange(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	flow := `{"name": "checked", "phases": [{"name": "work", "engine": "claude", "prompt": "work", "gates": [{"name": "true", "command": "true"}]}]}`

	if err := os.MkdirAll(w.store.FlowDir(), 0o755); err != nil {
		t.Fatalf("make the flows directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(w.store.FlowDir(), "checked.json"), []byte(flow), 0o600); err != nil {
		t.Fatalf("write the flow: %v", err)
	}

	mustAsk(t, w, "board new", In{
		Args: map[string]string{"id": "ACME-13", "text": "check both sides", "repo": r.Path, "flow": "checked"},
		By:   "operator",
	})

	dir, err := w.store.WorktreeDir(r.Path, "ACME-13")
	if err != nil {
		t.Fatalf("worktree dir: %v", err)
	}

	add := exec.Command("git", "worktree", "add", dir, "-b", "orbit/ACME-13")
	add.Dir = r.Path

	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v (%s)", err, out)
	}

	if err := os.WriteFile(filepath.Join(dir, "done.txt"), []byte("did it\n"), 0o600); err != nil {
		t.Fatalf("write the work: %v", err)
	}

	compared := mustAsk(t, w, "task compare", In{Task: "ACME-13", Repo: r.Path, By: "operator"})
	if compared.Said == "" {
		t.Error("compare said nothing at all")
	}
}

func TestTheBoardListsWhatIsThere(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-11", r.Path, "pay the thing")
	w.wrote(t, "ACME-12", r.Path, "ship it")

	out := mustAsk(t, w, "board list", In{By: "operator"})
	if !strings.Contains(out.Said, "ACME-11") || !strings.Contains(out.Said, "ACME-12") {
		t.Errorf("list answered:\n%s", out.Said)
	}
}
