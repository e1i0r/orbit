package task

import (
	"context"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

func TestDirectRefusesEmptyMessage(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIR-1", "direct test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Direct(s, tk, "supervisor", "   "); err == nil {
		t.Error("Direct on empty text answered nil, want error")
	}
}

func TestDirectRecordsDialogueAndNoteWhenNotRunning(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIR-2", "direct test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Direct(s, tk, "supervisor", "change approach to use redis"); err != nil {
		t.Fatalf("Direct: %v", err)
	}

	path, err := s.EventsPath(tk.Repo.Path, tk.ID)
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	events, err := record.Read(path)
	if err != nil {
		t.Fatalf("record.Read: %v", err)
	}

	var foundDialogue, foundNote bool

	for _, e := range events {
		if e.Kind == record.TaskDialogue && e.Data["by"] == "supervisor" {
			foundDialogue = true
		}

		if e.Kind == record.TaskNoted && e.Text == "[supervisor] change approach to use redis" {
			foundNote = true
		}
	}

	if !foundDialogue {
		t.Errorf("task.dialogue event not found in record")
	}

	if !foundNote {
		t.Errorf("task.noted event not found in record")
	}

	notes, err := unconsumedNotes(s, tk)
	if err != nil {
		t.Fatal(err)
	}

	if len(notes) != 1 || notes[0] != "[supervisor] change approach to use redis" {
		t.Errorf("unconsumedNotes = %v", notes)
	}
}

func TestDirectDefaultBy(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIR-3", "direct test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Direct(s, tk, "", "focus on unit tests"); err != nil {
		t.Fatalf("Direct with empty by: %v", err)
	}

	notes, err := unconsumedNotes(s, tk)
	if err != nil {
		t.Fatal(err)
	}

	if len(notes) != 1 || notes[0] != "[supervisor] focus on unit tests" {
		t.Errorf("unconsumedNotes with default by = %v", notes)
	}
}

func TestReopenReturnsErrorWhenDirectFails(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DIR-4", "direct test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := Reopen(context.Background(), s, tk, "supervisor", "", "quick", 0); err == nil {
		t.Error("Reopen on empty message answered nil, want error")
	}
}
