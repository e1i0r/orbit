package supervisor

// What Orbit knows, down the side of the supervisor screen.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/e1i0r/orbit/internal/knowledge"
)

// knowing is the screen open on a store that holds these facts.
func knowing(t *testing.T, facts ...knowledge.Fact) (State, Env) {
	t.Helper()

	e := world(t, &held{})
	e.Knows = func() []knowledge.Fact { return facts }

	return Open(0, e), e
}

func known(phrase string, sc knowledge.Scope) knowledge.Fact {
	return knowledge.Fact{Scope: sc, Source: knowledge.Human, Phrase: phrase}
}

func stopper(phrase string, sc knowledge.Scope) knowledge.Fact {
	f := known(phrase, sc)
	f.Stops, f.Check = true, "false"

	return f
}

// sideOf is the whole screen as plain text, which is where the side is.
func sideOf(t *testing.T, s State, e Env, w int) string {
	t.Helper()

	return ansi.Strip(strings.Join(s.rows(30, w, e), "\n"))
}

// TestTheSideSaysWhatOrbitKnowsHere.
//
// The same facts a phase started in this repository would be told, in the
// place the operator is already looking when they write another one. A rule
// nobody can see is a rule nobody trusts.
func TestTheSideSaysWhatOrbitKnowsHere(t *testing.T) {
	s, e := knowing(t,
		known("the PRs are written in English", knowledge.Scope{Kind: knowledge.General}),
		known("never discard an error with _", knowledge.Scope{Kind: knowledge.Language, Lang: "go"}),
		known("the fuzz tests hang sometimes", knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}),
	)
	drawn := sideOf(t, s, e, 180)

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
	s, e := knowing(t,
		known("something to keep in mind", knowledge.Scope{Kind: knowledge.General}),
		stopper("no UPDATE in ledger", knowledge.Scope{Kind: knowledge.General}),
	)

	drawn := sideOf(t, s, e, 180)

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
	s, e := knowing(t,
		known("of everything", knowledge.Scope{Kind: knowledge.General}),
		known("of Go", knowledge.Scope{Kind: knowledge.Language, Lang: "go"}),
	)
	drawn := sideOf(t, s, e, 180)

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
	s, e := knowing(t, known("of everything", knowledge.Scope{Kind: knowledge.General}))

	if drawn := sideOf(t, s, e, 120); !strings.Contains(drawn, "of everything") {
		t.Errorf("a hundred and twenty column window draws no side:\n%s", drawn)
	}
}

// TestATerminalTooNarrowKeepsItsConversation. Below the point where the
// thread stops being readable the panel is a bad trade, however useful it is.
func TestATerminalTooNarrowKeepsItsConversation(t *testing.T) {
	s, e := knowing(t, known("of everything", knowledge.Scope{Kind: knowledge.General}))

	if drawn := sideOf(t, s, e, 90); strings.Contains(drawn, "of everything") {
		t.Errorf("a ninety column window gave its width to the side:\n%s", drawn)
	}
}

// TestNothingKnownIsNoSide, so a fresh install is the conversation and
// nothing else.
func TestNothingKnownIsNoSide(t *testing.T) {
	s, e := knowing(t)

	if drawn := sideOf(t, s, e, 180); strings.Contains(drawn, "What Orbit knows") {
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

	s, e := knowing(t, asked)
	drawn := sideOf(t, s, e, 140)

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
	s, e := knowing(t, known("of everything", knowledge.Scope{Kind: knowledge.General}))
	s.lines = longThread(60)
	s.thread.invalidate()

	for _, row := range s.rows(30, 140, e) {
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

	s, e := knowing(t, many...)

	rows := s.knownSide(24, 140, e)
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

	s, e := knowing(t, facts...)

	drawn := ansi.Strip(strings.Join(s.knownSide(12, 140, e), "\n"))
	if !strings.Contains(drawn, "the one rule") {
		t.Errorf("the rule was cut before the aware ones:\n%s", drawn)
	}
}

// TestAFactARunWroteWhileYouWereAwayIsMarked. The question somebody comes
// back with is what happened, and a rule that appeared out of a run is part
// of the answer — the mark says nobody typed it.
func TestAFactARunWroteWhileYouWereAwayIsMarked(t *testing.T) {
	s, e := knowing(t)
	e.Now = time.Now()

	learned := knowledge.Fact{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.FromRecord,
		Phrase: "the api refuses a body over 1MB",
		At:     e.Now.Add(-2 * time.Hour),
	}

	typed := knowledge.Fact{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human,
		Phrase: "never force-push",
		At:     e.Now.Add(-time.Hour),
	}

	old := learned
	old.At = e.Now.Add(-72 * time.Hour)

	if !s.learnedRecently(learned, e) {
		t.Error("a fact a run wrote two hours ago is not marked")
	}

	if s.learnedRecently(typed, e) {
		t.Error("a fact the reader typed is marked as learned")
	}

	if s.learnedRecently(old, e) {
		t.Error("a fact from three days ago is part of what they missed")
	}

	rows := strings.Join(s.sideFact(learned, e), "\n")
	if !strings.Contains(rows, "learned") {
		t.Errorf("the side does not mark it:\n%s", rows)
	}
}
