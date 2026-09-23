//go:build linux

package machine

import "os"

// memoryUsed reads /proc/meminfo.
func memoryUsed() (int, bool) {
	text, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, false
	}

	return fromMeminfo(string(text))
}
