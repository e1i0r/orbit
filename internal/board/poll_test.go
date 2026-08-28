package board

// What the poll does with a read that failed, which is the only thing it
// does that is not a stat.

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAFailedReadIsRetriedOnceAndThenLetGoOf is the bookkeeping the fix
// turns on, and both halves of it matter.
//
// The size used to be committed before the read was even attempted, so a
// failure was never retried: the bytes went on the ledger as read and the
// next stat found nothing new to do. Committing it only on success is the
// other mistake — a line longer than record.MaxLine is a log no later read
// will get past, and four megabytes of pointless I/O twice a second, for as
// long as the window is open, is what "just retry" costs.
func TestAFailedReadIsRetriedOnceAndThenLetGoOf(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))
	path := eventsPath(t, s, repoPath, "ACME-1")
	r := NewReader(s, work)
	st := &taskState{repo: &repoState{name: "payments", path: repoPath}, id: "ACME-1", path: path}

	if _, err := r.poll(st); err != nil {
		t.Fatalf("the first poll of a readable log: %v", err)
	}
	read := st.size
	if read == 0 {
		t.Fatal("the first poll accounted for no bytes at all")
	}

	tooLongLine(t, path)
	if _, err := r.poll(st); err == nil {
		t.Fatal("a line over record.MaxLine was read without complaint")
	}
	if st.size != read {
		t.Errorf("size = %d, want it left at %d: bytes nobody read are not bytes accounted for", st.size, read)
	}
	if !st.retried {
		t.Error("the failure was not marked for its retry, so it will never get one")
	}

	if _, err := r.poll(st); err == nil {
		t.Fatal("the retry of an unreadable log passed")
	}
	if st.size != sizeOf(t, path) {
		t.Errorf("size = %d, want the whole %d: a log no read will get past must be let go of", st.size, sizeOf(t, path))
	}
	if st.retried {
		t.Error("the retry is still owed after it was taken")
	}

	// And now the log is skipped on the stat, which is what letting go of
	// it was for: the verdict the caller kept comes back rather than a
	// fourth megabyte of reading.
	kept := errors.New("the verdict from the read that failed")
	st.err = kept
	events, err := r.poll(st)
	if events != nil || !errors.Is(err, kept) {
		t.Errorf("poll = %v, %v — want the kept verdict and no re-read", events, err)
	}
}

// TestAnEndingIsNotLostToAReadThatFailedOnce is the same bug from the row's
// side. A fault that lasted a moment — a descriptor Orbit could not get, a
// filesystem that blinked — used to strand whatever had been appended in the
// meantime until something else was written. When that write was the last
// one of a run, nothing else ever was: the row went on saying the task was
// running over a log that says it finished, beside an error blaming a fault
// that was already over.
func TestAnEndingIsNotLostToAReadThatFailedOnce(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	r := NewReader(s, work)
	if b, _ := refresh(t, r); view.BandOf(b.Tasks[0]) != view.Running {
		t.Fatalf("the first refresh put the task in %v, want it running", view.BandOf(b.Tasks[0]))
	}

	path := eventsPath(t, s, repoPath, "ACME-1")
	appendTo(t, s, repoPath, "ACME-1", finishedEvent())
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	b, _ := refresh(t, r)
	if len(b.Errs) == 0 {
		t.Fatal("a log that could not be opened was read without complaint")
	}
	if !strings.Contains(b.Errs[0].Error(), "ACME-1") {
		t.Errorf("the failure says %q, want it to name the task", b.Errs[0])
	}

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod back: %v", err)
	}
	b, _ = refresh(t, r)
	if got := view.BandOf(b.Tasks[0]); got != view.Done {
		t.Errorf("the task is in %v, want it done: its ending was written before the read that failed", got)
	}
	if len(b.Errs) != 0 {
		t.Errorf("the board still carries %v, want the fault gone with the fault", b.Errs)
	}
}
