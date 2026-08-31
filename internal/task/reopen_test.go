package task

// Directing a task and starting it again: the front door `orbit direct` and
// the cockpit both come through, and the half of it that runs only when the
// task is genuinely in flight.

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// child starts a process that will sit there until something signals it, and
// hands back its pid and a channel that closes when it is gone. Reaping it is
// the point of the goroutine: a child nobody waits on becomes a zombie, and a
// zombie answers kill(pid, 0) as though it were alive — which would make
// Alive say a stopped run is still running for the rest of the test.
func child(t *testing.T) (int, <-chan struct{}) {
	t.Helper()

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start a stand-in for a run: %v", err)
	}

	gone := make(chan struct{})

	go func() {
		_ = cmd.Wait() //nolint:errcheck // the exit status is not the question; being gone is

		close(gone)
	}()

	t.Cleanup(func() {
		_ = cmd.Process.Kill() //nolint:errcheck // already dead is the ordinary case

		<-gone
	})

	return cmd.Process.Pid, gone
}

// TestReopenStartsTheTaskAgainOnceNothingHoldsIt is the ordinary path, and it
// had no test: the whole of Reopen past its first line ran for the first time
// in production every time.
func TestReopenStartsTheTaskAgainOnceNothingHoldsIt(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REOPEN-1", "reopen test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	pid, err := Reopen(context.Background(), s, tk, "operator", "try the other approach", "task", 0)
	if err != nil {
		t.Fatalf("Reopen on a task nothing holds: %v", err)
	}

	if pid <= 0 {
		t.Errorf("Reopen returned pid %d and no error", pid)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatal(err)
	}

	var dialogue, noted bool

	for _, e := range events {
		switch {
		case e.Kind == "task.dialogue" && strings.Contains(e.Text, "try the other approach"):
			dialogue = true
		case e.Kind == "task.noted" && strings.Contains(e.Text, "try the other approach"):
			noted = true
		}
	}

	if !dialogue {
		t.Error("the directive was not written to the record as dialogue")
	}
	// The note is the copy the next phase's prompt is built from. A reopen
	// that starts a run without it starts the same run again.
	if !noted {
		t.Error("the directive left no note for the phase about to start")
	}
}

// TestReopenUsesTheFlowTheTaskCarriesWhenNoneIsNamed. Both callers pass
// t.Flow through, and one of them may pass nothing at all — an empty name
// would reach `orbit run -flow ""`, which is a run with no phases.
func TestReopenUsesTheFlowTheTaskCarriesWhenNoneIsNamed(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REOPEN-2", "reopen with no flow named", "quick")
	if err != nil {
		t.Fatal(err)
	}

	pid, err := Reopen(context.Background(), s, tk, "", "again please", "", 0)
	if err != nil {
		t.Fatalf("Reopen with no flow named: %v", err)
	}

	if pid <= 0 {
		t.Errorf("Reopen returned pid %d and no error", pid)
	}
}

// TestReopenAtTheUnreadCapIsRefusedAndSaysSo. The cap is the one brake in the
// product, and directing a task is the easiest way to walk around it: the
// directive is recorded either way, so a refusal that came back as nil would
// leave a task that looks directed and was never run.
func TestReopenAtTheUnreadCapIsRefusedAndSaysSo(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REOPEN-3", "reopen at the cap", "quick")
	if err != nil {
		t.Fatal(err)
	}

	pid, err := Reopen(context.Background(), s, tk, "operator", "again", "quick", 99)
	if err == nil {
		t.Fatal("Reopen started a run past the unread cap")
	}

	if pid != 0 {
		t.Errorf("Reopen returned pid %d alongside its refusal", pid)
	}

	if !strings.Contains(err.Error(), "was not started") {
		t.Errorf("the refusal does not read as the cap's: %v", err)
	}
}

// TestADirectiveOnARunningTaskStopsTheRunFirst. A note left under a phase
// that is still running is a note the phase has already read past; the run
// has to end before the next one can carry it.
func TestADirectiveOnARunningTaskStopsTheRunFirst(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIRECT-RUN-1", "direct a running task", "quick")
	if err != nil {
		t.Fatal(err)
	}

	pid, gone := child(t)

	release, err := mark(s, tk, pid)
	if err != nil {
		t.Fatal(err)
	}

	defer release()

	if err := Direct(s, tk, "operator", "stop, you are on the wrong file"); err != nil {
		t.Fatalf("Direct on a running task: %v", err)
	}

	select {
	case <-gone:
	case <-time.After(10 * time.Second):
		t.Fatal("the process holding the task was never asked to stop")
	}
}

// TestADirectiveOnAMarkerThatWillNotReadIsRefused. A claim that cannot be
// read is a claim that cannot be ruled out: answering "not running" here
// would leave the old run walking the worktree the next one is about to open.
func TestADirectiveOnAMarkerThatWillNotReadIsRefused(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIRECT-RUN-2", "direct over a broken marker", "quick")
	if err != nil {
		t.Fatal(err)
	}

	path, err := s.RunPath(tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("pid: not a number\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err = Direct(s, tk, "operator", "stop")
	if err == nil {
		t.Fatal("a marker that will not parse was directed over as though nothing held the task")
	}

	if !strings.Contains(err.Error(), "liveness") {
		t.Errorf("the failure does not say which question could not be answered: %v", err)
	}
}

// TestADirectiveThatCannotBeWrittenDownIsRefused. The directive's only effect
// is the two lines it leaves in the record. A Direct that returned nil having
// written neither would report a correction that was never delivered.
func TestADirectiveThatCannotBeWrittenDownIsRefused(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIRECT-RUN-3", "direct onto a read-only record", "quick")
	if err != nil {
		t.Fatal(err)
	}

	path, err := s.EventsPath(tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chmod(path, 0o600) }) //nolint:errcheck // best effort, the root is a temp dir

	if err := Direct(s, tk, "operator", "stop"); err == nil {
		t.Fatal("Direct answered nil over a record it could not write to")
	}
}
