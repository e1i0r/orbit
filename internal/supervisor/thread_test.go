package supervisor

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

func TestRecordSupervisorAppendsEvents(t *testing.T) {
	s := fixture(t)

	if err := Record(nil, "", "elio", "tui", "", "", "hello"); err == nil {
		t.Error("Record on nil store answered nil, want error")
	}

	if err := Record(s, "", "elio", "tui", "", "", "   "); err == nil {
		t.Error("Record on blank text answered nil, want error")
	}

	if err := Record(s, record.SupervisorBriefing, "elio", "tui", "TASK-1", "payments", "validate all edge cases"); err != nil {
		t.Fatalf("Record: %v", err)
	}

	events, err := Events(s)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("events length = %d, want 1", len(events))
	}

	e := events[0]
	if e.Kind != record.SupervisorBriefing {
		t.Errorf("e.Kind = %q, want %q", e.Kind, record.SupervisorBriefing)
	}

	if e.Text != "validate all edge cases" {
		t.Errorf("e.Text = %q", e.Text)
	}

	if e.Data["by"] != "elio" || e.Data["channel"] != "tui" || e.Data["task_id"] != "TASK-1" || e.Data["repo"] != "payments" {
		t.Errorf("e.Data = %+v", e.Data)
	}
}

func TestSupervisorEventsReturnsEmptyOnMissingLog(t *testing.T) {
	s := fixture(t)

	events, err := Events(s)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("events length = %d, want 0", len(events))
	}

	if _, err := Events(nil); err == nil {
		t.Error("Events on nil store answered nil, want error")
	}
}

// TestRemovingAConversationTakesItOffTheList. The turns stay in the
// record; the list stops naming it.
func TestRemovingAConversationTakesItOffTheList(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	if err := Record(s, "", "operator", "cli", "", "", "never force-push"); err != nil {
		t.Fatalf("record: %v", err)
	}

	id, err := Current(s)
	if err != nil {
		t.Fatalf("current: %v", err)
	}

	if err := Remove(s, id); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if err := Remove(nil, id); err == nil {
		t.Error("removing with no store was accepted")
	}
}
