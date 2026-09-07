package panes

// Every pane, drawn over a record that has one of everything in it.
//
// The panes are read and never pressed, so what a test of one can ask is
// what it says: that the fact is on the screen, that it is on the row the
// hit test counts it at, and that what is folded is the part a reader has
// not asked for yet.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// busy is a run that did one of everything: two attempts, a gate that
// passed and one that failed, a refusal, a note, a session beside it, a
// question the model stopped on, and a delivery verb still out.
func busy() []view.Entry {
	return []view.Entry{
		{Kind: "task.created", At: ago(time.Hour), Text: "Reconcile settlements\n\nMatch on the bank reference."},
		{Kind: "task.started", Attempt: 1, At: ago(55 * time.Minute)},
		{Kind: "phase.started", Phase: "implement", At: ago(50 * time.Minute), Engine: "claude", Model: "opus"},
		{Kind: "gate.passed", Phase: "implement", Gate: "build", Text: "go build ./...", At: ago(45 * time.Minute)},
		{
			Kind: "gate.failed", Phase: "implement", Gate: "tests", At: ago(44 * time.Minute),
			Text:  "go test ./... -run TestSomethingWithAVeryLongNameThatWillNotFitOnOneRowOfThisPaneAtAll",
			Cause: "two tests failed in internal/store, and the fixture they share was left half written",
		},
		{
			Kind: "phase.refused", Phase: "implement", Tool: "Bash", At: ago(43 * time.Minute),
			Text: "git push origin main\nthe branch belongs to the operator or the runner,\nand nothing in a run may publish one",
		},
		{Kind: "phase.retried", Phase: "implement", Attempt: 2, Gate: "tests", At: ago(42 * time.Minute), Cost: 0.25},
		{Kind: "task.started", Attempt: 2, At: ago(41*time.Minute + time.Second)},
		{Kind: "phase.started", Phase: "implement", Attempt: 2, At: ago(41 * time.Minute), Engine: "claude", Model: "opus"},
		{
			Kind: "phase.finished", Phase: "implement", Attempt: 2, At: ago(30 * time.Minute), Cost: 0.5,
			Session: "sess-7", Text: "# What I did\n\nMatched on the bank reference and added the upsert.",
			Kept: 1200, Full: 48000,
		},
		{Kind: "task.noted", At: ago(25 * time.Minute), Attempt: 2, Text: "Check the index on the reference column."},
		{Kind: "task.dialogue", At: ago(24 * time.Minute), By: "cli", Text: "opened a session\nand looked at the schema"},
		{Kind: "phase.waiting", Phase: "review", At: ago(20 * time.Minute), Cause: "should I drop the old column?"},
		{Kind: "deliver.asked", Verb: "PR", By: "supervisor", At: ago(10 * time.Minute)},
	}
}

// TestEveryPaneSaysWhenTheRecordCouldNotBeRead. A pane that is empty because
// nothing happened and one that is empty because the read failed are two
// different facts, and the second is the one a reader can act on.
func TestEveryPaneSaysWhenTheRecordCouldNotBeRead(t *testing.T) {
	e := world(t, busy())
	e.Failed = "orbit.log: permission denied"

	for name, drawn := range map[string]string{
		"gates":    text(first(Gates(e))),
		"refused":  text(first(Refused(e))),
		"report":   text(first(Report(e))),
		"notes":    text(first(Notes(e))),
		"timeline": text(Timeline(e).Rows),
		"overview": text(Overview(e)),
	} {
		if !strings.Contains(drawn, "permission denied") {
			t.Errorf("the %s pane does not say why it is empty:\n%s", name, drawn)
		}
	}
}

// TestAGateThatFailedCarriesTheReasonAndTheCommand, and folds because it has
// more to say than a row: a check whose whole story fits beside its name is
// offered no arrow, because opening it would put nothing new on the screen.
func TestAGateThatFailedCarriesTheReasonAndTheCommand(t *testing.T) {
	e := world(t, busy())

	shut, heads := Gates(e)
	if len(heads) == 0 {
		t.Fatal("no gate on the pane offers to open, and one of them has two lines of reason")
	}

	if got := text(shut); !strings.Contains(got, "1/2") {
		t.Errorf("the pane does not count what passed:\n%s", got)
	}

	e.RowOpen = func(int) bool { return true }

	open := text(first(Gates(e)))
	if !strings.Contains(open, "was left half written") {
		t.Errorf("an opened gate still hides why it failed:\n%s", open)
	}
}

