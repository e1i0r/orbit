package known

// The list moves: under the arrows, under the wheel, and under a pointer.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// many is more rules than any terminal can show at once, each with a
// sentence of its own so that a test can look for one of them.
func many(n int) []knowledge.Fact {
	out := make([]knowledge.Fact, 0, n)

	for i := range n {
		out = append(out, knowledge.Fact{
			ID:     fmt.Sprintf("rule%04d", i),
			Scope:  knowledge.Scope{Kind: knowledge.Repo, Repo: "/w/orbit"},
			Source: knowledge.Human,
			Phrase: fmt.Sprintf("the rule numbered %d", i),
		})
	}

	return out
}

// shown is the screen at the size the Env says the terminal is.
func shown(t *testing.T, s State, e Env) string {
	t.Helper()

	return ansi.Strip(strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))
}

// rowAt is the terminal row the pointer has to be on to hit one row of the
// list, and false when that row is not on screen at all.
func rowAt(s State, e Env, i int) (int, bool) {
	for y := e.Frame.Body.Y; y < e.Frame.Body.Y+e.Frame.Body.H; y++ {
		if t := s.Hit(0, y, e); t.Kind == point.KnowledgeRow && t.Pane == i {
			return y, true
		}
	}

	return 0, false
}

// TestTheListScrollsRatherThanStoppingAtTheBottomOfTheScreen. Orbit learns
// four ways and keeps everything it is told, so this is the screen that
// grows on its own — and a list with a ceiling nobody declared is a list
// where the rules past the bottom are the ones nobody can find.
func TestTheListScrollsRatherThanStoppingAtTheBottomOfTheScreen(t *testing.T) {
	s, e := onScreen(t, many(40)...)

	if got := shown(t, s, e); !strings.Contains(got, "the rule numbered 0") {
		t.Fatalf("the first rule is not on the screen it opens at:\n%s", got)
	}

	for range 39 {
		s = s.Move(1, e)
	}

	got := shown(t, s, e)
	if !strings.Contains(got, "the rule numbered 39") {
		t.Errorf("the cursor walked to the last rule and it is not on screen:\n%s", got)
	}

	if strings.Contains(got, "the rule numbered 0") {
		t.Errorf("the list did not move: the first rule is still on screen:\n%s", got)
	}
}

// TestWhatTheKeysDoStaysOnScreenWhileTheListMoves. A reader going down forty
// rules is exactly the reader who needs to be told what [r] does, and a hint
// that scrolled off the bottom is a hint nobody has.
func TestWhatTheKeysDoStaysOnScreenWhileTheListMoves(t *testing.T) {
	s, e := onScreen(t, many(40)...)

	for range 39 {
		s = s.Move(1, e)
	}

	got := shown(t, s, e)
	for _, want := range []string{"What Orbit knows", "decide about it"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q left the screen when the list did:\n%s", want, got)
		}
	}
}

// TestTheWheelMovesOneRuleANotch. A rule is its sentence wrapped, so counted
// in lines the wheel crawled through a single row.
func TestTheWheelMovesOneRuleANotch(t *testing.T) {
	s, e := onScreen(t, many(40)...)

	if s = s.Scroll(1, e); s.sel != 1 {
		t.Errorf("a notch down left the cursor on row %d, want the second rule", s.sel)
	}

	if s = s.Scroll(-1, e); s.sel != 0 {
		t.Errorf("a notch back up left the cursor on row %d, want the first rule", s.sel)
	}

	// The wheel does not run off either end of the list.
	if s = s.Scroll(-1, e); s.sel != 0 {
		t.Errorf("the wheel pushed the cursor past the top of the list, to %d", s.sel)
	}
}

// TestAClickPutsTheCursorOnTheRowAndASecondOpensIt. The two-step every list
// in the window has: it is not a double-click, because a timer would make
// the same two clicks do different things depending on how fast somebody is.
func TestAClickPutsTheCursorOnTheRowAndASecondOpensIt(t *testing.T) {
	s, e := onScreen(t, many(6)...)

	y, on := rowAt(s, e, 3)
	if !on {
		t.Fatalf("the fourth rule is not on a screen with six on it")
	}

	s, already := s.PointAt(s.Hit(0, y, e).Pane, e)
	if already {
		t.Fatal("the first click on a row that was not the cursor's said it already was")
	}

	if s.sel != 3 {
		t.Fatalf("a click on the fourth rule put the cursor on row %d", s.sel)
	}

	if _, already = s.PointAt(3, e); !already {
		t.Fatal("a second click on the same row did not open it")
	}

	s, _ = s.Chosen(e)
	if !s.reviewing {
		t.Error("opening a rule did not open its review")
	}
}

// TestAClickOnFurnitureDoesNothing. A click that landed on a heading and
// moved the cursor to whatever was nearest is the gesture a reader learns
// not to trust.
func TestAClickOnFurnitureDoesNothing(t *testing.T) {
	s, e := onScreen(t, many(6)...)

	y, on := rowAt(s, e, 0)
	if !on {
		t.Fatal("the first rule is not on screen")
	}

	// The row above the first rule is its group's heading.
	if got := s.Hit(0, y-1, e); got.Kind != point.None {
		t.Errorf("the heading above the list answered %+v, want nothing", got)
	}
}

// TestAListThatGotShorterDoesNotLeaveTheViewPastItsEnd. A sentence kept
// leaves the tray and a repository emptied loses a heading, and a view
// parked past the end of a list that shrank is a screen of blank rows with
// no way to tell why.
func TestAListThatGotShorterDoesNotLeaveTheViewPastItsEnd(t *testing.T) {
	facts := many(40)

	e := world(t, facts...)
	e.All = func() []knowledge.Fact { return facts }

	s := Open(e)
	for range 39 {
		s = s.Move(1, e)
	}

	facts = facts[:3]

	if got := shown(t, s.Sync(e), e); !strings.Contains(got, "the rule numbered 0") {
		t.Errorf("the list shrank and the view stayed where it was:\n%s", got)
	}
}
