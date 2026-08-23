package task

import (
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
