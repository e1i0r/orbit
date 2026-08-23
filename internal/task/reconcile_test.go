package task

import (
	"os"
	"os/exec"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// deadPid returns the pid of a process that has certainly finished and been
// reaped, which is what a marker left behind by a run that was killed names.
func deadPid(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run a process that exits immediately: %v", err)
	}
	return cmd.ProcessState.Pid()
}

// openRun is a task whose log says a run is under way and stops there, which
// is what SIGKILL leaves behind: phase.started, and then nothing.
func openRun(t *testing.T, s *store.Store, tk Task) {
	t.Helper()
	for _, e := range []record.Event{
		{Kind: record.TaskStarted, Data: map[string]string{"flow": "task"}},
		{Kind: record.PhaseStarted, Phase: "implement", Data: map[string]string{"engine": "fake", "n": "1"}},
	} {
		if err := emit(s, tk, e); err != nil {
			t.Fatalf("emit %s: %v", e.Kind, err)
		}
	}
}

func markerExists(t *testing.T, s *store.Store, tk Task) bool {
	t.Helper()
	path, err := s.RunPath(tk.Repo.Path, tk.ID)
	if err != nil {
		t.Fatalf("RunPath: %v", err)
	}
	_, err = os.Stat(path)
	return err == nil
}

func TestReconcileClosesTheRecordOfARunWhoseProcessIsGone(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	openRun(t, s, tk)
	if _, err := mark(s, tk, deadPid(t)); err != nil {
		t.Fatalf("mark: %v", err)
	}

	wrote, err := Reconcile(s, tk)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if !wrote {
		t.Error("Reconcile reported nothing to do for a run whose process is gone")
	}
	wantKinds(t, eventsOf(t, s, tk),
		record.TaskCreated, record.TaskStarted, record.PhaseStarted, record.TaskAbandoned)
	if markerExists(t, s, tk) {
		t.Error("the stale marker is still there, so the next reader will come back to it")
	}
}

func TestReconcileLeavesARunThatIsStillGoingAlone(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	openRun(t, s, tk)
	// This test process is the liveliest pid there is.
	if _, err := mark(s, tk, os.Getpid()); err != nil {
		t.Fatalf("mark: %v", err)
	}

	wrote, err := Reconcile(s, tk)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if wrote {
		t.Error("Reconcile declared a running task abandoned")
	}
	wantKinds(t, eventsOf(t, s, tk),
		record.TaskCreated, record.TaskStarted, record.PhaseStarted)
	if !markerExists(t, s, tk) {
		t.Error("the marker of a live run was removed, so nothing knows who holds the task any more")
	}
}

func TestReconcileWritesNothingWhenTheLogAlreadyEnds(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	openRun(t, s, tk)
	if err := emit(s, tk, record.Event{Kind: record.TaskFailed, Text: "the model fell over"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if _, err := mark(s, tk, deadPid(t)); err != nil {
		t.Fatalf("mark: %v", err)
	}

	wrote, err := Reconcile(s, tk)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if wrote {
		t.Error("Reconcile appended task.abandoned to a log that already said how the run ended")
	}
	wantKinds(t, eventsOf(t, s, tk),
		record.TaskCreated, record.TaskStarted, record.PhaseStarted, record.TaskFailed)
	if markerExists(t, s, tk) {
		t.Error("a claim that outlived its process was left on disk")
	}
}

func TestReconcileWritesNothingWhenNobodyClaimedTheTask(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	openRun(t, s, tk)

	wrote, err := Reconcile(s, tk)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if wrote {
		t.Error("Reconcile closed a record on the strength of no evidence at all")
	}
	wantKinds(t, eventsOf(t, s, tk),
		record.TaskCreated, record.TaskStarted, record.PhaseStarted)
}

func TestAliveTellsTheThreeAnswersApart(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	pid, ok, err := Alive(s, tk)
	if err != nil || ok || pid != 0 {
		t.Errorf("with no marker Alive = (%d, %v, %v), want (0, false, nil)", pid, ok, err)
	}

	live := os.Getpid()
	if _, err := mark(s, tk, live); err != nil {
		t.Fatalf("mark: %v", err)
	}
	pid, ok, err = Alive(s, tk)
	if err != nil || !ok || pid != live {
		t.Errorf("with a live marker Alive = (%d, %v, %v), want (%d, true, nil)", pid, ok, err, live)
	}

	gone := deadPid(t)
	if _, err := mark(s, tk, gone); err != nil {
		t.Fatalf("mark: %v", err)
	}
	pid, ok, err = Alive(s, tk)
	if err != nil || ok || pid != gone {
		t.Errorf("with a stale marker Alive = (%d, %v, %v), want (%d, false, nil)", pid, ok, err, gone)
	}
}

func TestAliveRefusesAMarkerItCannotUnderstand(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	path, err := s.RunPath(tk.Repo.Path, tk.ID)
	if err != nil {
		t.Fatalf("RunPath: %v", err)
	}
	// Zero is the number kill(2) reads as "every process in my group", so a
	// marker carrying it must never reach a signal.
	if err := os.WriteFile(path, []byte("pid: 0\nstarted: whenever\n"), 0o600); err != nil {
		t.Fatalf("write a damaged marker: %v", err)
	}

	if _, _, err := Alive(s, tk); err == nil {
		t.Error("Alive accepted a marker naming pid 0")
	}
	if err := Cancel(s, tk); err == nil {
		t.Error("Cancel was willing to signal on the strength of a damaged marker")
	}
}
