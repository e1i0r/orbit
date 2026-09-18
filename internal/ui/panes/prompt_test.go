package panes

// The prompt a phase was given, word for word.
//
// The pane read zero. It is the one side of a run that used to be invisible
// — everything else can be read back, so a phase that came out strange could
// be examined from every direction except the one that caused it — and the
// whole reason to look at it is that what Orbit believes it sent and what it
// sent might differ.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// asked is one phase.asked entry: the prompt that phase was handed.
//
// The kind is the record's own spelling rather than the constant, because
// internal/ui/panes may not reach internal/record — the layers say so, and
// every other test in this package writes them the same way.
//
// Kept and Full are the same number, which is a prompt the record holds
// whole. The test that is about a prompt it did not hold widens Full itself.
func asked(phase, engine, text string) view.Entry {
	return view.Entry{
		Kind: "phase.asked", Phase: phase, Engine: engine, Text: text,
		Kept: 1, Full: 1,
	}
}

// plain is the pane as a reader sees it, with the colour taken out.
func plain(rows []string) string { return ansi.Strip(strings.Join(rows, "\n")) }

// TestTheLastAttemptIsTheOneAtTheTop. A phase that ran three times ran three
// times for a reason, and the reader is almost always asking about the run
// they just watched.
func TestTheLastAttemptIsTheOneAtTheTop(t *testing.T) {
	e := world(t, []view.Entry{
		asked("implement", "claude", "the first attempt"),
		asked("implement", "claude", "the second attempt"),
		asked("review", "codex", "the third attempt"),
	})

	rows, heads := Prompt(e)
	drawn := plain(rows)

	third := strings.Index(drawn, "the third attempt")
	second := strings.Index(drawn, "the second attempt")
	first := strings.Index(drawn, "the first attempt")

	if third < 0 || second < 0 || first < 0 {
		t.Fatalf("a prompt is missing from the pane:\n%s", drawn)
	}

	if third >= second || second >= first {
		t.Errorf("the prompts are in the order %d, %d, %d, want the newest first",
			third, second, first)
	}

	// Each row that folds stands for one of them, so a reader collapsing the
	// top one collapses the run they just watched.
	if len(heads) != 3 {
		t.Errorf("%d rows fold, want one per prompt", len(heads))
	}
}

// TestAPromptIsDrawnWordForWord. A pane that drew the headings would be
// drawing what this program believes it sent, and a line clipped to the
// measure is a line the engine was never handed.
func TestAPromptIsDrawnWordForWord(t *testing.T) {
	long := strings.Repeat("the prompt goes on ", 30)

	e := world(t, []view.Entry{asked("implement", "claude", "## what to do\n\n"+long)})

	drawn := plain(rows(t, e))

	if !strings.Contains(drawn, "## what to do") {
		t.Errorf("the prompt's own heading was not drawn:\n%s", drawn)
	}

	// Wrapped rather than cut: every word of it is on the screen somewhere.
	flat := strings.Join(strings.Fields(drawn), " ")
	if !strings.Contains(flat, strings.TrimSpace(long)) {
		t.Errorf("the prompt was cut rather than wrapped:\n%s", drawn)
	}
}

// TestAPromptSaysWhichPhaseAndWhoWasGivenIt. A run with four phases has four
// prompts, and a pane that did not say which was which would be four blocks
// of text a reader has to tell apart by reading them.
func TestAPromptSaysWhichPhaseAndWhoWasGivenIt(t *testing.T) {
	e := world(t, []view.Entry{asked("implement", "claude", "do the thing")})

	drawn := plain(rows(t, e))

	for _, want := range []string{"implement", "claude", "lines"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the head does not say %q:\n%s", want, drawn)
		}
	}
}

// TestAPhaseWithNoNameStillHasAHead. The record does not always say which
// phase a prompt belonged to, and a head that drew an empty column would
// read as a pane that failed rather than as a fact about the record.
func TestAPhaseWithNoNameStillHasAHead(t *testing.T) {
	e := world(t, []view.Entry{asked("", "claude", "do the thing")})

	if !strings.Contains(plain(rows(t, e)), "?") {
		t.Errorf("a prompt with no phase drew no stand-in:\n%s", plain(rows(t, e)))
	}
}

// TestAPromptTheRecordCutSaysSo. The size is what a reader checks before
// they scroll, and a length that silently meant "some of it" is the pane
// lying about the one thing it exists to show.
func TestAPromptTheRecordCutSaysSo(t *testing.T) {
	one := asked("implement", "claude", "do the thing")
	one.Full = one.Kept + 4096 //nolint:mnd // any number over Kept is "cut"

	e := world(t, []view.Entry{one})

	drawn := plain(rows(t, e))
	if !strings.Contains(drawn, "kept less than was sent") {
		t.Errorf("a prompt the record cut does not say so:\n%s", drawn)
	}

	// And one the record kept whole says nothing about it.
	whole := plain(rows(t, world(t, []view.Entry{asked("implement", "claude", "do the thing")})))
	if strings.Contains(whole, "kept less than was sent") {
		t.Errorf("a prompt kept whole claims it was cut:\n%s", whole)
	}
}

// TestARunWithNoPromptYetSaysSo. An empty pane is a reader wondering whether
// the run has not been asked anything or the pane is broken.
func TestARunWithNoPromptYetSaysSo(t *testing.T) {
	e := world(t, []view.Entry{{Kind: "task.started", Attempt: 1}})

	drawn, heads := Prompt(e)
	if heads != nil {
		t.Errorf("a run with no prompt offered %d rows to fold", len(heads))
	}

	if !strings.Contains(plain(drawn), "no phase of this run has been given a prompt") {
		t.Errorf("it drew:\n%s", plain(drawn))
	}
}

// TestAPaneThatCouldNotReadTheRecordSaysWhat. Nothing is hidden: a reading
// that failed says what failed, rather than drawing an empty pane that reads
// as a run with nothing in it.
func TestAPaneThatCouldNotReadTheRecordSaysWhat(t *testing.T) {
	e := world(t, nil)
	e.Failed = "the record of ACME-7 could not be read"

	drawn, heads := Prompt(e)
	if heads != nil {
		t.Error("a pane that failed still offered rows to fold")
	}

	if !strings.Contains(plain(drawn), e.Failed) {
		t.Errorf("it drew %q, want what stopped it", plain(drawn))
	}
}

// rows is the pane's lines, for the tests that do not care which fold.
func rows(t *testing.T, e Env) []string {
	t.Helper()

	out, _ := Prompt(e)

	return out
}
