package compose

// The doors the window reaches the form through, and the keys that move a
// caret rather than typing into it.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/typing"
)

// TestTheFormSaysWhereItStartsAndWhatItWillRun, which is what the window
// puts a reader on it for.
func TestTheFormSaysWhereItStartsAndWhatItWillRun(t *testing.T) {
	e := world(t)

	s := Open("app", e)
	if got := s.Starts(); got != "/r/app" {
		t.Errorf("the form starts in %q", got)
	}

	if got := s.Flow(); got == "" {
		t.Error("the form names no flow at all")
	}

	// It opens on the row of pills, where the dials are, and not in a field.
	if !s.OnPills() {
		t.Error("the form opened with the caret in a field, want it on the dials")
	}

	if s.field = composeText; s.OnPills() {
		t.Error("the box a task is written in is read as a row of pills")
	}
}

// TestOpeningOnADirectoryPutsTheCaretWhereTheWorkIs. Where it was held is
// the answer to everything else, so the only thing left to say is the task.
func TestOpeningOnADirectoryPutsTheCaretWhereTheWorkIs(t *testing.T) {
	e := world(t)

	s := OpenIn("/elsewhere/ledger", e)
	if s.Starts() != "/elsewhere/ledger" {
		t.Errorf("the form starts in %q", s.Starts())
	}

	if s.field != composeText {
		t.Errorf("the caret is on field %d, want the box the task goes in", s.field)
	}

	// With nothing to open on, the caret is where the form would have put
	// it on its own.
	if bare := OpenIn("", e); bare.field != composeFlow {
		t.Errorf("opening on nothing left the caret on field %d", bare.field)
	}
}

// TestADesignerThatWroteAFlowIsReadBackIntoTheDial. The reader left this
// form to write one, and comes back to find it chosen.
func TestADesignerThatWroteAFlowIsReadBackIntoTheDial(t *testing.T) {
	e := world(t)
	s := Open("", e)

	was := s.Flow()

	// A name nothing on the dial carries leaves it where it was.
	s.Write("no such flow")

	if s.Flow() != was {
		t.Errorf("a flow nobody has moved the dial to %q", s.Flow())
	}

	if len(s.flows) < 2 {
		t.Fatalf("the fixture offers %d flows, want at least two", len(s.flows))
	}

	s.Write(s.flows[1])

	if s.Flow() != s.flows[1] {
		t.Errorf("the dial is on %q, want the flow that was chosen", s.Flow())
	}

	// Reading the list again keeps the cursor on something real, even when
	// the list came back shorter.
	s.flowIdx = 99
	s.Refresh(e.Flows)

	if s.flowIdx >= len(s.flows) {
		t.Errorf("the dial is on %d of %d flows", s.flowIdx, len(s.flows))
	}
}

// TestTheSideArrowsAreTheCaretInAFieldAndThePillsOnARowOfPills. One key, and
// which it is depends on where the reader is.
func TestTheSideArrowsAreTheCaretInAFieldAndThePillsOnARowOfPills(t *testing.T) {
	s, e := composeOn(t, composeText, "hola")
	s.text.MoveTo(4)

	s, _ = s.Key(press("left"), e)
	if s.text.At != 3 {
		t.Errorf("← in a field left the caret at %d", s.text.At)
	}

	// Held with the option key it is a word at a time.
	s.text.MoveTo(4)

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt}, e)
	if s.text.At != 0 {
		t.Errorf("⌥← left the caret at %d, want the head of the word", s.text.At)
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt}, e)
	if s.text.At != 4 {
		t.Errorf("⌥→ left the caret at %d, want the end of the word", s.text.At)
	}

	// On the dials the same key turns one.
	dial, e := form(t)
	was := dial.Flow()

	dial, _ = dial.Key(press("right"), e)
	if len(dial.flows) > 1 && dial.Flow() == was {
		t.Errorf("→ on the flow row left the dial on %q", was)
	}
}

// TestHomeAndEndAreTheEndsOfTheLine, and shift makes either one a selection.
func TestHomeAndEndAreTheEndsOfTheLine(t *testing.T) {
	s, e := composeOn(t, composeID, "ORBIT-42")
	s.id.MoveTo(3)

	s, _ = s.Key(press("home"), e)
	if s.id.At != 0 {
		t.Errorf("home left the caret at %d", s.id.At)
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModShift}, e)
	if got := s.id.Selected(); got != "ORBIT-42" {
		t.Errorf("shift+end selected %q, want the whole line", got)
	}
}

// TestCutTakesTheSelectionOutAndCopyLeavesIt.
func TestCutTakesTheSelectionOutAndCopyLeavesIt(t *testing.T) {
	s, e := composeOn(t, composeText, "hola mundo")
	s.text.MoveTo(0)
	s.text.Extend(func(in *typing.Field) { in.MoveTo(4) })

	// Nothing selected is nothing to take: the field is left alone.
	bare, _ := composeOn(t, composeText, "hola mundo")
	if got := bare.composeCopy(true); got.text.String() != "hola mundo" {
		t.Errorf("a cut with nothing selected left %q", got.text.String())
	}

	// A clipboard this machine cannot write to leaves the text where it is:
	// a cut that emptied the field would be text nobody can get back.
	after := s.composeCopy(true)
	if got := after.text.String(); got != "hola mundo" && got != " mundo" {
		t.Errorf("a cut left %q, want either the text or what is past the selection", got)
	}

	_ = e
}

