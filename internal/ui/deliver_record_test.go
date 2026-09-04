package ui

// The delivery keys write what they asked for onto the task they were
// pressed about. Everything else about them happens somewhere the reader is
// not looking — a thread, an engine, a command — so these events are the
// whole of what the task view can show about a key that was pressed.

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// deliverWindow is a window open on one task in a checkout, with the
// supervisor thread and the record both willing to be written to.
func deliverWindow(t *testing.T) (Model, *[]Delivery) {
	t.Helper()

	m, _ := testModel(t, 120, 30)
	m.board = fixtureBoard([]view.Task{{
		ID: "ACME-9", Title: "a task in a checkout", Band: view.NeedsYou,
		Repo: "payments", RepoPath: "/work/payments", Since: ago(time.Minute),
	}}, 1)
	m.detail = "ACME-9"
	m.opts.RecordSupervisor = func(_, _, _ string) error { return nil }
	m.opts.AskSupervisor = func(_, _ string) (string, error) { return "", nil }

	written := &[]Delivery{}
	m.opts.RecordDeliver = func(_ view.Task, d Delivery) error {
		*written = append(*written, d)
		return nil
	}

	return m, written
}

func TestAVerbHandedToTheSupervisorIsWrittenOnTheTask(t *testing.T) {
	m, written := deliverWindow(t)

	next, cmd := m.fixChecks()
	if cmd == nil {
		t.Fatal("fix checks asked for nothing")
	}

	if len(*written) != 1 {
		t.Fatalf("wrote %d events, want the ask", len(*written))
	}

	if got := (*written)[0]; got.Verb != "FIX CHECKS" || got.By != "supervisor" || got.Done {
		t.Errorf("wrote %+v, want the ask, under the caption, handed to the supervisor", got)
	}

	// The answer lands minutes later, down the same wire every other
	// supervisor reply comes back on.
	done, _ := asModel(t, next).Update(supervisorReplyMsg{Text: "the checks are green"})

	if len(*written) != 2 {
		t.Fatalf("wrote %d events, want the answer as well", len(*written))
	}

	if got := (*written)[1]; !got.Done || got.Verb != "FIX CHECKS" ||
		got.Text != "the checks are green" {
		t.Errorf("wrote %+v, want what came back, against the verb that asked", got)
	}

	// And nothing is left waiting, so the next reply — one the operator
	// typed for, or a turn of autopilot — closes nothing.
	if v := asModel(t, done).delivering.verb; v != "" {
		t.Errorf("still waiting on %q after it was answered", v)
	}
}

// A supervisor that broke still answers: the verb came back, and why it
// broke is what the reader has to act on.
func TestABrokenVerbIsWrittenDownAsAnAnswer(t *testing.T) {
	m, written := deliverWindow(t)

	next, _ := m.deliverPR()
	asModel(t, next).Update(supervisorReplyMsg{Err: errors.New("gh is not logged in")})

	if len(*written) != 2 {
		t.Fatalf("wrote %d events, want the ask and the answer", len(*written))
	}

	if got := (*written)[1]; got.Failure == nil {
		t.Errorf("wrote %+v, want the reason it broke", got)
	}
}

// Merge and close are commands rather than asks, and they are written down
// the same way: the reader pressed the same kind of key and is owed the same
// account of it.
func TestTheCommandVerbsAreWrittenDownToo(t *testing.T) {
	m, written := deliverWindow(t)

	next, _ := m.mergePR()

	if len(*written) != 1 || (*written)[0].Verb != "MERGE PR" || (*written)[0].By != "merge" {
		t.Fatalf("wrote %+v, want the ask naming the command that carries it", *written)
	}

	asModel(t, next).Update(commandMsg{Name: "merge", Text: "merged"})

	if len(*written) != 2 || !(*written)[1].Done {
		t.Fatalf("wrote %+v, want the command's answer against the verb", *written)
	}
}

// A window handed no port is a window that writes nothing, and the verb
// still goes ahead: the work was asked for either way.
func TestAVerbWithNoRecordPortStillAsks(t *testing.T) {
	m, _ := deliverWindow(t)
	m.opts.RecordDeliver = nil

	if _, cmd := m.fixChecks(); cmd == nil {
		t.Error("fix checks asked for nothing when the record could not be written")
	}
}

