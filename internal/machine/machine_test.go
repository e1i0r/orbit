package machine

import "testing"

// TestLinuxMemoryIsWhatTheKernelSaysCanBeHad: MemAvailable, not MemFree,
// which leaves out the cache the kernel gives back the moment it is asked.
func TestLinuxMemoryIsWhatTheKernelSaysCanBeHad(t *testing.T) {
	meminfo := "MemTotal: 16000000 kB\n" +
		"MemFree: 500000 kB\n" +
		"MemAvailable: 4000000 kB\n"

	got, ok := fromMeminfo(meminfo)
	if !ok || got != 75 {
		t.Errorf("memory in use read as %d%% (%v), want 75%%", got, ok)
	}
}

// TestMacMemoryLeavesOutWhatCanBeHandedBack: free, inactive, speculative
// and purgeable pages, in the page size vm_stat names.
func TestMacMemoryLeavesOutWhatCanBeHandedBack(t *testing.T) {
	vmstat := `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                                     4000.
Pages active:                                 267017.
Pages inactive:                               240000.
Pages speculative:                             18000.
Pages wired down:                             153447.
Pages purgeable:                                 144.
`
	// 262144 pages of 16384 bytes free, out of 16 GiB: a quarter.
	got, ok := fromVMStat(vmstat, 16<<30)
	if !ok || got != 75 {
		t.Errorf("memory in use read as %d%% (%v), want 75%%", got, ok)
	}
}

// TestWhatCannotBeReadSaysSo rather than reading as empty or full.
func TestWhatCannotBeReadSaysSo(t *testing.T) {
	if _, ok := fromMeminfo("nothing here"); ok {
		t.Error("a meminfo with no totals was read")
	}

	if _, ok := fromVMStat("no header", 16<<30); ok {
		t.Error("a vm_stat with no page size was read")
	}
}

// TestThisMachineCanBeRead where the system is one this package knows.
func TestThisMachineCanBeRead(t *testing.T) {
	got, ok := MemoryUsed()
	if ok && (got < 0 || got > 100) {
		t.Errorf("memory in use read as %d%%", got)
	}
}
