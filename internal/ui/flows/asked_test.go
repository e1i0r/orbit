package flows

// The handful of questions the window asks about a screen it does not own:
// is a paste going into a form, is a question out at an engine, does the
// wheel scroll a list.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestTheDesignerSaysWhichOfItsThreeThingsItIsDoing. The window routes a
// paste, a wheel and a spinner off these three answers, and a screen that
// said the wrong one would send a pasted line into a list.
func TestTheDesignerSaysWhichOfItsThreeThingsItIsDoing(t *testing.T) {
	s, e := designer(t)

	if s.Creating() || s.Previewing() || s.Saying() {
		t.Errorf("the list says it is creating=%v previewing=%v saying=%v",
			s.Creating(), s.Previewing(), s.Saying())
	}

	if !s.Listing() {
		t.Error("the list does not say it is a list")
	}

	writing := s.startCreateFlow(e)
	if !writing.Creating() || writing.Listing() {
		t.Errorf("the form says creating=%v listing=%v", writing.Creating(), writing.Listing())
	}

	looking := Preview("careful", FromCompose, e)
	if !looking.Previewing() || looking.Listing() {
		t.Errorf("the preview says previewing=%v listing=%v", looking.Previewing(), looking.Listing())
	}
}

// TestAQuestionOutAtAnEngineIsAScreenTheWindowCanDrawWithoutPayingForOne.
//
// What the window draws while it waits — the spinner, the count of seconds,
// the way to stop — is the window's own line, and there has to be a way to
// put a screen in that state without an engine answering.
func TestAQuestionOutAtAnEngineIsAScreenTheWindowCanDrawWithoutPayingForOne(t *testing.T) {
	went := time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC)

	s := Asking("zeta", went)
	if !s.Saying() {
		t.Error("a screen with a question out does not say so")
	}

	if !s.Since().Equal(went) {
		t.Errorf("the question went out at %v, want %v", s.Since(), went)
	}

	// The list is neither: nothing is out and the wheel scrolls it.
	idle, _ := designer(t)
	if idle.Saying() || !idle.Since().IsZero() {
		t.Errorf("an idle list says saying=%v since=%v", idle.Saying(), idle.Since())
	}
}

// TestTheWheelScrollsTheListAndStopsAtTheTop.
func TestTheWheelScrollsTheListAndStopsAtTheTop(t *testing.T) {
	s, _ := designer(t)

	if got := s.Scroll(notch).scroll; got != notch {
		t.Errorf("one notch down left the list at %d", got)
	}

	if got := s.Scroll(-notch).scroll; got != 0 {
		t.Errorf("one notch up from the top left the list at %d", got)
	}
}

// TestTheTabWhereAFlowIsEditedByHandIsAskedForByName. Creating one opens on
// the tab that describes it in words, which is the right first move for an
// empty form and the wrong one for a reader who came to change a field.
func TestTheTabWhereAFlowIsEditedByHandIsAskedForByName(t *testing.T) {
	s, e := designer(t)

	writing := s.startCreateFlow(e)
	if writing.tab == flowTabFields {
		t.Error("creating a flow opened on the fields, want the tab that describes it in words")
	}

	if writing.OnFields().tab != flowTabFields {
		t.Error("OnFields did not open the tab a flow is edited on")
	}
}

// TestTheDesignerSaysWhichFlowIsBeingChanged, and says nothing while a new
// one is being written: the two are different sentences in the band.
func TestTheDesignerSaysWhichFlowIsBeingChanged(t *testing.T) {
	s, e := designer(t)

	if got := s.startCreateFlow(e).Editing(); got != "" {
		t.Errorf("a flow nobody has saved yet is named %q", got)
	}
}

// TestTheWaitSaysWhichEngineWasAsked. The band counts the seconds while a
// draft is out, and a wait that cannot name what it is waiting for is a
// window that looks stuck.
func TestTheWaitSaysWhichEngineWasAsked(t *testing.T) {
	_, e := designer(t)

	if got := Asking("zeta", fixtureNow).AskingOf(e); got != "zeta" {
		t.Errorf("the wait says it is asking %q", got)
	}

	// With nothing named it falls back to the engine a run would use, which
	// is the one it asked.
	if got := Asking("", fixtureNow).AskingOf(e); got == "" {
		t.Error("a wait with no engine named says nothing at all")
	}

	// And the form drawn while it waits says so on the screen itself.
	s, e := designer(t)
	s = s.startCreateFlow(e)
	s.saying, s.sayEngine, s.sayAt = true, "zeta", fixtureNow.Add(-42*time.Second)

	drawn := ansi.Strip(strings.Join(s.View(30, 100, e), "\n"))
	if !strings.Contains(drawn, "zeta") {
		t.Errorf("the form does not say which engine is answering:\n%s", drawn)
	}
}

// TestEscapeStopsWaitingRatherThanLeavingTheForm. The answer may still land,
// and a reader who gave up on it is not a reader who left what they were
// writing.
func TestEscapeStopsWaitingRatherThanLeavingTheForm(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e)
	s.saying, s.sayAt = true, fixtureNow

	after, out := s.Key(tea.KeyPressMsg{Code: tea.KeyEscape}, e)
	if after.Saying() {
		t.Error("escape left the question out")
	}

	if out.Leave || !after.Creating() {
		t.Errorf("escape answered leave=%v and left creating=%v", out.Leave, after.Creating())
	}

	if !strings.Contains(out.Said, "dropped") {
		t.Errorf("giving up on the answer said %q", out.Said)
	}

	// And what lands afterwards is dropped rather than overwriting a form
	// the reader has gone back to editing by hand.
	written, _ := after.Took(DraftedMsg{}, e)
	if written.Saying() {
		t.Error("an answer nobody was waiting for was taken in")
	}
}
