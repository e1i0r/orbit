package supervisor

// Where a click lands on this screen.
//
// It answered nothing at all until now, which meant three gestures — the
// conversations, the offers over an unfinished word, and picking a line to
// take back — could only be reached from the keyboard. Every one of them is
// a list with one row chosen, and every other list in this window is clicked.

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/view"
)

// at is the terminal row of the nth line of the body, which is what the
// window hands Hit.
func at(e Env, body int) int { return e.Frame.Body.Y + body }

// twoThreads is a record with two conversations in it, which is what the
// list has two of anything to show.
func twoThreads() *held {
	return &held{lines: []view.SupervisorLine{
		spoke(fixtureNow.Add(-5*time.Minute), "c1", "operator", "use codex here"),
		spoke(fixtureNow.Add(-4*time.Minute), "c1", "zeta", "nothing is stuck"),
		spoke(fixtureNow.Add(-3*time.Minute), "c2", "operator", "what about the ledger"),
		spoke(fixtureNow.Add(-2*time.Minute), "c2", "zeta", "it is waiting on you"),
	}}
}

// TestAConversationIsOpenedByPointingAtIt. The list is two rows per
// conversation — its title, and what was last said under it — and a reader
// who pointed at either of them pointed at that conversation.
func TestAConversationIsOpenedByPointingAtIt(t *testing.T) {
	kept := twoThreads()

	s, e := opened(t, kept)

	s, _ = s.Key(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, e)
	if !s.list {
		t.Fatal("ctrl-l did not open the conversation list")
	}

	convs := conversationsOf(s.all)
	if len(convs) != 2 {
		t.Fatalf("the list holds %d conversations, want two", len(convs))
	}

	// Both rows of the second conversation name the second conversation.
	// The list starts under the heading, which is the same four rows the
	// drawing puts above it.
	for _, row := range []int{supervisorHeadRows + 2, supervisorHeadRows + 3} {
		got := s.Hit(0, at(e, row), e)
		if got.Kind != point.SupervisorConversation {
			t.Fatalf("row %d of the list is %v, want a conversation", row, got.Kind)
		}

		if got.Pane != 1 {
			t.Errorf("row %d names conversation %d, want the second", row, got.Pane)
		}
	}

	next, _ := s.Click(point.Target{Kind: point.SupervisorConversation, Pane: 1}, e)

	if next.list {
		t.Error("choosing a conversation left the list open")
	}

	if next.Conversation() != convs[1].id {
		t.Errorf("it opened %q, want the one that was pointed at", next.Conversation())
	}
}

// TestAConversationRowPastTheEndOfTheListIsNothing. A list of two in a body
// of thirty is mostly blank, and a click on the blank has to be a click on
// nothing rather than on whatever the arithmetic last computed.
func TestAConversationRowPastTheEndOfTheListIsNothing(t *testing.T) {
	s, e := opened(t, twoThreads())

	s, _ = s.Key(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, e)

	if got := s.Hit(0, at(e, 20), e); got.Kind != point.None {
		t.Errorf("a row past the last conversation is %v, want nothing", got.Kind)
	}

	// And taking one the list does not hold changes nothing.
	next, out := s.Click(point.Target{Kind: point.SupervisorConversation, Pane: 9}, e)
	if !next.list || out.Said != "" {
		t.Errorf("a conversation the list does not hold was opened: %+v", out)
	}
}

// TestAnOfferIsTakenByPointingAtIt. The list is over the word being typed,
// which is where the reader is already looking — and a list they can see and
// cannot click is the one place on this screen their hand has to go back to
// the keyboard.
func TestAnOfferIsTakenByPointingAtIt(t *testing.T) {
	s, e := open(t)

	// A slash opens the gestures, which is the list that is always there.
	s = s.Type("/")

	offers := s.completions(e)
	if len(offers) < 2 {
		t.Fatalf("typing a slash offered %d gestures, want at least two", len(offers))
	}

	cw, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

	first := supervisorHeadRows + threadH + 1

	if len(s.drawCompletions(cw, e)) == 0 {
		t.Fatal("the offers are not drawn at all")
	}

	got := s.Hit(0, at(e, first+1), e)
	if got.Kind != point.SupervisorOffer {
		t.Fatalf("the second row of the list is %v, want an offer", got.Kind)
	}

	if got.Pane != 1 {
		t.Errorf("it names offer %d, want the second", got.Pane)
	}

	next, _ := s.Click(got, e)

	if next.input == s.input {
		t.Errorf("taking an offer left the line as %q", next.input)
	}
}

// TestAnOfferPastTheEndOfTheListIsNothing, and taking one that is not there
// leaves the line alone.
func TestAnOfferPastTheEndOfTheListIsNothing(t *testing.T) {
	s, e := open(t)
	s = s.Type("/")

	cw, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

	drawn := s.drawCompletions(cw, e)
	first := supervisorHeadRows + threadH + 1

	// The blank row the list ends with is the box's own chrome.
	if got := s.Hit(0, at(e, first+len(drawn)-1), e); got.Kind == point.SupervisorOffer {
		t.Error("the blank row under the list reads as an offer")
	}

	next, _ := s.Click(point.Target{Kind: point.SupervisorOffer, Pane: 99}, e)
	if next.input != s.input {
		t.Errorf("an offer that is not there left the line as %q", next.input)
	}
}

