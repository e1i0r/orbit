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

// lots is more rules than any terminal can show at once, each with a
// sentence of its own so that a test can look for one of them.
func lots(n int) []knowledge.Rule {
	out := make([]knowledge.Rule, 0, n)

	for i := range n {
		out = append(out, knowledge.Rule{
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
	s, e := onScreen(t, lots(40)...)

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
	s, e := onScreen(t, lots(40)...)

	for range 39 {
		s = s.Move(1, e)
	}

	got := shown(t, s, e)
	for _, want := range []string{"Brain", "open it"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q left the screen when the list did:\n%s", want, got)
		}
	}
}

// TestTheWheelMovesOneRuleANotch. A rule is its sentence wrapped, so counted
// in lines the wheel crawled through a single row.
func TestTheWheelMovesOneRuleANotch(t *testing.T) {
	s, e := onScreen(t, lots(40)...)

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
	s, e := onScreen(t, lots(6)...)

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
	if !s.reading {
		t.Error("opening a rule did not open its review")
	}
}

// TestAClickOnFurnitureDoesNothing. A click that landed on a heading and
// moved the cursor to whatever was nearest is the gesture a reader learns
// not to trust.
func TestAClickOnFurnitureDoesNothing(t *testing.T) {
	s, e := onScreen(t, lots(6)...)

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
	facts := lots(40)

	e := world(t, facts...)
	e.All = func() []knowledge.Rule { return facts }

	s := Open(e)
	for range 39 {
		s = s.Move(1, e)
	}

	facts = facts[:3]

	if got := shown(t, s.Sync(e), e); !strings.Contains(got, "the rule numbered 0") {
		t.Errorf("the list shrank and the view stayed where it was:\n%s", got)
	}
}

// TestARuleIsClickedWhereItIsDrawn, on every row of the body and at every
// scroll.
//
// The line a click landed on was counted from the head above the list and
// the distance the list had been scrolled, and against no floor: the foot
// says what the keys do and is drawn under the list, so a click on it put
// the cursor on the rule below the window — and a second click there opened
// a rule the reader had never seen.
func TestARuleIsClickedWhereItIsDrawn(t *testing.T) {
	s, e := onScreen(t, lots(40)...)

	for _, walk := range []int{0, 4, 20, 39} {
		at := s
		for range walk {
			at = at.Move(1, e)
		}

		drawn := at.View(e.Frame.Body.H, e.Frame.Body.W, e)

		for line := range e.Frame.Body.H {
			got := at.Hit(0, e.Frame.Body.Y+line, e)
			if got.Kind != point.KnowledgeRow {
				continue
			}

			_, rows := at.body(content(e.Frame.Body.W), e)
			if got.Pane < 0 || got.Pane >= len(rows) {
				t.Fatalf("walked %d, row %d answers rule %d of %d", walk, line, got.Pane, len(rows))
			}

			// The sentence of the rule that was answered has to be the
			// one drawn on the row that was clicked.
			phrase := fmt.Sprintf("the rule numbered %d", got.Pane)
			if !strings.Contains(ansi.Strip(drawn[line]), phrase) {
				t.Errorf("walked %d, row %d draws %q and a click there is rule %d",
					walk, line, strings.TrimSpace(ansi.Strip(drawn[line])), got.Pane)
			}
		}
	}
}
