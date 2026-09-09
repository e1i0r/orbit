//go:build integration

package integration

// Ending the processes these tests start.
//
// A run is built to outlive whoever launched it: `task.Start` gives it its
// own process group so a closed terminal does not take a working run with
// it. A test binary that dies without running its cleanups therefore leaves
// a live run behind, reparented to init, with nothing left that will ever
// end it. Four were found alive on a laptop on 2026-09-09, the oldest from a
// `go test` interrupted a day and a half earlier, still holding a temporary
// directory that TestMain had already deleted.
//
// Three doors, because no single one covers every way a test run ends.
// t.Cleanup covers the test that finishes. An interrupt handler covers the
// Ctrl-C that skips cleanups. Neither is reached by the timeout panic or by
// a SIGKILL, so what those leave behind is swept by the next test binary to
// start: a process is a leftover of this harness when it runs a binary out
// of an orbit-integration directory and init has adopted it, which is a
// thing only a corpse is. A run of this package happening at the same time
// still has its own binary as a parent, so nothing here reaches into it.

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// watched is every child a test asked to outlive its call, so that something
// other than t.Cleanup can end them.
var watched struct {
	sync.Mutex

	cmds []*exec.Cmd
}

// watch remembers a started child.
func watch(cmd *exec.Cmd) {
	watched.Lock()
	defer watched.Unlock()

	watched.cmds = append(watched.cmds, cmd)
}

// end kills a child and everything it spawned, and waits for it.
//
// The signal goes to the group rather than the process because a run spawns
// an engine, and a SIGKILL to the run alone would leave the engine behind —
// the same leak one level down. Wait is what takes the process off the
// system rather than leaving it a zombie; it is called for its effect, and
// answers an error on every child already ended, which is the normal case
// when both a cleanup and the interrupt handler reach the same run.
func end(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	// Gone is the point, and gone already is fine.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck
	// Reaping a process another door may have reaped already.
	_ = cmd.Wait() //nolint:errcheck
}

// endWatched ends every child started so far.
func endWatched() {
	watched.Lock()
	defer watched.Unlock()

	for _, cmd := range watched.cmds {
		end(cmd)
	}

	watched.cmds = nil
}

// catchInterrupt ends the started children on a Ctrl-C, which testing does
// not: an interrupted binary runs no t.Cleanup and returns from no m.Run.
//
// The directory holding the binaries under test is taken away here too, for
// the same reason. Exiting 130 is the shell's word for "interrupted", and
// what a caller that interrupted a test expects to read.
func catchInterrupt(dir string) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals

		endWatched()
		os.RemoveAll(dir) //nolint:errcheck // going away either way

		os.Exit(130)
	}()
}

// sweep ends the runs an earlier test binary left behind.
//
// It reads the process table rather than a file of pids, because the case it
// is for is the one where nothing got the chance to write anything down.
func sweep() {
	out, err := exec.Command("ps", "-A", "-o", "pid=,ppid=,command=").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "look for runs left by an earlier test:", err)

		return
	}

	for _, pid := range leftovers(string(out)) {
		// A corpse, whose group may be gone; and the process itself, for
		// the one that leads no group of its own.
		_ = syscall.Kill(-pid, syscall.SIGKILL) //nolint:errcheck
		_ = syscall.Kill(pid, syscall.SIGKILL)  //nolint:errcheck
	}
}

// leftovers is the pids in a ps listing that belong to a test binary that is
// no longer there.
//
// Two things have to be true together. The command runs out of one of this
// harness's temporary directories, which is a path no other program has; and
// its parent is init, which for a child of a test binary means the test
// binary is gone. A run of these tests happening right now fails the second
// and is left alone.
func leftovers(listing string) []int {
	var pids []int

	for _, line := range strings.Split(listing, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || !strings.Contains(line, harnessDir) {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil || fields[1] != "1" || pid <= 1 {
			continue
		}

		pids = append(pids, pid)
	}

	return pids
}

// harnessDir is the mark of a binary built by these tests: the prefix
// MkdirTemp is given in TestMain, which nothing else on the machine uses.
const harnessDir = "orbit-integration-"
