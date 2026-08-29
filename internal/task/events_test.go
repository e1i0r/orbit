package task

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// find returns the one event of a kind, failing if there is not exactly one.
func find(t *testing.T, events []record.Event, kind string) record.Event {
	t.Helper()

	var got []record.Event

	for _, e := range events {
		if e.Kind == kind {
			got = append(got, e)
		}
	}

	if len(got) != 1 {
		t.Fatalf("found %d %s events, want 1", len(got), kind)
	}

	return got[0]
}

// TestRunRecordsTheDataKeysTheWindowWillRead pins the names, not just the
// values. Every reading surface plan 2 builds comes out of these keys, and
// renaming one is a silent break: the log still parses, the window just
// stops finding what it needs.
func TestRunRecordsTheDataKeysTheWindowWillRead(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": engine.NewFake("done")}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	// task.started carries the flow and nothing else. It cannot carry the
	// worktree, because it is written at the top of Run — before there is a
	// worktree to name. Nothing reads the key, and the path is a pure
	// function of the state root, the repository and the id, so
	// s.WorktreeDir answers it for anyone who wants it; an attempt with no
	// line in the log is the thing that could not be recovered.
	started := find(t, events, record.TaskStarted)
	for key, want := range map[string]string{"flow": "task"} {
		if started.Data[key] != want {
			t.Errorf(`task.started Data[%q] = %q, want %q`, key, started.Data[key], want)
		}
	}

	phase := find(t, events, record.PhaseStarted)
	for key, want := range map[string]string{"engine": "fake", "model": "sonnet", "n": "1"} {
		if phase.Data[key] != want {
			t.Errorf(`phase.started Data[%q] = %q, want %q`, key, phase.Data[key], want)
		}
	}
}

func TestRunTruncatesAnEnormousOutputAndSaysSoInTheRecord(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Comfortably past the cap and past record.MaxLine, so that recording
	// it verbatim would be refused and the phase would finish with nothing
	// written down at all.
	huge := strings.Repeat("x", 5<<20)

	eng := resultEngine{result: engine.Result{Output: huge}}
	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v — an enormous phase output made the whole record unreadable", err)
	}

	finished := find(t, events, "phase.finished")
	if len(finished.Text) >= len(huge) {
		t.Errorf("recorded %d bytes of a %d byte output — nothing was cut", len(finished.Text), len(huge))
	}

	if !strings.HasSuffix(finished.Text, "…[truncated, full output was "+strconv.Itoa(len(huge))+" bytes]") {
		t.Errorf("the text ends %q — truncation must announce itself", finished.Text[max(0, len(finished.Text)-60):])
	}

	if finished.Data["output_bytes"] != strconv.Itoa(len(huge)) {
		t.Errorf(`Data["output_bytes"] = %q, want %q`, finished.Data["output_bytes"], strconv.Itoa(len(huge)))
	}

	if find(t, events, "task.finished").Kind != "task.finished" {
		t.Error("the run did not finish")
	}
}

func TestAnOrdinaryOutputIsRecordedWhole(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": engine.NewFake("wrote the retry")}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	finished := find(t, events, "phase.finished")
	if finished.Text != "wrote the retry" {
		t.Errorf("phase.finished text = %q, want it word for word", finished.Text)
	}

	if _, ok := finished.Data["output_bytes"]; ok {
		t.Error("output_bytes was recorded though nothing was truncated")
	}
}

// A run has four ways to fail, and every one of them has to reach the log.
// Two are refused before a phase starts: an invalid flow and an engine
// nobody configured. A run that fails without a task.failed in the
// record is not a failed task to anything downstream — the record is the
// only thing the window reads, so it is a task that started and never came
// back.

func TestRunRecordsAFlowThatDoesNotValidate(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "task"} // no phases
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake("")}, nil); err == nil {
		t.Fatal("Run walked a flow with no phases")
	}

	failed := find(t, mustEvents(t, s, tk), "task.failed")
	if !strings.Contains(failed.Text, "no phases") {
		t.Errorf("task.failed does not say what was wrong with the flow: %q", failed.Text)
	}
}

func TestRunRecordsAnEngineNobodyConfigured(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "implement", Engine: "opencode"}}}
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake("")}, nil); err == nil {
		t.Fatal("Run accepted a phase naming an engine that is not configured")
	}

	failed := find(t, mustEvents(t, s, tk), "task.failed")
	if !strings.Contains(failed.Text, "opencode") {
		t.Errorf("task.failed does not name the missing engine: %q", failed.Text)
	}
}

// mustEvents reads a task's record or fails the test.
func mustEvents(t *testing.T, s *store.Store, tk Task) []record.Event {
	t.Helper()

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	return events
}

// permissionFlow is a one-phase flow that states a posture, so the tests
// below can watch it travel from the flow file to the engine and into the
// record.
func permissionFlow(perms ...string) flow.Flow {
	return flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake", Model: "sonnet", Permissions: perms},
	}}
}

// TestPhaseStartedSaysWhatThePhaseWasAllowedToTouch puts the posture in the
// record. The log is the only thing the window reads and the only account of
// a run that survives it, so "what was this agent allowed to do?" has to be
// answerable from the log rather than from whichever flow file happened to
// be on disk at the time.
func TestPhaseStartedSaysWhatThePhaseWasAllowedToTouch(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := permissionFlow(flow.PermissionRead, flow.PermissionNetwork)
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake("done")}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if got := find(t, events, record.PhaseStarted).Data["permissions"]; got != "read,network" {
		t.Errorf(`phase.started Data["permissions"] = %q, want "read,network"`, got)
	}
}

// TestPhaseStartedOmitsPermissionsWhenThePhaseAsksForNothing matches how the
// ending events omit session and cost: a key that is not there says nothing,
// where a key holding an empty string would look like a posture somebody
// wrote down.
func TestPhaseStartedOmitsPermissionsWhenThePhaseAsksForNothing(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": engine.NewFake("done")}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if _, ok := find(t, events, record.PhaseStarted).Data["permissions"]; ok {
		t.Error(`phase.started carries an empty "permissions" key`)
	}
}

// TestTheEngineIsHandedThePhasesPermissions is the half of the change that
// the record cannot show: the posture has to reach the process, not only the
// log. Before this, the flow files shipped "permissions": ["repo"] and
// engine.Request had no field that could carry it.
func TestTheEngineIsHandedThePhasesPermissions(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("done")

	f := permissionFlow(flow.PermissionRepo)
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("the engine was called %d times, want 1", len(fake.Calls))
	}

	got := fake.Calls[0].Permissions
	if len(got) != 1 || got[0] != flow.PermissionRepo {
		t.Errorf("the engine was handed %v, want [repo]", got)
	}
}
