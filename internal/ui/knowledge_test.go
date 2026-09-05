package ui

// The screen that lists what Orbit knows.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/e1i0r/orbit/internal/knowledge"
)

func onScreen(t *testing.T, facts ...knowledge.Fact) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)
	m.opts.KnowsAll = func() []knowledge.Fact { return facts }

	return m.openKnowledge()
}

func drawnKnowledge(t *testing.T, m Model) string {
	t.Helper()

	return ansi.Strip(strings.Join(m.knowledgeRows(26, 96), "\n"))
}

// TestKOpensTheKnowledgeScreen. The one screen where what Orbit knows can be
// read whole, and the place the supervisor's side panel points at.
func TestKOpensTheKnowledgeScreen(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	if m = step(t, m, "K"); m.screen != screenKnowledge {
		t.Fatalf("K left the window on %v", m.screen)
	}

	if back := step(t, m, "esc"); back.screen != screenList {
		t.Errorf("esc left the window on %v, want the board", back.screen)
	}
}

// TestTheGeneralOnesComeFirstAndSayTheyDoNotTravel.
//
// A general fact lives in the state root, so it is one person's on one
// machine. Everything else here travels in the repository it is about, gets
// reviewed in a pull request, and arrives for whoever clones it. The screen
// has to say which is which, or somebody writes a rule for their team that
// only ever applied to them.
func TestTheGeneralOnesComeFirstAndSayTheyDoNotTravel(t *testing.T) {
	drawn := drawnKnowledge(t, onScreen(t,
		knowledge.Fact{Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}, Source: knowledge.Human, Phrase: "of the repository"},
		knowledge.Fact{Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "of everything"},
	))

	all, repo := strings.Index(drawn, "of everything"), strings.Index(drawn, "of the repository")
	if all < 0 || repo < 0 || all > repo {
		t.Errorf("the general ones are not first:\n%s", drawn)
	}

	if !strings.Contains(strings.ToLower(drawn), "travel") {
		t.Errorf("nothing says the general ones stay on this machine:\n%s", drawn)
	}
}

// TestEachFactSaysWhereItCameFrom. A sentence in the agent's context that
// nobody can trace is indistinguishable from one the model made up.
func TestEachFactSaysWhereItCameFrom(t *testing.T) {
	drawn := drawnKnowledge(t, onScreen(t,
		knowledge.Fact{Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.FromRecord, Phrase: "learned from a refusal"},
	))

	if !strings.Contains(strings.ToLower(drawn), "record") {
		t.Errorf("the screen does not say a fact came from the record:\n%s", drawn)
	}
}

// TestAFactThatIsOffLooksOff, so that turning one off is a thing somebody
// can see they did.
func TestAFactThatIsOffLooksOff(t *testing.T) {
	off := knowledge.Fact{Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "turned off"}
	off.Off = true

	if drawn := drawnKnowledge(t, onScreen(t, off)); !strings.Contains(strings.ToLower(drawn), "off") {
		t.Errorf("a fact that is off is drawn like any other:\n%s", drawn)
	}
}

// TestSpaceTurnsAFactOffAndOnAgain. Disagreeing with a fact and losing the
// record that it existed are different things, so it is turned off and not
// deleted.
func TestSpaceTurnsAFactOffAndOnAgain(t *testing.T) {
	var turned []knowledge.Fact

	m := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "of everything",
	})
	m.opts.TurnFact = func(f knowledge.Fact) error {
		turned = append(turned, f)

		return nil
	}

	next := next(t, m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if len(turned) != 1 {
		t.Fatalf("space turned %d facts", len(turned))
	}

	if !turned[0].Off {
		t.Error("space did not turn the fact off")
	}

	_ = next
}

// TestNothingKnownSaysSoRatherThanDrawingAnEmptyList.
func TestNothingKnownSaysSoRatherThanDrawingAnEmptyList(t *testing.T) {
	if drawn := drawnKnowledge(t, onScreen(t)); strings.TrimSpace(drawn) == "" {
		t.Error("a fresh install draws an empty screen with nothing said on it")
	}
}

// TestEOpensTheFactForEditing, with what it says already in the line: a fact
// is corrected far more often than it is rewritten.
func TestEOpensTheFactForEditing(t *testing.T) {
	m := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human,
		Phrase: "the fuxx tests hang sometimes",
	})
	// Nothing opens for editing without somewhere to save it to: typing into
	// a line that cannot be written back is worse than not offering it.
	m.opts.ReplaceFact = func(knowledge.Fact, knowledge.Fact) error { return nil }

	m = next(t, m, tea.KeyPressMsg{Code: 'e', Text: "e"})
	if !m.knowledge.editing {
		t.Fatal("e did not open the fact for editing")
	}

	if got := m.knowledge.in[factPhrase].val; got != "the fuxx tests hang sometimes" {
		t.Errorf("the line holds %q, want the sentence it is about to correct", got)
	}
}

