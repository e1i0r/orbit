package cli

// Upgrading. A state root written by an older Orbit — a task filed under the
// repository it was written against — and the first command run against it.

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestACommandOverAnOlderStateRootFindsItsTasks(t *testing.T) {
	root, orbitHome := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// The path git answers with, which is what the store files a repository
	// under: on macOS the temporary directory is reached through a symlink,
	// and the two spellings hash to two different keys.
	abs, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}

	sum := sha256.Sum256([]byte(abs))
	key := hex.EncodeToString(sum[:])[:12]

	old := filepath.Join(orbitHome, "repos", key)
	if err := os.MkdirAll(filepath.Join(old, "tasks", "ACME-1"), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	write := map[string]string{
		filepath.Join(old, "repo"):                            "path: " + abs + "\n",
		filepath.Join(old, "tasks", "ACME-1", "task.md"):      "retry the webhook on 5xx\n",
		filepath.Join(old, "tasks", "ACME-1", "events.jsonl"): `{"at":"2026-08-01T10:00:00Z","kind":"task.created","text":"retry the webhook on 5xx"}` + "\n",
	}
	for path, body := range write {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	code, out, errOut := run(t, "list", "-repo", repoDir)
	if code != 0 {
		t.Fatalf("list over an older state root exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-1") {
		t.Fatalf("list did not find the task the older tree held:\n%s", out)
	}

	// And what it says about it is what the old log said.
	code, out, errOut = run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show over an older state root exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "retry the webhook on 5xx") {
		t.Errorf("show read nothing of the old record:\n%s", out)
	}
}
