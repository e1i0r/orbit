package task

// Start is how something that is not a terminal begins a run: the window
// today, and anything else that wants a task walked without holding the
// process itself.

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/e1i0r/orbit/internal/store"
)

// Start spawns `orbit run` for one task and returns the pid it was given.
//
// It runs the same subcommand a person would type, on purpose. A gesture in
// the window and a line in a terminal have to reach the same code, or the
// window becomes a second implementation of running a task and the two
// drift; and a run in a process of its own is a run that survives the window
// closing, which is the whole reason the window may start one at all.
//
// Nothing is wired to the child's stdout or stderr, also on purpose. The
// record is the channel a run reports through — that is what makes the run
// readable by `orbit show`, by `cat`, and by the window at the same time —
// and a background process writing over a full-screen terminal is the
// alternative.
func Start(s *store.Store, t Task, flowName string) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("find the orbit binary to start task %s: %w", t.ID, err)
	}
	cmd := runCommand(exe, s.Root(), t, flowName)
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start a run of task %s: %w", t.ID, err)
	}
	pid := cmd.Process.Pid
	// The child is waited on in a goroutine, and its verdict is thrown
	// away. Nothing here wants the exit status — the record carries the
	// outcome, and the caller is a window that will read it from there —
	// but a child nobody waits on becomes a zombie, and a zombie answers
	// kill(pid, 0) as though it were alive. That would make Alive say a
	// dead run is running for as long as the window stays open, which is
	// exactly the lie this task exists to remove.
	go func() {
		_ = cmd.Wait() //nolint:errcheck // deliberate: see above
	}()
	return pid, nil
}

// runCommand is the command line Start spawns. It is split out so it can be
// asserted without a binary to run, exactly as claudeArgs was split out of
// engine.Claude.Run so the claude command line could be tested without
// claude installed.
func runCommand(exe, root string, t Task, flowName string) *exec.Cmd {
	cmd := exec.Command(exe, "run", "-repo", t.Repo.Path, "-flow", flowName, t.ID)
	// The repository the task is against, and not whatever directory the
	// caller happens to be in. -repo is absolute so nothing depends on this,
	// but a child inheriting a working directory that may be deleted while
	// it runs is a run that dies for a reason nobody can see.
	cmd.Dir = t.Repo.Path
	// The state root is passed rather than inherited. A run that wrote its
	// events into a different root than its reader is reading would be a
	// task that vanished at the moment it started; the environment is where
	// the root already comes from, so this is the same door, held open.
	cmd.Env = append(os.Environ(), "ORBIT_HOME="+root)
	// Its own process group, which is the point of spawning it this way. A
	// terminal that goes away sends SIGHUP to its foreground group, and a
	// run sharing that group dies with the window that started it — with
	// nothing written down, because the default action for SIGHUP leaves no
	// time to write. In a group of its own it survives, and `orbit cancel
	// -now` has one group to signal rather than a process tree to walk.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}
