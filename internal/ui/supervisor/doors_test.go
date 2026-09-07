package supervisor

// The doors the window reaches this screen through, and the two modes the
// keyboard has that the thread itself does not: the list of conversations,
// and the offers over a half-typed word.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/spoken"
	"github.com/e1i0r/orbit/internal/view"
)

// TestTheScreenHandsBackWhereItWasOpenedFrom, which is the window's own
// number: which screens there are is the window's business.
func TestTheScreenHandsBackWhereItWasOpenedFrom(t *testing.T) {
	const fromTheBoard = 4

	e := world(t, &held{})

	s := Open(fromTheBoard, e)
	if s.Back() != fromTheBoard {
		t.Errorf("the screen says it came from %d", s.Back())
	}

	next, out := s.Key(press("esc"), e)
	if !out.Leave || out.Back != fromTheBoard {
		t.Errorf("esc answered leave=%v back=%d", out.Leave, out.Back)
	}

	if next.input != "" || next.conversation != "" || len(next.lines) != 0 {
		t.Errorf("leaving kept %+v", next)
	}
}

// TestTheConversationIsWhereALineFromElsewhereGoes. The breaker and the
// delivery keys write into the thread the reader has open, not into one of
// their own.
func TestTheConversationIsWhereALineFromElsewhereGoes(t *testing.T) {
	kept := &held{lines: []view.SupervisorLine{
		spoke(fixtureNow, "c1", "operator", "the one that is open"),
	}}

	s, _ := opened(t, kept)
	if got := s.Conversation(); got != "c1" {
		t.Errorf("the screen has %q open", got)
	}
}

// TestTypingIntoTheLineFromOutsideIsAPaste, and it is refused while a line
// is being picked: there is nothing being typed into then, and text arriving
// would land in a field nobody can see.
func TestTypingIntoTheLineFromOutsideIsAPaste(t *testing.T) {
	s, _ := open(t)

	if got := s.Type("orbit/pull/104").input; got != "orbit/pull/104" {
		t.Errorf("the line holds %q after a paste", got)
	}

	s.picking = true
	if got := s.Type("more").input; got != "" {
		t.Errorf("a paste while a line is being picked left %q", got)
	}
}

// TestFollowPinsTheThreadToItsEnd, for an answer that has just landed.
func TestFollowPinsTheThreadToItsEnd(t *testing.T) {
	s, e := opened(t, &held{lines: longThread(40)})

	s, _ = s.Key(press("pgup"), e)
	if s.follow {
		t.Fatal("reading back through the thread left it following")
	}

	if !s.Follow().follow {
		t.Error("an answer that landed did not pin the thread to its end")
	}
}

// TestTheWholeScreenIsDrawnThroughOneDoor.
func TestTheWholeScreenIsDrawnThroughOneDoor(t *testing.T) {
	s, e := opened(t, &held{lines: longThread(6)})

	rows := s.View(30, 100, e)
	if len(rows) != 30 {
		t.Errorf("the screen filled %d rows of the 30 it was given", len(rows))
	}

	if !strings.Contains(ansi.Strip(strings.Join(rows, "\n")), "turn 00") {
		t.Error("what was said is not on the screen")
	}
}

// TestTheEngineIsShownThinkingWhileAnAnswerIsOut. A question that went out
// with nothing on screen to say so reads as a window that dropped it.
func TestTheEngineIsShownThinkingWhileAnAnswerIsOut(t *testing.T) {
	s, e := opened(t, &held{lines: longThread(3)})
	e.Busy = true

	drawn := ansi.Strip(strings.Join(s.View(30, 100, e), "\n"))
	for _, want := range []string{"zeta", "thinking"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the screen does not say %q while an answer is out:\n%s", want, drawn)
		}
	}
}

