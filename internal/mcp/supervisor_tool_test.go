package mcp

import (
	"strings"
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

// TestALimitIsReadTheWayEveryOtherArgumentIs. The limit used to be the one
// argument read inline, with its own float64 assertion in the middle of the
// handler, while everything else went through a helper. Every case below is
// something a client can send, and the reading of each one is what keeps a
// bad limit from being a different bug than a bad anything else.
func TestALimitIsReadTheWayEveryOtherArgumentIs(t *testing.T) {
	for _, c := range []struct {
		in   any
		want int
		why  string
	}{
		{float64(3), 3, "JSON has one number type and it decodes to float64"},
		{float64(0), 0, "zero is what the caller reads as no limit"},
		{float64(-5), 0, "a negative limit would slice from the wrong end"},
		{float64(2.7), 2, "a fraction is a count of messages rounded down, not an error"},
		{"3", 0, "a number a model quoted is not a number"},
		{true, 0, "a boolean is not a count"},
		{nil, 0, "a key that is there and null is a key that said nothing"},
	} {
		if got := intArg(map[string]any{"limit": c.in}, "limit"); got != c.want {
			t.Errorf("intArg(%#v) = %d, want %d — %s", c.in, got, c.want, c.why)
		}
	}

	if got := intArg(nil, "limit"); got != 0 {
		t.Errorf("intArg(nil) = %d, want 0", got)
	}

	if got := intArg(map[string]any{}, "limit"); got != 0 {
		t.Errorf("intArg of a call with no arguments = %d, want 0", got)
	}
}

// TestAHistoryLimitKeepsTheEndOfTheThreadAndNotItsStart. A supervisor asking
// for the last two lines wants what it was last told; head-truncating gives
// it the two oldest, which is the half it already knew.
func TestAHistoryLimitKeepsTheEndOfTheThreadAndNotItsStart(t *testing.T) {
	_, sn, _ := oneRepo(t)

	for _, said := range []string{"first", "second", "third"} {
		if res := sn.Call("orbit_supervisor_say", map[string]any{"message": said}); res.IsError {
			t.Fatalf("orbit_supervisor_say %q: %s", said, text(t, res))
		}
	}

	got := call(t, sn, "orbit_supervisor_history", map[string]any{"limit": float64(2)})

	msgs, ok := got["messages"].([]any)
	if !ok || len(msgs) != 2 {
		t.Fatalf("history with limit 2 = %v, want 2 messages", got["messages"])
	}

	joined := str(t, obj(t, msgs[0])["text"]) + "\n" + str(t, obj(t, msgs[1])["text"])
	if strings.Contains(joined, "first") || !strings.Contains(joined, "third") {
		t.Errorf("a limit of 2 answered the start of the thread, not its end:\n%s", joined)
	}
}
