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

	if err := registerJSON(path, "mcpServers", entry("/usr/local/bin/orbit")); err != nil {
		t.Fatalf("registerJSON: %v", err)
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
		if err := registerJSON(path, "mcpServers", entry("/usr/local/bin/orbit")); err != nil {
			t.Fatalf("registerJSON %d: %v", i+1, err)
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
	if err := registerJSON(path, "mcpServers", entry("/usr/local/bin/orbit")); err != nil {
		t.Fatalf("registerJSON: %v", err)
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

	err := registerJSON(path, "mcpServers", entry("/usr/local/bin/orbit"))
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

	if err := registerJSON(path, "mcpServers", entry("/usr/local/bin/orbit")); err != nil {
		t.Fatalf("registerJSON: %v", err)
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
// against a home directory of the test's own, and reads each file back the
// way the client that owns it would: an install is not a file that was
// written, it is an entry that client can find.
func TestInstallWritesEveryClientUnderTheGivenHome(t *testing.T) {
	home := t.TempDir()

	results := Install("/opt/orbit/orbit", home)
	if len(results) != len(ClientNames()) {
		t.Fatalf("Install touched %d clients, want %d", len(results), len(ClientNames()))
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

		if !registered(t, res) {
			t.Errorf("%s at %q has no orbit entry the client would find", res.Target, res.Path)
		}
	}
}

// registered asks the file whether the client that reads it would find
// Orbit, in whichever of the three shapes that client uses.
func registered(t *testing.T, res InstallResult) bool {
	t.Helper()

	switch res.Target {
	case "Codex":
		data, err := os.ReadFile(res.Path)
		if err != nil {
			t.Fatalf("read %q: %v", res.Path, err)
		}

		return strings.Contains(string(data), "[mcp_servers.orbit]")
	case "OpenCode":
		return mapOf(t, res.Path, "mcp")["orbit"] != nil
	default:
		return servers(t, res.Path)["orbit"] != nil
	}
}

// TestEveryClientIsGivenTheFileItActuallyReads is the finding, and the
// evidence for it was three files on a real machine: ~/.codex/config.json
// beside the config.toml Codex reads, ~/.gemini/config.json beside the
// settings.json Gemini reads, and an OpenCode file under a key OpenCode
// ignores. All three installs reported success.
func TestEveryClientIsGivenTheFileItActuallyReads(t *testing.T) {
	home := t.TempDir()

	want := map[string]string{
		"Claude Code": filepath.Join(home, ".claude.json"),
		"Codex":       filepath.Join(home, ".codex", "config.toml"),
		"Gemini":      filepath.Join(home, ".gemini", "settings.json"),
		"OpenCode":    filepath.Join(home, ".config", "opencode", "opencode.json"),
	}
	for _, c := range clientConfigs(home) {
		if expected, ok := want[c.name]; ok && c.path != expected {
			t.Errorf("%s is installed into %q, and it reads %q", c.name, c.path, expected)
		}
	}
}

// TestOpenCodeIsWrittenInTheShapeOpenCodeReads. The path was right and the
// contents were Claude's: a servers map called mcpServers, holding a command
// and its arguments. OpenCode reads a map called mcp, holding a type and one
// argv, and parsed the other quite happily while running nothing.
func TestOpenCodeIsWrittenInTheShapeOpenCodeReads(t *testing.T) {
	home := t.TempDir()
	Install("/opt/orbit/orbit", home)

	got, ok := mapOf(t, filepath.Join(home, ".config", "opencode", "opencode.json"), "mcp")["orbit"].(map[string]any)
	if !ok {
		t.Fatal("opencode.json has no orbit entry under mcp")
	}

	if got["type"] != "local" {
		t.Errorf("orbit is registered as type %v, want local", got["type"])
	}

	command, ok := got["command"].([]any)
	if !ok || len(command) != 2 || command[0] != "/opt/orbit/orbit" || command[1] != "mcp" {
		t.Errorf("orbit is registered with command %v, want the whole argv as a list", got["command"])
	}
}

// TestOpenCodeWritesIntoTheConfigurationAlreadyThere: opencode.jsonc is the
// other name OpenCode reads, and writing opencode.json beside one leaves two
// configurations where there was one.
func TestOpenCodeWritesIntoTheConfigurationAlreadyThere(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	commented := filepath.Join(dir, "opencode.jsonc")
	if err := os.WriteFile(commented, []byte(`{"$schema": "https://opencode.ai/config.json"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	Install("/opt/orbit/orbit", home)

	if mapOf(t, commented, "mcp")["orbit"] == nil {
		t.Error("the configuration that was already there has no orbit entry")
	}

	if _, err := os.Stat(filepath.Join(dir, "opencode.json")); err == nil {
		t.Error("a second configuration file was written beside the one that was already there")
	}
}

// mapOf reads one map out of a JSON configuration file.
func mapOf(t *testing.T, path, key string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse %q: %v\n%s", path, err, data)
	}

	got, ok := config[key].(map[string]any)
	if !ok {
		t.Fatalf("%q has no %s map: %s", path, key, data)
	}

	return got
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
