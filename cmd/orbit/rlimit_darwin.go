//go:build darwin

package main

import (
	"syscall"
)

// raiseFileLimit raises the process's soft file descriptor limit from macOS's
// default of 256 up to 10240 (or the hard limit, whichever is lower) so that
// background scans, SQLite pools, and subprocesses do not exhaust descriptors.
func raiseFileLimit() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return
	}

	const target = 10240
	if rLimit.Cur >= target {
		return
	}

	rLimit.Cur = target
	if rLimit.Cur > rLimit.Max {
		rLimit.Cur = rLimit.Max
	}

	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return
	}
}
