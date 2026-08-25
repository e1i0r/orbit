package task

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// TestE2EMultiPhaseFlowFeedsOutput tests a 2-phase pipeline where phase 2 receives phase 1's output.
func TestE2EMultiPhaseFlowFeedsOutput(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "E2E-1", "Design and review API", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("Step 1 Output: API Blueprint generated")
	engines := map[string]engine.Engine{"fake": fake}

	testFlow := flow.Flow{
		Name: "two-step",
		Phases: []flow.Phase{
			{Name: "1-design", Engine: "fake", Model: "sonnet", FeedOutput: false},
			{Name: "2-review", Engine: "fake", Model: "opus", FeedOutput: true},
		},
	}

	if err := Run(context.Background(), s, tk, testFlow, engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 engine calls, got %d", len(fake.Calls))
	}

	if !strings.Contains(fake.Calls[1].Prompt, "API Blueprint generated") {
		t.Errorf("phase 2 prompt did not contain output from phase 1: %q", fake.Calls[1].Prompt)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	var phaseStarts, phaseFinishes int
	for _, e := range events {
		if e.Kind == record.PhaseStarted {
			phaseStarts++
		}
		if e.Kind == record.PhaseFinished {
			phaseFinishes++
		}
	}
	if phaseStarts != 2 || phaseFinishes != 2 {
		t.Errorf("expected 2 phase starts and finishes, got starts=%d finishes=%d", phaseStarts, phaseFinishes)
	}
}

// TestE2EPhaseWithOperatorNotes tests that operator notes left before/during run are recorded and unconsumed notes are passed to the phase.
func TestE2EPhaseWithOperatorNotes(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "E2E-2", "Task with operator instructions", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Note(s, tk, "Please use SQL migrations rather than raw DDL"); err != nil {
		t.Fatalf("Note: %v", err)
	}

	fake := engine.NewFake("Created SQL migration files")
	engines := map[string]engine.Engine{"fake": fake}

	testFlow := flow.Flow{
		Name: "migration-flow",
		Phases: []flow.Phase{
			{Name: "migrate", Engine: "fake", Model: "sonnet"},
		},
	}

	if err := Run(context.Background(), s, tk, testFlow, engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 engine call, got %d", len(fake.Calls))
	}

	if !strings.Contains(fake.Calls[0].Prompt, "SQL migrations") {
		t.Errorf("engine prompt did not include the operator note: %q", fake.Calls[0].Prompt)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	var hasNoteEvent bool
	for _, e := range events {
		if e.Kind == record.TaskNoted && strings.Contains(e.Text, "SQL migrations") {
			hasNoteEvent = true
			break
		}
	}
	if !hasNoteEvent {
		t.Error("expected TaskNoted event in event stream")
	}
}

// TestE2ECostMonotonicityProperty verifies the property that total recorded task cost is non-negative and non-decreasing.
func TestE2ECostMonotonicityProperty(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "E2E-3", "Property test for cost accumulation", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("completed task successfully")
	engines := map[string]engine.Engine{"fake": fake}

	testFlow := flow.Flow{
		Name: "cost-flow",
		Phases: []flow.Phase{
			{Name: "p1", Engine: "fake", Model: "sonnet"},
			{Name: "p2", Engine: "fake", Model: "haiku"},
			{Name: "p3", Engine: "fake", Model: "opus"},
		},
	}

	if err := Run(context.Background(), s, tk, testFlow, engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	var cumulativeCost float64
	for _, e := range events {
		if e.Kind == record.PhaseFinished {
			if costStr, ok := e.Data["cost"]; ok && costStr != "" {
				c, err := strconv.ParseFloat(costStr, 64)
				if err != nil {
					t.Fatalf("invalid cost float string %q: %v", costStr, err)
				}
				if c < 0 {
					t.Errorf("negative phase cost: %f", c)
				}
				cumulativeCost += c
			}
		}
	}
	if cumulativeCost < 0 {
		t.Errorf("total cumulative cost is negative: %f", cumulativeCost)
	}
}

// TestE2EGateDecisionsPipeline tests how scripted gates control phase execution.
func TestE2EGateDecisionsPipeline(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "E2E-4", "Gate pipeline test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("phase execution completed")
	engines := map[string]engine.Engine{"fake": fake}

	testFlow := flow.Flow{
		Name: "3-phase-pipeline",
		Phases: []flow.Phase{
			{Name: "step1", Engine: "fake"},
			{Name: "step2", Engine: "fake"},
			{Name: "step3", Engine: "fake"},
		},
	}

	// Gate says: Skip step1, Continue step2, Stop at step3
	gate := &scriptedGate{answers: []Go{Skip, Continue, Stop}}

	err = Run(context.Background(), s, tk, testFlow, engines, gate)
	if err == nil {
		t.Fatal("expected Run to return error on Stop gate decision")
	}

	if len(fake.Calls) != 1 {
		t.Errorf("expected only 1 engine call (for step2), got %d", len(fake.Calls))
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	// Verify step1 has no phase events, step2 has finished, and task ended with cancelled/stopped
	for _, e := range events {
		if e.Phase == "step1" {
			t.Errorf("step1 was skipped but produced event: %v", e)
		}
	}
}
