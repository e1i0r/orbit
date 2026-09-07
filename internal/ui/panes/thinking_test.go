package panes

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// TestEachKindOfThoughtIsMarkedForWhatItIs. The pane is read down its left
// edge: what was decided, what was turned down, what is being looked into.
func TestEachKindOfThoughtIsMarkedForWhatItIs(t *testing.T) {
	for _, c := range []struct {
		said string
		want string
		role theme.Role
	}{
		{"decided to refactor", "🎯 decided to refactor", theme.OK},
		{"rejected outdated plan", "🚫 rejected outdated plan", theme.Warn},
		{"investigating the cache issue", "🔍 investigating the cache issue", theme.Live},
		{"because the disk was full", "💡 because the disk was full", theme.Accent},
		{"plain thinking text", "• plain thinking text", theme.Dim},
	} {
		got, role := formatThoughtLine(c.said)
		if got != c.want || role != c.role {
			t.Errorf("formatThoughtLine(%q) = (%q, %v), want (%q, %v)", c.said, got, role, c.want, c.role)
		}
	}
}

// TestABlockWithMoreToSayIsTheOnlyOneOfferedAnArrow. Opening a block that
// says all it has in one row would put nothing new on the screen, and an
// arrow that does nothing is an arrow a reader presses twice.
func TestABlockWithMoreToSayIsTheOnlyOneOfferedAnArrow(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "phase.thought", Phase: "plan", At: ago(time.Minute), Text: "one line and no more"},
		{
			Kind: "phase.thought", Phase: "plan", At: ago(time.Minute),
			Text: "decided to implement caching\nreject memory bloat\ninvestigate patterns",
		},
	})

	_, heads := Thinking(e)
	if len(heads) != 1 {
		t.Fatalf("the pane offers %d blocks to open, want only the one with more to say: %v", len(heads), heads)
	}

	for _, entry := range heads {
		if entry != 1 {
			t.Errorf("the block that folds was written by entry %d, want the second one", entry)
		}
	}
}

// TestAnOpenedBlockShowsEverythingItHas, and a shut one its first line.
func TestAnOpenedBlockShowsEverythingItHas(t *testing.T) {
	e := world(t, []view.Entry{{
		Kind: "phase.thought", Phase: "plan", At: ago(time.Minute),
		Text: "decided to implement caching\nreject memory bloat\ninvestigate patterns",
	}})

	shut, _ := Thinking(e)
	if strings.Contains(strings.Join(shut, "\n"), "investigate patterns") {
		t.Error("a shut block drew its last line, want only the first")
	}

	e.RowOpen = func(int) bool { return true }

	open, _ := Thinking(e)
	if !strings.Contains(strings.Join(open, "\n"), "investigate patterns") {
		t.Error("an opened block still hides its last line")
	}
}

// TestTheThoughtInFlightIsOnThePaneToo. It has not been written to the
// record yet, and a reader watching a run wants the sentence it is on now.
func TestTheThoughtInFlightIsOnThePaneToo(t *testing.T) {
	e := world(t, nil)
	e.Task.CurrentThought = "weighing the two schemas"

	got := strings.Join(first(Thinking(e)), "\n")
	if !strings.Contains(got, "weighing the two schemas") {
		t.Errorf("the pane does not show what the run is thinking now:\n%s", got)
	}
}

// TestAPaneWithNoThinkingSaysSo.
func TestAPaneWithNoThinkingSaysSo(t *testing.T) {
	got := strings.Join(first(Thinking(world(t, nil))), "\n")
	if !strings.Contains(got, "no thinking blocks") {
		t.Errorf("the empty pane drew %q, want it to say nothing was captured", got)
	}
}

// first is the lines of a pane that answers with its rows as well.
func first(lines []string, _ map[int]int) []string { return lines }
