package task

import (
	"context"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// TestPipelineThoughtsAndRefusalsEmitted tests that thoughts and tool refusals are properly recorded.
func TestPipelineThoughtsAndRefusalsEmitted(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "COV-1", "Emit thoughts and refusals", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	richEngine := &richResultEngine{
		res: engine.Result{
			Output:    "Execution completed",
			Thoughts:  []string{"Decided to use index on users table", "Avoided table lock"},
			Refusals:  []engine.StreamRefusal{{Tool: "psql", Input: "DROP DATABASE test"}},
			Cost:      0.045,
			SessionID: "sess-123",
		},
	}

	testFlow := flow.Flow{
		Name: "rich-flow",
		Phases: []flow.Phase{
			{Name: "execute", Engine: "rich"},
		},
	}

	if err := Run(context.Background(), s, tk, testFlow, map[string]engine.Engine{"rich": richEngine}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	var thoughtCount, refusalCount int
	for _, e := range events {
		if e.Kind == record.PhaseThought {
			thoughtCount++
		}
		if e.Kind == record.PhaseRefused {
			refusalCount++
			if e.Data["tool"] != "psql" {
				t.Errorf("refusal tool = %q, want psql", e.Data["tool"])
			}
		}
	}

	if thoughtCount != 2 {
		t.Errorf("expected 2 thought events, got %d", thoughtCount)
	}
	if refusalCount != 1 {
		t.Errorf("expected 1 refusal event, got %d", refusalCount)
	}
}

// TestPipelineGatesPassingAndFailing tests runGates with real shell commands.
func TestPipelineGatesPassingAndFailing(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "COV-2", "Gates test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("passed phase")
	engines := map[string]engine.Engine{"fake": fake}

	// 1. Flow with passing gate
	passFlow := flow.Flow{
		Name: "pass-gate-flow",
		Phases: []flow.Phase{
			{
				Name:   "phase1",
				Engine: "fake",
				Gates:  []flow.Gate{{Name: "lint", Command: "true"}},
			},
		},
	}

	if err := Run(context.Background(), s, tk, passFlow, engines, nil); err != nil {
		t.Fatalf("Run pass flow: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	var gatePassedFound bool
	for _, e := range events {
		if e.Kind == record.GatePassed && e.Data["gate"] == "lint" {
			gatePassedFound = true
			break
		}
	}
	if !gatePassedFound {
		t.Error("expected GatePassed event for lint gate")
	}

	// 2. Flow with failing gate
	tk2, err := Create(s, r, "COV-2B", "Failing gate test", "")
	if err != nil {
		t.Fatalf("Create tk2: %v", err)
	}
	failFlow := flow.Flow{
		Name: "fail-gate-flow",
		Phases: []flow.Phase{
			{
				Name:   "phase-fail",
				Engine: "fake",
				Gates:  []flow.Gate{{Name: "typecheck", Command: "false"}},
			},
		},
	}
	err = Run(context.Background(), s, tk2, failFlow, engines, nil)
	if err == nil {
		t.Fatal("expected Run to fail on failing gate command")
	}

	events2, err := Events(s, tk2)
	if err != nil {
		t.Fatalf("Events tk2: %v", err)
	}
	var gateFailedFound bool
	for _, e := range events2 {
		if e.Kind == record.GateFailed && e.Data["gate"] == "typecheck" {
			gateFailedFound = true
			break
		}
	}
	if !gateFailedFound {
		t.Error("expected GateFailed event for typecheck gate")
	}
}

// TestStartCapAndCommand asserts runCommand properties and start unread cap.
func TestStartCapAndCommand(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "COV-3", "Start test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Configure cap to 2
	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	cfg.UnreadCap = 2
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	// Start with unread=3 should fail due to cap
	_, err = Start(s, tk, "task", 3)
	if err == nil {
		t.Error("expected Start to fail when unread >= cap")
	}

	// Test runCommand structure
	cmd := runCommand("/bin/orbit", s.Root(), tk, "quick")
	if cmd.Dir != tk.Repo.Path {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, tk.Repo.Path)
	}
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Error("expected cmd.SysProcAttr.Setpgid to be true")
	}
}

// TestAliveMarkersAndBoot asserts marker reading, pid parsing, and stale detection.
func TestAliveMarkersAndBoot(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "COV-4", "Alive markers test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	release, err := mark(s, tk, 99999999)
	if err != nil {
		t.Fatalf("mark: %v", err)
	}
	defer release()

	pid, _, found, err := readMarker(s, tk)
	if err != nil || !found {
		t.Fatalf("readMarker: found=%v, err=%v", found, err)
	}
	if pid != 99999999 {
		t.Errorf("pid = %d, want 99999999", pid)
	}

	// Parsing tests
	if _, err := parsePid("not-a-pid"); err == nil {
		t.Error("expected parsePid to fail on non-integer")
	}
	if started := parseStarted("started: not-a-date"); !started.IsZero() {
		t.Error("expected parseStarted to return zero time on invalid date")
	}

	// Stale across boot for zero time
	if staleAcrossBoot(time.Time{}) {
		t.Error("expected staleAcrossBoot(zero) to be false")
	}
}

type richResultEngine struct {
	res engine.Result
}

func (e *richResultEngine) Name() string { return "rich" }

func (e *richResultEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	return e.res, nil
}

func (e *richResultEngine) Models() []engine.Choice  { return nil }
func (e *richResultEngine) Efforts() []engine.Choice { return nil }
func (e *richResultEngine) CanThink() bool           { return true }
func (e *richResultEngine) CanResume() bool          { return false }
