package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// serverName is the key Orbit registers itself under in every client's
// mcpServers map. It is one constant so a second install cannot leave two
// entries behind under two spellings.
const serverName = "orbit"

// The two things that can happen to a client configuration.
const (
	StatusInstalled = "installed"
	StatusFailed    = "failed"
)

// InstallResult is one client configuration the installer touched, and what
// happened to it.
type InstallResult struct {
	Target string // the client's name, as the installer prints it
	Path   string // the configuration file it wrote
	Status string // StatusInstalled or StatusFailed
	Err    error  // why it failed, and nil when it did not
}

// Install registers binaryPath as an MCP server in every client
// configuration under home.
//
// home is a parameter rather than something this function asks the operating
// system for, and that is the whole reason it is one. An installer that
// finds its own home directory cannot be exercised without writing into the
// reader's real Claude and Codex configuration — which is not hypothetical:
// the first version of this file shipped a test that did exactly that, and
// `go test ./...` replaced a working orbit entry with a path to a binary
// that does not exist.
//
// binaryPath may be empty, in which case the running executable is
// registered. A client spawns this command years after the install, so the
// path has to be absolute and it has to be resolved now.
func Install(binaryPath, home string) []InstallResult {
	binaryPath = binary(binaryPath)
	targets := clientConfigs(home)

	results := make([]InstallResult, 0, len(targets))
	for _, t := range targets {
		res := InstallResult{Target: t.name, Path: t.path, Status: StatusInstalled}
		if err := t.register(t.path, binaryPath); err != nil {
			res.Status = StatusFailed
			res.Err = err
		}

		results = append(results, res)
	}

	return results
}

// binary is the orbit a client should spawn: the one that was asked for, or
// the one running now.
//
// "orbit" is the honest fallback and not a good one: it works only if the
// client inherits a PATH that has orbit on it, which a desktop application
// launched from Finder does not. It is still better than refusing to install
// at all, and the printed path tells the reader what was registered.
func binary(chosen string) string {
	if chosen != "" {
		return chosen
	}

	exe, err := os.Executable()
	if err != nil {
		return "orbit"
	}

	return exe
}

// entry is Orbit as one line of a client's mcpServers map.
//
// The args are what `orbit mcp` has to go on being for ever: a client spawns
// this command years after the install, so the entry written today decides
// what the binary must still do then. It is one function because the
// installer and the launcher both write it, and two spellings of one entry
// is a session that has the server and a client that does not.
func entry(binaryPath string) map[string]any {
	return map[string]any{
		"command": binaryPath,
		"args":    []string{"mcp"},
	}
}

// registerJSON merges Orbit into one map of a JSON configuration file,
// leaving every other key and every other server exactly as it found them.
//
// Which map, and what Orbit looks like inside it, are the client's to say:
// Claude keeps servers under mcpServers as a command and its arguments,
// OpenCode keeps them under mcp as a type and one argv. A single hardcoded
// shape is what made the OpenCode file this installer wrote inert.
//
// A file that will not parse is a file somebody else is responsible for, and
// it is refused rather than replaced. The earlier version of this function
// started from an empty map when the decode failed, which turned one
// unreadable ~/.claude.json — a hundred kilobytes of a client's own state —
// into a file holding nothing but an orbit entry.
func registerJSON(configPath, key string, value map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create config directory for %q: %w", configPath, err)
	}

	config := map[string]any{}
	mode := os.FileMode(0o600)

	data, err := os.ReadFile(configPath)
	switch {
	case err == nil:
		if len(data) > 0 {
			if err := json.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("read %q: %w", configPath, err)
			}
		}

		if info, statErr := os.Stat(configPath); statErr == nil {
			mode = info.Mode().Perm()
		}
	case os.IsNotExist(err):
		// A client that has never had a server configured has no file, and
		// making it is the install.
	default:
		return fmt.Errorf("read %q: %w", configPath, err)
	}

	servers, ok := config[key].(map[string]any)
	if !ok {
		servers = map[string]any{}
	}

	servers[serverName] = value
	config[key] = servers

	return writeJSON(configPath, config, mode)
}

// writeJSON encodes the configuration and hands it to writeAtomically.
func writeJSON(path string, config map[string]any, mode os.FileMode) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config for %q: %w", path, err)
	}

	return writeAtomically(path, append(data, '\n'), mode, ".orbit-mcp-*.json")
}

// writeAtomically writes through a temporary file in the same directory and
// renames it into place.
//
// The rename is the point. These files hold state the client wrote and Orbit
// only borrows — ~/.claude.json is the largest of them, and a Codex
// config.toml holds somebody's model and their trusted projects — and a
// process that dies between truncating one and finishing the write has
// destroyed something it does not own. A rename either happened or did not.
func writeAtomically(path string, data []byte, mode os.FileMode, pattern string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return fmt.Errorf("create temporary file beside %q: %w", path, err)
	}

	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()     //nolint:errcheck // the write already failed; the close cannot add to it
		_ = os.Remove(name) //nolint:errcheck // best-effort cleanup of a file nothing points at

		return fmt.Errorf("write %q: %w", name, err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(name) //nolint:errcheck // best-effort cleanup of a file nothing points at
		return fmt.Errorf("close %q: %w", name, err)
	}

	if err := os.Chmod(name, mode); err != nil {
		_ = os.Remove(name) //nolint:errcheck // best-effort cleanup of a file nothing points at
		return fmt.Errorf("set mode on %q: %w", name, err)
	}

	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name) //nolint:errcheck // best-effort cleanup of a file nothing points at
		return fmt.Errorf("move %q into place at %q: %w", name, path, err)
	}

	return nil
}
