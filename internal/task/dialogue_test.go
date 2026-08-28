package task

// Dialogue writes into the same record a note does, and the one thing that
// must never be true of it is that a phase reads it.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

func TestDialogueRecordsWhatActedAndWhatItDid(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "DIA-1", "dialogue test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := Dialogue(s, tk, "mcp", "a model cancelled this task over mcp"); err != nil {
		t.Fatalf("Dialogue: %v", err)
	}
	e := lastEvent(t, s, tk)
	if e.Kind != record.TaskDialogue {
		t.Errorf("kind = %q, want %q", e.Kind, record.TaskDialogue)
	}
	if e.Text != "a model cancelled this task over mcp" {
		t.Errorf("text = %q", e.Text)
	}
	if e.Data["by"] != "mcp" {
		t.Errorf("by = %q, want the thing that acted", e.Data["by"])
	}
}

// The whole reason this is not a note: the next phase to start is handed
// every note recorded since the last one, and "a model paused this" read as
// an instruction is the engine being told to do something nobody asked for.
func TestDialogueIsNotHandedToTheNextPhase(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "DIA-2", "dialogue test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := Note(s, tk, "use the sql migration"); err != nil {
		t.Fatalf("Note: %v", err)
	}
	if err := Dialogue(s, tk, "mcp", "a model asked this task to pause over mcp"); err != nil {
		t.Fatalf("Dialogue: %v", err)
	}
	notes := unconsumedNotes(s, tk)
	if len(notes) != 1 || notes[0] != "use the sql migration" {
		t.Errorf("the next phase is handed %v, want only the note somebody wrote", notes)
	}
}

func TestDialogueRefusesEmptyTextAndKeepsByOptional(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "DIA-3", "dialogue test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := Dialogue(s, tk, "mcp", "   "); err == nil {
		t.Error("Dialogue on blank text answered nil, want it refused")
	}
	if err := Dialogue(s, tk, "  ", "somebody did something"); err != nil {
		t.Fatalf("Dialogue with no actor: %v", err)
	}
	if e := lastEvent(t, s, tk); e.Data != nil {
		t.Errorf("data = %v, want none: nothing is known about what acted", e.Data)
	}
	if err := Dialogue(s, Task{ID: "has/slash", Repo: r}, "mcp", "did something"); err == nil {
		t.Error("Dialogue on an id with no events path answered nil, want it refused")
	}
}

// lastEvent is the most recent line of a task's record.
func lastEvent(t *testing.T, s *store.Store, tk Task) record.Event {
	t.Helper()
	path, err := s.EventsPath(tk.Repo.Path, tk.ID)
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}
	events, err := record.Read(path)
	if err != nil {
		t.Fatalf("record.Read: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("the record is empty")
	}
	return events[len(events)-1]
}
