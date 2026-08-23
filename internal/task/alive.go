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
// number since, this reports a dead run as alive, and the run stays in the
// window as running until somebody looks. And the process it finds might not
// be Orbit at all, for the same reason. Both are the cost of a pid file, and
// both fail in the safe direction — towards leaving a record alone rather
// than towards writing "abandoned" over a run that is working.
func Alive(s *store.Store, t Task) (pid int, ok bool, err error) {
	pid, found, err := readMarker(s, t)
	if err != nil || !found {
		return 0, false, err
	}
	return pid, running(pid), nil
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
func hold(s *store.Store, t Task) (release func(), err error) {
	return mark(s, t, os.Getpid())
}

// mark writes the marker naming a pid. It is separate from hold so a test
// can plant a marker for a process that is not this one.
func mark(s *store.Store, t Task, pid int) (func(), error) {
	path, err := s.RunPath(t.Repo.Path, t.ID)
	if err != nil {
		return nil, err
	}
	body := fmt.Sprintf("pid: %d\nstarted: %s\n", pid, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return nil, fmt.Errorf("claim task %s for this process: %w", t.ID, err)
	}
	return func() {
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
func readMarker(s *store.Store, t Task) (pid int, found bool, err error) {
	path, err := s.RunPath(t.Repo.Path, t.ID)
	if err != nil {
		return 0, false, err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read the run marker of task %s: %w", t.ID, err)
	}
	pid, err = parsePid(string(body))
	if err != nil {
		return 0, false, fmt.Errorf("read the run marker of task %s: %w", t.ID, err)
	}
	return pid, true, nil
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
		// A number below 1 is never a process. Zero and negative numbers are
		// how kill(2) is told to signal a whole process group or every
		// process on the machine, so a damaged marker must be stopped here
		// rather than handed to a signal.
		if pid < 1 {
			return 0, fmt.Errorf("the marker names pid %d, which is not a process", pid)
		}
		return pid, nil
	}
	return 0, errors.New("the marker has no pid line")
}
