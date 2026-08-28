package task

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// TestKillAndCancelEdgeCases tests Kill, Cancel and killTarget.
func TestKillAndCancelEdgeCases(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "KILL-1", "Kill test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// 1. Kill a task that is not running
	if err := Kill(s, tk); err == nil {
		t.Error("expected Kill on non-running task to report not running")
	}

	// 2. Plant marker with non-existent PID and test Cancel
	release, err := mark(s, tk, 99999999)
	if err != nil {
		t.Fatalf("mark: %v", err)
	}
	defer release()

	if err := Cancel(s, tk); err == nil {
		t.Error("expected Cancel on dead pid to return error")
	}

	// 3. killTarget helper
	if got := killTarget(100, 100, nil); got != -100 {
		t.Errorf("killTarget(100, 100, nil) = %d, want -100", got)
	}

	if got := killTarget(100, 50, nil); got != 100 {
		t.Errorf("killTarget(100, 50, nil) = %d, want 100", got)
	}

	if got := killTarget(100, 1, nil); got != 100 {
		t.Errorf("killTarget(100, 1, nil) = %d, want 100", got)
	}

	if got := killTarget(100, 100, errors.New("err")); got != 100 {
		t.Errorf("killTarget with error = %d, want 100", got)
	}
}

// TestControlAndTakeCommands tests writing and taking control words.
func TestControlAndTakeCommands(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "CTRL-1", "Control test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Write control word
	if err := Control(s, tk, "pause"); err != nil {
		t.Fatalf("Control pause: %v", err)
	}

	// Take control word
	word, err := take(s, tk)
	if err != nil {
		t.Fatalf("take: %v", err)
	}

	if word != "pause" {
		t.Errorf("take word = %q, want pause", word)
	}

	// Second take should be empty
	second, err := take(s, tk)
	if err != nil {
		t.Fatalf("second take: %v", err)
	}

	if second != "" {
		t.Errorf("second take word = %q, want empty", second)
	}
}

// TestReconcileAbandonedTasks tests reconciling dead processes and clearing markers.
func TestReconcileAbandonedTasks(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REC-1", "Reconcile test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Plant dead pid marker
	release, err := mark(s, tk, 99999999)
	if err != nil {
		t.Fatalf("mark: %v", err)
	}
	defer release()

	// Emit task started event so it appears in flight
	if err := emit(s, tk, record.Event{Kind: record.TaskStarted}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	reconciled, err := Reconcile(s, tk)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	if !reconciled {
		t.Error("expected Reconcile to return true for in-flight dead task")
	}

	// Marker should be removed by Reconcile
	_, _, found, err := readMarker(s, tk)
	if err != nil {
		t.Fatalf("readMarker: %v", err)
	}

	if found {
		t.Error("expected marker to be removed after Reconcile")
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	var abandonedFound bool

	for _, e := range events {
		if e.Kind == record.TaskAbandoned {
			abandonedFound = true
			break
		}
	}

	if !abandonedFound {
		t.Error("expected TaskAbandoned event emitted by Reconcile")
	}
}

// TestTaskListAndFlowDefaults tests List, Load and flow resolution logic.
func TestTaskListAndFlowDefaults(t *testing.T) {
	s, r := fixture(t)

	// Create multiple tasks
	tk1, err := Create(s, r, "LST-1", "Task one", "quick")
	if err != nil {
		t.Fatalf("Create tk1: %v", err)
	}

	tk2, err := Create(s, r, "LST-2", "Task two", "")
	if err != nil {
		t.Fatalf("Create tk2: %v", err)
	}

	tasks, err := List(s, r)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("List returned %d tasks, want 2", len(tasks))
	}

	// Load individual task
	loaded, err := Load(s, r, tk1.ID)
	if err != nil {
		t.Fatalf("Load tk1: %v", err)
	}

	if loaded.Flow != "quick" {
		t.Errorf("loaded.Flow = %q, want quick", loaded.Flow)
	}

	// Default flow when none specified
	if tk2.Flow != flow.Default {
		t.Errorf("tk2.Flow = %q, want %q", tk2.Flow, flow.Default)
	}

	// chosenFlow helper
	chosen, err := chosenFlow(s, "custom")
	if err != nil || chosen != "custom" {
		t.Errorf("chosenFlow(custom) = %q, want custom", chosen)
	}

	chosenDef, err := chosenFlow(s, "")
	if err != nil || chosenDef != flow.Default {
		t.Errorf("chosenFlow(\"\") = %q, want %q", chosenDef, flow.Default)
	}

	// Non-existent task dir load error
	_, err = Load(s, r, "DOES-NOT-EXIST")
	if err == nil {
		t.Error("expected Load on non-existent task to error")
	}
}

// TestEventsReadingAndCorruptedLines tests event retrieval and corrupted line toleration.
func TestEventsReadingAndCorruptedLines(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "EVT-1", "Events test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Append valid and invalid JSON lines to events.jsonl
	eventsPath, err := s.EventsPath(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	corruptedData := "{\"at\":\"2026-08-24T12:00:00Z\",\"kind\":\"task.started\"}\nnot valid json\n{\"at\":\"2026-08-24T12:01:00Z\",\"kind\":\"task.finished\"}\n"
	if err := os.WriteFile(eventsPath, []byte(corruptedData), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events with unreadable line: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("expected 3 events (including unreadable marker), got %d: %v", len(events), events)
	}
}

// TestRunPrepareAndErrorReturns tests Run execution when engine returns error.
func TestRunPrepareAndErrorReturns(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "RUN-ERR-1", "Run error test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	testFlow := flow.Flow{
		Name: "fail-flow",
		Phases: []flow.Phase{
			{Name: "crash-phase", Engine: "fail"},
		},
	}

	// Missing engine in engines map returns error
	err = Run(context.Background(), s, tk, testFlow, map[string]engine.Engine{}, nil)
	if err == nil {
		t.Error("expected Run to fail when engine is missing from map")
	}

	// Engine returning error
	errEngineFail := &failingEngine{err: os.ErrPermission}

	err = Run(context.Background(), s, tk, testFlow, map[string]engine.Engine{"fail": errEngineFail}, nil)
	if err == nil {
		t.Error("expected Run to fail when engine returns error")
	}
}

type failingEngine struct {
	err error
}

func (e *failingEngine) Name() string { return "fail" }
func (e *failingEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	return engine.Result{}, e.err
}
func (e *failingEngine) Models() []engine.Choice  { return nil }
func (e *failingEngine) Efforts() []engine.Choice { return nil }
func (e *failingEngine) CanThink() bool           { return true }
func (e *failingEngine) CanResume() bool          { return false }
