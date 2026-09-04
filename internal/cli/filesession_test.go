package cli

// The conversation an interactive session leaves behind, on its way into
// the task it was had on.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// sessionDir is where claude keeps the sessions opened in dir: the path
// with every character that is not a letter or a digit turned into a dash.
// The rule lives in internal/engine, and it is claude's rather than either
// package's — this test writes a file that program would have written.
func sessionDir(home, dir string) string {
	slug := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		}

		return '-'
	}, dir)

	return filepath.Join(home, ".claude", "projects", slug)
}

// TestASessionsTurnsGoIntoTheTaskItWasOpenedOn, both sides of them, and the
// sentence Orbit itself opened the session with does not.
func TestASessionsTurnsGoIntoTheTaskItWasOpenedOn(t *testing.T) {
	root, _ := workspace(t)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	repo := filepath.Join(root, "payments")
	task := view.Task{ID: "ACME-1", RepoPath: repo, Repo: "payments", Title: "Retry the webhook"}

	worktree, err := s.CreateWorktreeParent(repo, "ACME-1")
	if err != nil {
		t.Fatalf("CreateWorktreeParent: %v", err)
	}

	if err := os.MkdirAll(worktree, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := sessionDir(home, worktree)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	at := time.Now().Format(time.RFC3339)
	said := func(kind, content string) string {
		return `{"type":"` + kind + `","timestamp":"` + at + `","message":{"content":` + content + `}}` + "\n"
	}

	body := said("user", `"`+openContext(task)+`"`) +
		said("user", `"the review gate keeps failing"`) +
		said("assistant", `[{"type":"text","text":"it is the line ceiling"}]`)

	if err := os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the transcript: %v", err)
	}

	filed, err := fileSessionPort(s, board.NewReader(s, root), newEngines())(task, "claude", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("reading the session back: %v", err)
	}

	if filed != 2 {
		t.Errorf("%d turns went into the task, want the two that were said in the session", filed)
	}

	entries, err := board.NewReader(s, root).Log(repo, "ACME-1")
	if err != nil {
		t.Fatalf("reading the record: %v", err)
	}

	var written []string

	for _, e := range entries {
		if e.What() == view.EntryDialogue {
			written = append(written, e.By+": "+e.Text)
		}
	}

	want := []string{"operator: the review gate keeps failing", "claude: it is the line ceiling"}
	if len(written) != len(want) {
		t.Fatalf("the record holds %v, want %v", written, want)
	}

	for i, w := range want {
		if written[i] != w {
			t.Errorf("dialogue %d is %q, want %q", i, written[i], w)
		}
	}
}

// TestASessionOnNoTaskIsFiledNowhere: there is no record to write it into,
// and the reader is asked to write the task down instead.
func TestASessionOnNoTaskIsFiledNowhere(t *testing.T) {
	root, _ := workspace(t)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	filed, err := fileSessionPort(s, board.NewReader(s, root), newEngines())(view.Task{}, "claude", time.Time{})
	if err != nil {
		t.Fatalf("reading back a session on no task: %v", err)
	}

	if filed != 0 {
		t.Errorf("%d turns were filed for a session on no task", filed)
	}
}
