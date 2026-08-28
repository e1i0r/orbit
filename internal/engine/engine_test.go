package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFakeReturnsItsOutputAndRecordsTheCall(t *testing.T) {
	f := NewFake("done")

	got, err := f.Run(context.Background(), Request{Prompt: "do it", Model: "sonnet", Dir: "/tmp/wt"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got.Output != "done" {
		t.Errorf("Output = %q, want done", got.Output)
	}

	if len(f.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1", len(f.Calls))
	}

	if f.Calls[0].Prompt != "do it" || f.Calls[0].Dir != "/tmp/wt" {
		t.Errorf("recorded %+v", f.Calls[0])
	}
}

func TestFakeReturnsItsError(t *testing.T) {
	want := errors.New("the model fell over")
	f := NewFake("")

	f.Err = want
	if _, err := f.Run(context.Background(), Request{}); !errors.Is(err, want) {
		t.Errorf("Run returned %v, want %v", err, want)
	}
}

func TestFakeStopsWhenTheContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := NewFake("done").Run(ctx, Request{}); err == nil {
		t.Error("Run ignored a cancelled context")
	}
}

func TestEnginesAreNamed(t *testing.T) {
	if NewFake("").Name() != "fake" {
		t.Errorf("fake is called %q", NewFake("").Name())
	}

	if NewClaude().Name() != "claude" {
		t.Errorf("claude is called %q", NewClaude().Name())
	}
}

func TestClaudeArgsCarryThePrompt(t *testing.T) {
	args, err := claudeArgs(Request{Prompt: "retry on 5xx", Model: "sonnet"})
	if err != nil {
		t.Fatalf("claudeArgs: %v", err)
	}

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-p") {
		t.Errorf("args %v do not run headless", args)
	}

	if !strings.Contains(joined, "retry on 5xx") {
		t.Errorf("args %v do not carry the prompt", args)
	}

	if !strings.Contains(joined, "sonnet") {
		t.Errorf("args %v do not carry the model", args)
	}
}

func TestClaudeArgsOmitAnEmptyModel(t *testing.T) {
	args, err := claudeArgs(Request{Prompt: "x"})
	if err != nil {
		t.Fatalf("claudeArgs: %v", err)
	}

	for _, a := range args {
		if a == "--model" {
			t.Error("an empty model was passed as a flag with no value")
		}
	}
}

// TestClaudeArgsAskForTheStreamThatCarriesTheSessionAndTheCost pins the
// output format, because it is not a preference. Plain text is prose and
// nothing else; the session id and what the run cost are reported in the
// JSON stream and nowhere else, and without a session id there is no taking
// the keyboard.
func TestClaudeArgsAskForTheStreamThatCarriesTheSessionAndTheCost(t *testing.T) {
	args, err := claudeArgs(Request{Prompt: "x"})
	if err != nil {
		t.Fatalf("claudeArgs: %v", err)
	}

	joined := strings.Join(args, " ")
	for _, want := range []string{"--output-format stream-json", "--verbose"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q do not carry %q", joined, want)
		}
	}
}

// TestClaudeArgsCarryASessionToResume wires the field the window's take-the-
// keyboard gesture will set. Nothing sets it yet, which is why it is tested
// here rather than through a caller: the argv is the contract, and it is the
// half that can be got wrong now and only discovered later.
func TestClaudeArgsCarryASessionToResume(t *testing.T) {
	args, err := claudeArgs(Request{Prompt: "x", Resume: "9c1f8f2a-4d3b-4a77-9a52-2f0f6f9b5c31"})
	if err != nil {
		t.Fatalf("claudeArgs: %v", err)
	}

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--resume 9c1f8f2a-4d3b-4a77-9a52-2f0f6f9b5c31") {
		t.Errorf("args %q do not resume the session they were given", joined)
	}
}

func TestClaudeArgsOmitAnEmptySession(t *testing.T) {
	args, err := claudeArgs(Request{Prompt: "x"})
	if err != nil {
		t.Fatalf("claudeArgs: %v", err)
	}

	for _, a := range args {
		if a == "--resume" {
			t.Error("an empty session id was passed as a flag with no value")
		}
	}
}
