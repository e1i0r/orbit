package task

// The run marker: two lines of text under a task's directory saying which
// process holds it and since when.
//
//	pid: 4711
//	started: 2026-08-23T09:14:02Z
//
// It is written when a run begins and taken off on every way out of one, and
// it is the only thing in Orbit that can tell a phase still running from a
// phase whose process is gone. The record cannot: a run killed with SIGKILL
// writes nothing on its way out, so a log ending at phase.started is either
// a run in flight or a run that died three days ago, and no amount of
// reading events tells those apart.
//
// Text, and one fact per line, like everything else Orbit writes down. `cat`
// is a supported way to answer "who is running this?".

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/e1i0r/orbit/internal/store"
)

// Alive says which process holds a task, and whether it is still there.
//
// There are three answers and they are different facts. No marker at all is
// (0, false, nil): nothing claims this task. A marker whose process is gone
// is (pid, false, nil): a claim that outlived the run that made it, which is
// what Reconcile is for. A marker whose process answers is (pid, true, nil).
//
// Two things it cannot answer, said out loud rather than hidden. A pid can
// be reused: if the machine went down and something else has taken that
// number since, this reports a dead run as alive. And the process it finds
// might not be Orbit at all, for the same reason. Both are the cost of a pid
// file.
//
// What that costs depends on who is asking, and the two answers are not the
// same. The board only draws a row, so it fails in the safe direction: the
// run sits there as running until somebody looks, which is better than
// writing "abandoned" over a run that is working. Cancel and Kill do not
// fail safe. They send SIGTERM and SIGKILL to whatever holds that number
// now, and this function is the only thing that told them it was a run.
//
// The window for that is narrow and it is real: a marker outlives its run
// (SIGKILL, or the machine went down), the number is recycled by something
// unrelated, and a cancel arrives before Reconcile has swept the marker
// away.
//
// The larger half of that window is closed here, by the marker's own start
// time. A machine that went down and came back has renumbered everything:
// pids are handed out from the low numbers again, so a marker written before
// the last boot names a number that now belongs to whatever happened to take
// it — very often something that started early and is still running, which
// is the worst case, because it answers yes. So a marker older than the boot
// is not consulted about its pid at all. It is a claim from a previous life
// of this machine, and the honest answer is that its run is gone.
//
// What is left open is a pid recycled without a reboot, on a machine up long
// enough to wrap the number round. That is rarer by orders of magnitude, and
// closing it needs the start time of the process rather than of the machine —
// which the standard library does not expose on darwin, and this build takes
// no dependency to reach. What guards it meanwhile is narrowing what a
// marker may name at all: see parsePid, and killTarget in cancel.go.
func Alive(s *store.Store, t Task) (pid int, ok bool, err error) {
	pid, started, found, err := readMarker(s, t)
	if err != nil || !found {
		return 0, false, err
	}
	if staleAcrossBoot(started) {
		return pid, false, nil
	}
	return pid, running(pid), nil
}

// staleAcrossBoot is whether a marker was written before the machine last
// started.
//
// Both unknowns answer false, and both are deliberate. A marker with no
// start time was written by a version of Orbit that did not record one, and
// declaring every one of those dead would abandon runs that are working. A
// machine that cannot say when it booted is the same shape of ignorance from
// the other side. In both cases Alive falls back to asking the pid, which is
// what it did before this check existed.
func staleAcrossBoot(started time.Time) bool {
	if started.IsZero() {
		return false
	}
	boot, ok := bootTime()
	if !ok {
		return false
	}
	// A second of slack, because the marker's timestamp is written to the
	// second and the boot time is read to the second: a run that began in
	// the same second the machine finished booting must not be read as
	// having begun before it.
	return started.Before(boot.Add(-time.Second))
}

// running asks the operating system whether a pid is still there. Signal 0
// delivers nothing and only performs the checks kill(2) would perform, which
// is exactly the question.
func running(pid int) bool {
	err := syscall.Kill(pid, 0)
	// EPERM is a process that exists and is not ours to signal, which is
	// still an answer of yes. ESRCH — no such process — is the no.
	return err == nil || errors.Is(err, syscall.EPERM)
}

// hold claims a task for this process and hands back the one way to let go.
// The release is safe to call twice and says nothing when there was never a
// marker, because unwinding is not the moment to start reporting problems.
//
// A task something else is already running is refused. Two runs of one task
// share a worktree, a branch and a log: their phases interleave in the
// record, and the first of them to finish used to take the other's marker
// off on its way out, leaving a live run that nothing claimed and nobody
// could cancel. The marker is the only thing that can say a task is taken,
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
		return nil, fmt.Errorf("task %s is already being run by process %d", t.ID, pid)
	}
	return mark(s, t, os.Getpid())
}

