//go:build !windows

package kin

// Signalling one process, and one group of them.

import "syscall"

// kill stops one process, whatever group it put itself in.
func kill(pid int) {
	if pid <= 1 {
		return
	}

	// Nothing to report: a process that has already gone is the ordinary
	// case here, and this runs while a run is being cancelled.
	_ = syscall.Kill(pid, syscall.SIGKILL) //nolint:errcheck // see above
}

// killGroup stops the group a process leads, which reaches the children
// that stayed in it — including any spawned between reading the table and
// signalling it.
//
// Only a group the process leads. A run that is one process of somebody
// else's group would otherwise take that group down with it, which is the
// rule cancel.go keeps for the same reason.
func killGroup(pid int) {
	if pgid, err := syscall.Getpgid(pid); err == nil && pgid == pid && pgid > 1 {
		_ = syscall.Kill(-pgid, syscall.SIGKILL) //nolint:errcheck // see kill
	}
}