// TestALineIsPickedByPointingAtIt. Picking one to take back is the gesture
// on this screen that cannot be undone, so it is the one where pointing at
// the wrong row costs the most — and until now the pointer could not reach
// it at all.
func TestALineIsPickedByPointingAtIt(t *testing.T) {
	kept := saidThree()

	s, e := picking(t, kept)

	cw, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

	rendered, starts := s.threadLines(cw, e)
	if len(starts) != 3 {
		t.Fatalf("the thread rendered %d messages, want three", len(starts))
	}

	// The thread is shorter than the body, so it sits against the floor:
	// the first row of the first message is that far down.
	blank := threadH - len(rendered)
	if blank < 0 {
		t.Fatalf("the thread is %d rows in a body of %d; this test wants it to fit", len(rendered), threadH)
	}

	got := s.Hit(0, at(e, supervisorHeadRows+blank+starts[0]), e)
	if got.Kind != point.SupervisorLine {
		t.Fatalf("the first line of the thread is %v, want a line", got.Kind)
	}

	if got.Pane != 0 {
		t.Errorf("it names line %d, want the first", got.Pane)
	}

	next, out := s.Click(got, e)

	if len(kept.back) != 1 {
		t.Fatalf("%d lines were taken back, want the one that was pointed at", len(kept.back))
	}

	if !kept.back[0].Equal(kept.lines[0].At) {
		t.Errorf("it took back the line said at %s, want the first", kept.back[0])
	}

	if next.picking {
		t.Error("taking a line back left the screen picking one")
	}

	if out.Said == "" {
		t.Error("it took a line back and said nothing")
	}
}

// TestAClickOnTheThreadWhileNothingIsBeingPickedIsNothing. The thread is
// read, and a click that quietly armed a retraction would be the one gesture
// on this screen that cannot be undone, fired by a reader scrolling.
func TestAClickOnTheThreadWhileNothingIsBeingPickedIsNothing(t *testing.T) {
	kept := saidThree()

	s, e := opened(t, kept)

	for row := range 20 {
		if got := s.Hit(0, at(e, row), e); got.Kind != point.None {
			t.Errorf("row %d of a thread nobody is picking from is %v", row, got.Kind)
		}
	}

	next, _ := s.Click(point.Target{Kind: point.SupervisorLine, Pane: 0}, e)

	if len(kept.back) != 0 {
		t.Errorf("a click took a line back with nothing being picked: %v", kept.back)
	}

	if next.picking {
		t.Error("it left the screen picking a line")
	}
}

// TestACellOutsideTheBodyIsNothing. The header, the activity band and the
// key bar are the window's, and this screen must not answer for them.
func TestACellOutsideTheBodyIsNothing(t *testing.T) {
	s, e := opened(t, saidThree())

	for _, y := range []int{0, 1, e.Frame.Body.Y - 1, e.Frame.Body.Y + e.Frame.Body.H + 1} {
		if got := s.Hit(0, y, e); got.Kind != point.None {
			t.Errorf("the cell at row %d is %v, and it is not in the body", y, got.Kind)
		}
	}
}

// TestAClickOnNothingChangesNothing. A target this screen does not answer —
// one the window made for another screen, or none at all — leaves it alone
// rather than doing whatever the last case happened to be.
func TestAClickOnNothingChangesNothing(t *testing.T) {
	kept := saidThree()

	s, e := opened(t, kept)

	next, out := s.Click(point.Target{Kind: point.Task, ID: "ACME-1"}, e)

	if out.Said != "" || out.Leave {
		t.Errorf("a target from another screen answered %+v", out)
	}

	if next.list || next.picking || len(kept.back) != 0 {
		t.Error("a target from another screen changed the screen")
	}
}

// TestTheColumnBesideTheThreadIsNotTheThread.
//
// What Orbit knows is drawn down the right of this screen. It is a reading
// and not a list, so it has nothing to point at — and a click on it while a
// line was being picked would have taken back a line the pointer was nowhere
// near, which is the one gesture here that cannot be undone.
func TestTheColumnBesideTheThreadIsNotTheThread(t *testing.T) {
	kept := saidThree()

	e := wideWorld(t, kept)
	e.Knows = func() []knowledge.Rule {
		return []knowledge.Rule{{
			ID: "aaaa1111", Scope: knowledge.Scope{Kind: knowledge.General},
			Source: knowledge.Human, Phrase: "never log a card number",
		}}
	}

	s := Open(0, e)

	s, _ = s.Key(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}, e)
	if !s.picking {
		t.Fatal("ctrl-r did not start picking a line")
	}

	if !s.sideFits(e.Frame.Body.W) {
		t.Fatalf("a body of %d columns does not fit the side, and this test is about it",
			e.Frame.Body.W)
	}

	cw, threadH := s.layout(e.Frame.Body.H, e.Frame.Body.W, e)

	rendered, starts := s.threadLines(cw, e)
	row := supervisorHeadRows + (threadH - len(rendered)) + starts[0]

	// The same row, one column inside the thread and one past its rail.
	if got := s.Hit(gutter, at(e, row), e); got.Kind != point.SupervisorLine {
		t.Fatalf("a cell inside the thread is %v, want a line", got.Kind)
	}

	if got := s.Hit(gutter+cw+2, at(e, row), e); got.Kind != point.None {
		t.Errorf("a cell in the column beside the thread is %v, want nothing", got.Kind)
	}
}
