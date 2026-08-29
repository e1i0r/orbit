package task

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
)

// logs is the log and the errors file, as text, after fn has run against a
// logger of its own. The logger is a package-level global, so no test here
// runs in parallel with another.
func logs(t *testing.T, fn func()) (string, string) {
	t.Helper()

	dir := t.TempDir()
	all, bad := filepath.Join(dir, "orbit.log"), filepath.Join(dir, "errors.log")

	if err := logger.Init(all, bad); err != nil {
		t.Fatalf("logger.Init: %v", err)
	}

	fn()

	if err := logger.CloseGlobal(); err != nil {
		t.Fatalf("logger.CloseGlobal: %v", err)
	}

	return read(t, all), read(t, bad)
}

func read(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(b)
}

// TestARunThatWorkedIsInTheLogAndNotInTheErrors. Every phase boundary is a
// line, so a run that took forty minutes can be read as the four phases it
// was; and nothing about it reaches the errors file, which is worth less for
// every line in it that nothing went wrong about.
func TestARunThatWorkedIsInTheLogAndNotInTheErrors(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	all, bad := logs(t, func() {
		engines := map[string]engine.Engine{"fake": engine.NewFake("wrote the retry")}
		if err := Run(context.Background(), s, tk, oneFlow(), engines, nil); err != nil {
			t.Errorf("Run: %v", err)
		}
	})

	for _, want := range []string{
		"[INFO] [task/run] " + tk.ID + ": task.started",
		"[INFO] [task/run] " + tk.ID + ": phase.started in phase implement",
		"[INFO] [task/run] " + tk.ID + ": phase.finished in phase implement",
		"[INFO] [task/run] " + tk.ID + ": task.finished",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the log does not say %q:\n%s", want, all)
		}
	}

	if bad != "" {
		t.Errorf("a run that worked wrote to the errors file:\n%s", bad)
	}
}

// TestAFailedRunSaysWhyInTheErrorsFile. The reason and not the model's
// prose: phase.failed keeps what the engine printed in Text and why it
// stopped in Data["error"], and a line that carried the first would name a
// failure without saying what it was.
func TestAFailedRunSaysWhyInTheErrorsFile(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	all, bad := logs(t, func() {
		fake := engine.NewFake("half an answer")
		fake.Err = errors.New("the model fell over")

		engines := map[string]engine.Engine{"fake": fake}
		if err := Run(context.Background(), s, tk, oneFlow(), engines, nil); err == nil {
			t.Error("Run reported success after the engine failed")
		}
	})

	for _, want := range []string{
		"[ERROR] [task/run] " + tk.ID + ": phase.failed in phase implement: the model fell over",
		"[ERROR] [task/run] " + tk.ID + ": task.failed",
	} {
		if !strings.Contains(bad, want) {
			t.Errorf("the errors file does not say %q:\n%s", want, bad)
		}
	}

	if !strings.Contains(all, "phase.failed") {
		t.Errorf("the failure is in the errors file and not in the log:\n%s", all)
	}
}

