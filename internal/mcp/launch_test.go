package mcp

// What an interactive session is handed so that it can ask Orbit anything.

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestASessionIsHandedOrbitAsOneLineOfConfiguration.
//
// The interactive session `c` opens is a coding agent started in a task's
// worktree, and what makes it able to ask Orbit anything is this: the server
// named, the binary that answers, and the one argument that starts it.
func TestASessionIsHandedOrbitAsOneLineOfConfiguration(t *testing.T) {
	raw, err := LaunchConfig("/usr/local/bin/orbit")
	if err != nil {
		t.Fatalf("the configuration could not be written: %v", err)
	}

	var doc struct {
		Servers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}

	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("what was handed over is not JSON: %v", err)
	}

	got, held := doc.Servers[serverName]
	if !held {
		t.Fatalf("the configuration names %v, want orbit among them", doc.Servers)
	}

	if got.Command != "/usr/local/bin/orbit" {
		t.Errorf("the session would run %q", got.Command)
	}

	if len(got.Args) != 1 || got.Args[0] != "mcp" {
		t.Errorf("the session would run it with %v", got.Args)
	}

	// With no binary named it falls back to this one, so a session started
	// from a build in a temp directory reaches that build and not whatever
	// orbit is on the path.
	bare, err := LaunchConfig("")
	if err != nil {
		t.Fatalf("a configuration with no binary named: %v", err)
	}

	if !strings.Contains(bare, "command") {
		t.Errorf("the configuration is %q", bare)
	}
}
