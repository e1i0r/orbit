package task

// Taking the run marker, and letting it go.
//
// alive.go reads the marker; this writes it. They are two files because
// they are two questions — "who holds this task" is asked by a board twice
// a second, and "may this run have it" is asked once, by the run itself,
// and only one of them can refuse.

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/e1i0r/orbit/internal/store"
)

// hold claims a task for this process and hands back the one way to let go.
// The release is safe to call twice and says nothing when there was never a
// marker, because unwinding is not the moment to start reporting problems.
//
// A task something else is already running is refused. Two runs of one task
// share a worktree, a branch and a log: their phases interleave in the
// record, and the first of them to finish takes the other's marker off on
// its way out, leaving a live run that nothing claims and nobody can
// cancel. The marker is the only thing that can say a task is taken,
// so this is the only place it can be caught.
//
// It is not a lock and does not pretend to be. Two processes arriving here
// in the same instant both see no marker and both go on. What it stops is
// the case that actually happens: a second run started seconds or minutes
// after the first, by a window, a tool call, or a person.
//
// A marker that will not parse is refused too, by way of Alive's error. That
// is the stance readMarker already takes and the reason it gives holds here
// more strongly, not less: a claim that cannot be read is a claim that
// cannot be ruled out, and starting a second run over one is the mistake
// this function exists to prevent.
func hold(s *store.Store, t Task) (release func(), err error) {
	pid, alive, err := Alive(s, t)
	if err != nil {
		return nil, err
	}

	if alive {
		// This process's own pid: whoever started it wrote the marker in
		// its name the moment it was spawned, so that it counted as a run
		// from that instant and not from here. See pledge.
		if pid == os.Getpid() {
			path, pathErr := s.RunPath(t.ID)
			if pathErr != nil {
				return nil, pathErr
			}

			return released(s, t, pid, path), nil
		}

		return nil, fmt.Errorf("task %s is already being run by process %d", t.ID, pid)
	}

	return claim(s, t, os.Getpid())
}

// claim takes the marker for this process, and refuses if somebody else
// already has it.
//
// The look above and the write below are two steps, and a run is spawned
// between them: `orbit task start` asks Alive, spawns a process, and that
// process asks again and writes. Two starts of one task inside that window
// — the window's autopilot and a reader pressing enter, a second later — both
// saw nothing and both wrote, and the second marker replaced the first. Two
// engines then worked in one worktree, on one task, writing over each
// other, and only one of them could be cancelled.
//
// So the claim is the exclusive act rather than the look before it: the
// marker is created only where there is no marker, and the operating system
// decides which of two writers got there. A marker left behind by a run
// that is gone is taken off first — that is what Alive said when it
// answered not alive — and if claiming still fails after that, somebody
// else won the race and this run is the one that stops.
func claim(s *store.Store, t Task, pid int) (func(), error) {
	if err := claimFor(s, t, pid); err != nil {
		return nil, err
	}

	path, err := s.RunPath(t.ID)
	if err != nil {
		return nil, err
	}

	return released(s, t, pid, path), nil
}

// pledge writes the marker for a run this process has just spawned, in the
// child's name, before the child has done anything.
//
// The child used to write its own, a few hundred milliseconds in, and at
// the lowest priority on a busy machine later still. Until then the run
// was invisible: three starts close together each found a free slot and
// all three ran against a limit of one, and a cancel in that window took a
// task out of the queue that then started anyway. Written here, the run
// counts from the moment it exists. The child finds its own pid on it and
// keeps it: see hold.
func pledge(s *store.Store, t Task, pid int) error {
	err := claimFor(s, t, pid)
	if err == nil {
		return nil
	}

	// A child quick enough to have claimed it first claimed it in the
	// same name, which is the marker this was going to write.
	if held, alive, aliveErr := Alive(s, t); aliveErr == nil && alive && held == pid {
		return nil
	}

	return err
}

