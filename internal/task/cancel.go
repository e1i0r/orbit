package task

// Stopping a run, the two ways there are: ask, and insist.

import (
	"fmt"
	"syscall"

	"github.com/e1i0r/orbit/internal/store"
)

// Cancel asks the process holding a task to stop, and to say so on its way
// out.
//
// SIGTERM, to the process itself and deliberately not to its group. `orbit
// run` turns SIGTERM into a cancelled context (cli/run.go), the context
// kills the engine it is waiting on, and Run writes phase.cancelled and
// task.cancelled with whatever the engine had printed — which is the whole
// point of asking rather than killing. Signalling the group would reach the
// engine first, and Orbit would see an engine that died of its own accord
// with a context that is still fine: a phase.failed, and a lie.
//
// It returns without waiting. Whether the run stopped is a question the
// record answers, and the caller is already reading it.
func Cancel(s *store.Store, t Task) error {
	pid, ok, err := Alive(s, t)
	if err != nil {
		return err
	}
	if !ok {
		return notRunning(t, pid)
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("ask process %d holding task %s to stop: %w", pid, t.ID, err)
	}
	return nil
}

// Kill stops a run and everything it started, now, and leaves nothing
// written down.
//
// SIGKILL cannot be caught, so the run gets no chance to record that it was
// killed and its log keeps whatever it had — which is precisely the hole
// Reconcile fills, and why `orbit cancel -now` tells the reader to run it.
// It is the answer for a run that has stopped listening: an engine ignoring
// SIGTERM, or a phase wedged on a network read.
func Kill(s *store.Store, t Task) error {
	pid, ok, err := Alive(s, t)
	if err != nil {
		return err
	}
	if !ok {
		return notRunning(t, pid)
	}
	// The whole group, but only when the run leads one. Start makes it the
	// leader of a group of its own (start.go), and then the group is exactly
	// the run and the engine it spawned — killing the engine is the point,
	// since it is the process actually holding the work.
	//
	// What the leader test excludes is a run that is in a group it did not
	// start: launched from a script, or as one process of a pipeline, where
	// the group also holds work that has nothing to do with this task and
	// signalling it would take that down too. It does not exclude a run
	// typed at a terminal — an interactive shell puts each job in a new
	// group led by its first process, so a hand-typed `orbit run` leads one
	// as surely as a spawned one does, and killing that group is right for
	// the same reason.
	pgid, gerr := syscall.Getpgid(pid)
	if err := syscall.Kill(killTarget(pid, pgid, gerr), syscall.SIGKILL); err != nil {
		return fmt.Errorf("stop process %d holding task %s: %w", pid, t.ID, err)
	}
	return nil
}

// killTarget says what Kill signals: the run's process group, when the run
// leads one, or the run's process alone when it does not. A negative number
// is how kill(2) is told to name a group, so this is the only place in Orbit
// where a minus sign belongs in front of a pid — and it is a function of its
// own so that boundary can be asserted without sending a signal to anything.
//
// A pgid of 1 is never negated, whatever Getpgid said. -1 is not a group: it
// is every process this user may signal, which on a developer's machine is
// their session and in a container is the container. parsePid already
// refuses a marker naming a pid below 2, so nothing should reach here with
// one; this refuses it again, because a defence written in two places
// survives one of them being edited.
func killTarget(pid, pgid int, gerr error) int {
	if gerr != nil || pgid != pid || pgid <= 1 {
		return pid
	}
	return -pid
}

// notRunning says which kind of "not running" this is, because they are
// different situations for the reader: nothing ever claimed the task, or
// something claimed it and is gone.
func notRunning(t Task, pid int) error {
	if pid > 0 {
		return fmt.Errorf("task %s is not running: process %d, which held it, is gone — run `orbit reconcile` to close its record", t.ID, pid)
	}
	return fmt.Errorf("task %s is not running", t.ID)
}
