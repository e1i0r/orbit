package mcp

import (
	"testing"
)

func TestSupervisorSayAndHistoryTools(t *testing.T) {
	_, sn, _ := oneRepo(t)

	// Post message 1
	res1 := sn.Call("orbit_supervisor_say", map[string]any{
		"message": "keep strict TDD on all features",
		"by":      "elio",
		"channel": "tui",
	})
	if res1.IsError {
		t.Fatalf("orbit_supervisor_say failed: %s", text(t, res1))
	}

	// Post message 2
	res2 := sn.Call("orbit_supervisor_say", map[string]any{
		"message": "understood, monitoring tasks",
		"by":      "claude",
		"channel": "mcp",
	})
	if res2.IsError {
		t.Fatalf("orbit_supervisor_say 2 failed: %s", text(t, res2))
	}

	// Read full history
	hRes := sn.Call("orbit_supervisor_history", nil)
	if hRes.IsError {
		t.Fatalf("orbit_supervisor_history failed: %s", text(t, hRes))
	}

	history := call(t, sn, "orbit_supervisor_history", nil)

	msgs, ok := history["messages"].([]any)
	if !ok || len(msgs) != 2 {
		t.Fatalf("history messages = %v, want 2", history["messages"])
	}

	// Read with limit 1
	hLimit := call(t, sn, "orbit_supervisor_history", map[string]any{"limit": float64(1)})

	limitMsgs, ok := hLimit["messages"].([]any)
	if !ok || len(limitMsgs) != 1 {
		t.Fatalf("history with limit 1 = %v, want 1", hLimit["messages"])
	}
}

func TestSupervisorSayRefusesEmptyMessage(t *testing.T) {
	_, sn, _ := oneRepo(t)

	res := sn.Call("orbit_supervisor_say", map[string]any{
		"message": "   ",
	})
	if !res.IsError {
		t.Fatal("orbit_supervisor_say on empty message did not refuse")
	}
}
