package logger

// How many files this process holds open.
//
// It is here, in the package that writes the log, because it is a fact about
// the process that only the log has anywhere to put — and because of the bug
// that asked for it. A server opened the state root for every call it
// answered and closed it in none of them. Every one of those opens
// succeeded: nothing failed, so nothing was written down, until the
// machine's own file table was full and the failures that finally appeared
// belonged to every process except the one holding the files.
//
// A number that climbs while nothing is going wrong is the only warning such
// a leak gives, and an errors file is the wrong place to look for it.

import "os"

// fdDir is where a process's own descriptors are listed. Both systems this
// runs on have it; on Linux it is the link to /proc/self/fd.
const fdDir = "/dev/fd"

// OpenFiles is how many descriptors this process has open, and -1 where that
// cannot be asked.
//
// The names are read rather than the entries, because reading the directory
// opens a descriptor of its own: it is counted, which is honest, but it is
// gone by the time anything could stat it, and asking about each entry is
// what turns that race into an error.
func OpenFiles() int {
	d, err := os.Open(fdDir)
	if err != nil {
		return -1
	}

	defer func() { _ = d.Close() }()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return -1
	}

	return len(names)
}
