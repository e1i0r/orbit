package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAWaitingTaskSaysItIsQueuedAndWhereInLine. A row that said "running"
// over a task with no process, or "to do" over one that was asked for, is
// a row the reader waits in front of for nothing.
func TestAWaitingTaskSaysItIsQueuedAndWhereInLine(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	at := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	queued := view.Reason{Key: view.ReasonQueued}
	first := view.Task{ID: "Q-1", Reason: queued, Since: at}
	second := view.Task{ID: "Q-2", Reason: queued, Since: at.Add(time.Minute)}
	m.board.Tasks = append(m.board.Tasks, first, second)

	word, _ := m.stateWord(second)
	if !strings.Contains(word, "queued") || !strings.Contains(word, "2") {
		t.Errorf("the second in line reads %q, want queued and its place, 2", word)
	}

	if word, _ := m.stateWord(first); !strings.Contains(word, "1") {
		t.Errorf("the first in line reads %q, want its place, 1", word)
	}
}

// TestAWaitingTaskCanBeCancelled: there is no process, and x still has to
// take it out of the queue.
func TestAWaitingTaskCanBeCancelled(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	queued := view.Task{ID: "Q-1", Reason: view.Reason{Key: view.ReasonQueued}}
	if a, ok := m.affordance(queued, m.keys.Cancel); !ok || !a.OK {
		t.Errorf("x is refused on a waiting task: %+v", a)
	}
}
