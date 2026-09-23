package queue

import (
	"fmt"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
)

// line is a queue under test: the runs going, by task, and how many times
// the service was asked for.
type line struct {
	s        *store.Store
	tasks    []task.Task
	going    map[string]bool
	services int
}

// aQueue is a store with n tasks written down and a machine whose memory
// reads as used. Runs are not started: a started run is a task marked as
// going, and finish takes the mark off.
func aQueue(t *testing.T, n, used int) *line {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	r := repo.Repo{Name: "app", Path: t.TempDir()}
	q := &line{s: s, going: map[string]bool{}}

	for i := range n {
		tk, err := task.Create(s, r, fmt.Sprintf("ACME-%d", 90+i), "work", "")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		q.tasks = append(q.tasks, tk)
	}

	oldMem, oldAlive, oldSpawn, oldService := memoryUsed, alive, spawnRun, startService

	t.Cleanup(func() {
		memoryUsed, alive, spawnRun, startService = oldMem, oldAlive, oldSpawn, oldService
	})

	memoryUsed = func() (int, bool) { return used, true }
	alive = func(_ *store.Store, tk task.Task) (int, bool, error) {
		return 4242, q.going[tk.ID], nil
	}
	spawnRun = func(s *store.Store, tk task.Task, _ runAsk) (int, error) {
		q.going[tk.ID] = true

		// A run writes task.started as it begins, which is what takes it
		// out of the line.
		return 4242, write(s, tk, record.Event{Kind: record.TaskStarted})
	}
	startService = func(*store.Store) error { q.services++; return nil }

	return q
}

// write puts one event in a task's record, as a run or a requeue would.
func write(s *store.Store, tk task.Task, e record.Event) error {
	d, err := s.Record()
	if err != nil {
		return err
	}

	return d.Append(tk.ID, e)
}

// waitingIn is whether the record says tk is waiting.
func (q *line) waitingIn(t *testing.T, tk task.Task) bool {
	t.Helper()

	events, err := task.Events(q.s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	_, isWaiting := queuedAsk(events)

	return isWaiting
}

// TestThreeGoAtOnceAndTheRestWait. Five tasks started on a machine that
// freezes under five make checks: three run, two wait, and the service is
// started to move them along.
func TestThreeGoAtOnceAndTheRestWait(t *testing.T) {
	q := aQueue(t, 5, 10)

	for i, tk := range q.tasks {
		pid, err := Start(q.s, tk, "", 0)
		if err != nil {
			t.Fatalf("start %s: %v", tk.ID, err)
		}

		if ran := pid != 0; ran != (i < 3) {
			t.Errorf("%s started=%v, want the first three to start and the rest to wait",
				tk.ID, ran)
		}
	}

	if len(q.going) != 3 {
		t.Errorf("%d runs are going, want 3", len(q.going))
	}

	if q.services == 0 {
		t.Error("tasks were left waiting and nothing was started to move them along")
	}

	for _, tk := range q.tasks[3:] {
		if !q.waitingIn(t, tk) {
			t.Errorf("%s is not written down as waiting", tk.ID)
		}
	}
}

// TestARunThatEndsLetsTheOldestWaitingOneGo.
func TestARunThatEndsLetsTheOldestWaitingOneGo(t *testing.T) {
	q := aQueue(t, 5, 10)

	for _, tk := range q.tasks {
		if _, err := Start(q.s, tk, "", 0); err != nil {
			t.Fatalf("start %s: %v", tk.ID, err)
		}
	}

	delete(q.going, q.tasks[0].ID)

	started, left, err := Dispatch(q.s)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if _, ok := started[q.tasks[3].ID]; !ok || len(started) != 1 {
		t.Errorf("the queue started %v, want only %s, the oldest waiting", started, q.tasks[3].ID)
	}

	if left != 1 {
		t.Errorf("%d are left waiting, want 1", left)
	}
}

// TestNothingStartsAboveTheMemoryCeiling, and the task says it is waiting.
func TestNothingStartsAboveTheMemoryCeiling(t *testing.T) {
	q := aQueue(t, 1, 91)

	pid, err := Start(q.s, q.tasks[0], "", 0)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if pid != 0 || len(q.going) != 0 {
		t.Errorf("a run started with memory at 91%% and a ceiling of 85%%")
	}

	if !q.waitingIn(t, q.tasks[0]) {
		t.Error("the task that could not start is not waiting")
	}
}

// TestATaskTakenBackLeavesTheQueue, and one already waiting is not queued
// twice.
func TestATaskTakenBackLeavesTheQueue(t *testing.T) {
	q := aQueue(t, 4, 10)

	for _, tk := range q.tasks {
		if _, err := Start(q.s, tk, "", 0); err != nil {
			t.Fatalf("start %s: %v", tk.ID, err)
		}
	}

	if _, err := Start(q.s, q.tasks[3], "", 0); err == nil {
		t.Error("a task already waiting was queued a second time")
	}

	if err := write(q.s, q.tasks[3], record.Event{Kind: record.TaskRequeued}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	if q.waitingIn(t, q.tasks[3]) {
		t.Error("a task put back in To Do is still waiting in the queue")
	}
}

// TestCancellingAWaitingTaskTakesItOutOfTheQueue. There is no process to
// signal, and "not running" would leave it to start later all the same.
func TestCancellingAWaitingTaskTakesItOutOfTheQueue(t *testing.T) {
	q := aQueue(t, 4, 10)

	for _, tk := range q.tasks {
		if _, err := Start(q.s, tk, "", 0); err != nil {
			t.Fatalf("start %s: %v", tk.ID, err)
		}
	}

	if err := Cancel(q.s, q.tasks[3]); err != nil {
		t.Fatalf("cancel a waiting task: %v", err)
	}

	if q.waitingIn(t, q.tasks[3]) {
		t.Error("a cancelled task is still waiting in the queue")
	}
}
