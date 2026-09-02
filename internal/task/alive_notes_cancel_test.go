package task

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

func TestAliveMarkerAndNotes(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	r := repo.Repo{Path: filepath.Join(t.TempDir(), "repo"), Name: "repo"}

	tk, err := Create(s, r, "TASK-ALIVE", "Testing alive marker", "quick")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Alive on fresh task (no marker)
	pid, ok, err := Alive(s, tk)
	if err != nil || ok || pid != 0 {
		t.Errorf("Alive on fresh task = (%d, %v, %v), want (0, false, nil)", pid, ok, err)
	}

	// 2. Plant marker with current process PID
	release, err := mark(s, tk, os.Getpid())
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	pid, ok, err = Alive(s, tk)
	if err != nil || !ok || pid != os.Getpid() {
		t.Errorf("Alive with active marker = (%d, %v, %v), want (%d, true, nil)", pid, ok, err, os.Getpid())
	}

	release()

	// 3. removeMarker when marker is absent and present
	if err := removeMarker(s, tk); err != nil {
		t.Errorf("removeMarker on absent marker failed: %v", err)
	}

	_, _ = mark(s, tk, os.Getpid()) //nolint:errcheck
	if err := removeMarker(s, tk); err != nil {
		t.Errorf("removeMarker on present marker failed: %v", err)
	}

	// 4. Stale across boot check with ancient timestamp
	ancient := time.Now().Add(-3000 * time.Hour)
	if !staleAcrossBoot(ancient) {
		// on darwin, if bootTime is available, ancient time should report stale
		if _, hasBoot := bootTime(); hasBoot {
			t.Error("expected staleAcrossBoot to be true for ancient time")
		}
	}

	if staleAcrossBoot(time.Time{}) {
		t.Error("expected staleAcrossBoot to be false for zero time")
	}

	// 5. Note appending and reading
	if err := Note(s, tk, "First operator note"); err != nil {
		t.Fatalf("Note failed: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events failed: %v", err)
	}

	if len(events) < 2 { // TaskCreated + OperatorNote
		t.Errorf("expected at least 2 events, got %d", len(events))
	}

	var foundNote bool

	for _, e := range events {
		if e.Kind == record.TaskNoted && e.Text == "First operator note" {
			foundNote = true
			break
		}
	}

	if !foundNote {
		t.Error("operator note event not found in Events()")
	}

	// 6. running helper
	if !running(os.Getpid()) {
		t.Errorf("running(%d) = false, want true", os.Getpid())
	}

	if running(9999999) {
		t.Error("running(9999999) = true, want false")
	}
}

func TestPhaseHelpersAndRunGates(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	r := repo.Repo{Path: filepath.Join(t.TempDir(), "repo"), Name: "repo"}

	tk, err := Create(s, r, "TASK-PHASE", "Testing phase events", "quick")
	if err != nil {
		t.Fatal(err)
	}

	// 1. phaseStart with full options
	pFull := flow.Phase{
		Name:        "build",
		Engine:      "claude",
		Model:       "sonnet",
		Effort:      "high",
		Thinking:    "enabled",
		Permissions: []string{"bash", "write"},
	}

	evStart := phaseStart(pFull, 2, []string{"note1", "note2"})
	if evStart.Data["model"] != "sonnet" || evStart.Data["effort"] != "high" || evStart.Data["notes"] != "2" {
		t.Errorf("unexpected phaseStart data: %+v", evStart.Data)
	}

	// 2. phaseEnd with session, cost, and error
	res := engine.Result{
		Output:    "Engine execution output",
		SessionID: "sess-1234",
		Cost:      0.045,
	}

	evEnd := phaseEnd(record.PhaseFailed, "build", res, errors.New("timeout reached"))
	if evEnd.Data["session"] != "sess-1234" || evEnd.Data["error"] != "timeout reached" {
		t.Errorf("unexpected phaseEnd data: %+v", evEnd.Data)
	}

	// 3. phaseThought & phaseRefused
	evTh := phaseThought("plan", 1, "Detailed thought trace")
	if evTh.Kind != record.PhaseThought || evTh.Text != "Detailed thought trace" {
		t.Errorf("unexpected phaseThought event: %+v", evTh)
	}

	evRef := phaseRefused("plan", 1, engine.StreamRefusal{Tool: "rm", Input: "-rf /"})
	if evRef.Kind != record.PhaseRefused || evRef.Data["tool"] != "rm" {
		t.Errorf("unexpected phaseRefused event: %+v", evRef)
	}

	// 4. runGates passing and failing
	pGates := flow.Phase{
		Name: "test",
		Gates: []flow.Gate{
			{Name: "pass-gate", Command: "echo ok"},
			{Name: "fail-gate", Command: "exit 42"},
		},
	}
	wtDir := t.TempDir()

	refused, err := runGates(context.Background(), s, tk, pGates, 1, wtDir, res)
	if err != nil {
		t.Errorf("runGates: %v", err)
	}

	if refused == nil || refused.Gate != "fail-gate" || refused.Exit != 42 {
		t.Errorf("expected fail-gate to refuse with exit 42, got %+v", refused)
	}

	// 5. Note validation and unconsumedNotes reset
	if err := Note(s, tk, "   "); err == nil {
		t.Error("expected error on whitespace note")
	}

	if err := Note(s, tk, "Note before phase"); err != nil {
		t.Fatal(err)
	}

	notesBefore, err := unconsumedNotes(s, tk)
	if err != nil {
		t.Fatal(err)
	}

	if len(notesBefore) == 0 {
		t.Error("expected unconsumed notes before phase start")
	}
	// Simulate phase start
	_ = emit(s, tk, record.Event{Kind: record.PhaseStarted, Phase: "build"}) //nolint:errcheck

	notesAfter, err := unconsumedNotes(s, tk)
	if err != nil {
		t.Fatal(err)
	}

	if len(notesAfter) != 0 {
		t.Errorf("expected 0 notes after phase started, got %d", len(notesAfter))
	}

	// 6. List on empty repo and repo with non-dir files
	freshRepo := repo.Repo{Path: filepath.Join(t.TempDir(), "fresh"), Name: "fresh"}

	ids, err := List(s, freshRepo)
	if err != nil || len(ids) != 0 {
		t.Errorf("List on fresh repo = (%v, %v), want (nil, nil)", ids, err)
	}
}

func TestCancelAndKillProcessExecution(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	r := repo.Repo{Path: filepath.Join(t.TempDir(), "repo"), Name: "repo"}

	tk, err := Create(s, r, "TASK-KILL", "Testing kill and cancel", "quick")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Cancel on task with dead PID in marker
	if _, err := mark(s, tk, 9999999); err != nil {
		t.Fatal(err)
	}

	err = Cancel(s, tk)
	if err == nil || !strings.Contains(err.Error(), "which held it, is gone") {
		t.Errorf("expected gone process error on Cancel, got %v", err)
	}

	// 2. Kill on task with dead PID in marker
	err = Kill(s, tk)
	if err == nil || !strings.Contains(err.Error(), "which held it, is gone") {
		t.Errorf("expected gone process error on Kill, got %v", err)
	}

	// 3. Kill on running child process
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }() //nolint:errcheck

	if _, err := mark(s, tk, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	if err := Kill(s, tk); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
}
