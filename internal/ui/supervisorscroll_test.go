package ui

// The supervisor thread's scrolling, which is its own file because it is its
// own bug: the offset used to be an unclamped counter with 999999 for "at the
// bottom", so the number moved and the screen did not.

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

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
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()
	m.supervisor.lines = longThread(40)
	if !m.supervisor.follow {
		t.Fatal("the screen did not open at the newest message")
	}
	total, view := m.threadSize()
	last := total - view
	if last <= 0 {
		t.Fatalf("the fixture thread fits on screen (%d rows in %d), so nothing here is scrolling", total, view)
	}

	// One press of up moves exactly one row off the end, immediately.
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.supervisor.offset != last-1 || m.supervisor.follow {
		t.Fatalf("after one up: offset %d follow %v, want %d and not following", m.supervisor.offset, m.supervisor.follow, last-1)
	}

	// Down returns to the end and starts following again.
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.supervisor.offset != last || !m.supervisor.follow {
		t.Fatalf("after down: offset %d follow %v, want %d and following", m.supervisor.offset, m.supervisor.follow, last)
	}

	// Ten more downs at the end do not run the counter past it, so the
	// next up is felt on the first press rather than the eleventh.
	for range 10 {
		m = next(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if m.supervisor.offset != last {
		t.Errorf("pressing down at the end ran the offset to %d, past %d", m.supervisor.offset, last)
	}
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.supervisor.offset != last-1 {
		t.Errorf("up after ten downs at the end = %d, want %d", m.supervisor.offset, last-1)
	}

	// And the top is a wall too.
	for range total + 5 {
		m = next(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	}
	if m.supervisor.offset != 0 {
		t.Errorf("up past the first row = %d, want 0", m.supervisor.offset)
	}
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.supervisor.offset != 1 {
		t.Errorf("down after walking off the top = %d, want 1", m.supervisor.offset)
	}
}

func TestSupervisorScrollPagesAndEnds(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()
	m.supervisor.lines = longThread(40)
	total, view := m.threadSize()
	last := total - view

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	if want := last - (view - 1); m.supervisor.offset != want {
		t.Errorf("page up = %d, want %d", m.supervisor.offset, want)
	}
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	if m.supervisor.offset != last || !m.supervisor.follow {
		t.Errorf("page down = %d follow %v, want %d and following", m.supervisor.offset, m.supervisor.follow, last)
	}

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyHome})
	if m.supervisor.offset != 0 || m.supervisor.follow {
		t.Errorf("home = %d follow %v, want 0 and not following", m.supervisor.offset, m.supervisor.follow)
	}
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEnd})
	if !m.supervisor.follow {
		t.Error("end did not go back to following the newest message")
	}
}

// threadReader is a supervisor thread that grows between polls, which is the
// only way to test what an arriving reply does to where you were reading.
type threadReader struct {
	fakeReader
	lines []view.SupervisorLine
}

func (r *threadReader) SupervisorLog() ([]view.SupervisorLine, error) { return r.lines, nil }

// TestSupervisorScrollStaysWhereYouWereReading: a reply arriving while you
// are reading back through the thread must not yank you to the bottom, and
// one arriving while you are at the bottom must not leave behind it.
func TestSupervisorScrollStaysWhereYouWereReading(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	r := &threadReader{lines: longThread(40)}
	m.opts.Reader = r
	m = m.openSupervisor()

	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	parked := m.supervisor.offset

	r.lines = longThread(48)
	m = m.syncSupervisor()
	if m.supervisor.offset != parked || m.supervisor.follow {
		t.Errorf("eight replies arrived and moved the reader to %d, from %d", m.supervisor.offset, parked)
	}

	// At the end, the same eight replies must be scrolled to rather than
	// left below the bottom edge.
	m = next(t, m, tea.KeyPressMsg{Code: tea.KeyEnd})
	before, rows := m.threadSize()
	r.lines = longThread(56)
	m = m.syncSupervisor()
	after, _ := m.threadSize()
	if after <= before {
		t.Fatalf("the thread did not grow: %d rows then %d", before, after)
	}
	if got := m.threadOffset(after, rows, nil); got != after-rows {
		t.Errorf("following, the window starts at row %d, want the end at %d", got, after-rows)
	}
}

// TestSupervisorWheelScrollsTheThread: the wheel used to do nothing at all
// on this screen — wheel() returned early for anything that was not the
// board.
func TestSupervisorWheelScrollsTheThread(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openSupervisor()
	m.supervisor.lines = longThread(40)
	total, view := m.threadSize()
	last := total - view

	y := m.frame.Body.Y
	m = m.wheel(tea.Mouse{X: 10, Y: y, Button: tea.MouseWheelUp})
	if want := last - wheelRows; m.supervisor.offset != want {
		t.Errorf("one notch up = %d, want %d", m.supervisor.offset, want)
	}
	m = m.wheel(tea.Mouse{X: 10, Y: y, Button: tea.MouseWheelDown})
	if m.supervisor.offset != last || !m.supervisor.follow {
		t.Errorf("one notch down = %d follow %v, want %d and following", m.supervisor.offset, m.supervisor.follow, last)
	}

	// While picking, the wheel moves the pick instead.
	m.opts.RetractSupervisor = func(time.Time) error { return nil }
	m = m.startPicking()
	m = m.wheel(tea.Mouse{X: 10, Y: y, Button: tea.MouseWheelUp})
	if want := len(m.supervisor.lines) - 1 - wheelRows; m.supervisor.pick != want {
		t.Errorf("wheeling while picking chose %d, want %d", m.supervisor.pick, want)
	}
}
