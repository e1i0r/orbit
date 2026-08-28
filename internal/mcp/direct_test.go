package mcp

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

func TestDirectTaskRecordsDirectiveAndNote(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "DIR-MCP-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	res := sn.Call("orbit_direct_task", map[string]any{
		"task_id": "DIR-MCP-1",
		"message": "please use context timeout here",
	})
	if res.IsError {
		t.Fatalf("orbit_direct_task failed: %s", text(t, res))
	}

	path, err := s.EventsPath(r.Path, "DIR-MCP-1")
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	events, err := record.Read(path)
	if err != nil {
		t.Fatalf("record.Read: %v", err)
	}

	var foundDialogue, foundNote bool

	for _, e := range events {
		if e.Kind == record.TaskDialogue && e.Data["by"] == "mcp" {
			foundDialogue = true
		}

		if e.Kind == record.TaskNoted && e.Text == "[mcp] please use context timeout here" {
			foundNote = true
		}
	}

	if !foundDialogue {
		t.Error("orbit_direct_task did not record task.dialogue")
	}

	if !foundNote {
		t.Error("orbit_direct_task did not record task.noted")
	}
}

func TestDirectTaskRefusesEmptyMessage(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "DIR-MCP-2", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	res := sn.Call("orbit_direct_task", map[string]any{
		"task_id": "DIR-MCP-2",
		"message": "   ",
	})
	if !res.IsError {
		t.Fatal("orbit_direct_task on empty message did not return error")
	}
}
