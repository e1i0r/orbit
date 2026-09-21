//go:build !windows

package engine

// Killing an engine that will not stop, and everything it started.

import (
	"os/exec"
	"syscall"

	"github.com/e1i0r/orbit/internal/kin"
)

// ownGroup puts an engine in a process group of its own and makes a
// cancelled phase stop the engine and everything under it.
//
// A tool call is a process: the engine spawns a shell, a build, a test
// suite, and those are what the work actually is. Killing the engine on its
// own left them running in a worktree the run had finished with, still
// writing to it — and the run itself waited on an output pipe they had
// inherited, so a cancelled phase did not end until they did.
//
// The group is not enough on its own, which a real engine showed and a
// fake could not: opencode runs every shell command its bash tool is asked
// for in a new group, so a cancel that signalled the engine's group left
// `/bin/zsh -c "sleep 120 && echo done >> notes.txt"` behind with ppid 1,
// writing into the worktree of a task the record already called cancelled.
// internal/kin stops the family rather than the group.
//
// A daemon an engine means to keep across runs is caught by this where the
// group would have missed it. That is the trade, and it is the right way
// round: a process still running inside a worktree Orbit has finished with
// is the failure, and an engine that wants something to outlive a run can
// have it survive the run it belongs to rather than every run there is.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return kin.Stop(cmd.Process.Pid) }
}
