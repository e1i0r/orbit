package ui

// What Orbit knows, down the side of the supervisor screen.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/e1i0r/orbit/internal/knowledge"
)

func knowing(t *testing.T, w int, facts ...knowledge.Fact) Model {
	t.Helper()

	m, _ := testModel(t, w, 30)
	m.opts.RecordSupervisor = func(string, string, string) error { return nil }
	m.opts.Knows = func() []knowledge.Fact { return facts }

	return m.openSupervisor()
}

func known(phrase string, sc knowledge.Scope) knowledge.Fact {
	return knowledge.Fact{Scope: sc, Source: knowledge.Human, Phrase: phrase}
}

func stopper(phrase string, sc knowledge.Scope) knowledge.Fact {
	f := known(phrase, sc)
	f.Stops, f.Check = true, "false"

	return f
}

func sideOf(t *testing.T, m Model, w int) string {
	t.Helper()

	return ansi.Strip(strings.Join(m.supervisorRows(30, w), "\n"))
}

// TestTheSideSaysWhatOrbitKnowsHere.
//
// The same facts a phase started in this repository would be told, in the
// place the operator is already looking when they write another one. A rule
// nobody can see is a rule nobody trusts.
func TestTheSideSaysWhatOrbitKnowsHere(t *testing.T) {
	drawn := sideOf(t, knowing(t, 180,
		known("the PRs are written in English", knowledge.Scope{Kind: knowledge.General}),
		known("never discard an error with _", knowledge.Scope{Kind: knowledge.Language, Lang: "go"}),
		known("the fuzz tests hang sometimes", knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}),
	), 180)

	for _, said := range []string{"the PRs are written", "never discard an error", "the fuzz tests hang"} {
		if !strings.Contains(drawn, said) {
			t.Errorf("the side does not say %q:\n%s", said, drawn)
		}
	}
}

// TestTheSideIsInTwoCategories: the ones that stop the work, and the ones
// that only reach the prompt. It is the difference the whole store is built
// around, so it is the first thing the side draws.
func TestTheSideIsInTwoCategories(t *testing.T) {
	m := knowing(t, 180,
		known("something to keep in mind", knowledge.Scope{Kind: knowledge.General}),
		stopper("no UPDATE in ledger", knowledge.Scope{Kind: knowledge.General}),
	)

	drawn := sideOf(t, m, 180)

	rules, aware := strings.Index(drawn, "Rules"), strings.Index(drawn, "Aware")
	if rules < 0 || aware < 0 {
		t.Fatalf("the side has no two categories:\n%s", drawn)
	}

	// The ones that stop come first: they are the ones that will send work
	// back, and the ones somebody most needs to know are standing.
	if rules > aware {
		t.Errorf("the side reads Aware before Rules:\n%s", drawn)
	}

	stops, warns := strings.Index(drawn, "no UPDATE in ledger"), strings.Index(drawn, "something to keep in mind")
	if stops < rules || stops > aware || warns < aware {
		t.Errorf("the facts are not under their own headings:\n%s", drawn)
	}
}

// TestTheSideSaysWhereEachFactApplies. Generals and the repository's own are
// side by side, so a line that did not say which is which would read as a
// rule about everything.
func TestTheSideSaysWhereEachFactApplies(t *testing.T) {
	drawn := sideOf(t, knowing(t, 180,
		known("of everything", knowledge.Scope{Kind: knowledge.General}),
		known("of Go", knowledge.Scope{Kind: knowledge.Language, Lang: "go"}),
	), 180)

	if !strings.Contains(drawn, "go") {
		t.Errorf("the side does not say a Go fact is about Go:\n%s", drawn)
	}
}

// TestTheSideAppearsOnAnOrdinaryTerminal.
//
// It waited for width going spare, and there is none: the content is capped
// at 110 columns and a terminal narrower than that has nothing left over, so
// the panel needed 150 columns and never appeared on the screen it was built
// for. It takes its columns from the thread now.
func TestTheSideAppearsOnAnOrdinaryTerminal(t *testing.T) {
	m := knowing(t, 120, known("of everything", knowledge.Scope{Kind: knowledge.General}))

	if drawn := sideOf(t, m, 120); !strings.Contains(drawn, "of everything") {
		t.Errorf("a hundred and twenty column window draws no side:\n%s", drawn)
	}
}

