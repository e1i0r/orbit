package mcp

// The boundary a reader asked for, as the clients record it.

import "testing"

// TestTheRootIsRegisteredWithTheServer. `orbit mcp install -root ~/work` is
// asking for a server that acts on nothing outside that tree; an entry
// without it registered one that acts on everything and said it had worked.
func TestTheRootIsRegisteredWithTheServer(t *testing.T) {
	got := entry("/opt/orbit/orbit", "/home/me/work")

	args, ok := got["args"].([]string)
	if !ok {
		t.Fatalf("the entry carries args as %T", got["args"])
	}

	if len(args) != 3 || args[0] != "mcp" || args[1] != "-root" || args[2] != "/home/me/work" {
		t.Errorf("the entry runs %v, want the root behind mcp", args)
	}

	bare, ok := entry("/opt/orbit/orbit", "")["args"].([]string)
	if !ok {
		t.Fatalf("an install with no root carries args as %T", entry("/opt/orbit/orbit", "")["args"])
	}

	if len(bare) != 1 {
		t.Errorf("an install with no root registered %v", bare)
	}
}
