package task

// Waiting for a run to be gone, which is the half of Reopen that had no test
// at all — a function that stood still for thirty seconds to reach its own
// deadline is a function nobody writes a test for.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// TestAWaitForATaskNobodyHoldsIsOverAtOnce.
func TestAWaitForATaskNobodyHoldsIsOverAtOnce(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "AWAIT-1", "await test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	began := time.Now()

	if err := awaitStopped(context.Background(), s, tk, time.Minute, time.Second); err != nil {
		t.Fatalf("awaitStopped on a task nothing claims: %v", err)
	}

	if took := time.Since(began); took > time.Second {
		t.Errorf("waiting on a task nobody holds took %s, and there was nothing to wait for", took)
	}
}

// TestARunThatWillNotStopIsReportedAndNotRestartedOver names the verb that
// ends it, because the caller's next move is to end it.
func TestARunThatWillNotStopIsReportedAndNotRestartedOver(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "AWAIT-2", "await test", "quick")
	if err != nil {
		t.Fatal(err)
	}
	// This process is alive by definition, so the marker never goes stale
	// and the wait has to reach its deadline.
	release, err := mark(s, tk, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	defer release()

	err = awaitStopped(context.Background(), s, tk, 30*time.Millisecond, time.Millisecond)
	if err == nil {
		t.Fatal("a run that never stopped was waited on and reported as gone")
	}

	for _, want := range []string{tk.ID, "still running", "orbit cancel -now"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal is %q, and it does not say %q", err, want)
		}
	}
}

// TestAWaitEndsWhenTheCallerStopsWaiting. Half a minute is a long time to
// hold a caller who has already given up, and which of the two happened has
// to survive into what they print.
func TestAWaitEndsWhenTheCallerStopsWaiting(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "AWAIT-3", "await test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	release, err := mark(s, tk, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	defer release()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	began := time.Now()

	err = awaitStopped(ctx, s, tk, time.Hour, 5*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("awaitStopped = %v, want the context's own error", err)
	}

	if took := time.Since(began); took > 5*time.Second {
		t.Errorf("the wait took %s after the caller gave up", took)
	}
}

// TestAWaitOnAMarkerItCannotReadStops. A claim that cannot be read is a
// claim that cannot be ruled out, which is the stance readMarker takes and
// the reason it gives holds here too: starting a second run over one is the
// mistake Reopen's wait exists to prevent.
func TestAWaitOnAMarkerItCannotReadStops(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "AWAIT-4", "await test", "quick")
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

	if err := awaitStopped(context.Background(), s, tk, time.Minute, time.Millisecond); err == nil {
		t.Error("a marker that will not parse was waited on and reported as gone")
	}
}
