//go:build windows

package engine

// Killing an engine that will not stop, where processes have no groups.

import "os/exec"

// ownGroup is nothing here: Windows has no process group to signal, and
// os/exec's own kill reaches the engine alone. What bounds the wait is
// Cmd.WaitDelay, which run sets either way — so a phase whose engine left
// work behind stops holding the run, and the work is left to the operating
// system.
func ownGroup(cmd *exec.Cmd) {}
