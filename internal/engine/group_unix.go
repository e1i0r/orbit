//go:build !windows

package engine

// Killing an engine that will not stop, where processes have groups.

import (
	"os/exec"
	"syscall"
)

// ownGroup puts an engine in a process group of its own and makes a
// cancelled phase signal the group rather than the engine alone.
//
// A tool call is a process: the engine spawns a shell, a build, a test
// suite, and those are what the work actually is. Killing the engine on its
// own left them running in a worktree the run had finished with, still
// writing to it — and the run itself waited on an output pipe they had
// inherited, so a cancelled phase did not end until they did.
//
// A daemon an engine wants to keep across runs is not caught by this: a
// process that means to outlive its parent leaves the group, which is what
// leaving the group is for.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}
