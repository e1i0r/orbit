//go:build darwin

package task

// When the machine last started, on darwin.

import (
	"encoding/binary"
	"syscall"
	"time"
)

// bootTime asks the kernel when it came up.
//
// kern.boottime is a struct timeval — a seconds field and a microseconds
// field, in the machine's own byte order — and syscall.Sysctl hands it back
// as the raw bytes in a string. Only the seconds are read: this is compared
// against a timestamp Orbit wrote to the second, so microseconds would be
// precision with nothing on the other side to match it.
//
// A failure is (zero, false) rather than an error. Every caller's fallback
// is the behaviour Orbit had before this file existed, which is a worse
// answer but never a wrong one, and a machine that cannot say when it booted
// is not a reason to refuse to draw a board.
func bootTime() (time.Time, bool) {
	raw, err := syscall.Sysctl("kern.boottime")
	if err != nil || len(raw) < 8 {
		return time.Time{}, false
	}
	sec := binary.LittleEndian.Uint64([]byte(raw)[:8])
	if sec == 0 {
		return time.Time{}, false
	}
	return time.Unix(int64(sec), 0).UTC(), true
}