// TestTheOffersWalkUnderTheArrowsAndAreTakenWithEnter.
func TestTheOffersWalkUnderTheArrowsAndAreTakenWithEnter(t *testing.T) {
	s, e := open(t)
	s.input = "/"

	offers := s.completions(e)
	if len(offers) < 2 {
		t.Fatalf("a slash offered %d things", len(offers))
	}

	down, _ := s.Key(press("down"), e)
	if down.pick != 1 {
		t.Errorf("down on the offers left the cursor on %d", down.pick)
	}

	// It stops at the ends rather than wrapping: a list that comes back
	// round has no bottom.
	up, _ := down.Key(press("up"), e)
	if up.pick != 0 {
		t.Errorf("up from the first offer left the cursor on %d", up.pick)
	}

	if again, _ := up.Key(press("up"), e); again.pick != 0 {
		t.Errorf("up past the first offer left the cursor on %d", again.pick)
	}

	took, _ := down.Key(press("enter"), e)
	if got := took.input; got != offers[1].Text+" " {
		t.Errorf("⏎ on the second offer left %q on the line", got)
	}
}

// TestTheOffersAreDrawnWhereTheReaderIsLooking, with what each one does
// beside it: the descriptions are the reason the list exists.
func TestTheOffersAreDrawnWhereTheReaderIsLooking(t *testing.T) {
	s, e := open(t)
	s.input = "/"

	drawn := ansi.Strip(strings.Join(s.drawCompletions(90, e), "\n"))
	if !strings.Contains(drawn, spoken.RuleWord) {
		t.Errorf("the offers do not carry the gestures:\n%s", drawn)
	}

	if !strings.Contains(drawn, "a rule with a check") {
		t.Errorf("the offers do not say what each one does:\n%s", drawn)
	}

	// More than fit says how many are left rather than dropping them
	// silently.
	if !strings.Contains(drawn, "more") {
		t.Errorf("a list longer than the screen does not say so:\n%s", drawn)
	}

	// And an ordinary sentence offers nothing at all.
	s.input = "what happened"
	if got := s.drawCompletions(90, e); got != nil {
		t.Errorf("an ordinary sentence drew %q", got)
	}
}

// TestANoteGoesOnTheTaskItNames.
func TestANoteGoesOnTheTaskItNames(t *testing.T) {
	kept := &held{}
	s, e := opened(t, kept)

	var onTask, said string

	e.Note = func(id, text string) error {
		onTask, said = id, text

		return nil
	}

	_, out := s.Say(spoken.AtWord+"ORB-1 the webhook retries", e)

	if onTask == "" || said == "" {
		t.Fatalf("the note went to task %q saying %q", onTask, said)
	}

	wantBand(t, out, "noted on")
}

// TestAWindowThatCannotWriteSaysSoRatherThanLosingTheLine.
func TestAWindowThatCannotWriteSaysSoRatherThanLosingTheLine(t *testing.T) {
	s, e := opened(t, &held{})
	e.Note, e.Learn = nil, nil

	if _, out := s.Say(spoken.AtWord+"ORB-1 something", e); !strings.Contains(out.Said, "cannot write notes") {
		t.Errorf("a window with no note door said %q", out.Said)
	}

	if _, out := s.Say(spoken.AwareWord+" something", e); !strings.Contains(out.Said, "cannot write down") {
		t.Errorf("a window with no store said %q", out.Said)
	}
}

// TestAGestureWithNothingAfterItSaysSo, rather than looking like a window
// that swallowed it.
func TestAGestureWithNothingAfterItSaysSo(t *testing.T) {
	s, e := opened(t, &held{})

	_, out := s.Say(spoken.RuleWord, e)
	wantBand(t, out, "nothing after it")
}

// TestTheConversationsSayHowToLeaveThem.
func TestTheConversationsSayHowToLeaveThem(t *testing.T) {
	s, e, _ := twoConversations(t)
	s = s.openConversationList()

	drawn := ansi.Strip(strings.Join(s.View(30, 100, e), "\n"))
	for _, want := range []string{"open", "back"} {
		if !strings.Contains(strings.ToLower(drawn), want) {
			t.Errorf("the list does not say how to %q:\n%s", want, drawn)
		}
	}
}