// claimFor writes the marker naming pid where there is none. A marker left
// by a run that is gone is taken off first; one held by a live run is
// refused.
func claimFor(s *store.Store, t Task, pid int) error {
	path, body, err := marker(s, t, pid)
	if err != nil {
		return err
	}

	took, err := store.WriteIfAbsent(path, body)
	if err != nil {
		return fmt.Errorf("claim task %s for this process: %w", t.ID, err)
	}

	if !took {
		// Somebody was there first. Who, and whether their run is still
		// going, is asked here and not taken from the look before this
		// one — that look is what the race got between.
		held, alive, aliveErr := Alive(s, t)
		if aliveErr != nil {
			return aliveErr
		}

		if alive {
			return fmt.Errorf("task %s is already being run by process %d", t.ID, held)
		}

		// A marker left behind by a run that is gone, which is what
		// Reconcile writes the ending for. Taking it off and claiming
		// again can still lose — another process may be doing exactly
		// this at the same moment — and losing is the answer.
		if rmErr := removeMarker(s, t); rmErr != nil {
			return rmErr
		}

		if took, err = store.WriteIfAbsent(path, body); err != nil {
			return fmt.Errorf("claim task %s for this process: %w", t.ID, err)
		}
	}

	if !took {
		return fmt.Errorf("task %s was claimed by another process while this one was starting", t.ID)
	}

	return nil
}

// marker is where a task's claim goes and what it says.
func marker(s *store.Store, t Task, pid int) (path string, body []byte, err error) {
	// The task's directory has to exist before a marker can go in it. It is
	// normally there already, made when the task was written — but the
	// marker is the first thing a run puts on disk, ahead of the
	// task.started that would otherwise make it, so the run has to be able
	// to make it itself. This is the store's own verb for that and not a MkdirAll on
	// a path taken apart here: which directories exist under the state root
	// is that package's business, not this one's.
	if _, err = s.CreateTaskDir(t.Repo.Path, t.ID); err != nil {
		return "", nil, err
	}

	path, err = s.RunPath(t.ID)
	if err != nil {
		return "", nil, err
	}

	return path, fmt.Appendf(nil, "pid: %d\nstarted: %s\n", pid, time.Now().UTC().Format(time.RFC3339)), nil
}

// mark writes the marker naming a pid, over whatever is there. It is how a
// test plants a claim for a process that is not this one; a run takes its
// claim through claim, which refuses to write over somebody else's.
func mark(s *store.Store, t Task, pid int) (func(), error) {
	// The task's directory has to exist before a marker can go in it. It is
	// normally there already, made when the task was written — but the
	// marker is the first thing a run puts on disk, ahead of the
	// task.started that would otherwise make it, so the run has to be able
	// to make it itself. This is the store's own verb for that and not a MkdirAll on
	// a path taken apart here: which directories exist under the state root
	// is that package's business, not this one's.
	path, body, err := marker(s, t, pid)
	if err != nil {
		return nil, err
	}

	// In one step, because this file is read while it is being written. A
	// plain write truncates first and writes second, and a marker caught
	// between the two has no pid line — which readMarker calls damage rather
	// than "not running", on purpose. The board asks Alive about every task
	// twice a second, so that instant is sampled constantly: it shows up as
	// an error under a run that is starting normally, and, worse, as hold
	// refusing to start a run over a claim it could not rule out. A rename is
	// atomic within a directory, so a reader sees the old marker or the new
	// one and never half of either.
	if err := store.WriteAtomically(path, body); err != nil {
		return nil, fmt.Errorf("claim task %s for this process: %w", t.ID, err)
	}

	return released(s, t, pid, path), nil
}

// released is what takes a claim off again when the run that made it ends.
func released(s *store.Store, t Task, pid int, path string) func() {
	return func() {
		// Only a marker that still names this process. A run asked to stop
		// takes as long to die as the engine it is waiting on, and another
		// run may have claimed the task in between; removing the marker
		// blindly took that claim off instead, which left a live run that
		// nothing claimed and nobody could cancel.
		//
		// The read and the remove are not one step, so a claim made between
		// the two is still taken off. That window is a file read wide,
		// against a shutdown measured in seconds, and closing it needs a
		// lock this design does not have. A claim lost that way is a stale
		// marker in reverse, and Reconcile is already the answer to it.
		if held, _, found, readErr := readMarker(s, t); readErr != nil || !found || held != pid {
			return
		}
		// Nobody is left to hand an error to: this runs while Run is
		// returning, on top of whatever answer Run already has, and a
		// failure to remove the marker must not become the run's verdict.
		// What it leaves behind if it does fail is a stale claim, which is
		// the case Reconcile already exists to clear up.
		_ = os.Remove(path) //nolint:errcheck // deliberate: see above
	}
}

// removeMarker takes a claim off a task and, unlike the release above,
// reports what happened. Reconcile calls it, and Reconcile has somebody to
// tell.
func removeMarker(s *store.Store, t Task) error {
	path, err := s.RunPath(t.ID)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear the run marker of task %s: %w", t.ID, err)
	}

	return nil
}
