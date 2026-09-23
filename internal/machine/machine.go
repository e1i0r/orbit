// Package machine reads how much of the machine is in use, for the queue to
// decide whether another run may start.
//
// Only what the queue asks, and read the cheapest way each system offers:
// /proc/meminfo on Linux, sysctl and vm_stat on macOS. A system that cannot
// be read answers that it could not, and the queue then does not hold runs
// back on a number it does not have.
package machine

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

// MemoryUsed is how full the machine's memory is, as a percentage, and
// whether it could be read at all.
func MemoryUsed() (int, bool) { return memoryUsed() }

// percent is used out of total, rounded down, and false for a total of
// nothing.
func percent(total, available uint64) (int, bool) {
	if total == 0 || available > total {
		return 0, false
	}

	return int((total - available) * 100 / total), true
}

// fromMeminfo reads /proc/meminfo: MemTotal and MemAvailable, in kB. The
// kernel's own estimate of what can be had without swapping is the honest
// answer, which is why MemFree is not used.
func fromMeminfo(text string) (int, bool) {
	var total, available uint64

	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}

		n, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			total = n
		case "MemAvailable:":
			available = n
		}
	}

	return percent(total, available)
}

// vmPage is vm_stat's page size and its page counts.
var (
	vmPageSize = regexp.MustCompile(`page size of (\d+) bytes`)
	vmCount    = regexp.MustCompile(`^Pages (free|inactive|speculative|purgeable):\s+(\d+)\.`)
)

// fromVMStat reads what macOS can hand out without swapping from vm_stat:
// free, inactive, speculative and purgeable pages, which is what Activity
// Monitor leaves out of "used". total is hw.memsize, in bytes.
func fromVMStat(text string, total uint64) (int, bool) {
	size := vmPageSize.FindStringSubmatch(text)
	if size == nil {
		return 0, false
	}

	pageSize, err := strconv.ParseUint(size[1], 10, 64)
	if err != nil {
		return 0, false
	}

	var pages uint64

	for _, line := range strings.Split(text, "\n") {
		if m := vmCount.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			n, parseErr := strconv.ParseUint(m[2], 10, 64)
			if parseErr == nil {
				pages += n
			}
		}
	}

	return percent(total, pages*pageSize)
}
