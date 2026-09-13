package verb

// The board as a whole: one repository's tasks, the sweep that closes the
// runs nothing is walking any more, and taking a task off it.

import (
	"strings"
	"testing"
)

// TestTheBoardCanBeAskedAboutOneRepository.
//
// Without a repository the listing is every task under the root, which is
// what the window draws. With one it is that checkout's own, which is what
// somebody standing in it means — and the two read the same way, because a
// reader who moved between them should not have to learn a second shape.
func TestTheBoardCanBeAskedAboutOneRepository(t *testing.T) {
	w := worldOf(t)
	here := w.gitRepo(t, "acme")
	elsewhere := w.gitRepo(t, "ledger")

	w.wrote(t, "ACME-30", here.Path, "pay the thing")
	w.wrote(t, "LED-30", elsewhere.Path, "ship it")

	out := mustAsk(t, w, "board list", In{Args: map[string]string{"repo": here.Path}, By: "operator"})

	if !strings.Contains(out.Said, "ACME-30") {
		t.Errorf("the repository's own task is not in %q", out.Said)
	}

	if strings.Contains(out.Said, "LED-30") {
		t.Errorf("a task of another checkout is in %q", out.Said)
	}
}

// TestAskingAboutARepositoryThatIsNotOne is the typo: a path that is not a
// checkout is a refusal and not an empty board, which would read as "there
// is nothing here" about a question that was never asked.
func TestAskingAboutARepositoryThatIsNotOne(t *testing.T) {
	w := worldOf(t)

	if err := refuseErr(t, w, "board list",
		In{Args: map[string]string{"repo": t.TempDir()}, By: "operator"}); err == nil {
		t.Error("a directory that is no checkout answered as a board")
	}
}

// TestReconcileSweepsEveryTaskOfARepository.
//
// Named, it is one task; unnamed, it is the repository — the narrowing and
// not the whole of it. A sweep over tasks nothing is running finds nothing
// to close, and says so rather than saying nothing.
func TestReconcileSweepsEveryTaskOfARepository(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-31", r.Path, "pay the thing")
	w.wrote(t, "ACME-32", r.Path, "ship it")

	swept := mustAsk(t, w, "board reconcile", In{Repo: r.Path, By: "operator"})
	if swept.Said == "" {
		t.Error("a sweep of the whole repository said nothing at all")
	}

	one := mustAsk(t, w, "board reconcile",
		In{Repo: r.Path, Args: map[string]string{"task": "ACME-31"}, By: "operator"})
	if !strings.Contains(one.Said, "ACME-31") {
		t.Errorf("reconciling one task answered %q, which names no task", one.Said)
	}
}

// TestATaskCanBeTakenOffTheBoard, and the board is told: the set of tasks
// changed, and a surface that was not told shows a row for something that is
// not there any more.
func TestATaskCanBeTakenOffTheBoard(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-33", r.Path, "pay the thing")

	gone := mustAsk(t, w, "task delete", In{Task: "ACME-33", Repo: r.Path, By: "operator"})
	if !strings.Contains(gone.Said, "ACME-33") {
		t.Errorf("deleting answered %q, which names no task", gone.Said)
	}

	out := mustAsk(t, w, "board list", In{By: "operator"})
	if strings.Contains(out.Said, "ACME-33") {
		t.Errorf("a deleted task is still on the board: %q", out.Said)
	}
}

// TestARunDoesNotStartWhileFinishedTasksSitUnread.
//
// The cap is the whole of what holds a board back from running away from
// the person watching it, so the refusal has to say what it is and how to
// move it. A run refused with "no" and nothing else is a run somebody
// retries.
func TestARunDoesNotStartWhileFinishedTasksSitUnread(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-34", r.Path, "pay the thing")
	w.unread = 99

	err := refuseErr(t, w, "task start", In{Task: "ACME-34", Repo: r.Path, By: "operator"})
	if !strings.Contains(err.Error(), "unread") {
		t.Errorf("the refusal reads %q, without saying what held it back", err)
	}
}

// TestATaskWrittenAndStartedStandsWhenTheRunWillNotStart.
//
// Two gestures in one command, and they are not one thing: the task is
// written and that stands. A run that would not start is a second thing to
// tell the reader, not a reason to pretend the task is not there.
func TestATaskWrittenAndStartedStandsWhenTheRunWillNotStart(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.unread = 99

	out, err := Run(ctxOf(), w, "board new", In{
		Args: map[string]string{"id": "ACME-35", "text": "ship it", "repo": r.Path, "run": "true"},
		By:   "operator",
	})
	if err == nil {
		t.Fatal("the run started with the board over its unread cap")
	}

	if !strings.Contains(out.Said, "ACME-35") {
		t.Errorf("the answer reads %q, without saying the task was written", out.Said)
	}

	on := mustAsk(t, w, "board list", In{By: "operator"})
	if !strings.Contains(on.Said, "ACME-35") {
		t.Errorf("the task is not on the board: %q", on.Said)
	}
}

// TestCorrectingAndStartingAgainIsTwoThings.
//
// `-restart` is the second half of a correction: the run in flight is
// stopped so the next one starts having read it. When the next one will not
// start, the correction is still on the record — it is what the reader came
// to do, and losing it because a cap was reached would be the wrong half to
// drop.
func TestCorrectingAndStartingAgainIsTwoThings(t *testing.T) {
	w := worldOf(t)
	r := w.gitRepo(t, "acme")

	w.wrote(t, "ACME-37", r.Path, "pay the thing")
	w.unread = 99

	at := In{
		Task: "ACME-37", Repo: r.Path, By: "operator",
		Args: map[string]string{"text": "use redis", "restart": "true"},
	}

	if err := refuseErr(t, w, "task direct", at); !strings.Contains(err.Error(), "unread") {
		t.Errorf("the refusal reads %q, without saying what held the run back", err)
	}

	// The correction outlived the run that would not start.
	told := mustAsk(t, w, "task history", In{Task: "ACME-37", Repo: r.Path, By: "operator"})
	if !strings.Contains(told.Said, "use redis") {
		t.Errorf("the correction is not in the record:\n%s", told.Said)
	}
}