// TestPickingALineKeepsItOnScreen. Scrolling and picking are the same
// movement seen from two sides: while a line is being picked the offset is
// whatever keeps it in view, and the reader does not manage both.
func TestPickingALineKeepsItOnScreen(t *testing.T) {
	kept := &held{lines: longThread(40)}
	s, e := opened(t, kept)

	total, rows := s.threadSize(e)

	_, starts := s.threadLines(90, e)
	if len(starts) < 10 {
		t.Fatalf("the thread drew %d messages, want the fixture's forty", len(starts))
	}

	// A line near the top, with the thread scrolled to its end.
	s.picking, s.pick, s.follow = true, 0, true

	if got := s.threadOffset(total, rows, starts); got > starts[0] {
		t.Errorf("picking the first line left the window at row %d, past its row %d", got, starts[0])
	}

	// And one near the end, with the thread at its top.
	s.pick, s.follow, s.offset = len(starts)-1, false, 0

	if got := s.threadOffset(total, rows, starts); got == 0 {
		t.Error("picking the last line left the window at the top of the thread")
	}

	// The offset never runs past either end.
	s.picking, s.follow, s.offset = false, false, 9999
	if got := s.threadOffset(total, rows, starts); got > total-rows {
		t.Errorf("the window starts at row %d of %d", got, total)
	}
}

// TestTheConversationsAnswerTheirOwnKeys.
func TestTheConversationsAnswerTheirOwnKeys(t *testing.T) {
	s, e, kept := twoConversations(t)
	s = s.openConversationList()

	// The cursor stops at both ends rather than wrapping.
	if up, _ := s.conversationKey(press("up"), e); up.listSel != 0 {
		t.Errorf("up from the first row left the cursor on %d", up.listSel)
	}

	down, _ := s.conversationKey(press("down"), e)
	if down.listSel != 1 {
		t.Errorf("down left the cursor on %d", down.listSel)
	}

	if end, _ := down.conversationKey(press("down"), e); end.listSel != 1 {
		t.Errorf("down from the last row left the cursor on %d", end.listSel)
	}

	// esc puts the thread back without changing which one is open.
	back, _ := s.conversationKey(press("esc"), e)
	if back.list || back.conversation != s.conversation {
		t.Errorf("esc left list=%v on conversation %q", back.list, back.conversation)
	}

	// d removes the one under the cursor, and ^N starts one.
	if _, out := s.conversationKey(press("d"), e); out.Said == "" {
		t.Error("removing a conversation said nothing about it")
	}

	if len(kept.gone) != 1 {
		t.Errorf("d removed %v", kept.gone)
	}

	fresh, out := s.conversationKey(ctrl('n'), e)
	if fresh.conversation == s.conversation || out.Said == "" {
		t.Errorf("^N left the screen on %q saying %q", fresh.conversation, out.Said)
	}

	// A key the list has no use for changes nothing.
	if after, out := s.conversationKey(press("z"), e); after.listSel != s.listSel || out.Said != "" {
		t.Errorf("a key the list has no use for answered %+v", out)
	}
}

// TestAScopeIsSaidInTheWordsTheOperatorUsed, so the sentence that confirms a
// rule names the same thing they typed.
func TestAScopeIsSaidInTheWordsTheOperatorUsed(t *testing.T) {
	s, e := opened(t, &held{})
	e.Repo = "/w/orbit"

	for _, c := range []struct {
		scope string
		want  string
	}{
		{"", "orbit"},
		{"general", "everything"},
		{"go", "go"},
	} {
		if got := s.whereFact(spoken.Line{Scope: c.scope}, e); got != c.want {
			t.Errorf("a fact scoped %q is said as %q, want %q", c.scope, got, c.want)
		}
	}

	// A board with no repository to name says everything instead of
	// pretending to know which one.
	e.Repo = ""
	if got := s.whereFact(spoken.Line{}, e); got != "everything" {
		t.Errorf("a fact with nowhere to belong is said as %q", got)
	}
}