func TestTheFlowTreeDrawsWhatWasAskedForByHand(t *testing.T) {
	m, _ := deliverWindow(t)
	m.entries = []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Minute)},
		{
			Kind: "deliver.answered",
			Verb: "CREATE PR",
			Text: "opened pull request 12",
			At:   ago(time.Minute),
		},
		{Kind: "deliver.asked", Verb: "FIX CHECKS", By: "supervisor", At: ago(30 * time.Second)},
	}

	rows, _ := m.flowRows()
	drawn := strings.Join(rows, "\n")

	wants := []string{"Asked for by hand", "CREATE PR", "came back", "FIX CHECKS", "still out"}
	for _, want := range wants {
		if !strings.Contains(drawn, want) {
			t.Errorf("the flow tree does not say %q:\n%s", want, drawn)
		}
	}
}

// The answer pairs with the ask of the same verb, and a verb asked for twice
// is two rows: the second ask is not closed by the first answer.
func TestEachAskIsClosedByItsOwnAnswer(t *testing.T) {
	m, _ := deliverWindow(t)
	m.entries = []view.Entry{
		{Kind: "deliver.asked", Verb: "FIX CHECKS", At: ago(3 * time.Minute)},
		{Kind: "deliver.answered", Verb: "FIX CHECKS", Text: "green", At: ago(2 * time.Minute)},
		{Kind: "deliver.asked", Verb: "FIX CHECKS", At: ago(time.Minute)},
	}

	steps := m.byHand()
	if len(steps) != 2 {
		t.Fatalf("read %d steps, want one per ask", len(steps))
	}

	if !steps[0].done || steps[1].done {
		t.Errorf("steps = %+v, want the first answered and the second still out", steps)
	}
}

// The timeline is where a reader looks for what happened and when. A verb
// that is on the flow tree and not here is a verb with no moment attached
// to it.
func TestTheTimelineNamesTheVerbThatWasAskedFor(t *testing.T) {
	m, _ := deliverWindow(t)
	m.entries = []view.Entry{
		{Kind: "deliver.asked", Verb: "RESOLVE COMMENTS", By: "supervisor", At: ago(time.Minute)},
		{
			Kind:  "deliver.answered",
			Verb:  "RESOLVE COMMENTS",
			Cause: "no threads to answer",
			At:    ago(time.Second),
		},
	}

	rows, _, _ := m.logRows()
	drawn := strings.Join(rows, "\n")

	wants := []string{
		"asked for",
		"RESOLVE COMMENTS",
		"came back broken",
		"no threads to answer",
	}
	for _, want := range wants {
		if !strings.Contains(drawn, want) {
			t.Errorf("the timeline does not say %q:\n%s", want, drawn)
		}
	}
}

// The checkout handed to the supervisor is the task's worktree when one
// exists, not the parent repository, so git and gh commands run on the
// task's actual branch.
func TestSupervisorDeliveryVerbUsesWorktreeWhenPresent(t *testing.T) {
	m, _ := deliverWindow(t)
	dir := t.TempDir()

	var askedBody string

	m.opts.AskSupervisor = func(_, prompt string) (string, error) {
		askedBody = prompt
		return "done", nil
	}
	m.opts.Reader = &fakeReader{worktree: dir}

	_, cmd := m.fixChecks()
	if cmd == nil {
		t.Fatal("fix checks failed to ask supervisor")
	}

	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				c()
			}
		}
	}

	if !strings.Contains(askedBody, dir) {
		t.Errorf("prompt handed to supervisor does not name worktree %q:\n%s", dir, askedBody)
	}
}

// When the task's worktree does not exist on disk, the supervisor is not
// pointed to a phantom path or to the main repo; the window says there is no
// checkout.
func TestSupervisorDeliveryVerbRefusesWhenWorktreeMissing(t *testing.T) {
	m, _ := deliverWindow(t)
	m.opts.Reader = &fakeReader{worktree: "/nonexistent/path/to/worktree"}

	next, cmd := m.fixChecks()
	if cmd != nil {
		t.Error("asked supervisor when worktree does not exist on disk")
	}

	if got := asModel(t, next).message; !strings.Contains(got, "checkout") {
		t.Errorf("it said %q, want it saying there is nowhere to work", got)
	}
}
