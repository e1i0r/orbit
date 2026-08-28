package cli

// `orbit mcp` is two commands under one name: the server, which is what a
// client runs, and the installer, which is what tells the client to run it.
//
// Every test here redirects the home directory first. The installer writes
// into Claude's and Codex's real configuration, and the first version of
// this file did not: `go test ./...` rewrote a working orbit entry to point
// at the test binary, which is a path that stops existing when the test ends.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/mcp"
)

// installHome points everything the installer reads at a directory of the
// test's own.
func installHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", t.TempDir())
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))

	return home
}

// claudeCode is the entry the installer wrote for Claude Code, which keeps
// its server list in one file at the top of the home directory.
func claudeCode(t *testing.T, home string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		t.Fatalf("read the configuration the installer wrote: %v", err)
	}

	var config struct {
		Servers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("the installer wrote something that is not JSON: %v\n%s", err, raw)
	}

	entry, ok := config.Servers["orbit"]
	if !ok {
		t.Fatalf("the configuration has no orbit entry:\n%s", raw)
	}

	if !filepath.IsAbs(entry.Command) {
		t.Errorf("command = %q, want an absolute path: a client spawns this years later, from a working directory of its own", entry.Command)
	}

	if len(entry.Args) != 1 || entry.Args[0] != "mcp" {
		t.Errorf("args = %v, want [mcp] — that is the command the server is", entry.Args)
	}
}

func TestMCPInstallFlagRegistersTheServer(t *testing.T) {
	home := installHome(t)

	code, out, errOut := run(t, "mcp", "-install")
	if code != 0 {
		t.Fatalf("mcp -install exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, filepath.Join(home, ".claude.json")) {
		t.Errorf("the output does not say where it wrote:\n%s", out)
	}

	claudeCode(t, home)
}

func TestMCPInstallSubcommandDoesTheSameThing(t *testing.T) {
	home := installHome(t)

	code, _, errOut := run(t, "mcp", "install")
	if code != 0 {
		t.Fatalf("mcp install exited %d: %s", code, errOut)
	}

	claudeCode(t, home)
}

// Nothing outside the home the installer was given is touched, which is the
// property the whole redirection above exists to check.
func TestMCPInstallTouchesNothingElse(t *testing.T) {
	home := installHome(t)

	elsewhere := filepath.Join(t.TempDir(), "claude.json")
	if err := os.WriteFile(elsewhere, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if code, _, errOut := run(t, "mcp", "install"); code != 0 {
		t.Fatalf("mcp install exited %d: %s", code, errOut)
	}

	got, err := os.ReadFile(elsewhere)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(got) != "{}\n" {
		t.Errorf("a configuration outside the home directory was rewritten:\n%s", got)
	}

	if _, err := os.Stat(filepath.Join(home, ".claude.json")); err != nil {
		t.Errorf("the one inside it was not: %v", err)
	}
}

// `orbit mcp` with no flags is the server, for ever: it is the command line
// every install writes into every client, so anything else after mcp is a
// mistake and not a new subcommand.
func TestMCPTakesNoOtherArgument(t *testing.T) {
	installHome(t)

	code, _, errOut := run(t, "mcp", "serve")
	if code == 0 {
		t.Fatal("mcp serve exited 0, want it refused")
	}

	if !strings.Contains(errOut, "serve") {
		t.Errorf("the error does not name the argument it did not understand: %s", errOut)
	}
}

// TestTheInstallHelpNamesTheClientsItWritesTo. The flag carried a list of
// its own and it had drifted from the installer twice over: it offered
// Cursor, which this installer has never written a line of configuration
// for, and it named none of the three clients added since. Somebody read
// that help, installed for a client that was not touched, and never heard
// about the ones that were.
func TestTheInstallHelpNamesTheClientsItWritesTo(t *testing.T) {
	installHome(t)

	_, out, errOut := run(t, "mcp", "-h")

	help := out + errOut
	for _, name := range mcp.ClientNames() {
		if !strings.Contains(help, name) {
			t.Errorf("the help does not name %q, which `orbit mcp install` writes to:\n%s", name, help)
		}
	}

	if strings.Contains(help, "Cursor") {
		t.Errorf("the help offers Cursor and the installer does not write to it:\n%s", help)
	}
}
