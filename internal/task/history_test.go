package task

import (
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestTheHistoryCarriesEveryEngineTheTaskPassedThrough.
//
// The scenario this exists for: claude runs out of quota, gemini takes
// over, gemini runs out, and it comes back to claude. What the first two
// said has to still be there when the third arrives, or the task cannot be
// finished by anybody who was not watching.
func TestTheHistoryCarriesEveryEngineTheTaskPassedThrough(t *testing.T) {
	tk := Task{ID: "ACME-40", Text: "add Total, in cents\nand round nothing"}

	body := historyOf(tk, []record.Event{
		{Kind: record.TaskDialogue, Text: "use cents", Data: map[string]string{"by": "operator"}},
		{Kind: record.TaskDialogue, Text: "added Total", Data: map[string]string{"by": "claude"}},
		{Kind: record.TaskDialogue, Text: "the rounding was the bug", Data: map[string]string{"by": "gemini"}},
		{Kind: record.TaskDialogue, Text: "and the test covers it now", Data: map[string]string{"by": "claude"}},
	})

	for _, want := range []string{"use cents", "added Total", "the rounding was the bug", "and the test covers it now"} {
		if !strings.Contains(body, want) {
			t.Errorf("the history lost %q:\n%s", want, body)
		}
	}

	for _, who := range []string{"operator", "claude", "gemini"} {
		if !strings.Contains(body, who) {
			t.Errorf("the history does not say %q said anything:\n%s", who, body)
		}
	}

	if at, then := strings.Index(body, "use cents"), strings.Index(body, "the rounding"); at > then {
		t.Error("the history is out of order")
	}
}

// TestTheBriefIsTheHeadingAndNotTheFirstTurn. It is what the whole
// conversation is about; repeated as a turn it would read as somebody
// opening every session by reciting the task at it.
func TestTheBriefIsTheHeadingAndNotTheFirstTurn(t *testing.T) {
	tk := Task{ID: "ACME-41", Text: "make the endpoint reject negatives"}

	body := historyOf(tk, []record.Event{
		{Kind: record.TaskDialogue, Text: "start with the router", Data: map[string]string{"by": "operator"}},
	})

	if strings.Count(body, "make the endpoint reject negatives") != 1 {
		t.Errorf("the brief is in the history twice:\n%s", body)
	}
}

// TestATaskNobodyHasSpokenAboutSaysSo, rather than answering an empty file
// that reads as a history that failed to load.
func TestATaskNobodyHasSpokenAboutSaysSo(t *testing.T) {
	body := historyOf(Task{ID: "ACME-42", Text: "later"}, nil)
	if !strings.Contains(body, "Nothing has been said") {
		t.Errorf("an empty history says nothing about being empty:\n%s", body)
	}
}

// TestTheHistoryLandsInTheTasksOwnDirectory, which outlives every worktree:
// a checkout can be pruned, and the conversation must not go with it.
func TestTheHistoryLandsInTheTasksOwnDirectory(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-43", "add Total", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Dialogue(s, tk, "claude", "I added it"); err != nil {
		t.Fatalf("Dialogue: %v", err)
	}

	where, err := WriteHistory(s, tk)
	if err != nil {
		t.Fatalf("WriteHistory: %v", err)
	}

	dir, err := s.TaskDir(tk.ID)
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	if !strings.HasPrefix(where, dir) {
		t.Errorf("the history went to %q, outside the task's directory %q", where, dir)
	}

	body, err := os.ReadFile(where)
	if err != nil {
		t.Fatalf("read it back: %v", err)
	}

	if !strings.Contains(string(body), "I added it") {
		t.Errorf("the file does not carry what was said:\n%s", body)
	}
}
