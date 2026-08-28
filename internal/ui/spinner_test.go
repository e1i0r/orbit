package ui

// The frame clock, which is the part of the spinner that can go wrong
// without anybody noticing: a spinner that turns is obviously right, and a
// spinner that is not on screen is obviously absent, but a clock that ticks
// forever over a still board, or twice per frame, looks exactly like a
// clock that works.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// blank is a window that has never seen a board, so that what a board does
// to the frame clock is the only thing under test.
func blank() Model {
	return New(Options{Words: words.For("en"), Settings: &settings{}, Width: 100, Height: 30})
}

// after is the window a board left behind.
func after(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()

	next, cmd := m.Update(msg)

	got, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want a Model", next)
	}

	return got, cmd
}

// TestTheFrameClockRunsOnlyWhileSomethingIsMoving. The clock is not a
// standing ticker — a window watching nothing must not wake ten times a
// second to redraw a picture that has not changed — so a board arriving is
// what starts it, and a board with nothing working on it is what lets it
// stop.
func TestTheFrameClockRunsOnlyWhileSomethingIsMoving(t *testing.T) {
	for _, c := range []struct {
		name  string
		tasks int
		want  bool
	}{
		{"a live run starts it", len(fixtureTasks()), true},
		// The first three fixtures are all waiting on a person: no
		// process holds them, so there is nothing to animate.
		{"a board of stopped tasks leaves it stopped", 3, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			m, _ := after(t, blank(), boardMsg{Board: fixtureBoard(fixtureTasks()[:c.tasks], 4)})
			if m.spinning != c.want {
				t.Errorf("spinning=%v, want %v", m.spinning, c.want)
			}

			if m.moving() != c.want {
				t.Errorf("moving=%v, want %v", m.moving(), c.want)
			}
		})
	}
}

// TestAHeldRunIsNotAnimated. Live only says a process is believed to hold
// the task, and a run somebody paused still has its process. A spinner over
// it would say the opposite of what the pause said.
func TestAHeldRunIsNotAnimated(t *testing.T) {
	tasks := fixtureTasks()

	var held int

	for i := range tasks {
		if tasks[i].Live {
			tasks[i].Reason = view.Reason{Key: view.ReasonHeld, Args: []view.Arg{arg("phase", tasks[i].Phase)}}
			held++
		}
	}

	if held == 0 {
		t.Fatal("the fixtures have no live task to hold")
	}

	m, _ := after(t, blank(), boardMsg{Board: fixtureBoard(tasks, 4)})
	if m.moving() || m.spinning {
		t.Errorf("moving=%v spinning=%v, want a board of held runs to be still", m.moving(), m.spinning)
	}

	if got := m.runGlyph(m.anyWorking()); got != "⚡ " {
		t.Errorf("the RUNNING band is headed %q, want the static bolt", got)
	}
}

// TestTheFrameClockIsNeverAskedForTwice is the pile-up. The clock is a chain
// of one-shot ticks, each asking for the next, so a second asker while one
// is already in flight is a second chain that never ends — and every one of
// them makes the spinner turn faster for the rest of the session.
func TestTheFrameClockIsNeverAskedForTwice(t *testing.T) {
	m, _ := after(t, blank(), boardMsg{Board: fixtureBoard(fixtureTasks(), 4)})
	if !m.spinning {
		t.Fatal("a board with a live run did not start the frame clock")
	}

	if _, cmd := m.nextFrame(); cmd != nil {
		t.Error("a second asker got a second frame while one was already on its way")
	}

	// The tick landing is what frees the chain to be extended, and it
	// extends it itself rather than leaving the window still.
	m, cmd := after(t, m, spinnerTickMsg(fixtureNow))
	if cmd == nil || !m.spinning {
		t.Errorf("cmd=%v spinning=%v, want the tick to have asked for the frame after it", cmd != nil, m.spinning)
	}
}

// TestTheStatusHeartbeatIsAbsentRatherThanFrozen. The status line used to
// carry how many milliseconds the last board read took, painted red past
// 100ms — a number with no screen to give it context, flashing an alarm at a
// window that was working. What replaced it is one glyph, and it is drawn
// only while the clock that turns it is running: a spinner standing still
// reads as a wedged run, which is a worse lie than no spinner at all.
func TestTheStatusHeartbeatIsAbsentRatherThanFrozen(t *testing.T) {
	beat := func(m Model) bool {
		line := m.statusLine(100)
		for _, f := range spinnerFrames {
			if strings.Contains(line, f) {
				return true
			}
		}

		return false
	}

	live, _ := after(t, blank(), boardMsg{Board: fixtureBoard(fixtureTasks(), 4)})
	if !beat(live) {
		t.Error("the status line has no heartbeat while a run is working")
	}

	still, _ := after(t, blank(), boardMsg{Board: fixtureBoard(fixtureTasks()[:3], 4)})
	if beat(still) {
		t.Error("the status line is beating over a board where nothing is running")
	}

	if strings.Contains(live.statusLine(100), "ms read") {
		t.Error("the read time came back")
	}
}