// TestATerminalTooNarrowKeepsItsConversation. Below the point where the
// thread stops being readable the panel is a bad trade, however useful it is.
func TestATerminalTooNarrowKeepsItsConversation(t *testing.T) {
	m := knowing(t, 90, known("of everything", knowledge.Scope{Kind: knowledge.General}))

	if drawn := sideOf(t, m, 90); strings.Contains(drawn, "of everything") {
		t.Errorf("a ninety column window gave its width to the side:\n%s", drawn)
	}
}

// TestNothingKnownIsNoSide, so a fresh install is the conversation and
// nothing else.
func TestNothingKnownIsNoSide(t *testing.T) {
	m := knowing(t, 180)

	if drawn := sideOf(t, m, 180); strings.Contains(drawn, "What Orbit knows") {
		t.Errorf("a heading with nothing under it was drawn:\n%s", drawn)
	}
}

// TestARuleWaitingForACheckIsStillUnderRules.
//
// Written with /rule, it arrives under Aware if the side groups by what a
// fact does — because with no check it can only warn. From the operator's
// side that reads as the gesture having been ignored. It goes where they put
// it, marked as not yet able to fire: what was asked for decides the heading,
// and what it can do decides the mark.
func TestARuleWaitingForACheckIsStillUnderRules(t *testing.T) {
	asked := known("coverage stays above 90%", knowledge.Scope{Kind: knowledge.General})
	asked.Stops = true // and no check

	drawn := sideOf(t, knowing(t, 140, asked), 140)

	rules, aware := strings.Index(drawn, "Rules"), strings.Index(drawn, "Aware")
	at := strings.Index(drawn, "coverage stays above")

	if at < 0 || rules < 0 || at < rules || (aware >= 0 && at > aware) {
		t.Errorf("a rule with no check is not under Rules:\n%s", drawn)
	}

	line := ""

	for _, l := range strings.Split(drawn, "\n") {
		if strings.Contains(l, "coverage stays above") || strings.Contains(l, "no check") {
			line += l
		}
	}

	if !strings.Contains(strings.ToLower(line), "no check") {
		t.Errorf("nothing says the rule cannot fire yet: %q", line)
	}
}

// TestTheThreadIsNotCutByTheSide. The scroll rail sits one column past the
// text, so a row is a column wider than the text is: cutting at the text's
// width takes the rail off and leaves an ellipsis down the seam.
func TestTheThreadIsNotCutByTheSide(t *testing.T) {
	m := knowing(t, 140, known("of everything", knowledge.Scope{Kind: knowledge.General}))
	m.supervisor.lines = longThread(60)
	m.thread.invalidate()

	for _, row := range m.supervisorRows(30, 140) {
		if strings.Contains(ansi.Strip(row), "…") {
			t.Errorf("the side cut the thread short: %q", ansi.Strip(row))
			break
		}
	}
}

// TestTheSideSaysWhatItCouldNotFit.
//
// Twenty facts do not fit down the side of a thirty row terminal, and the
// rows past the bottom were dropped where nobody could see them go. A column
// that quietly stops listing is worse than a short one: somebody reading it
// believes they have seen what Orbit knows.
func TestTheSideSaysWhatItCouldNotFit(t *testing.T) {
	many := make([]knowledge.Fact, 0, 30)
	for i := range 30 {
		many = append(many, known(fmt.Sprintf("fact number %02d", i), knowledge.Scope{Kind: knowledge.General}))
	}

	m := knowing(t, 140, many...)

	rows := m.knownSide(24, 140)
	if len(rows) > 24 {
		t.Errorf("the side drew %d rows into 24", len(rows))
	}

	drawn := strings.Join(rows, "\n")
	if !strings.Contains(ansi.Strip(drawn), "more") {
		t.Errorf("the side dropped what did not fit without saying so:\n%s", ansi.Strip(drawn))
	}
}

// TestTheRulesSurviveTheCut. What stops the work is what somebody most needs
// to know is standing, so it is the last thing given up for room.
func TestTheRulesSurviveTheCut(t *testing.T) {
	facts := []knowledge.Fact{stopper("the one rule", knowledge.Scope{Kind: knowledge.General})}
	for i := range 30 {
		facts = append(facts, known(fmt.Sprintf("aware %02d", i), knowledge.Scope{Kind: knowledge.General}))
	}

	drawn := ansi.Strip(strings.Join(knowing(t, 140, facts...).knownSide(12, 140), "\n"))
	if !strings.Contains(drawn, "the one rule") {
		t.Errorf("the rule was cut before the aware ones:\n%s", drawn)
	}
}
