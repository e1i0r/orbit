package mcp

// What a tool call leaves behind. A server answers questions for as long as
// the client it was spawned by is running, so anything one call keeps is
// something every call keeps.

import (
	"os"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// openDescriptors is how many files this process has open. /dev/fd is the
// directory of them on both systems this runs on, and the names are read
// without asking about each one: the entry for the handle doing the reading
// is gone by the time anything could stat it.
func openDescriptors(t *testing.T) int {
	t.Helper()

	d, err := os.Open("/dev/fd")
	if err != nil {
		t.Skipf("this system does not list open descriptors: %v", err)
	}

	defer func() { _ = d.Close() }()

	names, err := d.Readdirnames(-1)
	if err != nil {
		t.Skipf("this system does not list open descriptors: %v", err)
	}

	return len(names)
}

// TestAToolCallGivesTheRecordBack.
//
// Every tool call opens the state root, and opening it opens the record: one
// SQLite connection, which is the database file, its write-ahead log and its
// shared-memory index — three descriptors a call. A server that never gives
// them back climbs until the machine's own file table is full, and then
// nothing on it can open a file: not the next tool call, not the window's
// run markers, not git.
func TestAToolCallGivesTheRecordBack(t *testing.T) {
	s, work := newRoot(t)
	payments := gitRepo(t, work, "payments")
	addTask(t, s, payments, "ACME-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	sn := Session{Root: work, Version: "test"}

	// One call first, so that what a process opens once is already open and
	// the count below is only what the calls after it added.
	if got := sn.Call("orbit_get_board_summary", nil); got.IsError {
		t.Fatalf("the first call failed: %s", text(t, got))
	}

	before := openDescriptors(t)

	// Both doors: the tools that only fold the board, and the tools that
	// find a row and write to it, which carry the record further.
	const calls = 20
	for range calls {
		if got := sn.Call("orbit_get_board_summary", nil); got.IsError {
			t.Fatalf("a call failed: %s", text(t, got))
		}

		if got := sn.Call("orbit_add_note", map[string]any{"task_id": "ACME-1", "text": "still here"}); got.IsError {
			t.Fatalf("a note failed: %s", text(t, got))
		}
	}

	if after := openDescriptors(t); after > before+2 {
		t.Errorf("%d tool calls left %d descriptors open (%d before, %d after); "+
			"a call has to give the record back", calls, after-before, before, after)
	}
}
