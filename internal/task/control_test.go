package task

import (
	"context"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// controlPath is where the word a reader left lives, asked of the store
// rather than built here — a test that builds its own path is a second
// opinion about the layout.
func controlPath(t *testing.T, s *store.Store, tk Task) string {
	t.Helper()
	path, err := s.ControlPath(tk.Repo.Path, tk.ID)
	if err != nil {
		t.Fatalf("ControlPath: %v", err)
	}
	return path
}

func TestAControlWordIsOneLineOfTextAnybodyCanRead(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	if err := Control(s, tk, "pause"); err != nil {
		t.Fatalf("Control: %v", err)
	}

	path := controlPath(t, s, tk)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the control file: %v", err)
	}
	if string(body) != "pause\n" {
		t.Errorf("the control file holds %q, want %q — one word, one line, cat-able", body, "pause\n")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat the control file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("the control file is mode %v, want 0600 like everything else under the state root", info.Mode().Perm())
	}
}

func TestAWordNoRunUnderstandsIsRefusedAtTheDoor(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	err := Control(s, tk, "halt")

	if err == nil {
		t.Fatal("Control accepted a word no run understands")
	}
	if !strings.Contains(err.Error(), "halt") {
		t.Errorf("the refusal does not name the word that was wrong: %v", err)
	}
	if _, statErr := os.Stat(controlPath(t, s, tk)); !os.IsNotExist(statErr) {
		t.Error("a refused word was written down anyway")
	}
}

func TestEveryWordTheGateActsOnIsAWordControlAccepts(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	for _, word := range []string{"pause", "resume", "cancel", "continue", "skip"} {
		if err := Control(s, tk, word); err != nil {
			t.Errorf("Control(%q): %v", word, err)
		}
	}
}

func TestAPauseWordStopsTheRunAndAResumeWordLetsItGo(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	// Written before the run starts, which is the case the file exists for:
	// a word survives a reader who is not there when it is read.
	if err := Control(s, tk, "pause"); err != nil {
		t.Fatalf("Control: %v", err)
	}
	fake := engine.NewFake("wrote the retry")

	synctest.Test(t, func(t *testing.T) {
		done := make(chan error, 1)
		go func() {
			done <- Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second))
		}()
		synctest.Wait()

		waiting := find(t, eventsOf(t, s, tk), record.PhaseWaiting)
		if got := waiting.Data["why"]; got != "paused" {
			t.Errorf("phase.waiting says why=%q, want paused — the reader's own pause is not a warning", got)
		}
		if _, statErr := os.Stat(controlPath(t, s, tk)); !os.IsNotExist(statErr) {
			t.Error("the control word is still there after it was consumed — one word moves a run once")
		}

		if err := Control(s, tk, "resume"); err != nil {
			t.Fatalf("Control: %v", err)
		}
		if err := <-done; err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted,
		record.PhaseWaiting, record.PhaseResumed,
		record.PhaseStarted, record.PhaseFinished,
		record.TaskFinished)
	if got := find(t, events, record.PhaseResumed).Data["how"]; got != "resume" {
		t.Errorf("phase.resumed says how=%q, want resume", got)
	}
	if len(fake.Calls) != 1 {
		t.Errorf("the engine was called %d times, want 1", len(fake.Calls))
	}
}

func TestACancelWordAtAGateIsACancellationAndNotAFailure(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	if err := Control(s, tk, "pause"); err != nil {
		t.Fatalf("Control: %v", err)
	}
	fake := engine.NewFake("")

	var runErr error
	synctest.Test(t, func(t *testing.T) {
		done := make(chan error, 1)
		go func() {
			done <- Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second))
		}()
		synctest.Wait()

		if err := Control(s, tk, "cancel"); err != nil {
			t.Fatalf("Control: %v", err)
		}
		runErr = <-done
	})

	if runErr == nil {
		t.Fatal("Run reported success after it was cancelled at its gate")
	}
	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted,
		record.PhaseWaiting, record.TaskCancelled)
	for _, e := range events {
		if e.Kind == record.TaskFailed {
			t.Error("a run cancelled at its gate was written down as failed")
		}
	}
	if len(fake.Calls) != 0 {
		t.Errorf("the engine was called %d times after the run was cancelled, want 0", len(fake.Calls))
	}
}

func TestAutopilotDoesNotReleaseARunTheReaderPaused(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	if err := Control(s, tk, "pause"); err != nil {
		t.Fatalf("Control: %v", err)
	}
	fake := engine.NewFake("wrote the retry")

	synctest.Test(t, func(t *testing.T) {
		done := make(chan error, 1)
		go func() {
			done <- Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second))
		}()
		synctest.Wait()

		// Autopilot answers the flow's gates. It is not an answer to the
		// reader's own hand on the brake, and only `resume` is.
		if err := s.SaveSettings(store.Settings{Autopilot: true, UnreadCap: 5}); err != nil {
			t.Fatalf("SaveSettings: %v", err)
		}
		// A sleep on the bubble's clock, so at least one poll goes by with
		// the switch on before the record is asked what happened.
		time.Sleep(3 * time.Second)
		synctest.Wait()
		for _, e := range eventsOf(t, s, tk) {
			if e.Kind == record.PhaseResumed {
				t.Fatal("autopilot released a run the reader had paused; the reader's pause is theirs to lift")
			}
		}

		if err := Control(s, tk, "resume"); err != nil {
			t.Fatalf("Control: %v", err)
		}
		if err := <-done; err != nil {
			t.Fatalf("Run: %v", err)
		}
	})
}

// TestASkipWordPutsTheRunPastTheNextPhaseAndNoFurther is the word's other
// half: skip is answered at a gate that was never going to stop, so the run
// steps over one phase without waiting for anybody and carries on into the
// next. One word moves a run once, which is why the second phase runs.
func TestASkipWordPutsTheRunPastTheNextPhaseAndNoFurther(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	if err := Control(s, tk, "skip"); err != nil {
		t.Fatalf("Control: %v", err)
	}
	fake := engine.NewFake("done")

	if err := Run(context.Background(), s, tk, twoFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted,
		record.PhaseStarted, record.PhaseFinished, record.TaskFinished)
	for _, e := range events {
		if e.Phase == "implement" {
			t.Errorf("the skipped phase left %s in the record; a phase that did not run records nothing", e.Kind)
		}
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("the engine was called %d times, want 1 — the skip word moved the run past one phase, not past the flow", len(fake.Calls))
	}
}
