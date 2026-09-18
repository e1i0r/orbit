package task

// What the next attempt is told about the one before it.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
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

	// An attempt that did something is never described as having done
	// nothing. "It left no trace" is what the next attempt reads to decide
	// whether to start over, and told it wrongly it starts over on top of
	// work that is already there.
	if strings.Contains(said, "left no trace") {
		t.Errorf("an attempt that ran a command was described as leaving nothing:\n%s", said)
	}

	// And what it is told is that attempt and not the one before it: the
	// event that opened it belongs to the attempt, not to its account.
	if n := strings.Count(said, "refused by a gate"); n != 1 {
		t.Errorf("the account of one attempt says how it ended %d times:\n%s", n, said)
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

	// The gate leaves something behind, because worked() streams a refusal
	// and a phase refused something that writes nothing does not finish.
	one := flow.Flow{Name: "task", Phases: []flow.Phase{{
		Name: "implement", Engine: "fake",
		Gates: []flow.Gate{{Name: "write", Command: "echo done > NOTES.md"}},
	}}}

	if err := Run(context.Background(), s, tk, one, fakes(fake), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if strings.Contains(fake.Calls[0].Prompt, "## The attempt before you") {
		t.Errorf("a first attempt was told about an attempt before it:\n%s", fake.Calls[0].Prompt)
	}
}

// TestHowAnAttemptEndedIsReadOffItsLastTerminalEvent.
//
// The newest one, because an attempt that was retried and then cancelled
// ended by being cancelled. And an attempt whose only event is the one that
// ended it is the ordinary shape of a phase that failed at once — read from
// the wrong end, or one event short, it would read as an attempt nobody can
// account for.
func TestHowAnAttemptEndedIsReadOffItsLastTerminalEvent(t *testing.T) {
	only := howItEnded([]record.Event{{Kind: record.PhaseFailed}})
	if only != endings[record.PhaseFailed] {
		t.Errorf("an attempt of one event reads as %q", only)
	}

	after := howItEnded([]record.Event{
		{Kind: record.PhaseToolCall},
		{Kind: record.PhaseRetried},
		{Kind: record.PhaseCancelled},
	})
	if after != endings[record.PhaseCancelled] {
		t.Errorf("an attempt that was retried and then stopped reads as %q", after)
	}

	// A record with no terminal event at all is what an attempt whose
	// process was killed outright leaves. Guessing there would be inventing
	// the one fact this section exists to get right.
	none := howItEnded([]record.Event{{Kind: record.PhaseToolCall}, {Kind: record.PhaseThought}})
	if !strings.Contains(none, "does not say what stopped it") {
		t.Errorf("an attempt the record cannot account for reads as %q", none)
	}
}

// attemptOf writes a record whose last two attempts at one phase bracket the
// events given, with a terminal event of an earlier phase in front of them.
func attemptOf(t *testing.T, between ...record.Event) (*store.Store, Task) {
	t.Helper()

	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-39", "read the attempt before", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The terminal event belongs to the phase before this one, so it must
	// not be read as how this attempt ended.
	events := []record.Event{
		{Kind: record.PhaseFinished, Phase: "plan"},
		{Kind: record.PhaseStarted, Phase: "implement"},
	}
	events = append(events, between...)
	events = append(events, record.Event{Kind: record.PhaseStarted, Phase: "implement"})

	for _, e := range events {
		if err := emit(s, tk, e); err != nil {
			t.Fatalf("emit %s: %v", e.Kind, err)
		}
	}

	return s, tk
}

// TestAnAttemptIsWhatHappenedInsideItAndNothingBeforeIt.
//
// The account is bracketed by the two times this phase started, and the
// event that opened the attempt is not part of what the attempt did. Read
// one event wider, the phase before it hands over its ending — and the
// reader is told the attempt finished when nothing in it did.
func TestAnAttemptIsWhatHappenedInsideItAndNothingBeforeIt(t *testing.T) {
	s, tk := attemptOf(t, record.Event{
		Kind: record.PhaseToolCall, Phase: "implement",
		Text: "go build ./...", Data: map[string]string{"tool": "Bash"},
	})

	said := soFar(s, tk, "implement", t.TempDir())
	if !strings.Contains(said, "does not say what stopped it") {
		t.Errorf("an attempt with no ending of its own reads as:\n%s", said)
	}

	if strings.Contains(said, "It finished") {
		t.Errorf("the phase before it handed over its ending:\n%s", said)
	}
}

// TestAnAttemptThatDidSomethingIsNeverSaidToHaveDoneNothing.
//
// "It left no trace" is what the next attempt reads to decide whether to
// start over, and told it wrongly it starts over on top of work already
// there. One command run and one tool refused are each a trace on their
// own, and together they are still one.
func TestAnAttemptThatDidSomethingIsNeverSaidToHaveDoneNothing(t *testing.T) {
	ran := record.Event{
		Kind: record.PhaseToolCall, Phase: "implement",
		Text: "go build ./...", Data: map[string]string{"tool": "Bash"},
	}
	refused := record.Event{
		Kind: record.PhaseRefused, Phase: "implement",
		Text: "https://example.test", Data: map[string]string{"tool": "WebFetch"},
	}

	for _, one := range []struct {
		why  string
		what []record.Event
	}{
		{"it ran a command", []record.Event{ran}},
		{"a tool it reached for was refused", []record.Event{refused}},
		{"both", []record.Event{ran, refused}},
	} {
		s, tk := attemptOf(t, one.what...)

		if said := soFar(s, tk, "implement", t.TempDir()); strings.Contains(said, "left no trace") {
			t.Errorf("an attempt where %s was said to have left nothing:\n%s", one.why, said)
		}
	}

	// And one that really did nothing says so, which is the other half: an
	// account listing nothing under three headings is a reader wondering
	// what they missed.
	s, tk := attemptOf(t)

	said := soFar(s, tk, "implement", t.TempDir())
	if !strings.Contains(said, "left no trace") {
		t.Errorf("an attempt that did nothing reads as:\n%s", said)
	}
}
