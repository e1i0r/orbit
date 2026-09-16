package known

// The screen that lists what Orbit knows.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/words"
)

// world is the Env: the words, the keys, and a store held in the test's own
// variables so that what a gesture wrote is read back rather than guessed at.
func world(t *testing.T, facts ...knowledge.Fact) Env {
	t.Helper()

	frame, err := layout.Fit(screenW, screenH)
	if err != nil {
		t.Fatalf("a terminal of %dx%d: %v", screenW, screenH, err)
	}

	return Env{
		Words: words.For("en"),
		Keys:  keymap.New(words.For("en")),
		Frame: frame,
		All:   func() []knowledge.Fact { return facts },
		Repos: []string{"/w/orbit"},
	}
}

// The terminal every test draws on: the width Elio works at, and a height
// short enough that a handful of rules is already more than fits.
const (
	screenW = 100
	screenH = 40
)

// known is one fact somebody typed.
func known(phrase string, sc knowledge.Scope) knowledge.Fact {
	return knowledge.Fact{Scope: sc, Source: knowledge.Human, Phrase: phrase}
}

// onScreen is the screen open on those facts.
func onScreen(t *testing.T, facts ...knowledge.Fact) (State, Env) {
	t.Helper()

	e := world(t, facts...)

	return Open(e), e
}

// drawnKnowledge is the screen as plain text.
func drawnKnowledge(t *testing.T, s State, e Env) string {
	t.Helper()

	return ansi.Strip(strings.Join(s.View(26, 96, e), "\n"))
}

