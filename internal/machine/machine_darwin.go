//go:build darwin

package machine

import (
	"os/exec"
	"strconv"
	"strings"
)

// memoryUsed asks sysctl for the size of memory and vm_stat for what is
// free in it.
func memoryUsed() (int, bool) {
	size, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, false
	}

	total, err := strconv.ParseUint(strings.TrimSpace(string(size)), 10, 64)
	if err != nil {
		return 0, false
	}

	stat, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0, false
	}

	return fromVMStat(string(stat), total)
}
