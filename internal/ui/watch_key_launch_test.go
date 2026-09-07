package ui

// watch_more_coverage_test.go is watch.go's remaining branches, split out of
// plain_and_watch_coverage_test.go once that file reached the 300-line
// ceiling: the key that is not the way out, launch's fallthrough to a
// watched run, reopening a command already running, the body's own
// collapse-and-trim, and the cap commandWatch's buffer is kept under.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/palette"
)

func TestWatchKeyIgnoresAnythingThatIsNotTheWayOut(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.watchUp = true

	next, cmd := m.watchKey(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if cmd != nil || !asModel(t, next).watchUp {
		t.Error("watchKey on an unrelated key should leave the run's output up")
	}
}

func TestLaunchFallsThroughToRunWatchedForAnyOtherCommand(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	next, cmd := m.launch(Command{Name: "custom"}, []string{"a"})

	after := asModel(t, next)
	if cmd == nil || after.watching == nil || after.watching.name != "custom" {
		t.Error("launch on a command that is not one of the screen-opening ones should run it watched")
	}
}

func TestRunWatchedReopensTheSameCommandStillRunning(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	first, _ := m.runWatched(Command{Name: "build"}, nil)
	running := asModel(t, first)
	running.watchUp = false // as if the reader closed the output to look at the board

	next, cmd := running.runWatched(Command{Name: "build"}, nil)
	if cmd != nil {
		t.Error("reopening the same running command should not start a second one")
	}

	if !asModel(t, next).watchUp {
		t.Error("runWatched on the same command should have reopened its output")
	}
}

func TestWatchRowsCollapsesAndTrims(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	if rows := m.watchRows(0, 40); rows != nil {
		t.Errorf("watchRows(0, ...) = %v, want nil", rows)
	}

	// Finished, with more lines than the body can hold: only the tail
	// survives, and the status line says it is over.
	m.watching = nil

	lines := make([]string, 0, 20)
	for i := range 20 {
		lines = append(lines, "line")
		_ = i
	}

	m.output = strings.Join(lines, "\n")

	drawn := m.watchRows(5, 40)
	if len(drawn) != 5 {
		t.Errorf("watchRows(5, ...) drew %d lines, want 5", len(drawn))
	}

	if !strings.Contains(drawn[len(drawn)-1], "finished") {
		t.Errorf("last row is %q, want it to say the run finished", drawn[len(drawn)-1])
	}
}

func TestWriteTrimsTheBufferPastItsCap(t *testing.T) {
	w := &commandWatch{name: "big"}
	_, _ = w.Write(make([]byte, outputCap+100)) //nolint:errcheck

	text, _ := w.snapshot()
	if len(text) != outputCap {
		t.Errorf("commandWatch kept %d bytes, want it trimmed to outputCap (%d)", len(text), outputCap)
	}
}

// TestTheLineRunsWhatWasChosenAndNothingWhenNothingWas. The window's half of
// the palette: a chosen command reaches the same port the keyboard and the
// pointer do, and a line with nothing under the selection runs nothing.
func TestTheLineRunsWhatWasChosenAndNothingWhenNothingWas(t *testing.T) {
	// 1. Nothing in the filtered list: staying open says "not yet".
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = nil
	m = m.openPalette()

	next, cmd := m.paletteKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil || !asModel(t, next).palette.Up() {
		t.Error("⏎ with nothing selected should leave the line up and do nothing")
	}

	// 2. A fully typed name is split on spaces so the name itself is not
	// passed to the command as its own first argument.
	m2, _ := testModel(t, 100, 30)
	m2.opts.Commands = []Command{{Name: "custom"}}
	m2.palette = palette.OpenWith("custom")

	next2, cmd2 := m2.paletteKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	after2 := asModel(t, next2)
	if cmd2 == nil || after2.watching == nil {
		t.Fatal("⏎ on a real command answered with nothing running")
	}

	if got := after2.watching.name; got != "custom" {
		t.Errorf("the watch is on %q, want the command that was chosen", got)
	}
}
