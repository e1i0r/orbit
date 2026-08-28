package mcp

// What the installer must never do: lose something it did not write. Every
// test here installs into a temporary home, which is the point — see the
// note at the top of fixture_test.go for what happened when one of them did
// not.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// servers reads one configuration file back and answers its mcpServers map.
func servers(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse %q: %v\n%s", path, err, data)
	}

	got, ok := config["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("%q has no mcpServers map: %s", path, data)
	}

	return got
}

// TestInstallLeavesEverythingElseAlone is the whole of what a merge means: a
// client's other servers and a client's other keys are still there
// afterwards, because they are not Orbit's to touch.
func TestInstallLeavesEverythingElseAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	before := `{
	  "numStartups": 41,
	  "mcpServers": {"chrome": {"command": "node", "args": ["chrome.js"]}},
	  "theme": "dark"
	}`
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := register(path, "/usr/local/bin/orbit"); err != nil {
		t.Fatalf("register: %v", err)
	}

	got := servers(t, path)
	if got["chrome"] == nil {
		t.Error("the client's own mcp server is gone: a merge that drops the other entries is a replacement")
	}

	orbit, ok := got["orbit"].(map[string]any)
	if !ok {
		t.Fatalf("orbit was not registered: %v", got)
	}

	if orbit["command"] != "/usr/local/bin/orbit" {
		t.Errorf("orbit is registered as %v, want /usr/local/bin/orbit", orbit["command"])
	}

	args, ok := orbit["args"].([]any)
	if !ok || len(args) != 1 || args[0] != "mcp" {
		t.Errorf("orbit is registered with args %v, want [mcp] — the installer writes the command line the client will run", orbit["args"])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}

	if config["theme"] != "dark" || config["numStartups"] == nil {
		t.Errorf("the client's own keys did not survive the install: %s", data)
	}
}

// TestInstallIsIdempotent: running it twice leaves one entry, not two under
// two spellings, and not a file that grows every time.
func TestInstallIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	for i := range 2 {
		if err := register(path, "/usr/local/bin/orbit"); err != nil {
			t.Fatalf("register %d: %v", i+1, err)
		}
	}

	got := servers(t, path)
	if len(got) != 1 {
		t.Errorf("two installs left %d servers, want 1: %v", len(got), got)
	}
}

// TestInstallCreatesTheFileAndItsDirectory: a client that has never had a
// server configured has neither, and making both is the install.
func TestInstallCreatesTheFileAndItsDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "never", "existed", "mcp.json")
	if err := register(path, "/usr/local/bin/orbit"); err != nil {
		t.Fatalf("register: %v", err)
	}

	if servers(t, path)["orbit"] == nil {
		t.Error("orbit was not registered into a configuration that had to be created")
	}
}

// TestInstallRefusesAConfigurationItCannotParse is the finding this test
// exists for: the first version started from an empty map when the decode
// failed, which turned one unreadable ~/.claude.json — a hundred kilobytes
// of a client's own state — into a file holding an orbit entry and nothing
// else. Refusing leaves the damage for whoever can repair it.
func TestInstallRefusesAConfigurationItCannotParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	damaged := `{"mcpServers": {"chrome": {`
	if err := os.WriteFile(path, []byte(damaged), 0o600); err != nil {
		t.Fatal(err)
	}

	err := register(path, "/usr/local/bin/orbit")
	if err == nil {
		t.Fatal("a configuration that will not parse was overwritten rather than refused")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("the refusal does not name the file: %v", err)
	}

	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}

	if string(after) != damaged {
		t.Errorf("the damaged configuration was modified:\n%s", after)
	}
}

// TestInstallKeepsTheFileMode: these files hold a client's credentials often
// enough that widening 0600 to 0644 on every install would be a real
// disclosure.
func TestInstallKeepsTheFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not what they are here on windows")
	}

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := register(path, "/usr/local/bin/orbit"); err != nil {
		t.Fatalf("register: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("the configuration is now mode %v, want 0600", info.Mode().Perm())
	}
}

// TestInstallWritesEveryClientUnderTheGivenHome walks the whole of Install
// against a home directory of the test's own.
func TestInstallWritesEveryClientUnderTheGivenHome(t *testing.T) {
	home := t.TempDir()

	results := Install("/opt/orbit/orbit", home)
	if len(results) < 2 {
		t.Fatalf("Install touched %d clients, want at least Claude Code and OpenCode", len(results))
	}

	for _, res := range results {
		if res.Status != StatusInstalled || res.Err != nil {
			t.Errorf("%s was not installed: %v", res.Target, res.Err)
			continue
		}

		if !strings.HasPrefix(res.Path, home) {
			t.Errorf("%s was written to %q, which is outside the home the installer was given", res.Target, res.Path)
			continue
		}

		if servers(t, res.Path)["orbit"] == nil {
			t.Errorf("%s at %q has no orbit entry", res.Target, res.Path)
		}
	}
}

// TestClaudeCodeIsWhereClaudeCodeLooks. The obvious guess is ~/.claude/mcp.json
// and it is wrong — nothing reads that file. Claude Code keeps its
// user-scope servers in ~/.claude.json, alongside the rest of its state,
// which is why register merges and writes through a rename.
func TestClaudeCodeIsWhereClaudeCodeLooks(t *testing.T) {
	home := t.TempDir()

	var found string

	for _, c := range clientConfigs(home) {
		if c.name == "Claude Code" {
			found = c.path
		}
	}

	if want := filepath.Join(home, ".claude.json"); found != want {
		t.Errorf("Claude Code is installed into %q, want %q", found, want)
	}
}

// TestInstallWithoutAHomeTouchesNothing: an operating system that cannot
// name a home directory is not a reason to write configuration into the
// filesystem root.
func TestInstallWithoutAHomeTouchesNothing(t *testing.T) {
	if results := Install("/opt/orbit/orbit", ""); len(results) != 0 {
		t.Errorf("Install with no home touched %d configurations: %+v", len(results), results)
	}
}
