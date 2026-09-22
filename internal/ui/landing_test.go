package ui

// A gesture says what it is doing, and says when it landed.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// fakeBoard answers took's one question without a board.
type fakeBoard struct {
	t  view.Task
	on bool
}

func (b fakeBoard) task(string) (view.Task, bool) { return b.t, b.on }

// TestAGestureSaysWhatItIsDoingAndThenThatItLanded.
//
// It said one thing, once, in the past tense — "asked ORB-121 to cancel" —
// and then nothing, while the run went on spending for another two minutes.
// Elio: I need to do it quickly, and with feedback, like cancelling...
// cancelled.
func TestAGestureSaysWhatItIsDoingAndThenThatItLanded(t *testing.T) {
	m := openOn(t, "PAY-1")
	m = m.awaiting("PAY-1", gestureCancel)

	if !strings.Contains(m.message, "cancelling") {
		t.Errorf("the band says %q when the key is pressed, want the present tense", m.message)
	}

	// The row says it too, and says it in the cell the reader is looking
	// at rather than only at the foot of the window.
	word, out := m.awaitedWord("PAY-1")
	if !out || !strings.Contains(word, "cancelling") {
		t.Errorf("the status cell says %q (out=%v), want cancelling…", word, out)
	}

	// Another task's row is untouched.
	if _, out := m.awaitedWord("PAY-2"); out {
		t.Error("a gesture on one task took over another task's status cell")
	}

	// Still running: nothing said, and the waiting is not over.
	running := view.Task{ID: "PAY-1", Band: view.Running}
	if said, over := m.tookAwaited(fakeBoard{running, true}); over || said != "" {
		t.Errorf("a task still running ended the wait with %q", said)
	}

	// Stopped: the past tense.
	stopped := view.Task{ID: "PAY-1", Band: view.Done}
	said, over := m.tookAwaited(fakeBoard{stopped, true})

	if !over {
		t.Fatal("the task stopped and the window is still waiting")
	}

	if !strings.Contains(said, "cancelled") {
		t.Errorf("it landed and the band says %q, want cancelled", said)
	}
}

// TestEachGestureKnowsWhatLandingLooksLike.
func TestEachGestureKnowsWhatLandingLooksLike(t *testing.T) {
	for _, c := range []struct {
		verb string
		task view.Task
		on   bool
		want bool
		why  string
	}{
		{gestureCancel, view.Task{Band: view.Running}, true, false, "still running"},
		{gestureCancel, view.Task{Band: view.Done}, true, true, "stopped"},
		{gestureCancel, view.Task{}, false, true, "gone from the board"},

		{gestureResume, view.Task{Band: view.NeedsYou}, true, false, "still waiting"},
		{gestureResume, view.Task{Band: view.Running}, true, true, "running again"},

		{gesturePause, view.Task{Band: view.Running, Live: view.LiveHeld}, true, false, "still held"},
		{gesturePause, view.Task{Band: view.NeedsYou}, true, true, "parked"},

		{gestureRequeue, view.Task{Band: view.Running}, true, false, "still running"},
		{gestureRequeue, view.Task{Band: view.ToDo}, true, true, "back in to do"},

		{gestureRead, view.Task{Band: view.Done}, true, false, "not read yet"},
		{gestureRead, view.Task{Band: view.Done, Read: true}, true, true, "read"},
		// Read is the one that does not treat a missing row as landed: a
		// task that left the board was not read, it was deleted.
		{gestureRead, view.Task{}, false, false, "gone, which is not read"},
	} {
		if got := landed(c.verb, c.task, c.on); got != c.want {
			t.Errorf("%s / %s: landed = %v, want %v", c.verb, c.why, got, c.want)
		}
	}
}

// TestTheWindowStopsPromisingAGestureThatNeverLands.
//
// A word written for a process that was already gone, or a run that died
// before it could act: the row would have said "cancelling…" for as long
// as the window stayed open.
func TestTheWindowStopsPromisingAGestureThatNeverLands(t *testing.T) {
	m := openOn(t, "PAY-1")
	m = m.awaiting("PAY-1", gestureCancel)

	running := fakeBoard{view.Task{ID: "PAY-1", Band: view.Running}, true}

	m.now = m.now.Add(awaitedFor / 2)
	if _, over := m.tookAwaited(running); over {
		t.Error("the window gave up on a gesture that has had half its time")
	}

	m.now = m.now.Add(awaitedFor + time.Second)

	said, over := m.tookAwaited(running)
	if !over {
		t.Fatal("the window is still promising a gesture that never landed")
	}

	if said != "" {
		t.Errorf("it gave up and claimed %q; the board underneath is the truth", said)
	}
}

// TestEveryGestureThatChangesATaskSaysSo is the list, so the next one
// added is noticed.
//
// Elio asked it plainly: do we support all the actions, deleting, etc? Six
// of them did and four did not, and nothing said which.
func TestEveryGestureThatChangesATaskSaysSo(t *testing.T) {
	m := openOn(t, "PAY-1")

	for _, verb := range []string{
		gesturePause, gestureResume, gestureContinue, gestureCancel,
		gestureSkip, gestureRequeue, gestureRead, gestureStart, gestureDelete,
	} {
		armed := m.awaiting("PAY-1", verb)

		if armed.message == "" || strings.HasPrefix(armed.message, "{") {
			t.Errorf("%s says %q on the band when the key is pressed", verb, armed.message)
		}

		word, out := armed.awaitedWord("PAY-1")
		if !out || word == "" {
			t.Errorf("%s puts nothing in the status cell", verb)
		}

		if !strings.HasSuffix(word, "…") {
			t.Errorf("%s says %q in the cell, want the present tense with an ellipsis", verb, word)
		}

		if done := armed.doneSaid("PAY-1", verb); done == "" || strings.Contains(done, "…") {
			t.Errorf("%s lands with %q, want a finished sentence", verb, done)
		}
	}
}

// TestADeletedTaskLandsWhenItsRowIsGone.
func TestADeletedTaskLandsWhenItsRowIsGone(t *testing.T) {
	m := openOn(t, "PAY-1").awaiting("PAY-1", gestureDelete)

	// Still listed: the delete has not finished, whatever else it removed.
	if _, over := m.tookAwaited(fakeBoard{view.Task{ID: "PAY-1", Band: view.Done}, true}); over {
		t.Error("a task still on the board reads as deleted")
	}

	said, over := m.tookAwaited(fakeBoard{on: false})
	if !over || !strings.Contains(said, "gone") {
		t.Errorf("a task off the board = (%q, %v), want it to say it is gone", said, over)
	}
}
