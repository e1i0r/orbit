// Package queue is how many runs go at once, and which starts next.
//
// A run used to start the moment it was asked for, however many were
// already going. Five tasks started were five engines and five make checks
// on one machine, and the machine froze. Now a start takes a slot if there
// is one, and otherwise waits in the queue, written down as task.queued, to
// be started by whoever frees the slot: the queue's service, see service.go.
//
// The record is the queue. A task is waiting while the last thing written
// about starting it is task.queued; a run that begins, a requeue, a cancel
// or a delete takes it out. Nothing else holds the order, so a window, a
// command line and the service all read the same queue.
//
// It is a package of its own above internal/task: task starts a run now,
// and the queue decides when. Every start goes through here.
package queue

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/machine"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
)

// runAsk is what a run was asked to walk: the flow, the engine to walk it
// on, and the phase to begin at. Empty is the task's own, in every field.
type runAsk struct {
	flow, engine, from string
}

// waiting is a task in the queue, with what it was asked to run.
type waiting struct {
	task  task.Task
	ask   runAsk
	since time.Time
}

// state is what the queue holds right now: how many runs are going, and
// who is waiting, oldest first.
type state struct {
	running int
	waiting []waiting
}

// The things the queue asks of the machine and does to it, as variables so a
// test can say how full the machine is, which runs are going, and watch what
// would have started without starting it.
var (
	memoryUsed   = machine.MemoryUsed
	alive        = task.Alive
	spawnRun     = spawn
	startService = serviceProcess
)

// Start is task.Start through the queue: now if there is room and nobody is
// waiting ahead, and otherwise written down as waiting. It answers the pid
// of a run that started, and zero for a task left waiting.
func Start(s *store.Store, t task.Task, flowName string, unread int) (int, error) {
	return enqueue(s, t, runAsk{flow: flowName}, unread)
}

// StartWith is task.StartWith through the queue.
func StartWith(s *store.Store, t task.Task, flowName, engineName string, unread int) (int, error) {
	return enqueue(s, t, runAsk{flow: flowName, engine: engineName}, unread)
}

// Retry is task.Retry through the queue.
func Retry(s *store.Store, t task.Task, flowName, phase string, unread int) (int, error) {
	return enqueue(s, t, runAsk{flow: flowName, from: phase}, unread)
}

// Reopen applies a directive, waits for the run it stops to be gone, and
// starts the task again through the queue: task.Redirect, then Start.
func Reopen(
	ctx context.Context, s *store.Store, t task.Task, by, message, flowName string, unread int,
) (int, error) {
	if err := task.Redirect(ctx, s, t, by, message); err != nil {
		return 0, err
	}

	return Start(s, t, task.Walks(flowName, t), unread)
}

// Cancel stops a task's run, or takes it out of the queue when it is only
// waiting for one: there is no process to signal then, and "not running"
// would leave it to start later all the same.
func Cancel(s *store.Store, t task.Task) error {
	left, err := Leave(s, t)
	if err != nil || left {
		return err
	}

	return task.Cancel(s, t)
}

// Leave takes a task out of the queue if it is waiting there, and answers
// whether it was: what a command that cancels says depends on which of the
// two a cancel did.
func Leave(s *store.Store, t task.Task) (bool, error) { return leave(s, t) }

// Dispatch starts as many waiting tasks as there is room for, oldest first,
// and answers which it started, with which pid, and how many still wait:
// the last is how the service knows it has nothing left to do.
func Dispatch(s *store.Store) (map[string]int, int, error) {
	unlock, err := lockQueue(s)
	if err != nil {
		return nil, 0, err
	}

	defer unlock()

	return dispatch(s)
}

// dispatch is Dispatch with the queue's lock already held.
func dispatch(s *store.Store) (map[string]int, int, error) {
	cfg, err := s.Settings()
	if err != nil {
		return nil, 0, err
	}

	st, err := look(s)
	if err != nil {
		return nil, 0, err
	}

	started := map[string]int{}

	for _, w := range st.waiting {
		if !room(cfg, st.running) {
			break
		}

		pid, spawnErr := spawnRun(s, w.task, w.ask)
		if spawnErr != nil {
			// One that will not start is left where it is and said in
			// the log; the ones behind it are not held up by it.
			logger.Error("queue", "%s could not be started from the queue: %v", w.task.ID, spawnErr)

			continue
		}

		logger.Info("queue", "%s started from the queue as process %d", w.task.ID, pid)

		started[w.task.ID] = pid
		st.running++
	}

	return started, len(st.waiting) - len(started), nil
}

