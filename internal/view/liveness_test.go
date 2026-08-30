package view

// The word each Liveness answers to. It leaves this package as a string —
// the MCP server hands it to a model as `live` — so the three states have to
// arrive as three words, and the one that matters most is the one a bool had
// no room for.

import "testing"

func TestEachLivenessSaysWhichOfTheThreeItIs(t *testing.T) {
	for state, want := range map[Liveness]string{
		LiveFree:    "free",
		LiveHeld:    "held",
		LiveUnknown: "unknown",
	} {
		if got := state.String(); got != want {
			t.Errorf("Liveness(%d) says %q, want %q", int(state), got, want)
		}
	}
}
