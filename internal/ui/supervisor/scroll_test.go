package supervisor

// The supervisor thread's scrolling, which is its own file because it fails
// on its own: an unclamped counter with 999999 for "at the bottom" is a
// number that moves while the screen does not.

import (
	"fmt"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// longThread is a conversation taller than any window in these tests.
func longThread(n int) []view.SupervisorLine {
	at := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)

	lines := make([]view.SupervisorLine, 0, n)
	for i := range n {
		lines = append(lines, view.SupervisorLine{
			At: at.Add(time.Duration(i) * time.Minute), By: "operator", Channel: "tui",
			Text: fmt.Sprintf("turn %02d", i),
		})
	}

	return lines
}

// TestSupervisorScrolling is the fix. The offset was a raw counter with a
// sentinel of 999999 for "at the bottom" and no ceiling on the way down, so
// one press of ↑ moved it to 999998 and the thread did not budge: the number
// moved and the screen did not, which is what "the scroll does not work"
// means. Every movement is clamped where it is made now.
func TestSupervisorScrolling(t *testing.T) {
	s, e := open(t)

	s.lines = longThread(40)
	if !s.follow {
		t.Fatal("the screen did not open at the newest message")
	}

	total, shown := s.threadSize(e)

	last := total - shown
	if last <= 0 {
		t.Fatalf("the fixture thread fits on screen (%d rows in %d), so nothing here is scrolling", total, shown)
	}

	// One press of up moves exactly one row off the end, immediately.
	s, _ = s.Key(press("up"), e)
	if s.offset != last-1 || s.follow {
		t.Fatalf("after one up: offset %d follow %v, want %d and not following", s.offset, s.follow, last-1)
	}

	// Down returns to the end and starts following again.
	s, _ = s.Key(press("down"), e)
	if s.offset != last || !s.follow {
		t.Fatalf("after down: offset %d follow %v, want %d and following", s.offset, s.follow, last)
	}

	// Ten more downs at the end do not run the counter past it, so the
	// next up is felt on the first press rather than the eleventh.
	for range 10 {
		s, _ = s.Key(press("down"), e)
	}

	if s.offset != last {
		t.Errorf("pressing down at the end ran the offset to %d, past %d", s.offset, last)
	}

	s, _ = s.Key(press("up"), e)
	if s.offset != last-1 {
		t.Errorf("up after ten downs at the end = %d, want %d", s.offset, last-1)
	}

	// And the top is a wall too.
	for range total + 5 {
		s, _ = s.Key(press("up"), e)
	}

	if s.offset != 0 {
		t.Errorf("up past the first row = %d, want 0", s.offset)
	}

	s, _ = s.Key(press("down"), e)
	if s.offset != 1 {
		t.Errorf("down after walking off the top = %d, want 1", s.offset)
	}
}

func TestSupervisorScrollPagesAndEnds(t *testing.T) {
	s, e := open(t)
	s.lines = longThread(40)
	total, shown := s.threadSize(e)
	last := total - shown

	s, _ = s.Key(press("pgup"), e)
	if want := last - (shown - 1); s.offset != want {
		t.Errorf("page up = %d, want %d", s.offset, want)
	}

	s, _ = s.Key(press("pgdown"), e)
	if s.offset != last || !s.follow {
		t.Errorf("page down = %d follow %v, want %d and following", s.offset, s.follow, last)
	}

	s, _ = s.Key(press("home"), e)
	if s.offset != 0 || s.follow {
		t.Errorf("home = %d follow %v, want 0 and not following", s.offset, s.follow)
	}

	s, _ = s.Key(press("end"), e)
	if !s.follow {
		t.Error("end did not go back to following the newest message")
	}
}

// TestSupervisorScrollStaysWhereYouWereReading: a reply arriving while you
// are reading back through the thread must not yank you to the bottom, and
// one arriving while you are at the bottom must not leave behind it.
func TestSupervisorScrollStaysWhereYouWereReading(t *testing.T) {
	kept := &held{lines: longThread(40)}
	s, e := opened(t, kept)

	s, _ = s.Key(press("pgup"), e)
	parked := s.offset

	kept.lines = longThread(48)

	s = s.Sync(e)
	if s.offset != parked || s.follow {
		t.Errorf("eight replies arrived and moved the reader to %d, from %d", s.offset, parked)
	}

	// At the end, the same eight replies must be scrolled to rather than
	// left below the bottom edge.
	s, _ = s.Key(press("end"), e)
	before, rows := s.threadSize(e)
	kept.lines = longThread(56)
	s = s.Sync(e)

	after, _ := s.threadSize(e)
	if after <= before {
		t.Fatalf("the thread did not grow: %d rows then %d", before, after)
	}

	if got := s.threadOffset(after, rows, nil); got != after-rows {
		t.Errorf("following, the window starts at row %d, want the end at %d", got, after-rows)
	}
}

// notch is how far one turn of the wheel moves the thread. The window
// decides it and hands it down, so a test that turns the wheel says the same
// number the window would.
const notch = 3

// TestTheWheelScrollsTheThreadAndTheChoice: the wheel does on this screen
// what the arrows do, including while a line is being picked — the hand
// should not have to know which of the two it is holding.
func TestTheWheelScrollsTheThreadAndTheChoice(t *testing.T) {
	kept := &held{}
	s, e := opened(t, kept)
	s.lines = longThread(40)
	total, shown := s.threadSize(e)
	last := total - shown

	s = s.Wheel(-notch, e)
	if want := last - notch; s.offset != want {
		t.Errorf("one notch up = %d, want %d", s.offset, want)
	}

	s = s.Wheel(notch, e)
	if s.offset != last || !s.follow {
		t.Errorf("one notch down = %d follow %v, want %d and following", s.offset, s.follow, last)
	}

	// While picking, the wheel moves the pick instead.
	s = s.startPicking(e)

	s = s.Wheel(-notch, e)
	if want := len(s.lines) - 1 - notch; s.pick != want {
		t.Errorf("wheeling while picking chose %d, want %d", s.pick, want)
	}
}
