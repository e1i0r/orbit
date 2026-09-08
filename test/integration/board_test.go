//go:build integration

package integration

// The world one test runs in: a state root, a git repository with a real Go
// module in it, and the script the stand-in engine follows.
//
// A module and not an empty directory, because the checks a shipped flow
// carries are `go test ./...` and a coverage figure read off `go tool
// cover`. A fixture those cannot run against would leave every loop in these
// tests turning on a command that answers the same thing for ever.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// board is one test's whole world.
type board struct {
	home   string // what the commands see as $HOME
	state  string // the state root: the record, the worktrees, the settings
	repo   string // the repository the task is written against
	script string // the file the stand-in engine reads
	turns  string // where it counts the turns of each phase
}

// newBoard makes that world and answers it.
func newBoard(t *testing.T, phases map[string]any) board {
	t.Helper()

	dir := t.TempDir()
	b := board{
		home:   filepath.Join(dir, "home"),
		state:  filepath.Join(dir, "state"),
		repo:   filepath.Join(dir, "ledger"),
		script: filepath.Join(dir, "script.json"),
		turns:  filepath.Join(dir, "turns"),
	}

	for _, d := range []string{b.home, b.state, b.repo} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("make %q: %v", d, err)
		}
	}

	b.write(t, phases)
	b.module(t)

	return b
}

// write puts the script where the stand-in engine will look for it.
func (b board) write(t *testing.T, phases map[string]any) {
	t.Helper()

	body, err := json.MarshalIndent(map[string]any{"phases": phases}, "", "  ")
	if err != nil {
		t.Fatalf("encode the script: %v", err)
	}

	if err := os.WriteFile(b.script, body, 0o600); err != nil {
		t.Fatalf("write the script: %v", err)
	}
}

// module makes the repository: a Go module with one package, a test that
// covers it, and a first commit for a worktree to be cut from.
func (b board) module(t *testing.T) {
	t.Helper()

	for name, body := range map[string]string{
		"go.mod":         "module example.com/ledger\n\ngo 1.24\n",
		"ledger.go":      "package ledger\n\n// Total adds a charge and a refund.\nfunc Total(charge, refund int) int {\n\treturn charge - refund\n}\n",
		"ledger_test.go": "package ledger\n\nimport \"testing\"\n\nfunc TestTotal(t *testing.T) {\n\tif Total(10, 3) != 7 {\n\t\tt.Fatal(\"the total is wrong\")\n\t}\n}\n",
	} {
		if err := os.WriteFile(filepath.Join(b.repo, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
	}

	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@orbit"},
		{"config", "user.name", "orbit test"},
		{"add", "-A"},
		{"commit", "-m", "the ledger"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = b.repo

		if said, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, said)
		}
	}
}
