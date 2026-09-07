package flows

// The tab that turns a sentence into a flow: the two dials on it, the
// warning it carries, and what a draft that came back does to the form.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// saying is the builder on the tab a flow is described in words on.
func saying(t *testing.T) (State, Env) {
	t.Helper()

	s, e := editing(t)
	s.tab = flowTabSay

	return s, e
}

// TestTheTabSaysWhatItIsForAndWhatItWillCost. Nothing is saved until Save
// Flow, and a draft replaces the phases that are there — the one thing this
// tab does that escape cannot undo.
func TestTheTabSaysWhatItIsForAndWhatItWillCost(t *testing.T) {
	s, e := saying(t)

	drawn := ansi.Strip(strings.Join(s.View(40, 100, e), "\n"))
	for _, want := range []string{"Say what the flow should do", "nothing is saved"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the tab does not say %q:\n%s", want, drawn)
		}
	}

	// A flow being changed, with phases already in it, is warned.
	s.isEditing = true
	s.flowName = "careful"

	if len(s.phases) == 0 {
		t.Fatal("the fixture flow has no phases to be replaced")
	}

	drawn = ansi.Strip(strings.Join(s.View(40, 100, e), "\n"))
	if !strings.Contains(drawn, "replaces") {
		t.Errorf("a flow with phases is not warned that a draft replaces them:\n%s", drawn)
	}
}

// TestTheTwoDialsWalkUnderTabAndTheArrowsTurnThem.
func TestTheTwoDialsWalkUnderTabAndTheArrowsTurnThem(t *testing.T) {
	s, e := saying(t)
	s.sayFocus = sayOnEngine

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyTab}, e)
	if s.sayFocus == sayOnEngine {
		t.Error("tab left the focus on the engine dial")
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, e)
	if s.sayFocus != sayOnEngine {
		t.Errorf("shift+tab left the focus on %d, want the engine", s.sayFocus)
	}

	// The arrows on the engine dial turn it, and say so.
	_, out := s.Key(press("right"), e)
	if out.Said == "" {
		t.Error("turning the engine dial said nothing about what would be asked")
	}
}

// TestWhatIsTypedIsTheSentenceTheEngineIsGiven, and an empty one is refused
// rather than paid for.
func TestWhatIsTypedIsTheSentenceTheEngineIsGiven(t *testing.T) {
	s, e := saying(t)
	s.sayFocus = sayThings - 1 // off the dials, in the box

	for _, r := range "plan it then test it" {
		s, _ = s.Key(press(string(r)), e)
	}

	if s.say != "plan it then test it" {
		t.Errorf("the box holds %q", s.say)
	}

	// Backspace takes a character back.
	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyBackspace}, e)
	if !strings.HasSuffix(s.say, "i") {
		t.Errorf("backspace left %q", s.say)
	}

	// And shift+↵ is a line break rather than the key that sends it.
	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}, e)
	if !strings.HasSuffix(s.say, "\n") {
		t.Errorf("shift+↵ left %q", s.say)
	}

	// Nothing typed is nothing to ask.
	empty, _ := saying(t)
	empty.sayFocus = sayThings - 1

	_, out := empty.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	if !strings.Contains(out.Said, "say what the flow should do") {
		t.Errorf("an empty box answered %q", out.Said)
	}
}

// TestNothingIsTypedWhileTheEngineIsOut. What comes back replaces the
// phases, and a sentence written in the meantime is a question nobody asked.
func TestNothingIsTypedWhileTheEngineIsOut(t *testing.T) {
	s, e := saying(t)
	s.sayFocus = sayThings - 1
	s.say = "plan it"
	s.saying = true

	after, _ := s.Key(press("x"), e)
	if after.say != "plan it" {
		t.Errorf("typing while the engine was out left %q", after.say)
	}
}

// TestADraftThatCameBackLandsInTheOtherTabsToCheck. Nothing is saved by it:
// the reader reads what arrived and presses Save Flow, or does not.
func TestADraftThatCameBackLandsInTheOtherTabsToCheck(t *testing.T) {
	s, e := saying(t)
	s.sayFocus = sayThings - 1
	s.say = "plan it then test it"

	e.Draft = func(string, string, string) (string, error) {
		return `{"name":"drafted","description":"one phase",` +
			`"phases":[{"name":"1-do","engine":"zeta","prompt":"do it"}]}`, nil
	}

	next, out := s.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	if !next.Saying() || out.Cmd == nil {
		t.Fatalf("⏎ on a written sentence answered saying=%v cmd=%v", next.Saying(), out.Cmd)
	}

	if !out.Waiting {
		t.Error("the question went out without asking the window to start its clock")
	}

	msg, ok := out.Cmd().(DraftedMsg)
	if !ok {
		t.Fatalf("the engine answered with %T, want the designer's own message", out.Cmd())
	}

	after, out := next.Took(msg, e)
	if after.Saying() {
		t.Error("the form is still waiting after the answer landed")
	}

	if out.Said == "" {
		t.Error("the answer landed and nothing was said about it")
	}
}
