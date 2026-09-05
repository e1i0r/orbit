package supervisor

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
)

func TestSuperviseExecutesEngineAndRecordsAnswer(t *testing.T) {
	s := fixture(t)
	fake := &engine.Fake{
		Output: "Reviewed all tasks, all systems nominal.",
	}

	res, err := Supervise(context.Background(), s, fake, "how is the board doing?")
	if err != nil {
		t.Fatalf("Supervise failed: %v", err)
	}

	if res != "Reviewed all tasks, all systems nominal." {
		t.Errorf("res = %q", res)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("fake.Calls len = %d, want 1", len(fake.Calls))
	}

	if !strings.Contains(fake.Calls[0].Prompt, "how is the board doing?") {
		t.Errorf("prompt missing question: %s", fake.Calls[0].Prompt)
	}

	// Verify recorded in supervisor thread
	events, err := Events(s)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}

	if events[0].Text != "Reviewed all tasks, all systems nominal." || events[0].Data["by"] != "fake" {
		t.Errorf("event[0] = %+v", events[0])
	}
}

func TestSuperviseRefusesNilStoreOrEngineOrEmptyPrompt(t *testing.T) {
	s := fixture(t)
	fake := &engine.Fake{Output: "ok"}

	if _, err := Supervise(context.Background(), nil, fake, "msg"); err == nil {
		t.Error("Supervise on nil store answered nil, want error")
	}

	if _, err := Supervise(context.Background(), s, nil, "msg"); err == nil {
		t.Error("Supervise on nil engine answered nil, want error")
	}

	if _, err := Supervise(context.Background(), s, fake, "   "); err == nil {
		t.Error("Supervise on empty prompt answered nil, want error")
	}
}

func TestAutoSuperviseBuildsPromptWithTaskIDs(t *testing.T) {
	s := fixture(t)
	fake := &engine.Fake{
		Output: "Addressed failures in ORB-10 and ORB-12.",
	}

	res, err := AutoSupervise(context.Background(), s, fake, []string{"ORB-10", "ORB-12"})
	if err != nil {
		t.Fatalf("AutoSupervise failed: %v", err)
	}

	if !strings.Contains(fake.Calls[0].Prompt, "ORB-10, ORB-12") {
		t.Errorf("prompt missing task list: %s", fake.Calls[0].Prompt)
	}

	if res != "Addressed failures in ORB-10 and ORB-12." {
		t.Errorf("res = %q", res)
	}
}

// TestTheSecondCallShowsTheModelWhatTheFirstOneSaid is what the thread is
// for. The supervisor does not only speak — it directs tasks, retries them
// and cancels them — so one that starts every call from nothing is one that
// does the same thing twice.
func TestTheSecondCallShowsTheModelWhatTheFirstOneSaid(t *testing.T) {
	s := fixture(t)
	fake := &engine.Fake{Output: "ORB-3 is stuck on a gate"}

	if _, err := Supervise(context.Background(), s, fake, "what is stuck?"); err != nil {
		t.Fatalf("first Supervise: %v", err)
	}

	if _, err := Supervise(context.Background(), s, fake, "and now?"); err != nil {
		t.Fatalf("second Supervise: %v", err)
	}

	second := fake.Calls[1].Prompt
	if !strings.Contains(second, "## Thread so far") {
		t.Errorf("the second prompt carries no history section:\n%s", second)
	}

	if !strings.Contains(second, "ORB-3 is stuck on a gate") {
		t.Errorf("the second prompt does not carry what the first call answered:\n%s", second)
	}
}

// TestTheSupervisorIsAskedInMarkdown. What it says is drawn as Markdown in
// the cockpit, so what it is asked is written as Markdown too: a prompt that
// asks in one shape for another is asking twice.
func TestTheSupervisorIsAskedInMarkdown(t *testing.T) {
	full := buildSupervisorPrompt("[operator via tui]: what is stuck?\n", "and now?")

	lines := strings.Split(full, "\n")
	for _, head := range []string{"# Supervisor", "## Thread so far", "## Operator message", "## How to answer"} {
		if !slices.Contains(lines, head) {
			t.Errorf("the prompt has no %q section:\n%s", head, full)
		}
	}

	// Its own contract and not the one a phase answers to: see answerContract.
	if !strings.HasSuffix(full, "\n"+answerContract) {
		t.Errorf("the prompt does not end on the contract:\n%s", full)
	}

	if n := strings.Count(full, "## How to answer"); n != 1 {
		t.Errorf("the prompt says how to answer %d times, want once:\n%s", n, full)
	}
}

// TestAThreadOfMarkdownIsFencedOffFromThePrompt. Every turn in the thread was
// written to the contract above, headings and code fences and all. Set loose
// under a heading of this prompt, its sections read as sections of the prompt.
func TestAThreadOfMarkdownIsFencedOffFromThePrompt(t *testing.T) {
	said := "[fake via supervisor]: ## Findings\n\n```go\nfmt.Println(\"kept\")\n```\n"

	full := buildSupervisorPrompt(said, "and now?")

	_, after, ok := strings.Cut(full, "````markdown\n")
	if !ok {
		t.Fatalf("the thread is not in a fence longer than its own:\n%s", full)
	}

	body, _, ok := strings.Cut(after, "\n````")
	if !ok || !strings.Contains(body, "fmt.Println") {
		t.Errorf("the fence closed before the end of the thread: %q", body)
	}
}

// TestAThreadWithNothingInItIsNotHeaded. An empty heading is a question the
// model has to answer for itself — whether nothing was said before, or
// whether something was and this program dropped it.
func TestAThreadWithNothingInItIsNotHeaded(t *testing.T) {
	full := buildSupervisorPrompt("", "what is stuck?")

	if slices.Contains(strings.Split(full, "\n"), "## Thread so far") {
		t.Errorf("the prompt heads a thread over nothing:\n%s", full)
	}

	if !strings.Contains(full, "what is stuck?") {
		t.Errorf("the prompt lost the operator's message:\n%s", full)
	}
}
