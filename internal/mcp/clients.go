package mcp

// Where each client keeps its MCP servers, and what it expects Orbit to look
// like once it is there.
//
// This was a list of names and paths, and every client on it was written the
// same JSON under the same key. Two of the five read it. The other three got
// a file at a path they do not open — ~/.codex/config.json beside the
// config.toml Codex actually reads, ~/.gemini/config.json beside the
// settings.json Gemini actually reads — or, in OpenCode's case, the right
// file carrying a key it ignores. The install reported success five times.

import (
	"os"
	"path/filepath"
	"runtime"
)

// clientConfig is one file an MCP client reads its server list from, and how
// that client expects Orbit to appear in it.
//
// The how is a function rather than a flag because the three shapes have
// nothing in common past the path of the binary: two JSON keys holding two
// different entries, and one TOML table.
type clientConfig struct {
	name     string
	path     string
	register func(configPath, binaryPath string) error
}

// clientConfigs is where each supported client keeps its MCP server list.
//
// Claude Code is the one worth writing down, because the obvious guess is
// wrong: it does not read ~/.claude/mcp.json. Its user-scope servers live in
// ~/.claude.json under the same mcpServers key Claude Desktop and Gemini
// use, alongside the rest of that file's state — which is why the registers
// below merge rather than write, and why they write through a rename.
func clientConfigs(home string) []clientConfig {
	if home == "" {
		return nil
	}

	var targets []clientConfig
	if p := claudeDesktopConfig(home); p != "" {
		targets = append(targets, jsonClient("Claude Desktop", p, "mcpServers", entry))
	}

	return append(targets,
		jsonClient("Claude Code", filepath.Join(home, ".claude.json"), "mcpServers", entry),
		jsonClient("OpenCode", opencodeConfig(home), "mcp", opencodeEntry),
		clientConfig{name: "Codex", path: filepath.Join(home, ".codex", "config.toml"), register: registerCodex},
		jsonClient("Gemini", filepath.Join(home, ".gemini", "settings.json"), "mcpServers", entry),
	)
}

// jsonClient is a client that keeps a map of servers in a JSON file.
func jsonClient(name, path, key string, shape func(binaryPath string) map[string]any) clientConfig {
	return clientConfig{
		name: name,
		path: path,
		register: func(configPath, binaryPath string) error {
			return registerJSON(configPath, key, shape(binaryPath))
		},
	}
}

// opencodeEntry is Orbit as OpenCode reads one: a local server whose command
// is one argv rather than a command and its arguments, under a key of its
// own. Written in the Claude shape it parses and does nothing.
func opencodeEntry(binaryPath string) map[string]any {
	return map[string]any{
		"type":    "local",
		"command": []string{binaryPath, "mcp"},
		"enabled": true,
	}
}

// ClientNames is every client the installer writes to, in the order it
// writes them.
//
// It is exported for the command line's help, which would otherwise carry a
// list of its own and drift from this one. A person reading that help would
// install for a client this installer does not touch, and never learn about
// the ones it does.
func ClientNames() []string {
	// Which clients there are does not depend on the home directory; only
	// where their files are does. Claude Desktop is the exception, and only
	// on a Windows with no APPDATA, so a placeholder keeps the list whole
	// on the machine that is merely asking what the list is.
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = string(filepath.Separator)
	}

	targets := clientConfigs(home)

	names := make([]string, 0, len(targets))
	for _, c := range targets {
		names = append(names, c.name)
	}

	return names
}

// claudeDesktopConfig is where Claude Desktop keeps its configuration on
// this operating system, or the empty string when there is nowhere to look —
// a Windows without APPDATA set, which is the one case where guessing a path
// would create a file no application will ever read.
func claudeDesktopConfig(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return ""
		}

		return filepath.Join(appData, "Claude", "claude_desktop_config.json")
	default:
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}
}

// opencodeConfig is OpenCode's global configuration, which it will read
// under either of two names.
//
// The one already on disk wins. Writing opencode.json next to somebody's
// opencode.jsonc leaves two configurations where there was one, and the
// reader is left to find out which of them their client picked up.
func opencodeConfig(home string) string {
	dir := filepath.Join(home, ".config", "opencode")

	commented := filepath.Join(dir, "opencode.jsonc")
	if _, err := os.Stat(commented); err == nil {
		return commented
	}

	return filepath.Join(dir, "opencode.json")
}
