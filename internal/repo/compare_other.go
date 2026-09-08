//go:build windows

package repo

// Killing a check that will not stop, where processes have no groups.

import "os/exec"

// ownGroup is nothing here: Windows has no process group to signal, and
// os/exec's own kill reaches the shell alone. What bounds the wait is
// Cmd.WaitDelay, which the caller sets either way — so a check that spawns
// work outliving its shell is reported as timed out on time, and the work
// itself is left to the operating system.
func ownGroup(cmd *exec.Cmd) {}
