package task

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

func TestRecordSupervisorAppendsEvents(t *testing.T) {
	s, _ := fixture(t)

	if err := RecordSupervisor(nil, "", "elio", "tui", "", "", "hello"); err == nil {
		t.Error("RecordSupervisor on nil store answered nil, want error")
	}
	if err := RecordSupervisor(s, "", "elio", "tui", "", "", "   "); err == nil {
		t.Error("RecordSupervisor on blank text answered nil, want error")
	}

	if err := RecordSupervisor(s, record.SupervisorBriefing, "elio", "tui", "TASK-1", "payments", "validate all edge cases"); err != nil {
		t.Fatalf("RecordSupervisor: %v", err)
	}

	events, err := SupervisorEvents(s)
	if err != nil {
		t.Fatalf("SupervisorEvents: %v", err)
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
	s, _ := fixture(t)
	events, err := SupervisorEvents(s)
	if err != nil {
		t.Fatalf("SupervisorEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("events length = %d, want 0", len(events))
	}
	if _, err := SupervisorEvents(nil); err == nil {
		t.Error("SupervisorEvents on nil store answered nil, want error")
	}
}
