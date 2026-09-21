//go:build !windows

package task

// Killing a gate that will not stop, where processes have groups.

import (
	"os/exec"
	"syscall"
)

// ownGroup puts a gate's shell in a process group of its own and makes a
// cancelled run signal the group rather than the shell alone.
//
// A gate is a shell line and the work is whatever it spawns: killing `sh`
// leaves the `go test` under it running in a worktree the run has finished
// with. internal/repo learned this on the checks it runs — see
// compare_unix.go, which is this file's twin — and the gates a flow carries
// went on running the old way.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}
