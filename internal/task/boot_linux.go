//go:build linux

package task

// When the machine last started, on linux.

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// bootTime reads btime out of /proc/stat, which is the boot moment as
// seconds since the epoch. /proc/uptime would need the current time to be
// turned into the same thing and would drift by however long the read took;
// btime is the number itself.
//
// A failure is (zero, false), for the reason the darwin twin gives.
func bootTime() (time.Time, bool) {
	body, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, false
	}
	for line := range strings.SplitSeq(string(body), "\n") {
		rest, ok := strings.CutPrefix(line, "btime ")
		if !ok {
			continue
		}
		sec, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64)
		if err != nil || sec == 0 {
			return time.Time{}, false
		}
		return time.Unix(sec, 0).UTC(), true
	}
	return time.Time{}, false
}
