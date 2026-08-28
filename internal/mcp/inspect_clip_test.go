package mcp

// The budgets an inspection is cut to, exercised on text that is not
// English. That is the only kind that can catch what these guard: a cut at a
// byte offset splits whatever character spans it, and every character in an
// ASCII log is one byte wide, so a test written in English passes either way.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/record"
)

func TestClipCutsCharactersRatherThanBytes(t *testing.T) {
	got := clip(strings.Repeat("é", 100), 5)

	if !utf8.ValidString(got) {
		t.Fatalf("clip answered text that is not valid UTF-8: %q", got)
	}

	if !strings.HasPrefix(got, strings.Repeat("é", 5)) {
		t.Errorf("clip kept %q, want the first five characters", got)
	}

	if !strings.Contains(got, "95 more characters") {
		t.Errorf("clip counted the remainder in bytes rather than characters: %q", got)
	}
}

func TestTailKeepsTheEndItPromises(t *testing.T) {
	got := tail(strings.Repeat("é", 100)+"FIN", 5)

	if !utf8.ValidString(got) {
		t.Fatalf("tail answered text that is not valid UTF-8: %q", got)
	}

	if !strings.HasSuffix(got, "ééFIN") {
		t.Errorf("tail kept %q, want the last five characters", got)
	}

	if !strings.Contains(got, "98 earlier characters omitted") {
		t.Errorf("tail counted what it dropped in bytes rather than characters: %q", got)
	}
}

func TestTextInsideTheBudgetIsLeftAlone(t *testing.T) {
	whole := strings.Repeat("é", 30)
	if got := clip(whole, 30); got != whole {
		t.Errorf("clip shortened text that fits: %q", got)
	}

	if got := tail(whole, 30); got != whole {
		t.Errorf("tail shortened text that fits: %q", got)
	}
}

// TestAnInspectionOfThinkingThatIsNotEnglishArrivesUndamaged is the defect
// as the caller meets it. A half character is not valid UTF-8, the encoder
// replaces it with U+FFFD on the way out, and the answer carries no field
// saying anything was damaged — so a supervisor reads a replacement mark
// where a word was and has no reason to doubt it.
func TestAnInspectionOfThinkingThatIsNotEnglishArrivesUndamaged(t *testing.T) {
	s, sn, r := oneRepo(t)

	addTask(t, s, r, "PAY-9",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"},
		record.Event{At: at(2), Kind: record.PhaseThought, Phase: "implement", Text: strings.Repeat("é", thoughtChars+50)},
	)

	thinking := list(t, call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-9"})["thinking"])
	if len(thinking) != 1 {
		t.Fatalf("the inspection carried %d thoughts, want 1", len(thinking))
	}

	thought := str(t, obj(t, thinking[0])["text"])
	if n := strings.Count(thought, "�"); n > 0 {
		t.Errorf("the thought came back with %d replacement characters in it, so the cut split one", n)
	}

	if !strings.HasPrefix(thought, strings.Repeat("é", thoughtChars)) {
		t.Error("the thought was not kept up to the budget it is cut at")
	}
}
