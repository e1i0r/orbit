package supervisor

// What the record says happened, which is the ground the answer to "what
// happened while I was out?" stands on.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// TestTheRecordBlockIsWhatWasWrittenDown, and only the kinds that say what
// became of the work: a phase's thinking and its tool calls are thousands of
// lines, and the task view is where somebody reads those.
func TestTheRecordBlockIsWhatWasWrittenDown(t *testing.T) {
	s := fixture(t)
	now := time.Now().UTC()

	d, err := s.Record()
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	for _, e := range []record.Event{
		{At: now.Add(-30 * time.Minute), Kind: record.TaskCreated, Text: "add the thing"},
		{At: now.Add(-20 * time.Minute), Kind: record.PhaseFinished, Phase: "implement", Text: "wrote it", Data: map[string]string{"engine": "claude"}},
		{At: now.Add(-19 * time.Minute), Kind: record.PhaseThought, Text: "thinking out loud"},
		{At: now.Add(-18 * time.Minute), Kind: record.GateFailed, Phase: "implement", Text: "3 tests failing", Data: map[string]string{"gate": "tests", "exit": "1"}},
		{At: now.Add(-2 * time.Minute), Kind: record.TaskFinished},
	} {
		if err := d.Append("ACME-1", e); err != nil {
			t.Fatalf("append %s: %v", e.Kind, err)
		}
	}

	lines, err := Happened(s, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("Happened: %v", err)
	}

	whole := strings.Join(lines, "\n")
	for _, want := range []string{"ACME-1", "phase.finished", "gate.failed", "gate=tests", "exit=1", "3 tests failing", "task.finished"} {
		if !strings.Contains(whole, want) {
			t.Errorf("the block does not say %q:\n%s", want, whole)
		}
	}

	if strings.Contains(whole, "thinking out loud") {
		t.Errorf("the block carries the engine's stream:\n%s", whole)
	}

	// Oldest first, so a run reads forwards.
	if !strings.Contains(lines[0], "phase.finished") {
		t.Errorf("the block starts at %q", lines[0])
	}
}

// TestTheWindowIsWhatItSays: a task nobody has touched since yesterday
// contributes nothing to what happened this afternoon.
func TestTheWindowIsWhatItSays(t *testing.T) {
	s := fixture(t)
	now := time.Now().UTC()

	d, err := s.Record()
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	if err := d.Append("OLD-1", record.Event{At: now.Add(-72 * time.Hour), Kind: record.TaskFinished}); err != nil {
		t.Fatalf("append: %v", err)
	}

	lines, err := Happened(s, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("Happened: %v", err)
	}

	if len(lines) != 0 {
		t.Errorf("the window let %d old lines in: %v", len(lines), lines)
	}
}

// TestTheAnswerIsAskedToKeepVerifiedApartFromItsOwnReading. The person
// coming back from lunch decides what to do next on the strength of which
// one it is.
func TestTheAnswerIsAskedToKeepVerifiedApartFromItsOwnReading(t *testing.T) {
	asked := buildSupervisorPrompt("", []string{"2026-09-05 14:00 · ACME-1 · gate.passed · gate=tests"}, "¿qué pasó?")

	for _, want := range []string{"What the record says", "gate.passed", "verified"} {
		if !strings.Contains(asked, want) {
			t.Errorf("the prompt does not carry %q", want)
		}
	}

	if !strings.Contains(asked, "yours, and say that it is") {
		t.Error("the contract does not ask it to mark what is its own reading")
	}
}

// TestTheAnswerFollowsTheLanguageTheQuestionWasAskedIn. The record and this
// prompt are in English because that is what the log is written in; the
// person reading the answer asked in Spanish.
func TestTheAnswerFollowsTheLanguageTheQuestionWasAskedIn(t *testing.T) {
	asked := buildSupervisorPrompt("", nil, "¿qué pasó mientras no estaba?")

	if !strings.Contains(asked, "Answer in the language the operator wrote in") {
		t.Error("the contract does not ask it to answer in the language it was asked in")
	}
}
