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
// A failure is (zero, false) rather than an error. Every caller's fallback
// is the behaviour Orbit had before this file existed, which is a worse
// answer but never a wrong one, and a machine that cannot say when it booted
// is not a reason to refuse to draw a board.
func bootTime() (time.Time, bool) {
	raw, err := syscall.Sysctl("kern.boottime")
	if err != nil {
		return time.Time{}, false
	}

	return parseBootTime(raw)
}

// parseBootTime reads the answer kern.boottime gives.
//
// It is a struct timeval — a seconds field and a microseconds field — and
// syscall.Sysctl hands it back as the raw bytes in a string. They are read
// little-endian, named outright rather than described as the machine's own
// byte order: every darwin Go builds for is little-endian (amd64 and arm64),
// so the two agree, and saying which one is being assumed is the point. Only the seconds are read: this is compared
// against a timestamp Orbit wrote to the second, so microseconds would be
// precision with nothing on the other side to match it.
//
// It is a function of its own for the reason codexArgs is one — that command
// line is asserted without codex installed. The two refusals below only
// happen on a machine that has stopped being able to answer kern.boottime,
// and no test can make this machine stop answering. Handed the bytes
// instead, they are two strings.
func parseBootTime(raw string) (time.Time, bool) {
	if len(raw) < 8 {
		return time.Time{}, false
	}

	sec := binary.LittleEndian.Uint64([]byte(raw)[:8])
	if sec == 0 {
		return time.Time{}, false
	}

	return time.Unix(int64(sec), 0).UTC(), true
}
