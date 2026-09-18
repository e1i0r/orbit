package task

// What the next attempt is told about the one before it.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
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

// TestACountAndTheWordForWhatItCounts.
//
// A prompt is read by something that reads English, and "1 files" is the
// kind of seam that makes a reader wonder what else was assembled without
// being looked at.
func TestACountAndTheWordForWhatItCounts(t *testing.T) {
	for _, one := range []struct {
		n    int
		want string
	}{
		{0, "0 files"},
		{1, "1 file"},
		{2, "2 files"},
		{12, "12 files"},
	} {
		if got := many(one.n, "file"); got != one.want {
			t.Errorf("many(%d) = %q, want %q", one.n, got, one.want)
		}
	}
}

// TestACommandIsShownByItsShapeAndNotItsWholeScript.
//
// The arguments are whatever the engine wrote and may be a whole script.
// What the reader wants is the shape of what was tried, and a heredoc pasted
// into a prompt is the room the files needed — so it is the first line, and
// no more of it than a command can be recognised by.
func TestACommandIsShownByItsShapeAndNotItsWholeScript(t *testing.T) {
	ran := func(tool, args string) record.Event {
		return record.Event{Kind: record.PhaseToolCall, Text: args, Data: map[string]string{"tool": tool}}
	}

	if got := command(ran("Bash", "go build ./...")); got != "go build ./..." {
		t.Errorf("a command came back as %q", got)
	}

	// A tool that reads a file is not a command somebody could type again.
	if got := command(ran("Read", "internal/task/run.go")); got != "" {
		t.Errorf("a file that was read came back as the command %q", got)
	}

	// The first line, and a mark saying there was more.
	script := command(ran("Bash", "cat <<'EOF' > f.txt\nline one\nline two\nEOF"))
	if script != "cat <<'EOF' > f.txt …" {
		t.Errorf("a script came back as %q", script)
	}

	// A single line exactly as wide as is shown is shown whole: a mark
	// saying something was cut, over nothing, is a reader going to look for
	// the rest of a command that is all there.
	exact := strings.Repeat("a", lineWidth)
	if got := command(ran("Bash", exact)); got != exact {
		t.Errorf("a command of exactly %d characters came back as %q", lineWidth, got)
	}

	over := command(ran("Bash", strings.Repeat("a", lineWidth+1)))
	if !strings.HasSuffix(over, "…") || len(over) != lineWidth+len("…") {
		t.Errorf("a command one character too long came back as %d characters", len(over))
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

// TestTheFilesAreCappedAndTheRestAreCounted.
//
// "… and 30 more" is itself a fact about the attempt, and a prompt listing
// four hundred paths is one where the files are the prompt. The cap has to
// bite once: a line after every file but the twentieth is a list with
// thirty counts in it and nothing to count.
func TestTheFilesAreCappedAndTheRestAreCounted(t *testing.T) {
	var all []repo.Change
	for i := range atMostFiles + 5 {
		all = append(all, repo.Change{Path: fmt.Sprintf("internal/f%02d.go", i), Added: 1})
	}

	var b strings.Builder

	writeFiles(&b, all)

	listing := b.String()
	if n := strings.Count(listing, "… and "); n != 1 {
		t.Errorf("the listing counts what was left out %d times:\n%s", n, listing)
	}

	if !strings.Contains(listing, "… and 5 more") {
		t.Errorf("the listing does not say how many were left out:\n%s", listing)
	}

	// The first twenty are there and the twenty-first is not.
	if !strings.Contains(listing, "internal/f19.go") {
		t.Errorf("the last file inside the cap is missing:\n%s", listing)
	}

	if strings.Contains(listing, "internal/f20.go") {
		t.Errorf("a file past the cap was listed anyway:\n%s", listing)
	}

	// And a tree with nothing in it writes no heading at all.
	var empty strings.Builder

	writeFiles(&empty, nil)

	if empty.String() != "" {
		t.Errorf("a tree holding nothing wrote %q", empty.String())
	}
}