// TestTheModelsStreamStaysOutOfTheLog. A phase writes one event per thought
// and one per tool call, hundreds of them, and the record has all of them.
// In the log they would bury the four lines that say what became of the run.
func TestTheModelsStreamStaysOutOfTheLog(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	all, _ := logs(t, func() {
		fake := engine.NewFake("done")
		fake.Events = []engine.StreamEvent{
			{Type: "thought", Thought: "considering the retry"},
			{Type: "tool_call", ToolCall: engine.StreamToolCall{Name: "Bash", Args: "go test ./..."}},
		}

		engines := map[string]engine.Engine{"fake": fake}
		if err := Run(context.Background(), s, tk, oneFlow(), engines, nil); err != nil {
			t.Errorf("Run: %v", err)
		}
	})

	for _, absent := range []string{"phase.thought", "phase.tool_call", "considering the retry", "go test ./..."} {
		if strings.Contains(all, absent) {
			t.Errorf("the log carries %q, which belongs in the record and nowhere else:\n%s", absent, all)
		}
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	var streamedKinds int

	for _, e := range events {
		if streamed[e.Kind] {
			streamedKinds++
		}
	}

	if streamedKinds != 2 {
		t.Errorf("the record kept %d streamed events, want 2 — this test would pass on a run that never streamed", streamedKinds)
	}
}

// TestNoEventPutsAWholeFileOnOneLine. task.created carries the whole of
// task.md, and every entry in the log is one line with one timestamp on it.
func TestNoEventPutsAWholeFileOnOneLine(t *testing.T) {
	s, r := fixture(t)

	body := strings.Repeat("a paragraph of the task description\n", 40)

	all, _ := logs(t, func() {
		if _, err := Create(s, r, "ACME-1", body, ""); err != nil {
			t.Errorf("Create: %v", err)
		}
	})

	if !strings.Contains(all, "task.created") {
		t.Fatalf("the log does not say a task was created:\n%s", all)
	}

	if lines := strings.Count(strings.TrimSpace(all), "\n") + 1; lines != 1 {
		t.Errorf("task.created wrote %d lines, want 1:\n%s", lines, all)
	}
}

// TestAFailureIsCutToOneLine. An engine can fail with the whole of its
// stderr, and a log entry wide enough to wrap in a terminal is one nobody
// skims. The record still holds all of it.
func TestAFailureIsCutToOneLine(t *testing.T) {
	wide := strings.Repeat("w", noteWidth+50)

	for _, c := range []struct {
		name string
		e    record.Event
		want string
	}{
		{
			"the reason and not the prose",
			record.Event{Text: "what the engine printed", Data: map[string]string{"error": "exit status 1"}},
			"exit status 1",
		},
		{
			"the output when there is no reason",
			record.Event{Text: "what the engine printed"},
			"what the engine printed",
		},
		{
			"the first line and no more",
			record.Event{Text: "the model fell over\nand a second line"},
			"the model fell over",
		},
		{
			"as wide as one line may be",
			record.Event{Text: wide},
			wide[:noteWidth],
		},
		{
			"nothing to say",
			record.Event{},
			"",
		},
	} {
		if got := oneLine(c.e); got != c.want {
			t.Errorf("%s: oneLine = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestNoEntrySpillsOntoASecondLine. Every entry in the log is one line with
// one timestamp on it, and a failure that arrives with a newline in it is
// the way that stops being true.
func TestNoEntrySpillsOntoASecondLine(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, bad := logs(t, func() {
		fake := engine.NewFake("")
		fake.Err = errors.New("the model fell over\n" + strings.Repeat("w", noteWidth+50))

		engines := map[string]engine.Engine{"fake": fake}
		if err := Run(context.Background(), s, tk, oneFlow(), engines, nil); err == nil {
			t.Error("Run reported success after the engine failed")
		}
	})

	for i, line := range strings.Split(strings.TrimSpace(bad), "\n") {
		if !strings.HasPrefix(line, "[") {
			t.Errorf("line %d of the errors file is the tail of the entry above it: %q", i, line)
		}
	}
}

// TestACancelledRunIsAWarningAndNotAFailure. A reader who typed cancel got
// what they asked for. The run is over either way and the log says so; what
// the errors file is for is the runs nobody chose to end.
func TestACancelledRunIsAWarningAndNotAFailure(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	all, bad := logs(t, func() {
		ctx, cancel := context.WithCancel(context.Background())

		fake := engine.NewFake("")
		fake.Err = errors.New("killed")

		cancel()

		engines := map[string]engine.Engine{"fake": fake}
		if err := Run(ctx, s, tk, oneFlow(), engines, nil); err == nil {
			t.Error("Run reported success after its context was cancelled")
		}
	})

	if !strings.Contains(all, "[WARN] [task/run] "+tk.ID+": task.cancelled") {
		t.Errorf("a cancelled run is not in the log as a warning:\n%s", all)
	}

	if strings.Contains(bad, "task.cancelled") {
		t.Errorf("a run somebody cancelled went to the errors file:\n%s", bad)
	}
}

// TestAKilledRunLeavesItsOnlyTraceInTheLog. SIGKILL cannot be caught, so the
// run records nothing about its own death and the record says nothing about
// it until somebody runs reconcile — which may be days later, or never.
func TestAKilledRunLeavesItsOnlyTraceInTheLog(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cmd := held(t, "sleep 5")
	if _, err := mark(s, tk, cmd.Process.Pid); err != nil {
		t.Fatalf("mark: %v", err)
	}

	all, _ := logs(t, func() {
		if err := Kill(s, tk); err != nil {
			t.Errorf("Kill: %v", err)
		}
	})

	if !strings.Contains(all, "[WARN] [task/cancel] "+tk.ID+": killed") {
		t.Errorf("a killed run left no trace anywhere:\n%s", all)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	for _, e := range events {
		if e.Kind == "task.cancelled" || e.Kind == "task.failed" {
			t.Errorf("the record has %q for a run that was killed — this test's premise is wrong", e.Kind)
		}
	}
}
