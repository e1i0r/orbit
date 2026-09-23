package repos

// The list is longer than the screen, and this is the suite that says so.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
)

// many is a board with n repositories on it, each with a task filed under
// it, which is more than a short terminal has room for.
func many(t *testing.T, n int) Env {
	t.Helper()

	e := world(t)

	var b board.Board

	for i := range n {
		name := string(rune('a'+i%26)) + strings.Repeat("o", 1+i/26) + "-repo"
		b.RepoList = append(b.RepoList, board.RepoInfo{Name: name, Path: "/r/" + name})
		b.Tasks = append(b.Tasks, view.Task{Repo: name, ID: "T-" + name, Band: view.Running})
	}

	e.Board = b
	e.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: 15, W: 100}}

	return e
}

// TestTheLastRepositoryCanBeSeen. The list outgrew the screen the moment
// somebody worked in a handful of repositories, and the rows past the
// bottom were reachable by a cursor that left no mark of having gone there:
// ⏎ filtered the board to a repository the reader could not see.
func TestTheLastRepositoryCanBeSeen(t *testing.T) {
	e := many(t, 12)

	s := Open()
	list := List(e)

	if on := drawnList(s, e); strings.Contains(on, list[len(list)-1].Name) {
		t.Fatal("the last repository is on a screen this test needs it to be off")
	}

	for range len(list) - 1 {
		s, _ = s.Key(press("down"), e)
	}

	if s.sel != len(list)-1 {
		t.Fatalf("the cursor is on row %d, want the last of %d", s.sel, len(list))
	}

	on := drawnList(s, e)
	if !strings.Contains(on, list[len(list)-1].Name) {
		t.Errorf("the cursor is on %q and the screen does not draw it:\n%s",
			list[len(list)-1].Name, on)
	}

	if !strings.Contains(on, cells.Mark) {
		t.Errorf("no row on the screen carries the cursor:\n%s", on)
	}
}

// TestTheTitleAndTheWayOutStayWhileTheListScrolls. Both are wanted most by
// the reader who has scrolled furthest, which is exactly the reader who
// would have lost them.
func TestTheTitleAndTheWayOutStayWhileTheListScrolls(t *testing.T) {
	e := many(t, 20)

	s := Open()
	for range 15 {
		s, _ = s.Key(press("down"), e)
	}

	on := drawnList(s, e)
	for _, want := range []string{"Repositories", "back"} {
		if !strings.Contains(on, want) {
			t.Errorf("a scrolled list does not say %q:\n%s", want, on)
		}
	}
}

// TestTheWheelMovesTheChoiceAndTheListFollows, which is what every other
// list in this window does under the same notch — and it stops at the ends
// rather than coming back round, because a wheel that wraps is a list the
// reader has to watch to know where they are.
func TestTheWheelMovesTheChoiceAndTheListFollows(t *testing.T) {
	e := many(t, 12)

	s := Open().Scroll(3, e)
	if s.sel != 3 {
		t.Errorf("three notches down left the cursor on row %d", s.sel)
	}

	if up := s.Scroll(-9, e); up.sel != 0 {
		t.Errorf("nine notches up from row 3 left the cursor on row %d, want the first", up.sel)
	}

	last := len(List(e)) - 1

	if down := s.Scroll(99, e); down.sel != last {
		t.Errorf("ninety-nine notches down left the cursor on row %d, want %d", down.sel, last)
	}

	// And the row it lands on is on the screen.
	end := s.Scroll(99, e)
	if !strings.Contains(drawnList(end, e), List(e)[last].Name) {
		t.Error("the wheel left the cursor on a row the screen does not draw")
	}
}

// TestAListThatFitsIsNotScrolled, and a screen with no body at all starts
// the list at the top rather than scrolling against a height the next frame
// will not be drawn at.
func TestAListThatFitsIsNotScrolled(t *testing.T) {
	e := many(t, 3)

	s := Open()
	for range 2 {
		s, _ = s.Key(press("down"), e)
	}

	if got := s.Off(e); got != 0 {
		t.Errorf("a list of three in a body of fifteen starts at row %d", got)
	}

	tall := many(t, 30)
	tall.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: 0, W: 100}}

	walked := Open()
	for range 20 {
		walked, _ = walked.Key(press("down"), tall)
	}

	if got := walked.Off(tall); got != 0 {
		t.Errorf("a window with no body starts the list at row %d", got)
	}
}

// drawnList is the screen as one string.
func drawnList(s State, e Env) string {
	return ansi.Strip(strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n"))
}
