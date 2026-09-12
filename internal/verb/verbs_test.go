package verb

// Every verb that acts, asked the way every door asks it.
//
// These call Run, not the bodies: what is covered is the dispatch and the
// doing together, so a verb that is declared and not done fails here rather
// than in one surface. The world is the fake in world_test.go — a real
// store, canned answers for everything past it.

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// TestWritingAndSteeringATask. Writing, stopping, taking back, noting,
// reading, deleting: the verbs that move a task between bands.
func TestWritingAndSteeringATask(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-1", r.Path, "pay the thing")

	for _, name := range []string{"pause", "resume", "continue", "skip"} {
		out := mustAsk(t, w, name, In{Task: "ACME-1", By: "operator"})
		if !strings.Contains(out.Said, "ACME-1") {
			t.Errorf("%s answered %q, which names no task", name, out.Said)
		}
	}

	mustAsk(t, w, "note", In{Task: "ACME-1", Args: map[string]string{"text": "cents, not floats"}, By: "operator"})
	mustAsk(t, w, "direct", In{Task: "ACME-1", Args: map[string]string{"text": "use redis"}, By: "operator"})
	mustAsk(t, w, "read", In{Task: "ACME-1", By: "operator"})
	mustAsk(t, w, "requeue", In{Task: "ACME-1", Args: map[string]string{"why": "wrong brief"}, By: "operator"})

	// Cancelling asks a live process to stop, so the test lends it one:
	// a sleep with a marker naming it, killed when the test ends.
	cmd := holdARun(t, w, "ACME-1")
	mustAsk(t, w, "cancel", In{Task: "ACME-1", By: "operator"})
	_ = cmd.Process.Kill() //nolint:errcheck // the cleanup kills it again; this only hurries it

	out := mustAsk(t, w, "history", In{Task: "ACME-1", By: "operator"})
	if !strings.Contains(out.Said, "ACME-1") {
		t.Errorf("history answered %q, which names no task", out.Said)
	}

	mustAsk(t, w, "delete", In{Task: "ACME-1", By: "operator"})
	mustRefuse(t, w, "read", In{Task: "ACME-1", By: "operator"})
}

// TestAnsweringWhatATaskWaitsFor. Approving, permitting, marking: the
// verbs that answer a question somebody else put. Asked with nothing
// waiting, they say so rather than deciding anything.
func TestAnsweringWhatATaskWaitsFor(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-2", r.Path, "take the keyboard")

	if err := mustRefuse(t, w, "approve", In{Task: "ACME-2", By: "operator"}); !strings.Contains(err.Error(), "ACME-2") {
		t.Errorf("approve refused with %q, which names no task", err)
	}

	if err := mustRefuse(t, w, "permit", In{Task: "ACME-2", By: "operator"}); !strings.Contains(err.Error(), "ACME-2") {
		t.Errorf("permit refused with %q, which names no task", err)
	}

	mustAsk(t, w, "critical", In{Task: "ACME-2", Args: map[string]string{"on": "true"}, By: "operator"})
	mustAsk(t, w, "critical", In{Task: "ACME-2", Args: map[string]string{"on": "false"}, By: "operator"})
}

// TestPermittingWhatWasStopped. A snapshot before a critical action is the
// question; permit answers it yes or no, and either way the question is
// gone afterwards.
func TestPermittingWhatWasStopped(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	tk := w.wrote(t, "ACME-20", r.Path, "merge it")

	dir, err := w.store.WorktreeDir(r.Path, "ACME-20")
	if err != nil {
		t.Fatalf("worktree dir: %v", err)
	}

	add := exec.Command("git", "worktree", "add", dir, "-b", "orbit/ACME-20")
	add.Dir = r.Path

	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v (%s)", err, out)
	}

	snapshot := func() {
		t.Helper()

		if _, err := task.Snapshot(w.store, tk, r, dir, task.Action{Name: "merge", Plan: "merge the pull request"}); err != nil {
			t.Fatalf("snapshot: %v", err)
		}
	}

	snapshot()

	yes := mustAsk(t, w, "permit", In{Task: "ACME-20", Args: map[string]string{"yes": "true"}, By: "operator"})
	if !strings.Contains(yes.Said, "may go ahead") {
		t.Errorf("permit answered %q", yes.Said)
	}

	snapshot()

	no := mustAsk(t, w, "permit", In{Task: "ACME-20", By: "operator"})
	if !strings.Contains(no.Said, "refused") {
		t.Errorf("permit answered %q", no.Said)
	}
}