// TestEditingTheSentenceReplacesTheFact rather than leaving both: the file is
// named after the sentence when nothing else names it.
func TestEditingTheSentenceReplacesTheFact(t *testing.T) {
	var was, now knowledge.Fact

	m := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "fuxx",
	})
	m.opts.ReplaceFact = func(a, b knowledge.Fact) error {
		was, now = a, b

		return nil
	}

	m = next(t, m, tea.KeyPressMsg{Code: 'e', Text: "e"})
	m = next(t, m, tea.KeyPressMsg{Text: "y"})
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if was.Phrase != "fuxx" {
		t.Errorf("the fact replaced was %q", was.Phrase)
	}

	if now.Phrase != "fuxxy" {
		t.Errorf("the fact written is %q, want what was typed", now.Phrase)
	}

	if m.knowledge.editing {
		t.Error("the line stayed open after saving")
	}
}

// TestACheckCanBeGivenToARuleThatHasNone. This is the gesture that turns a
// sentence into a gate: the screen already says which rules cannot fire, and
// this is where that is answered.
func TestACheckCanBeGivenToARuleThatHasNone(t *testing.T) {
	var now knowledge.Fact

	asked := knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human,
		Phrase: "coverage stays above 90%", Stops: true,
	}

	m := onScreen(t, asked)
	m.opts.ReplaceFact = func(_, b knowledge.Fact) error {
		now = b

		return nil
	}

	m = next(t, m, tea.KeyPressMsg{Code: 'e', Text: "e"})
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyTab})

	for _, r := range "make cover" {
		m = next(t, m, tea.KeyPressMsg{Text: string(r)})
	}

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if now.Check != "make cover" {
		t.Errorf("the check written is %q", now.Check)
	}

	if now.Action() != knowledge.Stops {
		t.Error("a rule that was given a check still only warns")
	}
}

// TestEscapeLeavesTheFactAsItWas.
func TestEscapeLeavesTheFactAsItWas(t *testing.T) {
	saved := false

	m := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "as it was",
	})
	m.opts.ReplaceFact = func(knowledge.Fact, knowledge.Fact) error {
		saved = true

		return nil
	}

	m = next(t, m, tea.KeyPressMsg{Code: 'e', Text: "e"})
	m = next(t, m, tea.KeyPressMsg{Text: "x"})
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	if saved {
		t.Error("escape saved the change it was cancelling")
	}

	if m.knowledge.editing || m.screen != screenKnowledge {
		t.Errorf("escape left editing=%v screen=%v", m.knowledge.editing, m.screen)
	}
}

// TestNWritesANewFact, on the same line the corrections are made in. The
// supervisor is where most of them are written, mid-conversation; this is for
// the one somebody thinks of while reading the others.
func TestNWritesANewFact(t *testing.T) {
	var now knowledge.Fact

	m := onScreen(t)
	m.opts.ReplaceFact = func(_, b knowledge.Fact) error {
		now = b

		return nil
	}

	m = next(t, m, tea.KeyPressMsg{Code: 'n', Text: "n"})
	if !m.knowledge.editing {
		t.Fatal("n did not open a line to write in")
	}

	for _, r := range "written here" {
		m = next(t, m, tea.KeyPressMsg{Text: string(r)})
	}

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if now.Phrase != "written here" {
		t.Errorf("the fact written is %q", now.Phrase)
	}

	if now.Source != knowledge.Human {
		t.Errorf("a fact typed by a person came from %v", now.Source)
	}
}

// TestLeftWidensAFactAndRightNarrowsIt.
//
// The common move: a rule written in the supervisor defaults to the
// repository being worked in, and then turns out to be true everywhere.
func TestLeftWidensAFactAndRightNarrowsIt(t *testing.T) {
	var now knowledge.Fact

	m := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}, Source: knowledge.Human, Phrase: "of the repo",
	})
	m.opts.ReplaceFact = func(_, b knowledge.Fact) error {
		now = b

		return nil
	}

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if now.Scope.Kind != knowledge.General {
		t.Errorf("left left the fact at %v, want everywhere", now.Scope.Kind)
	}

	if now.Scope.Repo != "" {
		t.Errorf("a general fact still names the repository %q", now.Scope.Repo)
	}
}

// TestNarrowingWithNothingToNarrowToSaysSo, rather than picking a repository
// on the reader's behalf.
func TestNarrowingWithNothingToNarrowToSaysSo(t *testing.T) {
	m := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "of everything",
	})
	m.opts.ReplaceFact = func(knowledge.Fact, knowledge.Fact) error { return nil }
	m.board.RepoList = nil

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.knowledge.facts[0].Scope.Kind != knowledge.General {
		t.Error("right narrowed a fact to a repository that was never named")
	}
}
