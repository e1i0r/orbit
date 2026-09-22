//go:build windows

package task

// Killing a gate that will not stop, where processes have no groups.

import "os/exec"

// ownGroup is nothing here: Windows has no process group to signal, and
// os/exec's own kill reaches the shell alone. What bounds the wait is
// Cmd.WaitDelay, which runGates sets either way — so a gate that spawns
// work outliving its shell stops holding the run, and the work itself is
// left to the operating system.
func ownGroup(cmd *exec.Cmd) {}