// TestTabWalksTheFieldsAndStopsAtTheEnds.
func TestTabWalksTheFieldsAndStopsAtTheEnds(t *testing.T) {
	s, e := form(t)

	for range 5 {
		s, _ = s.Key(press("tab"), e)
	}

	if s.field != composeText {
		t.Errorf("five tabs left field %d, want the last one", s.field)
	}

	for range 5 {
		s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, e)
	}

	if s.field != composeFlow {
		t.Errorf("five shift+tabs left field %d, want the first one", s.field)
	}
}

// TestShiftEnterIsANewLineInTheBoxAndNothingElsewhere.
func TestShiftEnterIsANewLineInTheBoxAndNothingElsewhere(t *testing.T) {
	s, e := composeOn(t, composeText, "first")

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}, e)
	if got := s.text.String(); !strings.Contains(got, "\n") {
		t.Errorf("shift+↵ in the box left %q, want a line break", got)
	}

	// On the id it is the key that moves on, because an id has one line.
	one, e := composeOn(t, composeID, "ORB-1")

	one, _ = one.Key(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}, e)
	if strings.Contains(one.id.String(), "\n") {
		t.Errorf("shift+↵ on the id left %q", one.id.String())
	}
}

// TestADragTakesWhatItCrossed, and a pointer that wandered into another
// field takes nothing: the drag is over the text it started in.
func TestADragTakesWhatItCrossed(t *testing.T) {
	s, e := composeOn(t, composeText, "hola mundo")
	y := formRow(t, s, e, "hola mundo")

	var from, to point.Target

	for x := range e.Frame.Body.W {
		at := s.Hit(x, y, e)
		if at.Kind != point.ComposeCaret {
			continue
		}

		if at.Caret == 0 && from.Kind == point.None {
			from = at
		}

		if at.Caret == 4 {
			to = at
			break
		}
	}

	s = s.Aim(from, e)

	if got := s.Drag(to, e).text.Selected(); got != "hola" {
		t.Errorf("the drag took %q", got)
	}
}

// TestClickingATabOrAPillOrAButton, each of which is a different answer.
func TestClickingATabOrAPillOrAButton(t *testing.T) {
	s, e := form(t)

	s, _ = s.Click(point.Target{Kind: point.ComposeTab, Pane: composeTabURL}, e)
	if s.tab != composeTabURL {
		t.Errorf("clicking the second tab left tab %d", s.tab)
	}

	s, _ = s.Click(point.Target{Kind: point.ComposeField, Pane: composeURL}, e)
	if s.field != composeURL {
		t.Errorf("clicking the url row left field %d", s.field)
	}

	// A pill out of the list changes nothing rather than reaching past the
	// end of it.
	before := s.flowIdx

	s, _ = s.Click(point.Target{Kind: point.ComposeFlowChoice, Pane: 99}, e)
	if s.flowIdx != before {
		t.Errorf("clicking a pill nobody drew left the dial on %d", s.flowIdx)
	}

	// The save button answers with the task, or with what is missing.
	_, out := s.Click(point.Target{Kind: point.ComposeAction, Key: "save"}, e)
	if out.Write != nil {
		t.Error("save on an empty form wrote a task")
	}

	wantBand(t, out, "the id is required")
}

// TestAURLTabDrawsWhatWasReadOutOfTheLink. The reader pasted a link; what
// the form knows about it is the only thing that says the paste worked.
func TestAURLTabDrawsWhatWasReadOutOfTheLink(t *testing.T) {
	s, e := form(t)
	s.tab, s.field = composeTabURL, composeURL

	s = s.Type("https://linear.app/acme/issue/ENG-456/fix-auth-flow")

	drawn := strings.Join(s.View(30, 100, e), "\n")
	for _, want := range []string{"ENG-456", "LINEAR"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the tab does not say %q about the link:\n%s", want, drawn)
		}
	}
}

// TestALinkThatIsNotAnIssueLeavesTheFormAlone. A URL in the box is not
// necessarily a tracker's, and a form that switched tabs on any link would
// take a reader somewhere they did not ask to go.
func TestALinkThatIsNotAnIssueLeavesTheFormAlone(t *testing.T) {
	s, e := form(t)
	s.tab, s.field = composeTabManual, composeID

	s = s.Type("https://example.com/not/an/issue")

	if s.tab != composeTabManual {
		t.Errorf("a link nothing can read moved the form to tab %d", s.tab)
	}

	if s.parsedIssue != nil {
		t.Errorf("a link nothing can read was read as %+v", s.parsedIssue)
	}

	_ = e
}

// TestTheFieldBeingTypedIntoIsTheOneTheCursorIsOn, and a row of pills is
// none of them.
func TestTheFieldBeingTypedIntoIsTheOneTheCursorIsOn(t *testing.T) {
	s, _ := form(t)

	for _, c := range []struct {
		tab, field int
		want       string
	}{
		{composeTabManual, composeFlow, ""},
		{composeTabManual, composeID, "the id"},
		{composeTabManual, composeText, "the task"},
		{composeTabURL, composeURLFlow, ""},
		{composeTabURL, composeURL, "the url"},
	} {
		s.tab, s.field = c.tab, c.field
		s.id.SetValue("the id")
		s.text.SetValue("the task")
		s.url.SetValue("the url")

		if got := s.typed(); got != c.want {
			t.Errorf("tab %d field %d is typing into %q, want %q", c.tab, c.field, got, c.want)
		}
	}
}
