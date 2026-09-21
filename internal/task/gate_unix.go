//go:build !windows

package task

// Killing a gate that will not stop, and everything it started.

import (
	"os/exec"
	"syscall"

	"github.com/e1i0r/orbit/internal/kin"
)

// ownGroup puts a gate's shell in a process group of its own and makes a
// cancelled run stop the shell and everything under it.
//
// A gate is a shell line and the work is whatever it spawns: killing `sh`
// leaves the `go test` under it running in a worktree the run has finished
// with. internal/repo learned this on the checks it runs — see
// compare_unix.go — and the gates a flow carries went on running the old
// way.
//
// The family and not the group, for the reason internal/engine gives: a
// child that puts itself in a group of its own is not reached by
// signalling the one it left, and that is what a real tool call does.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return kin.Stop(cmd.Process.Pid) }
}