// press is one keystroke as the event loop delivers it.
func press(keystroke string) tea.KeyPressMsg {
	switch keystroke {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}

	r := []rune(keystroke)[0]

	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// typed is a whole word going into the line being edited.
func typed(s State, e Env, word string) State {
	for _, r := range word {
		s, _ = s.Key(press(string(r)), e)
	}

	return s
}

// TestTheGeneralOnesComeFirstAndSayTheyDoNotTravel.
//
// A general fact lives in the state root, so it is one person's on one
// machine. Everything else here travels in the repository it is about, gets
// reviewed in a pull request, and arrives for whoever clones it. The screen
// has to say which is which, or somebody writes a rule for their team that
// only ever applied to them.
func TestTheGeneralOnesComeFirstAndSayTheyDoNotTravel(t *testing.T) {
	s, e := onScreen(t,
		knowledge.Fact{Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"}, Source: knowledge.Human, Phrase: "of the repository"},
		knowledge.Fact{Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "of everything"},
	)
	drawn := drawnKnowledge(t, s, e)

	all, repo := strings.Index(drawn, "of everything"), strings.Index(drawn, "of the repository")
	if all < 0 || repo < 0 {
		t.Errorf("one of the two rules is not on the screen:\n%s", drawn)
	}

	// How far each reaches is on the rule's own screen, in the strip: it is
	// a fact about one rule, and a column of it beside every row was a
	// column nobody read.
	for at, want := range map[int]string{0: "with the repo", 1: "this machine"} {
		opened := ansi.Strip(strings.Join(s.Move(at, e).openDetail(e).View(30, 96, e), "\n"))
		if !strings.Contains(opened, want) {
			t.Errorf("the rule at %d does not say it reaches %q:\n%s", at, want, opened)
		}
	}
}

// TestEachFactSaysWhereItCameFrom. A sentence in the agent's context that
// nobody can trace is indistinguishable from one the model made up.
func TestEachFactSaysWhereItCameFrom(t *testing.T) {
	s, e := onScreen(t,
		knowledge.Fact{Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.FromRecord, Phrase: "learned from a refusal"},
	)
	// On the rule's own screen and not beside every row: a column of
	// sources is a column nobody reads, and the question "can I trust this"
	// is asked of one rule at a time.
	drawn := ansi.Strip(strings.Join(s.openDetail(e).View(30, 96, e), "\n"))

	if !strings.Contains(strings.ToLower(drawn), "model") {
		t.Errorf("the rule does not say a model worked it out:\n%s", drawn)
	}
}

// TestAFactThatIsOffLooksOff, so that turning one off is a thing somebody
// can see they did.
func TestAFactThatIsOffLooksOff(t *testing.T) {
	off := knowledge.Fact{Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "turned off"}
	off.State = knowledge.Off

	s, e := onScreen(t, off)
	if drawn := drawnKnowledge(t, s, e); !strings.Contains(strings.ToLower(drawn), "off") {
		t.Errorf("a fact that is off is drawn like any other:\n%s", drawn)
	}
}

// TestNothingKnownSaysSoRatherThanDrawingAnEmptyList.
func TestNothingKnownSaysSoRatherThanDrawingAnEmptyList(t *testing.T) {
	s, e := onScreen(t)
	if drawn := drawnKnowledge(t, s, e); strings.TrimSpace(drawn) == "" {
		t.Error("a fresh install draws an empty screen with nothing said on it")
	}
}

// TestCorrectingOpensTheRuleWithWhatItSays, so that a correction is a
// correction and not a retyping.
func TestCorrectingOpensTheRuleWithWhatItSays(t *testing.T) {
	s, _ := correcting(t, "the fuxx tests hang sometimes")

	if got := s.in[factPhrase].Val; got != "the fuxx tests hang sometimes" {
		t.Errorf("the line holds %q, want the sentence it is about to correct", got)
	}
}

// TestEditingTheSentenceReplacesTheFact rather than leaving both: the file is
// named after the sentence when nothing else names it.
func TestEditingTheSentenceReplacesTheFact(t *testing.T) {
	var was, now knowledge.Fact

	s, e := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "fuxx",
	})
	e.Replace = func(a, b knowledge.Fact) error {
		was, now = a, b

		return nil
	}

	s, _ = s.Key(press("enter"), e)
	s, _ = s.Key(press("c"), e)
	s = typed(s, e, "y")
	s, _ = s.Key(press("enter"), e)

	if was.Phrase != "fuxx" {
		t.Errorf("the fact replaced was %q", was.Phrase)
	}

	if now.Phrase != "fuxxy" {
		t.Errorf("the fact written is %q, want what was typed", now.Phrase)
	}

	if s.editing {
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

	s, e := onScreen(t, asked)
	e.Replace = func(_, b knowledge.Fact) error {
		now = b

		return nil
	}

	s, _ = s.Key(press("enter"), e)
	s, _ = s.Key(press("c"), e)

	for s.field != rowCheck {
		s, _ = s.Key(press("tab"), e)
	}

	s = typed(s, e, "make cover")

	if _, _ = s.Key(press("enter"), e); now.Check != "make cover" {
		t.Errorf("the check written is %q", now.Check)
	}

	if now.Action() != knowledge.Stops {
		t.Error("a rule that was given a check still only warns")
	}
}

// TestEscapeLeavesTheFactAsItWas.
func TestEscapeLeavesTheFactAsItWas(t *testing.T) {
	saved := false

	s, e := onScreen(t, knowledge.Fact{
		Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "as it was",
	})
	e.Replace = func(knowledge.Fact, knowledge.Fact) error {
		saved = true

		return nil
	}

	s, _ = s.Key(press("enter"), e)
	s, _ = s.Key(press("c"), e)
	s = typed(s, e, "x")

	s, out := s.Key(press("esc"), e)

	if saved {
		t.Error("escape saved the change it was cancelling")
	}

	// Escape closes the line and not the screen: the reader is still
	// reading the list they were correcting from.
	if s.editing || out.Leave {
		t.Errorf("escape left editing=%v leave=%v", s.editing, out.Leave)
	}
}

// TestNWritesANewFact, on the same line the corrections are made in. The
// supervisor is where most of them are written, mid-conversation; this is for
// the one somebody thinks of while reading the others.
func TestNWritesANewFact(t *testing.T) {
	var now knowledge.Fact

	s, e := onScreen(t)
	e.Replace = func(_, b knowledge.Fact) error {
		now = b

		return nil
	}

	s, _ = s.Key(press("n"), e)
	if !s.editing {
		t.Fatal("n did not open a line to write in")
	}

	s = typed(s, e, "written here")
	s, _ = s.Key(press("enter"), e)

	if now.Phrase != "written here" {
		t.Errorf("the fact written is %q", now.Phrase)
	}

	if now.Source != knowledge.Human {
		t.Errorf("a fact typed by a person came from %v", now.Source)
	}
}
