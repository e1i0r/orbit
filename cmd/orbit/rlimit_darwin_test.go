//go:build darwin

package main

import (
	"syscall"
	"testing"
)

func TestRaiseFileLimitFromLowerValue(t *testing.T) {
	var original syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &original); err != nil {
		t.Fatalf("getrlimit: %v", err)
	}

	// Lower the soft limit to 512.
	low := original
	low.Cur = 512

	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &low); err != nil {
		t.Fatalf("setrlimit lower: %v", err)
	}
	defer func() {
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &original); err != nil {
			t.Errorf("restore rlimit: %v", err)
		}
	}()

	raiseFileLimit()

	var after syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &after); err != nil {
		t.Fatalf("getrlimit after: %v", err)
	}

	if after.Cur < 10240 && after.Cur < original.Max {
		t.Errorf("rlimit after raiseFileLimit = %d, want >= 10240", after.Cur)
	}
}
