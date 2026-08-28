package task

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

func TestMarkReadWritesDownThatSomebodyLooked(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	if err := MarkRead(s, tk); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}

	events := eventsOf(t, s, tk)

	got := find(t, events, record.TaskRead)
	if got.Phase != "" {
		t.Errorf("task.read names phase %q — being read is a fact about the task, not about a phase", got.Phase)
	}
}

func TestStartRefusesAtTheUnreadCapAndSaysWhy(t *testing.T) {
	s, r := fixture(t)

	tk := written(t, s, r)
	if err := s.SaveSettings(store.Settings{UnreadCap: 3}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	pid, err := Start(s, tk, "task", 3)
	if err == nil {
		t.Fatal("Start spawned a run at the unread cap — the one brake in the product did nothing")
	}

	if pid != 0 {
		t.Errorf("Start returned pid %d alongside its refusal", pid)
	}
	// The numbers, because a brake that says only "no" is a brake people
	// route around rather than clear.
	if !strings.Contains(err.Error(), "3") {
		t.Errorf("the refusal does not say how many are unread or what the cap is: %v", err)
	}
}

func TestTheUnreadCap(t *testing.T) {
	for _, tc := range []struct {
		name   string
		unread int
		limit  int
		want   bool
	}{
		// Zero is the setting a user chooses, and it means no cap at all —
		// which is why store.Settings fills in 5 for a file that was never
		// written rather than leaving the Go zero to speak for it.
		{"no cap at all", 99, 0, false},
		{"below the cap", 2, 5, false},
		{"at the cap", 5, 5, true},
		{"past the cap", 6, 5, true},
		{"nothing unread", 0, 5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := atCap(tc.unread, tc.limit); got != tc.want {
				t.Errorf("atCap(%d, %d) = %v, want %v", tc.unread, tc.limit, got, tc.want)
			}
		})
	}
}

// TestStartingATaskAnotherRunHoldsIsRefused is the regression, and what makes
// it one is how quiet the old answer was.
//
// Run refuses a task that is already claimed and writes nothing about it —
// deliberately, since the log already describes the run that is happening —
// and Start wires the child's stderr to nothing. So the spawn succeeded, the
// pid came back with a nil error, the child died a moment later, and the only
// trace of any of it was a window that said the task was running again.
func TestStartingATaskAnotherRunHoldsIsRefused(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	// This test process is the live claim: Alive asks the kernel about the
	// pid, and there is no more certainly running process to name.
	release, err := mark(s, tk, os.Getpid())
	if err != nil {
		t.Fatalf("mark: %v", err)
	}
	defer release()

	pid, err := Start(s, tk, "task", 0)
	if err == nil {
		t.Fatal("Start spawned a second run of a task that is already running")
	}

	if pid != 0 {
		t.Errorf("Start returned pid %d alongside its refusal", pid)
	}
	// The same sentence hold would have given, so that a person is told the
	// same thing whichever door refused them, and it names the process so
	// they can go and look at it.
	if got := err.Error(); !strings.Contains(got, "already being run") || !strings.Contains(got, strconv.Itoa(os.Getpid())) {
		t.Errorf("the refusal reads %q, want hold's own words and the pid holding the task", got)
	}
}
