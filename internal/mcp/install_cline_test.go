package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

// TestClineKeepsTheServersItAlreadyHad. cline's MCP file holds the servers
// the reader set up in cline itself, some with their sign-ins; orbit is
// added beside them, never over them.
func TestClineKeepsTheServersItAlreadyHad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CLINE_MCP_SETTINGS_PATH", "")
	t.Setenv("CLINE_DATA_DIR", "")

	path := filepath.Join(home, ".cline", "data", "settings", "cline_mcp_settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(`{"mcpServers":{"linear":{"command":"npx","args":["linear"]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	Install("/opt/orbit/orbit", home, "")

	got := mapOf(t, path, "mcpServers")
	if got["linear"] == nil {
		t.Error("the server cline already had was dropped")
	}

	orbit, ok := got["orbit"].(map[string]any)
	if !ok || orbit["command"] != "/opt/orbit/orbit" {
		t.Errorf("orbit is registered as %v, want its command", got["orbit"])
	}
}

// TestClineIsFoundWhereCLINE_DIRPutsIt, as the transcript is: data under it.
func TestClineIsFoundWhereCLINE_DIRPutsIt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLINE_MCP_SETTINGS_PATH", "")
	t.Setenv("CLINE_DATA_DIR", "")
	t.Setenv("CLINE_DIR", root)

	want := filepath.Join(root, "data", "settings", "cline_mcp_settings.json")
	if got := clineConfig(t.TempDir()); got != want {
		t.Errorf("clineConfig = %q, want %q", got, want)
	}
}
