package db

// Asking again for a turn at the lock.
//
// A retry nobody measures is a retry that can become one attempt, or none,
// and the failure it was written for — an event lost because another process
// held the lock across work it should not have — is silent either way.

import (
	"errors"
	"testing"
	"time"
)

// errBusy is what the driver says when somebody else holds the write lock.
// The text is what refused matches on, so it is written here as the driver
// writes it rather than as a sentinel this package made up.
var errBusy = errors.New("database is locked (SQLITE_BUSY)")

// TestAWriteRefusedForWantOfATurnAsksAgain. The caller has nowhere else to
// put the event, so a refusal that is merely reported is an event gone with
// nothing to say so.
func TestAWriteRefusedForWantOfATurnAsksAgain(t *testing.T) {
	tries := 0

	err := keepTrying(func() error {
		tries++
		if tries < 3 {
			return errBusy
		}

		return nil
	})
	if err != nil {
		t.Fatalf("a write that got its turn on the third ask failed: %v", err)
	}

	if tries != 3 {
		t.Errorf("it asked %d times, want the three it took", tries)
	}
}

// TestAskingStopsAfterTheLastTry. Four, backing off — and the number matters
// in both directions: one attempt fewer loses events under contention, and a
// loop with no end is a command that never returns.
func TestAskingStopsAfterTheLastTry(t *testing.T) {
	tries := 0

	started := time.Now()

	err := keepTrying(func() error {
		tries++

		return errBusy
	})

	if !errors.Is(err, errBusy) {
		t.Fatalf("a write refused every time answered %v, want the refusal itself", err)
	}

	// The first attempt plus one per retry.
	if tries != retries+1 {
		t.Errorf("it asked %d times, want %d", tries, retries+1)
	}

	// And it backed off between them rather than spinning: the lock is held
	// by somebody doing something, and asking again at once only adds a
	// wakeup to whatever they are doing.
	least := 250 * time.Millisecond * time.Duration(retries*(retries+1)/2)
	if waited := time.Since(started); waited < least {
		t.Errorf("it gave up after %s, want it to have backed off for at least %s", waited, least)
	}
}

// TestAFailureThatIsNotABusyLockIsNotAskedAgain. A constraint, a broken
// file, a column that is not there — all of them fail the same way the
// second time, and asking again spends a second and a half learning it.
func TestAFailureThatIsNotABusyLockIsNotAskedAgain(t *testing.T) {
	stuck := errors.New("no such column: worktree")

	tries := 0

	err := keepTrying(func() error {
		tries++

		return stuck
	})

	if !errors.Is(err, stuck) {
		t.Fatalf("it answered %v, want the failure itself", err)
	}

	if tries != 1 {
		t.Errorf("it asked %d times about a failure that will not change, want one", tries)
	}
}

// TestWhatCountsAsWantingATurn. The two sentences the driver uses, and
// nothing else: a reader that matched more widely would retry a broken file
// for a second and a half before saying so.
func TestWhatCountsAsWantingATurn(t *testing.T) {
	cases := []struct {
		said string
		want bool
	}{
		{"database is locked (5)", true},
		{"SQLITE_BUSY: database is locked", true},
		{"no such table: proposal", false},
		{"UNIQUE constraint failed: task.task_id", false},
		{"", false},
	}

	for _, c := range cases {
		if got := refused(errors.New(c.said)); got != c.want {
			t.Errorf("%q reads as wanting a turn: %v, want %v", c.said, got, c.want)
		}
	}
}
