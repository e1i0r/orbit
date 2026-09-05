package logger

import (
	"os"
	"testing"
)

// TestOpenFilesCountsWhatThisProcessHolds. The count is the one number that
// would have named the leak that took the machine's file table: it climbs
// while nothing at all is failing, which is the shape of bug an errors file
// cannot hold.
func TestOpenFilesCountsWhatThisProcessHolds(t *testing.T) {
	before := OpenFiles()
	if before < 0 {
		t.Skip("this system does not list a process's own descriptors")
	}

	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}

	after := OpenFiles()

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if after <= before {
		t.Errorf("one more file open counted %d, and %d before it", after, before)
	}
}
