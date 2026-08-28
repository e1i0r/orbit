package mcp

// The Codex installer, which edits TOML as text. What it must never do is
// give back a file that is not the one it was handed, plus its own table.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// existing is a Codex configuration with the things a real one has in it:
// comments, settings, and a table whose name is a path.
const existing = `# my settings, do not lose them
model = "gpt-5.3-codex"
model_reasoning_effort = "xhigh"

[projects."/Users/someone/work/payments"]
trust_level = "trusted"
`

func codexFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func read(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}

	return string(data)
}

// TestCodexKeepsEverythingItDoesNotOwn is why this is a text edit and not a
// decode and an encode: the file holds somebody's model, their reasoning
// effort and their trusted projects, and a round trip through a TOML library
// gives all of that back with the comments gone and the keys reordered.
func TestCodexKeepsEverythingItDoesNotOwn(t *testing.T) {
	path := codexFile(t, existing)
	if err := registerCodex(path, "/usr/local/bin/orbit"); err != nil {
		t.Fatalf("registerCodex: %v", err)
	}

	got := read(t, path)
	if !strings.HasPrefix(got, existing) {
		t.Errorf("the file somebody else wrote came back changed:\n%s", got)
	}

	if !strings.Contains(got, "[mcp_servers.orbit]") {
		t.Errorf("orbit has no table in the file:\n%s", got)
	}

	if !strings.Contains(got, `command = "/usr/local/bin/orbit"`) || !strings.Contains(got, `args = ["mcp"]`) {
		t.Errorf("the table does not say what to run:\n%s", got)
	}
}

// TestCodexReplacesTheTableItWroteBefore: an install after an install is one
// table with the new path in it, and the tables around it untouched.
func TestCodexReplacesTheTableItWroteBefore(t *testing.T) {
	path := codexFile(t, existing+`
[mcp_servers.orbit]
command = "/old/orbit"
args = ["mcp"]

[mcp_servers.chrome]
command = "node"
args = ["chrome.js"]
`)
	if err := registerCodex(path, "/usr/local/bin/orbit"); err != nil {
		t.Fatalf("registerCodex: %v", err)
	}

	got := read(t, path)
	if n := strings.Count(got, "[mcp_servers.orbit]"); n != 1 {
		t.Errorf("the file has %d orbit tables, want 1:\n%s", n, got)
	}

	if strings.Contains(got, "/old/orbit") {
		t.Errorf("the old command is still there:\n%s", got)
	}

	if !strings.Contains(got, "[mcp_servers.chrome]") || !strings.Contains(got, `args = ["chrome.js"]`) {
		t.Errorf("somebody else's server did not survive the edit:\n%s", got)
	}
}

// TestCodexLeavesAnInlineOrbitEntryAlone. Orbit written as a key of
// [mcp_servers] rather than as a table of its own is the same entry in a
// shape this cannot replace, and appending a table beside it would be a
// duplicate key that stops the whole file parsing.
func TestCodexLeavesAnInlineOrbitEntryAlone(t *testing.T) {
	before := existing + `
[mcp_servers]
orbit = { command = "/somewhere/orbit", args = ["mcp"] }
`
	path := codexFile(t, before)

	err := registerCodex(path, "/usr/local/bin/orbit")
	if err == nil {
		t.Fatal("an entry this cannot edit was written over rather than refused")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("the refusal does not name the file: %v", err)
	}

	if got := read(t, path); got != before {
		t.Errorf("the file was modified anyway:\n%s", got)
	}
}

// TestCodexCreatesTheFileWhenThereIsNone: a Codex that has never been
// configured has no config.toml, and making one is the install.
func TestCodexCreatesTheFileWhenThereIsNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "never", "existed", "config.toml")
	if err := registerCodex(path, "/usr/local/bin/orbit"); err != nil {
		t.Fatalf("registerCodex: %v", err)
	}

	if got := read(t, path); !strings.HasPrefix(got, "[mcp_servers.orbit]") {
		t.Errorf("the file that was created is not the table:\n%s", got)
	}
}

// TestAPathIsQuotedForATOMLReader. A Windows path is mostly backslashes and
// every one of them is an escape to a TOML reader, so the unescaped path
// would come back as something else or not parse at all.
func TestAPathIsQuotedForATOMLReader(t *testing.T) {
	got := tomlString(`C:\Users\someone\bin\orbit.exe`)
	if want := `"C:\\Users\\someone\\bin\\orbit.exe"`; got != want {
		t.Errorf("tomlString = %s, want %s", got, want)
	}
}

// TestATableIsTheSameTableHoweverItIsSpelled: [mcp_servers."orbit"] is the
// table [mcp_servers.orbit] is, and an installer that could not see that
// would write a second one.
func TestATableIsTheSameTableHoweverItIsSpelled(t *testing.T) {
	path := codexFile(t, "[mcp_servers.\"orbit\"]\ncommand = \"/old/orbit\"\nargs = [\"mcp\"]\n")
	if err := registerCodex(path, "/usr/local/bin/orbit"); err != nil {
		t.Fatalf("registerCodex: %v", err)
	}

	if n := strings.Count(read(t, path), "orbit]"); n != 1 {
		t.Errorf("the quoted table was not recognised as ours:\n%s", read(t, path))
	}
}
