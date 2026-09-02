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
//
// unread is how many finished tasks nobody has looked at, and it is the one
// place the cap is enforced. It is a parameter rather than something this
// function works out, because working it out means folding every record in
// the state root — which the caller has already done, and which this package
// must not do a second time: internal/board and internal/view are the
// readers of that format, and internal/board imports this package, so a fold
// here would be both a second opinion and a cycle. Whoever holds the board
// passes the number; internal/board has Unread for exactly that.
//
// The cap itself comes from the settings file and is read here, so that a
// caller cannot forget it. At the cap Start refuses and says the two numbers,
// because a brake that says only "no" is a brake people route around.
//
// A task something else is already running is refused too, in the same words
// the run itself would use. Run's hold is still the authority and this is
// still not a lock — a run that begins in the instant between the look and
// the spawn is caught there, not here. What this closes is that hold's
// refusal is invisible from out here by design: it writes nothing to the
// record, because the log already describes the run that is happening, and
// its stderr goes nowhere by the paragraph above. So the caller was handed a
// pid and no error for a child that had already exited, and both callers say
// the same thing with it — the window draws the task as running again, and
// mcp answers "task X is running again" — over a run that never began.
func Start(s *store.Store, t Task, flowName string, unread int) (int, error) {
	holder, alive, err := Alive(s, t)
	if err != nil {
		return 0, err
	}

	if alive {
		return 0, fmt.Errorf("task %s is already being run by process %d", t.ID, holder)
	}

	cfg, err := s.Settings()
	if err != nil {
		return 0, err
	}

	if atCap(unread, cfg.UnreadCap) {
		return 0, fmt.Errorf("task %s was not started: %d finished tasks are unread and the cap is %d — read one with `orbit read`, or change the cap with `orbit set unread-cap <n>`", t.ID, unread, cfg.UnreadCap)
	}

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
// A task with no repository is started without -repo, which is the flag
// saying which one it is against: passing an empty path would have `orbit
// run` open the current directory and hand the run whatever repository the
// window happens to be sitting in. The run reads the flag the same way, so
// the absence is the answer.
func runCommand(exe, root string, t Task, flowName string) *exec.Cmd {
	args := []string{"run"}
	if t.Repo.Path != "" {
		args = append(args, "-repo", t.Repo.Path)
	}

	args = append(args, "-flow", flowName, t.ID)

	cmd := exec.Command(exe, args...)
	// The repository the task is against, and not whatever directory the
	// caller happens to be in. -repo is absolute so nothing depends on this,
	// but a child inheriting a working directory that may be deleted while
	// it runs is a run that dies for a reason nobody can see. A task with no
	// repository is run from the state root, which is the one directory such
	// a run is certain of.
	cmd.Dir = t.Repo.Path
	if cmd.Dir == "" {
		cmd.Dir = root
	}
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
