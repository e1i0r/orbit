package board

// Live is the one field on a row the log cannot answer. These tests are
// about where it does come from: the run marker beside the log, and the
// process that marker names.

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// claim writes a run marker naming a pid, the way `orbit run` does while it
// holds a task.
func claim(t *testing.T, s *store.Store, repoPath, id string, pid int) {
	t.Helper()

	body := "pid: " + strconv.Itoa(pid) + "\nstarted: " + time.Now().UTC().Format(time.RFC3339) + "\n"
	writeMarker(t, s, repoPath, id, body)
}

func writeMarker(t *testing.T, s *store.Store, repoPath, id, body string) {
	t.Helper()

	path, err := s.RunPath(repoPath, id)
	if err != nil {
		t.Fatalf("run path of task %s: %v", id, err)
	}

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write the marker of task %s: %v", id, err)
	}
}

// gonePid is a pid that has certainly finished and been reaped: what a run
// killed outright leaves behind in its marker.
func gonePid(t *testing.T) int {
	t.Helper()

	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run a process that exits immediately: %v", err)
	}

	return cmd.ProcessState.Pid()
}

func oneRow(t *testing.T, b Board) view.Task {
	t.Helper()

	if len(b.Tasks) != 1 {
		t.Fatalf("%d tasks on the board, want 1", len(b.Tasks))
	}

	return b.Tasks[0]
}

func TestARowIsLiveOnlyWhileAProcessHoldsIt(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("retry the webhook"), startedEvent())
	r := NewReader(s, work)

	if got := oneRow(t, first(t, r)).Live; got != view.LiveFree {
		t.Errorf("a task no marker claims is %v, want free", got)
	}

	claim(t, s, repoPath, "ACME-1", os.Getpid())

	if got := oneRow(t, next(t, r)).Live; got != view.LiveHeld {
		t.Errorf("a task whose process is running is %v, want held — its log did not move, and a marker appearing is not an event, so this has to be asked on every refresh", got)
	}

	claim(t, s, repoPath, "ACME-1", gonePid(t))

	if got := oneRow(t, next(t, r)).Live; got != view.LiveFree {
		t.Errorf("a task whose process is gone is %v, want free — a process dying changes no file, so a poll that only watches the log will never notice", got)
	}
}

// first and next name what a refresh is for, so the reads above stay one
// line each.
func first(t *testing.T, r *Reader) Board { t.Helper(); b, _ := refresh(t, r); return b }
func next(t *testing.T, r *Reader) Board  { t.Helper(); b, _ := refresh(t, r); return b }

func TestALiveRunIsStillBandedByTheRecord(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	// A log that ends where SIGKILL leaves one, and a marker whose process
	// is gone. The row is not live — and it is still Running, because
	// task.abandoned is not in the log and no reader may draw a conclusion
	// the record does not carry. Closing it is task.Reconcile's job.
	addTask(t, s, repoPath, "ACME-1", created("retry the webhook"), startedEvent())
	claim(t, s, repoPath, "ACME-1", gonePid(t))

	row := oneRow(t, first(t, NewReader(s, work)))
	if row.Live != view.LiveFree {
		t.Errorf("a task whose process is gone is %v, want free", row.Live)
	}

	if row.Band != view.Running {
		t.Errorf("band = %v, want Running — a board that banded on liveness would be the only reader of the record that knew", row.Band)
	}
}

// TestADamagedMarkerIsUnknownAndNotFree. A marker that is there and will not
// parse is the third answer, and it used to be folded into the second: the
// row said free, and free is what the window checks before it starts a task.
// So the one task nobody could say anything about was the one task orbit
// would start a second engine on.
//
// It is not held either. Held offers a stop, and stopping reads this same
// unreadable marker, which would leave the task with no way out of the state
// but deleting a file by hand.
func TestADamagedMarkerIsUnknownAndNotFree(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("retry the webhook"), startedEvent())
	writeMarker(t, s, repoPath, "ACME-1", "pid: not a number\n")

	b := first(t, NewReader(s, work))

	row := oneRow(t, b)
	if row.Live != view.LiveUnknown {
		t.Errorf("a marker nobody could read is %v, want unknown", row.Live)
	}

	if row.Band != view.Running {
		t.Errorf("band = %v, want Running — the row is drawn from its log, which is undamaged", row.Band)
	}

	if len(b.Errs) == 0 {
		t.Fatal("a damaged marker was passed over in silence")
	}

	if !strings.Contains(b.Errs[0].Error(), "ACME-1") {
		t.Errorf("the fault does not say which task it is about: %v", b.Errs[0])
	}
}
