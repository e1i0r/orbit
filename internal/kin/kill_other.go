//go:build windows

package kin

// Signalling, where processes have no groups.

import "os"

// kill stops one process. Windows has no signals worth the name, and
// os.Process.Kill is what there is.
func kill(pid int) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}

	_ = p.Kill() //nolint:errcheck // a process already gone is the ordinary case
}

// killGroup is nothing here: there is no group to signal, and the tree walk
// above is the whole of what reaches a child.
func killGroup(int) {}
