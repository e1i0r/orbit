package mcp

// The count of open descriptors, in the log, every so many calls.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
)

// TestTheDescriptorCountIsWrittenDownEverySoManyCalls.
//
// This server answers for as long as the client that spawned it runs, and
// the leak it had gave no other sign: every call succeeded, and the count of
// what they held open was the only number going anywhere. It is in the log
// now, where somebody reading a slow afternoon can see it climb.
func TestTheDescriptorCountIsWrittenDownEverySoManyCalls(t *testing.T) {
	if logger.OpenFiles() < 0 {
		t.Skip("this system does not list a process's own descriptors")
	}

	logs := t.TempDir()
	logPath := filepath.Join(logs, "orbit.log")

	if err := logger.Init(logPath, filepath.Join(logs, "errors.log")); err != nil {
		t.Fatalf("init the log: %v", err)
	}

	defer func() {
		if err := logger.CloseGlobal(); err != nil {
			t.Errorf("close the log: %v", err)
		}
	}()

	old := callsPerCensus
	callsPerCensus = 3

	defer func() { callsPerCensus = old }()

	s, work := newRoot(t)
	payments := gitRepo(t, work, "payments")
	addTask(t, s, payments, "ACME-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	sn := Session{Root: work, Version: "test"}
	for range callsPerCensus {
		if got := sn.Call("orbit_get_board_summary", nil); got.IsError {
			t.Fatalf("a call failed: %s", text(t, got))
		}
	}

	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read the log: %v", err)
	}

	if !strings.Contains(string(body), "descriptors open") {
		t.Errorf("three calls with a census every three said nothing about what they hold open; "+
			"the log holds:\n%s", body)
	}
}
