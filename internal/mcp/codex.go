package mcp

// Codex is the one client that does not keep JSON. Its servers live in
// ~/.codex/config.toml as [mcp_servers.<name>] tables; ~/.codex/config.json
// is a file Codex has never opened.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// codexTable is the table Codex keeps its servers under.
const codexTable = "mcp_servers"

// registerCodex adds or replaces Orbit's table in a Codex config.toml.
//
// The file is edited as text, and there is no TOML library in go.mod for the
// same reason: a round trip through a decoder and an encoder gives back a
// file with the comments gone and the keys reordered, and this one holds
// somebody's model, their reasoning effort and the list of projects they
// have marked trusted. Orbit borrows a corner of it. Every byte outside its
// own table comes out the way it went in.
func registerCodex(configPath, binaryPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create config directory for %q: %w", configPath, err)
	}

	mode := os.FileMode(0o600)

	data, err := os.ReadFile(configPath)
	switch {
	case err == nil:
		if info, statErr := os.Stat(configPath); statErr == nil {
			mode = info.Mode().Perm()
		}
	case os.IsNotExist(err):
		// Codex has never been configured here, and making the file is the
		// install.
	default:
		return fmt.Errorf("read %q: %w", configPath, err)
	}

	merged, err := codexMerge(string(data), binaryPath)
	if err != nil {
		return fmt.Errorf("%q: %w", configPath, err)
	}

	return writeAtomically(configPath, []byte(merged), mode, ".orbit-mcp-*.toml")
}

// codexMerge is the edit itself, as a function of the text: everything
// outside [mcp_servers.orbit] is carried through untouched, and that table
// is replaced if it is there and appended if it is not.
func codexMerge(text, binaryPath string) (string, error) {
	lines := strings.Split(text, "\n")
	ours := codexTable + "." + serverName
	start, end, table := -1, len(lines), ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "[") {
			// Orbit written as one key of [mcp_servers] rather than as a
			// table of its own is the same entry in a shape this cannot
			// replace, and appending a table would be a duplicate key that
			// stops the whole file parsing.
			if table == codexTable && isKey(trimmed, serverName) {
				return "", fmt.Errorf("orbit is already listed under [%s] as a key; leaving it as it is", codexTable)
			}

			continue
		}

		if start >= 0 && i > start && end == len(lines) {
			end = i
		}

		table = tableName(trimmed)
		if table == ours {
			// A second [mcp_servers.orbit] is a file TOML already refuses
			// to parse, and it is not this installer's to guess at: taking
			// the first left the second behind as a duplicate key, and the
			// arithmetic below quietly wrote both tables out with a third
			// between them. Say what is wrong and touch nothing.
			if start >= 0 {
				return "", fmt.Errorf("orbit is listed twice under [%s]; leaving it as it is", codexTable)
			}

			start = i
		}
	}

	block := codexBlock(binaryPath)

	if start < 0 {
		if strings.TrimSpace(text) == "" {
			return block, nil
		}

		return strings.TrimRight(text, "\n") + "\n\n" + block, nil
	}

	kept := make([]string, 0, len(lines))
	kept = append(kept, lines[:start]...)
	kept = append(kept, strings.Split(strings.TrimRight(block, "\n"), "\n")...)

	return strings.Join(append(kept, lines[end:]...), "\n"), nil
}

// codexBlock is Orbit as a Codex table.
func codexBlock(binaryPath string) string {
	return fmt.Sprintf("[%s.%s]\ncommand = %s\nargs = [\"mcp\"]\n", codexTable, serverName, tomlString(binaryPath))
}

// tableName is the dotted name of a table header with the quoting taken off,
// so that [mcp_servers."orbit"] and [mcp_servers.orbit] are one table. A
// quoted name with a dot inside it would defeat this, and no client writes
// one.
func tableName(header string) string {
	if i := strings.Index(header, "]"); i >= 0 {
		header = header[:i+1]
	}

	header = strings.TrimSuffix(strings.TrimPrefix(header, "["), "]")

	parts := strings.Split(header, ".")
	for i, p := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(p), `"'`)
	}

	return strings.Join(parts, ".")
}

// isKey says whether a line assigns the given name, however it is spelled.
func isKey(line, name string) bool {
	for _, form := range []string{name, `"` + name + `"`, `'` + name + `'`} {
		rest, ok := strings.CutPrefix(line, form)
		if ok && strings.HasPrefix(strings.TrimSpace(rest), "=") {
			return true
		}
	}

	return false
}

// tomlString quotes a path the way a TOML reader will take it back. A
// Windows path is mostly backslashes and every one of them is an escape.
func tomlString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
