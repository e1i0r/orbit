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
	args := claudeArgs(Request{Prompt: "retry on 5xx", Model: "sonnet"})
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
	for _, a := range claudeArgs(Request{Prompt: "x"}) {
		if a == "--model" {
			t.Error("an empty model was passed as a flag with no value")
		}
	}
}