// TestARefusalIsShownWholeAndNotItsFirstLine. What the sandbox writes down
// is a paragraph often enough that it cannot be set on one row, and cutting
// it at the first break put the rest of it on no row at all — which left the
// row with nothing to open.
func TestARefusalIsShownWholeAndNotItsFirstLine(t *testing.T) {
	e := world(t, busy())
	e.RowOpen = func(int) bool { return true }

	got := text(first(Refused(e)))
	if !strings.Contains(got, "nothing in a run may publish one") {
		t.Errorf("the refusal is cut off before the part that says why:\n%s", got)
	}

	// And the standing rules are under it, whatever this run did.
	if !strings.Contains(got, "git push") {
		t.Errorf("the sandbox's own rules are not on the pane:\n%s", got)
	}
}

// TestTheNotesTabCarriesBothSidesOfTheDialogue: what the operator filed,
// what happened beside the run, and what the model stopped to ask.
func TestTheNotesTabCarriesBothSidesOfTheDialogue(t *testing.T) {
	e := world(t, busy())
	e.RowOpen = func(int) bool { return true }

	got := text(first(Notes(e)))
	for _, want := range []string{
		"Check the index",              // the note
		"CLI",                          // the session beside the run
		"should I drop the old column", // what the model asked
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the notes tab does not carry %q:\n%s", want, got)
		}
	}

	// A note the run has read says so, because the next run reads the notes
	// and a reader wants to know which of theirs have landed.
	if !strings.Contains(got, "read by run 2") {
		t.Errorf("the notes tab does not say which run read the note:\n%s", got)
	}
}

// TestTheReportIsSeamedWhereOneAttemptEnds, and says what was thrown away
// where the engine printed more than the record kept.
func TestTheReportIsSeamedWhereOneAttemptEnds(t *testing.T) {
	e := world(t, busy())

	rows, seams := Report(e)
	if len(seams) == 0 {
		t.Fatal("a record with two attempts drew no seam between them")
	}

	got := text(rows)
	for _, want := range []string{"Matched on the bank reference", "1,200 of 48,000", "sess-7"} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not carry %q:\n%s", want, got)
		}
	}

	// An attempt the reader shut is not drawn, and the pane still knows it
	// holds a report: counted before it is drawn, or a closed attempt reads
	// as a run that wrote nothing.
	e.AttemptOpen = func(int) bool { return false }

	if shut := text(first(Report(e))); strings.Contains(shut, "no engine report") {
		t.Errorf("a shut attempt made the pane say the run wrote nothing:\n%s", shut)
	}
}

// TestTheTimelineIsOneRowPerEventAndFoldsTheLongOnes.
func TestTheTimelineIsOneRowPerEventAndFoldsTheLongOnes(t *testing.T) {
	e := world(t, busy())

	drawn := Timeline(e)
	if len(drawn.Seams) == 0 {
		t.Error("the timeline drew no rule between the two attempts")
	}

	if len(drawn.Heads) == 0 {
		t.Error("nothing on the timeline offers to open, and the refusal is a paragraph")
	}

	got := text(drawn.Rows)
	for _, want := range []string{"gate passed", "gate failed", "refused", "finished", "waiting"} {
		if !strings.Contains(got, want) {
			t.Errorf("the timeline does not say %q happened:\n%s", want, got)
		}
	}

	// Shut, the detail is a qualifier of the word beside it; opened, it is
	// what the reader asked to read.
	e.RowOpen = func(int) bool { return true }

	if open := text(Timeline(e).Rows); !strings.Contains(open, "the branch belongs to the operator") {
		t.Errorf("an opened row still hides the rest of what was refused:\n%s", open)
	}
}

// TestAPaneWithNothingInItSaysSoRatherThanDrawingAnEmptyPage.
func TestAPaneWithNothingInItSaysSoRatherThanDrawingAnEmptyPage(t *testing.T) {
	e := world(t, nil)

	for name, want := range map[string]string{
		"gates":    "no verification gates",
		"notes":    "no notes or dialogue",
		"report":   "no engine report",
		"timeline": "nothing has been recorded",
	} {
		drawn := map[string]string{
			"gates":    text(first(Gates(e))),
			"notes":    text(first(Notes(e))),
			"report":   text(first(Report(e))),
			"timeline": text(Timeline(e).Rows),
		}[name]

		if !strings.Contains(drawn, want) {
			t.Errorf("the empty %s pane drew %q, want it to say %q", name, drawn, want)
		}
	}

	// The refused pane is the one that says the opposite: nothing denied is
	// something worth telling the reader.
	if got := text(first(Refused(e))); !strings.Contains(got, "no commands or actions were denied") {
		t.Errorf("a run nothing was refused in drew %q", got)
	}
}