// enqueue refuses what could never start, starts the task now if nothing is
// waiting ahead of it and there is room, and otherwise writes it down as
// waiting and starts the service to move the line along.
func enqueue(s *store.Store, t task.Task, a runAsk, unread int) (int, error) {
	cfg, err := task.Refuse(s, t, unread)
	if err != nil {
		return 0, err
	}

	unlock, err := lockQueue(s)
	if err != nil {
		return 0, err
	}

	defer unlock()

	st, err := look(s)
	if err != nil {
		return 0, err
	}

	for _, w := range st.waiting {
		if w.task.ID == t.ID {
			return 0, fmt.Errorf("task %s is already waiting in the queue", t.ID)
		}
	}

	if len(st.waiting) == 0 && room(cfg, st.running) {
		return spawnRun(s, t, a)
	}

	if err := task.Enqueued(s, t, a.flow, a.engine, a.from); err != nil {
		return 0, err
	}

	started, left, err := dispatch(s)
	if err != nil {
		return 0, err
	}

	if pid, ok := started[t.ID]; ok {
		return pid, nil
	}

	logger.Info("queue", "%s waits in the queue, one of %d", t.ID, left)

	if err := startService(s); err != nil {
		logger.Error("queue", "the queue's service could not be started: %v", err)
	}

	return 0, nil
}

// leave takes a waiting task out of the queue, and answers whether it was
// waiting. Under the queue's lock, so a dispatch cannot start it in the
// moment between the look and the cancel.
func leave(s *store.Store, t task.Task) (bool, error) {
	unlock, err := lockQueue(s)
	if err != nil {
		return false, err
	}

	defer unlock()

	if _, going, aliveErr := alive(s, t); aliveErr != nil || going {
		return false, aliveErr
	}

	events, err := task.Events(s, t)
	if err != nil {
		return false, err
	}

	if _, isWaiting := queuedAsk(events); !isWaiting {
		return false, nil
	}

	logger.Info("queue", "%s taken out of the queue before it started", t.ID)

	return true, task.Dequeued(s, t)
}

// queuedAsk is the task.queued a task is waiting on, and whether it is
// waiting at all: the last word about starting it has to be the ask.
func queuedAsk(events []record.Event) (record.Event, bool) {
	var (
		ask       record.Event
		isWaiting bool
	)

	for _, e := range events {
		switch e.Kind {
		case record.TaskQueued:
			ask, isWaiting = e, true
		case record.TaskStarted, record.TaskRequeued, record.TaskCancelled, record.TaskDeleted:
			isWaiting = false
		}
	}

	return ask, isWaiting
}

// look reads the queue off the record: every task, whether its run is
// alive, and whether it is waiting.
func look(s *store.Store) (state, error) {
	d, err := s.Record()
	if err != nil {
		return state{}, err
	}

	ids, err := d.Tasks()
	if err != nil {
		return state{}, err
	}

	var st state

	for _, id := range ids {
		t := task.Task{ID: id}

		if _, going, aliveErr := alive(s, t); aliveErr == nil && going {
			st.running++

			continue
		}

		events, eventsErr := task.Events(s, t)
		if eventsErr != nil {
			return state{}, eventsErr
		}

		if ask, ok := queuedAsk(events); ok {
			t.Repo = repo.Repo{Path: ask.Data["repo"]}
			st.waiting = append(st.waiting, waiting{task: t, since: ask.At, ask: runAsk{
				flow: ask.Data["flow"], engine: ask.Data["engine"], from: ask.Data["from"],
			}})
		}
	}

	sort.SliceStable(st.waiting, func(i, j int) bool {
		return st.waiting[i].since.Before(st.waiting[j].since)
	})

	return st, nil
}

// room is whether one more run may start: a slot free, and memory under
// the ceiling. A machine whose memory cannot be read is not held back on a
// number nobody has.
func room(cfg store.Settings, running int) bool {
	if running >= cfg.Running() {
		return false
	}

	used, known := memoryUsed()

	return !known || used < cfg.MemoryBar()
}

// spawn starts the run now, through the door task keeps for it. The unread
// cap was asked when the task was queued, so it is not asked again here: a
// task that waited its turn is not refused for the reader's backlog.
func spawn(s *store.Store, t task.Task, a runAsk) (int, error) {
	if a.from != "" {
		return task.Retry(s, t, a.flow, a.from, 0)
	}

	return task.StartWith(s, t, a.flow, a.engine, 0)
}