// mark writes the marker naming a pid. It is separate from hold so a test
// can plant a marker for a process that is not this one.
func mark(s *store.Store, t Task, pid int) (func(), error) {
	// The task's directory has to exist before a marker can go in it. It is
	// normally there already, made when the task was written — but the
	// marker is now the first thing a run puts on disk, ahead of the
	// task.started that used to make it, so the run has to be able to make
	// it itself. This is the store's own verb for that and not a MkdirAll on
	// a path taken apart here: which directories exist under the state root
	// is that package's business, not this one's.
	if _, err := s.CreateTaskDir(t.Repo.Path, t.ID); err != nil {
		return nil, err
	}
	path, err := s.RunPath(t.Repo.Path, t.ID)
	if err != nil {
		return nil, err
	}
	body := fmt.Sprintf("pid: %d\nstarted: %s\n", pid, time.Now().UTC().Format(time.RFC3339))
	// In one step, because this file is read while it is being written. A
	// plain write truncates first and writes second, and a marker caught
	// between the two has no pid line — which readMarker calls damage rather
	// than "not running", on purpose. The board asks Alive about every task
	// twice a second, so that instant is sampled constantly: it shows up as
	// an error under a run that is starting normally, and, worse, as hold
	// refusing to start a run over a claim it could not rule out. A rename is
	// atomic within a directory, so a reader sees the old marker or the new
	// one and never half of either.
	if err := store.WriteAtomically(path, []byte(body)); err != nil {
		return nil, fmt.Errorf("claim task %s for this process: %w", t.ID, err)
	}
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
	}, nil
}

// removeMarker takes a claim off a task and, unlike the release above,
// reports what happened. Reconcile calls it, and Reconcile has somebody to
// tell.
func removeMarker(s *store.Store, t Task) error {
	path, err := s.RunPath(t.Repo.Path, t.ID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear the run marker of task %s: %w", t.ID, err)
	}
	return nil
}

// readMarker reads the marker, if there is one. A marker that is there and
// cannot be understood is an error rather than a silent "not running": the
// file is one line of text this package wrote, and a reader that shrugs at
// damage it cannot explain is how a running task gets declared abandoned.
func readMarker(s *store.Store, t Task) (pid int, started time.Time, found bool, err error) {
	path, err := s.RunPath(t.Repo.Path, t.ID)
	if err != nil {
		return 0, time.Time{}, false, err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, time.Time{}, false, nil
	}
	if err != nil {
		return 0, time.Time{}, false, fmt.Errorf("read the run marker of task %s: %w", t.ID, err)
	}
	pid, err = parsePid(string(body))
	if err != nil {
		return 0, time.Time{}, false, fmt.Errorf("read the run marker of task %s: %w", t.ID, err)
	}
	return pid, parseStarted(string(body)), true, nil
}

// parseStarted picks the start time out of a marker, and answers the zero
// time when there is not one to pick.
//
// Unlike parsePid this never fails. A missing or unreadable start time is a
// marker from before Orbit wrote one, or a marker somebody edited, and the
// caller's answer to "I do not know when this began" is already written: it
// falls back to asking the pid. Turning it into an error would make an old
// marker unreadable, which would declare a running task damaged.
func parseStarted(body string) time.Time {
	for line := range strings.SplitSeq(body, "\n") {
		rest, ok := strings.CutPrefix(line, "started: ")
		if !ok {
			continue
		}
		at, err := time.Parse(time.RFC3339, strings.TrimSpace(rest))
		if err != nil {
			return time.Time{}
		}
		return at.UTC()
	}
	return time.Time{}
}

// parsePid picks the pid line out of a marker.
func parsePid(body string) (int, error) {
	for line := range strings.SplitSeq(body, "\n") {
		rest, ok := strings.CutPrefix(line, "pid: ")
		if !ok {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return 0, fmt.Errorf("the pid line reads %q: %w", line, err)
		}
		// A number below 2 is never a run this may signal. Zero and
		// negative numbers are how kill(2) is told to signal a whole
		// process group or every process on the machine, and 1 is worse
		// than either: Kill negates a pid to reach its group, and -1 is
		// POSIX for every process this user may signal. `orbit run` is pid
		// 1 in a container, so a marker naming 1 is not only a marker
		// somebody hand-edited — and refusing it is the right answer there
		// too, because a run that is its container's init is stopped by
		// stopping the container, not by a signal from inside it.
		if pid <= 1 {
			return 0, fmt.Errorf("the marker names pid %d, which is not a process this may signal", pid)
		}
		return pid, nil
	}
	return 0, errors.New("the marker has no pid line")
}
