package compose

// The form from open to written: what is typed into it, what it refuses, and
// what it answers with when it is satisfied.

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/typing"
)

// TestATaskIsTypedAndWrittenDown, which is the whole of what this form is
// for.
func TestATaskIsTypedAndWrittenDown(t *testing.T) {
	s, e := form(t)
	s.field = composeID

	s = typed(s, e, "TASK-9X")
	s, _ = s.Key(press("backspace"), e)

	if got := s.id.String(); got != "TASK-9" {
		t.Errorf("the id reads %q, want TASK-9", got)
	}

	s.field = composeText
	s = typed(s, e, "Fix payment")

	if rows := s.View(20, 100, e); len(rows) == 0 {
		t.Error("the form drew nothing")
	}

	_, out := s.Submit(false, e)
	if out.Write == nil {
		t.Fatalf("the form refused a task with an id and a sentence: %q", out.Said)
	}

	if out.Write.ID != "TASK-9" || out.Write.Text != "Fix payment" {
		t.Errorf("the task written is %+v", out.Write)
	}

	if out.Write.Start {
		t.Error("⏎ started the task, and it is the key that only writes it")
	}

	// ^R is the other button, and it is what makes "save and run" true.
	if _, out := s.Submit(true, e); out.Write == nil || !out.Write.Start {
		t.Errorf("save and run answered %+v", out.Write)
	}
}

// TestAFormWithNothingInItSaysWhatIsMissing, one thing at a time: a refusal
// that named two would leave a reader wondering which one they had.
func TestAFormWithNothingInItSaysWhatIsMissing(t *testing.T) {
	s, e := form(t)

	_, out := s.Submit(false, e)
	wantBand(t, out, "the id is required")

	s.id = typing.New("TASK-1")

	_, out = s.Submit(false, e)
	wantBand(t, out, "needs something written")
}

// TestAnIdTheStoreRefusesIsNotWritten, in the store's own words: the window
// is a keyboard in front of the commands and not a second copy of their
// rules.
func TestAnIdTheStoreRefusesIsNotWritten(t *testing.T) {
	s, e := form(t)
	s.id, s.text = typing.New("TASK-10"), typing.New("Write some tests")

	e.ValidID = func(string) error { return errors.New("id taken") }

	_, out := s.Submit(false, e)
	if out.Write != nil {
		t.Fatal("a task with an id the store refused was written down")
	}

	wantBand(t, out, "id taken")

	e.ValidID = func(string) error { return nil }

	if _, out := s.Submit(false, e); out.Write == nil {
		t.Errorf("a task the store accepted was refused: %q", out.Said)
	}
}

// TestTheTwoTabsAreReachedByTheirNumbers, which is what the labels say.
func TestTheTwoTabsAreReachedByTheirNumbers(t *testing.T) {
	s, e := form(t)
	s.field = composeFlow

	s, _ = s.Key(press("2"), e)
	if s.tab != composeTabURL {
		t.Errorf("2 left tab %d, want the URL one", s.tab)
	}

	s, _ = s.Key(press("1"), e)
	if s.tab != composeTabManual {
		t.Errorf("1 left tab %d, want the manual one", s.tab)
	}
}

// TestPlusOnTheFlowRowAsksForTheDesigner, and i asks to look at the one that
// is chosen. Neither is this form's to open.
func TestPlusOnTheFlowRowAsksForTheDesigner(t *testing.T) {
	s, e := form(t)
	s.field = composeFlow

	if _, out := s.Key(press("+"), e); out.Flow != New {
		t.Errorf("+ asked for flow %q, want one nobody has written yet", out.Flow)
	}

	_, out := s.Key(press("i"), e)
	if out.Flow == "" || out.Flow == New {
		t.Errorf("i asked for flow %q, want the one the dial is on", out.Flow)
	}
}

// TestAURLPastedAnywhereIsAnIssue. A reader who pastes a Linear link into
// the id is not asking for it to be the id; the form follows it to the tab
// that reads one.
func TestAURLPastedAnywhereIsAnIssue(t *testing.T) {
	s, _ := form(t)
	s.tab, s.field = composeTabManual, composeID

	s = s.Type("https://linear.app/acme/issue/ENG-456/fix-auth-flow")

	if s.tab != composeTabURL {
		t.Errorf("pasting a URL left tab %d, want the one that reads an issue", s.tab)
	}

	if got := s.id.String(); got != "ENG-456" {
		t.Errorf("the id reads %q, want the issue the URL names", got)
	}
}

// TestAPasteKeepsItsLines: a paste is usually the reason somebody wants more
// than one.
func TestAPasteKeepsItsLines(t *testing.T) {
	s, _ := form(t)
	s.tab, s.field = composeTabManual, composeText

	s = s.Type("Pasted task requirement line 1\nLine 2")

	if got := s.text.String(); got != "Pasted task requirement line 1\nLine 2" {
		t.Errorf("the task reads %q, want both lines of the paste", got)
	}
}

// TestTheFlowDialTurnsUnderTheArrows.
func TestTheFlowDialTurnsUnderTheArrows(t *testing.T) {
	s, e := form(t)
	s.field = composeFlow

	was := s.chosenFlow()

	s, _ = s.Key(press("right"), e)
	if len(s.flows) > 1 && s.chosenFlow() == was {
		t.Errorf("the dial stayed on %q", was)
	}
}

// TestTheArrowsWalkTheFieldsAndBack.
func TestTheArrowsWalkTheFieldsAndBack(t *testing.T) {
	s, e := form(t)

	for _, want := range []int{composeID, composeText} {
		s, _ = s.Key(press("down"), e)
		if s.field != want {
			t.Fatalf("down left field %d, want %d", s.field, want)
		}
	}

	for _, want := range []int{composeID, composeFlow} {
		s, _ = s.Key(press("up"), e)
		if s.field != want {
			t.Fatalf("up left field %d, want %d", s.field, want)
		}
	}
}

// TestTheURLTabWalksDownToTheURL. The field is drawn under the flow it will
// be run with, and the cursor walks the rows in the order they are drawn: a
// reader who opens the tab is on the URL, because pasting one is what they
// opened it for, and the flow is the row above.
func TestTheURLTabWalksDownToTheURL(t *testing.T) {
	s, e := form(t)

	s.tab = composeTabURL
	s.field = firstComposeField(composeTabURL)

	if s.field != composeURL {
		t.Fatalf("opening the URL tab left field %d, want the url", s.field)
	}

	up := keyed(s, press("up"), e)
	if up.field != composeURLFlow {
		t.Fatalf("up from the url is field %d, want the flow above it", up.field)
	}

	if down := keyed(up, press("down"), e); down.field != composeURL {
		t.Errorf("down from the flow is field %d, want the url below it", down.field)
	}
}

// TestEscapeClosesTheFormAndKeepsNothing.
func TestEscapeClosesTheFormAndKeepsNothing(t *testing.T) {
	s, e := form(t)
	s.id = typing.New("half typed")

	next, out := s.Key(tea.KeyPressMsg{Code: tea.KeyEscape}, e)
	if !out.Leave {
		t.Fatal("escape left the form open")
	}

	if next.id.String() != "" {
		t.Errorf("escape kept %q on the line", next.id.String())
	}
}
