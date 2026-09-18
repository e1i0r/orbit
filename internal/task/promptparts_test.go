package task

// Two parts of a phase's prompt that only appear on the right kind of run,
// and the cut that keeps the record readable.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/flow"
)

// TestATaskWithNoCheckoutIsToldHowToGetOne.
//
// A task written against no repository is the ordinary way one starts when
// nobody knew yet which project the work was in. The prompt that told it
// about "other" repositories would be naming the checkout it has, and the
// model would go looking for one it was never given.
func TestATaskWithNoCheckoutIsToldHowToGetOne(t *testing.T) {
	none := workspace(Task{ID: "ACME-1"}, []string{"api", "ledger"})

	if !strings.Contains(none, "## Repositories in this workspace") {
		t.Errorf("a task with no checkout heads the section as though it had one:\n%s", none)
	}

	if strings.Contains(none, "## Other repositories") {
		t.Errorf("a task with no checkout is told about the others:\n%s", none)
	}

	if !strings.Contains(none, "no checkout of its own yet") {
		t.Errorf("it is not told it has no checkout:\n%s", none)
	}

	// And one that has a checkout is told about the rest, without being
	// told to go and get the one it is already standing in.
	some := workspace(Task{ID: "ACME-1", Repo: repoTaskRepo(t)}, []string{"api", "ledger"})

	if !strings.Contains(some, "## Other repositories in this workspace") {
		t.Errorf("a task with a checkout is not told about the others:\n%s", some)
	}

	if strings.Contains(some, "no checkout of its own yet") {
		t.Errorf("a task with a checkout is told it has none:\n%s", some)
	}

	// Both name the verb and say it may happen at any point, because the
	// alternative a model falls back on is asking permission the run has
	// nobody to give.
	for _, one := range []string{none, some} {
		if !strings.Contains(one, "orbit join <name>") || !strings.Contains(one, "any phase") {
			t.Errorf("the section does not say how to join, or when:\n%s", one)
		}
	}
}

// TestWhatReviewersAskedForIsItsOwnSection.
//
// A reviewer asking for something is the same kind of instruction a note is
// — it came from a person and it is about this task — but the operator is
// the one whose word settles a disagreement, so the two are kept apart and
// theirs is read first. A heading over nothing is a section a model looks
// for content under and finds the contract instead.
func TestWhatReviewersAskedForIsItsOwnSection(t *testing.T) {
	const head = "## What reviewers asked for"

	bare := build(Task{ID: "ACME-1", Text: "Retry the webhook on 5xx.", Repo: repoTaskRepo(t)},
		flow.Phase{Name: "implement"}, false, nil, nil, nil, "", "", nil)

	if strings.Contains(bare, head) {
		t.Errorf("a run nobody reviewed heads the section over nothing:\n%s", bare)
	}

	asked := build(Task{ID: "ACME-1", Text: "Retry the webhook on 5xx.", Repo: repoTaskRepo(t)},
		flow.Phase{Name: "implement"}, false, nil,
		[]string{"hold off on the backoff"},
		[]string{"name the error type", "add a test for the 429"}, "", "", nil)

	if !strings.Contains(asked, head) {
		t.Errorf("what reviewers asked for is not in the prompt:\n%s", asked)
	}

	for _, want := range []string{"- name the error type", "- add a test for the 429"} {
		if !strings.Contains(asked, want) {
			t.Errorf("the prompt does not set %q as a bullet:\n%s", want, asked)
		}
	}

	// The operator's own notes come first, because theirs is the word that
	// settles a disagreement between the two.
	if strings.Index(asked, "## Operator notes") > strings.Index(asked, head) {
		t.Errorf("the reviewers are read before the operator:\n%s", asked)
	}
}

// TestACutNeverSeversARune.
//
// The record is UTF-8 and the limit is a count of bytes, so a cut landing
// inside a character would come back from the log as a replacement
// character — in the middle of the one line somebody is reading to find out
// what went wrong.
func TestACutNeverSeversARune(t *testing.T) {
	// Every character is three bytes, so a limit that is not a multiple of
	// three lands inside one.
	long := strings.Repeat("ñ", 100)

	for _, limit := range []int{50, 51, 52, 99, 100, 101} {
		text, full := cut(long, limit)
		if !utf8.ValidString(text) {
			t.Errorf("a cut at %d severed a character: %q", limit, text)
		}

		if full != len(long) {
			t.Errorf("a cut at %d says there were %d bytes, want %d", limit, full, len(long))
		}

		if !strings.Contains(text, "truncated") {
			t.Errorf("a cut at %d says nothing about being cut: %q", limit, text)
		}
	}

	// And what fits is handed over whole, with nothing said about a cut
	// that did not happen.
	short := strings.Repeat("ñ", 10)
	if text, full := cut(short, len(short)); text != short || full != 0 {
		t.Errorf("a string of exactly the limit came back as %q with %d bytes reported", text, full)
	}
}
