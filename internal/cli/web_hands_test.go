package cli

// The browser's buttons, asked the way the browser asks them.
//
// hands.Ask is the whole of what the browser does: one method over the
// verbs, so a task started from a tab and one started from a terminal are
// the same run. These drive it directly — no server, no port — with a
// state root and a board reader of the test's own.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/web"
	"github.com/e1i0r/orbit/internal/words"
)

// handsOf is the browser's verbs port over a state root of the test's
// own, read against a workspace with one repository in it.
func handsOf(t *testing.T) (hands, string) {
	t.Helper()

	t.Setenv("ORBIT_HOME", t.TempDir())

	root := t.TempDir()
	repoDir := filepath.Join(root, "payments")

	if err := os.MkdirAll(repoDir, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@orbit.local"},
		{"config", "user.name", "Orbit Tester"},
		{"commit", "-q", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir

		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return hands{store: s, board: board.NewReader(s, root), words: words.For("en")}, repoDir
}

// TestTheBrowsersButtonsAskTheVerbs. Saying, writing, noting, reading and
// listing through Ask, so that the browser and the terminal report one act
// in one form of words.
func TestTheBrowsersButtonsAskTheVerbs(t *testing.T) {
	h, repoDir := handsOf(t)

	if _, err := h.Ask("supervisor say", web.Asked{Args: map[string]string{"text": "never force-push"}}); err != nil {
		t.Fatalf("say: %v", err)
	}

	out, err := h.Ask("board new", web.Asked{Args: map[string]string{
		"id": "ACME-1", "text": "pay the thing", "repo": repoDir,
	}})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if len(out.Of) != 1 || out.Of[0] != "ACME-1" {
		t.Errorf("new acted on %v, want it to act on ACME-1", out.Of)
	}

	if _, err := h.Ask("task note", web.Asked{Task: "ACME-1", Args: map[string]string{"text": "cents"}}); err != nil {
		t.Fatalf("note: %v", err)
	}

	if _, err := h.Ask("task read", web.Asked{Task: "ACME-1"}); err != nil {
		t.Fatalf("read: %v", err)
	}

	listed, err := h.Ask("board list", web.Asked{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if !strings.Contains(listed.Said, "ACME-1") {
		t.Errorf("list answered %q, want the row", listed.Said)
	}

	shown, err := h.Ask("task show", web.Asked{Task: "ACME-1"})
	if err != nil {
		t.Fatalf("show: %v", err)
	}

	if !strings.Contains(shown.Said, "ACME-1") {
		t.Errorf("show answered %q, want the task", shown.Said)
	}

	if _, err := h.Ask("task history", web.Asked{Task: "ACME-1"}); err != nil {
		t.Fatalf("history: %v", err)
	}

	if _, err := h.Ask("mergre", web.Asked{}); err == nil {
		t.Error("a name nothing answers to was accepted")
	}
}

// TestTheBrowsersReadingsReadTheSameRecord. History and standing beside
// the task: the same rendering the window draws, read through the port.
func TestTheBrowsersReadingsReadTheSameRecord(t *testing.T) {
	h, repoDir := handsOf(t)

	if _, err := h.Ask("board new", web.Asked{Args: map[string]string{
		"id": "ACME-2", "text": "ship it", "repo": repoDir,
	}}); err != nil {
		t.Fatalf("new: %v", err)
	}

	body, err := h.History("ACME-2", repoDir)
	if err != nil {
		t.Fatalf("history: %v", err)
	}

	if !strings.Contains(body, "ACME-2") {
		t.Errorf("history reads %q, want the task", body)
	}

	if now := h.Standing("ACME-2", repoDir); now.Held {
		t.Error("a task nobody started reads as held")
	}

	if now := h.Standing("ACME-404", repoDir); now.Held || len(now.Pending) != 0 {
		t.Errorf("a task nobody wrote reads as %+v", now)
	}
}
