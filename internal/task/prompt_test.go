package task

// What an engine is handed for one phase, and what it is asked to hand back.
// The prompt is Markdown and the answer is asked for in Markdown, because the
// panes that draw the answer render one and cannot lay out anything else.

import (
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

// asked is the prompt for a task with everything in it: a phase with its own
// instruction, something the phase before said, and notes the operator added
// while it ran.
func asked(t *testing.T, prev string, notes ...string) string {
	t.Helper()

	return prompt(
		Task{ID: "ACME-1", Text: "Retry the webhook on 5xx.", Repo: repoTaskRepo(t)},
		flow.Phase{Name: "implement", Prompt: "Keep the retry budget bounded."},
		notes, prev,
	)
}

// TestThePromptIsWrittenInMarkdown. Every part of it is under a heading of
// its own, so that a model reading it can tell the task from the phase from
// what somebody said about them afterwards.
func TestThePromptIsWrittenInMarkdown(t *testing.T) {
	full := asked(t, "wrote retry.go", "hold off on the backoff")

	// Each of them a line of its own and not a prefix of the next: "## Phase"
	// found inside "## Phase instructions" is a heading nobody wrote.
	lines := strings.Split(full, "\n")
	for _, head := range []string{
		"# ACME-1",
		"## Phase",
		"## Phase instructions",
		"## Previous phase output",
		"## Operator notes",
		"## How to answer",
	} {
		if !slices.Contains(lines, head) {
			t.Errorf("the prompt has no %q section:\n%s", head, full)
		}
	}

	for _, said := range []string{"Retry the webhook on 5xx.", "implement", "repo", "Keep the retry budget bounded."} {
		if !strings.Contains(full, said) {
			t.Errorf("the prompt lost %q:\n%s", said, full)
		}
	}
}

// TestASectionWithNothingInItIsNotDrawn. An empty heading is a question the
// model has to answer for itself — whether the phase before said nothing, or
// said something this program dropped.
func TestASectionWithNothingInItIsNotDrawn(t *testing.T) {
	bare := prompt(
		Task{ID: "ACME-1", Text: "Retry the webhook on 5xx.", Repo: repoTaskRepo(t)},
		flow.Phase{Name: "implement"}, nil, "",
	)

	bareLines := strings.Split(bare, "\n")
	for _, head := range []string{"## Phase instructions", "## Previous phase output", "## Operator notes"} {
		if slices.Contains(bareLines, head) {
			t.Errorf("the prompt heads %q over nothing:\n%s", head, bare)
		}
	}

	if !strings.Contains(bare, "## How to answer") {
		t.Errorf("a prompt with nothing added to it still has to say how to answer:\n%s", bare)
	}
}

// TestOperatorNotesAreABulletedList. They arrive as separate sentences and
// run together as a paragraph, which is how two notes become one instruction
// nobody wrote.
func TestOperatorNotesAreABulletedList(t *testing.T) {
	full := asked(t, "", "hold off on the backoff", "the staging key rotated")

	for _, want := range []string{"- hold off on the backoff", "- the staging key rotated"} {
		if !strings.Contains(full, want) {
			t.Errorf("the prompt does not set %q as a bullet:\n%s", want, full)
		}
	}
}

// TestWhatTheLastPhaseSaidIsFencedOffFromTheRest. The phase before answered
// in Markdown, headings and code fences and all. Set loose under a heading of
// this prompt, its sections read as sections of the prompt.
func TestWhatTheLastPhaseSaidIsFencedOffFromTheRest(t *testing.T) {
	said := "## Findings\n\n```go\nfmt.Println(\"kept\")\n```"

	full := asked(t, said)

	// A fence longer than the one inside it, or the block closes on the
	// answer's own fence and everything past that point is prose again.
	body, ok := carved(full, "````markdown\n", "\n````")
	if !ok {
		t.Fatalf("what the phase before said is not in a fence of its own:\n%s", full)
	}

	if !strings.Contains(body, "fmt.Println") || !strings.Contains(body, "## Findings") {
		t.Errorf("the fence closed before the end of what the phase said: %q", body)
	}
}

// TestTheAnswerContractIsTheLastThingSaid. An instruction about the shape of
// the answer, put before the task, is read as background to the task; put
// last, it is the last thing read before the answer is written.
func TestTheAnswerContractIsTheLastThingSaid(t *testing.T) {
	full := asked(t, "wrote retry.go", "hold off on the backoff")

	if !strings.HasSuffix(full, "\n"+engine.AnswerContract) {
		t.Errorf("the prompt does not end on the contract:\n%s", full)
	}

	if n := strings.Count(full, "## How to answer"); n != 1 {
		t.Errorf("the prompt says how to answer %d times, want once:\n%s", n, full)
	}
}

// carved is the text between an opening and the first closing after it.
func carved(s, open, close string) (string, bool) {
	_, after, found := strings.Cut(s, open)
	if !found {
		return "", false
	}

	body, _, found := strings.Cut(after, close)

	return body, found
}
