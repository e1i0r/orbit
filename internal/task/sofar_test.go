package task

// What the next attempt is told about the one before it.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

// worked is a fake that ran a command, was refused another, and streamed both
// the way a real engine does.
func worked(output string) *engine.Fake {
	f := engine.NewFake(output)
	f.Events = []engine.StreamEvent{
		{Type: "tool_call", ToolCall: engine.StreamToolCall{Name: "Bash", Args: "go build ./..."}},
		{Type: "tool_call", ToolCall: engine.StreamToolCall{Name: "Bash", Args: "go build ./..."}},
		{Type: "tool_call", ToolCall: engine.StreamToolCall{Name: "Read", Args: "internal/task/run.go"}},
		{Type: "refusal", Refusal: engine.StreamRefusal{Tool: "WebFetch", Input: "https://example.com"}},
	}

	return f
}

func TestTheSecondAttemptIsToldWhatTheFirstGotDone(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "make the build green", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := worked("wrote it")

	twice := retryFlow(countingGate("2"), 0)
	if err := Run(context.Background(), s, tk, twice, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("the engine ran %d times, want 2", len(fake.Calls))
	}

	// The first attempt has nothing before it, and says nothing rather than
	// putting up an empty heading for the model to wonder about.
	if strings.Contains(fake.Calls[0].Prompt, "## The attempt before you") {
		t.Error("the first attempt was told about an attempt before it, and there was none")
	}

	said := fake.Calls[1].Prompt
	for _, want := range []string{
		"## The attempt before you",
		"refused by a gate",
		"go build ./...",
		"WebFetch",
		"What the worktree holds now",
	} {
		if !strings.Contains(said, want) {
			t.Errorf("the second attempt was not told %q:\n%s", want, said)
		}
	}

	// Twice streamed, once said: nine identical lines say nothing the first
	// one did not, and they spend the room the files needed.
	if n := strings.Count(said, "go build ./..."); n != 1 {
		t.Errorf("the same command is listed %d times, want 1", n)
	}

	// A tool that reads a file is not a command somebody could type again,
	// and a list of every file the attempt opened is a list of nothing.
	if strings.Contains(said, "internal/task/run.go") {
		t.Errorf("a file the attempt read was listed as a command it ran:\n%s", said)
	}
}

// TestTheSummarySaysTheWorktreeIsEveryPhasesWork is the whole rule of this
// section in one claim: the diff cannot say which phase wrote which line, so
// the heading credits the tree and never the attempt.
func TestTheSummarySaysTheWorktreeIsEveryPhasesWork(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-2", "make the build green", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := worked("wrote it")

	twice := retryFlow(countingGate("2"), 0)
	if err := Run(context.Background(), s, tk, twice, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	said := fake.Calls[1].Prompt
	if !strings.Contains(said, "this phase and every one before it") {
		t.Errorf("the worktree's changes were credited to the last attempt alone:\n%s", said)
	}
}

// TestAPhaseWithNoAttemptBeforeItIsToldNothing is the case that is almost
// every case, and the one an empty heading would cost a turn on.
func TestAPhaseWithNoAttemptBeforeItIsToldNothing(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-3", "write it down", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := worked("wrote it")
	one := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "implement", Engine: "fake"}}}

	if err := Run(context.Background(), s, tk, one, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if strings.Contains(fake.Calls[0].Prompt, "## The attempt before you") {
		t.Errorf("a first attempt was told about an attempt before it:\n%s", fake.Calls[0].Prompt)
	}
}