// TestAskingAboutNothingKnown. Every verb refused for a task nobody wrote
// down, which is the whole of what the record has to say about it.
func TestAskingAboutNothingKnown(t *testing.T) {
	w := worldOf(t)

	for _, name := range []string{
		"pause", "cancel", "requeue", "note", "direct", "approve", "read",
		"history", "delete", "permit", "critical", "join", "diff",
		"tree", "impact", "compare", "flow", "show",
	} {
		in := In{Task: "ACME-404", By: "operator"}
		if name == "note" || name == "direct" {
			in.Args = map[string]string{"text": "hi"}
		}

		if _, err := Run(ctxOf(), w, name, in); err == nil {
			t.Errorf("%s was accepted about a task nobody wrote", name)
		}
	}
}

// TestJoiningAnotherRepository. Joining names a repository; one nobody
// knows is a refusal that lists what there is.
func TestJoiningAnotherRepository(t *testing.T) {
	w := worldOf(t)
	a := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-3", a.Path, "reach into both")

	if err := mustRefuse(t, w, "join", In{
		Task: "ACME-3", Args: map[string]string{"name": "nowhere"}, By: "operator",
	}); !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("join refused with %q, which names nothing", err)
	}
}

// TestReconcilingWhatIsLeft. Nothing running and nothing dead: the sweep
// says everything is accounted for.
func TestReconcilingWhatIsLeft(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-4", r.Path, "leave no process behind")

	out := mustAsk(t, w, "reconcile", In{Task: "ACME-4", By: "operator"})
	if out.Said == "" {
		t.Error("reconcile said nothing at all")
	}

	swept := mustAsk(t, w, "reconcile", In{Args: map[string]string{}, By: "operator"})
	if swept.Said == "" {
		t.Error("the sweep said nothing at all")
	}
}

// TestHandingWorkToTheWorld. Opening, merging and closing go through the
// Deliver port, which is the seam a test Borg would have to cut to reach
// gh — so the fake answers the URL and the verbs are the sentences.
func TestHandingWorkToTheWorld(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-5", r.Path, "ship it")

	for _, name := range []string{"pr", "merge", "close-pr"} {
		out := mustAsk(t, w, name, In{Task: "ACME-5", Repo: r.Path, By: "operator"})
		if !strings.Contains(out.Said, "example.test") {
			t.Errorf("%s answered %q, want the delivered URL", name, out.Said)
		}
	}
}

// TestSayingAndLearning. Saying records a line for the supervisor's loop;
// learning writes down what the next run should be told.
func TestSayingAndLearning(t *testing.T) {
	w := worldOf(t)

	mustAsk(t, w, "say", In{Args: map[string]string{"text": "never force-push"}, By: "operator"})

	if len(w.said) != 1 || w.said[0].text != "never force-push" {
		t.Errorf("say recorded %v, want the one line", w.said)
	}

	out := mustAsk(t, w, "learn", In{
		Args: map[string]string{"text": "amounts are cents", "repo": "/src/acme"}, By: "operator",
	})
	if !strings.Contains(out.Said, "cents") {
		t.Errorf("learn answered %q", out.Said)
	}

	if len(w.learnt) != 1 {
		t.Errorf("learn wrote down %d facts, want one", len(w.learnt))
	}
}

// TestTakingATerminal. The one verb that needs a terminal answers through
// the port, because a way in that has none says so rather than pretending.
func TestTakingATerminal(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-6", r.Path, "type at it yourself")

	out := mustAsk(t, w, "take", In{Task: "ACME-6", Repo: r.Path, By: "operator"})
	if out.Said == "" {
		t.Error("take said nothing at all")
	}
}

// TestTheBoardAsAReading. Listing, showing, and the board behind them:
// rows the record already folded, read back the same way.
func TestTheBoardAsAReading(t *testing.T) {
	w := worldOf(t)
	w.board = board.Board{
		Tasks: []view.Task{{ID: "ACME-7", Title: "pay the thing"}},
	}

	out := mustAsk(t, w, "list", In{By: "operator"})
	if !strings.Contains(out.Said, "ACME-7") {
		t.Errorf("list answered %q, want the row", out.Said)
	}

	w.log = map[string][]view.Entry{
		"ACME-7": {{Kind: "task.created", Text: "pay the thing"}},
	}

	shown := mustAsk(t, w, "show", In{Task: "ACME-7", By: "operator"})
	if !strings.Contains(shown.Said, "ACME-7") {
		t.Errorf("show answered %q, want the task", shown.Said)
	}

	mustRefuse(t, w, "show", In{Task: "ACME-404", By: "operator"})
}
