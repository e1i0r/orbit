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
	// The whole group, but only when the run made itself the leader of one.
	// Start does that (start.go), and a run typed into a shell by hand does
	// not — its group is the shell's job, or a script's, and signalling a
	// group Orbit did not create would take everything else in it down too.
	// Where the run leads its own group, the group is exactly the run and
	// the engine it spawned, and killing the engine is the point: it is the
	// process actually holding the work.
	target := pid
	if pgid, gerr := syscall.Getpgid(pid); gerr == nil && pgid == pid {
		target = -pid
	}
	if err := syscall.Kill(target, syscall.SIGKILL); err != nil {
		return fmt.Errorf("stop process %d holding task %s: %w", pid, t.ID, err)
	}
	return nil
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
