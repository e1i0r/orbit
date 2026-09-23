//go:build !windows

package repo

// Killing a check that will not stop, where processes have groups.

import (
	"os/exec"
	"syscall"
)

// ownGroup puts a check in a process group of its own and makes the deadline
// signal the group rather than the shell alone.
//
// A check is a shell line, and the work is whatever it spawns: killing `sh`
// leaves `go test` running and holding the output pipe, so the read waited
// past the deadline for ever. Signalling the group reaches the work.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}

// shellLine is how a check is started: through the shell, at the lowest
// priority there is. The comparison runs the repository's suite twice at
// once, and a suite like make check already takes every core; at the
// ordinary priority the window and the run beside it were left waiting on
// the CPU, and Elio watched orbit go heavy while the two sides ran. Nice
// only yields a core somebody else wants, so on an idle machine it costs
// nothing.
func shellLine(command string) []string {
	return []string{"nice", "-n", "19", "sh", "-c", command}
}
