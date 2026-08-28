package cli

// The session the cockpit's c gesture opens: where it opens, what the engine
// is told about the task, and the one thing that makes it more than a shell —
// Orbit's own server, configured for that session and no other.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// serverIn is the orbit entry of the configuration one argument carries.
func serverIn(t *testing.T, arg string) map[string]any {
	t.Helper()
	var doc struct {
		Servers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(arg), &doc); err != nil {
		t.Fatalf("the configuration handed to the engine is not JSON: %v\n%s", err, arg)
	}
	entry, ok := doc.Servers["orbit"]
	if !ok {
		t.Fatalf("the configuration names no orbit server: %s", arg)
	}
	return entry
}

func TestOpenCommandGivesClaudeOrbitsOwnServer(t *testing.T) {
	dir := t.TempDir()
	cmd, err := openCommand("claude", dir, "look at PAY-1")
	if err != nil {
		t.Fatalf("openCommand: %v", err)
	}
	if cmd.Dir != dir {
		t.Errorf("Dir = %q, want %q", cmd.Dir, dir)
	}
	args := cmd.Args[1:]
	at := slices.Index(args, "--mcp-config")
	if at < 0 || at+1 >= len(args) {
		t.Fatalf("args = %v, want the session to be handed a configuration", args)
	}
	entry := serverIn(t, args[at+1])
	if command, ok := entry["command"].(string); !ok || command == "" {
		t.Errorf("the orbit entry names no command: %v", entry)
	}
	callArgs, ok := entry["args"].([]any)
	if !ok || len(callArgs) != 1 || callArgs[0] != "mcp" {
		t.Errorf("the orbit entry runs %v, want [mcp] — the command `orbit mcp install` writes too", entry["args"])
	}
	if args[len(args)-1] != "look at PAY-1" {
		t.Errorf("the last argument is %q, want what the session is opening about", args[len(args)-1])
	}
}

// An engine with no such flag gets no such flag. The gesture is "hand me the
// terminal", and an argument the program does not know would turn that into
// a session that never starts.
func TestOpenCommandLeavesAnEngineWithoutTheFlagAlone(t *testing.T) {
	cmd, err := openCommand("codex", "", "look at PAY-1")
	if err != nil {
		t.Fatalf("openCommand: %v", err)
	}
	if got := cmd.Args[1:]; len(got) != 1 || got[0] != "look at PAY-1" {
		t.Errorf("args = %v, want only what the session is about", got)
	}
	bare, err := openCommand("codex", "", "")
	if err != nil {
		t.Fatalf("openCommand: %v", err)
	}
	if got := bare.Args[1:]; len(got) != 0 {
		t.Errorf("args = %v, want none: there is no task to open on", got)
	}
}

func TestOpenCommandNeedsAnEngine(t *testing.T) {
	if _, err := openCommand("", t.TempDir(), ""); err == nil {
		t.Error("openCommand with no engine answered a command, want it refused")
	}
}

// The worktree is the checkout the run made its changes in; opening the
// repository would show a tree that does not have them.
func TestOpenDirPrefersTheWorktreeThatIsThere(t *testing.T) {
	root, _ := workspace(t)
	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	r := board.NewReader(s, root)
	task := view.Task{ID: "ACME-1", RepoPath: root}

	if got := openDir(r, task, root); got != root {
		t.Errorf("openDir = %q, want the repository %q: nothing has run, so there is no worktree", got, root)
	}

	worktree, err := s.CreateWorktreeParent(root, "ACME-1")
	if err != nil {
		t.Fatalf("CreateWorktreeParent: %v", err)
	}
	if err := os.MkdirAll(worktree, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if got := openDir(r, task, root); got != worktree {
		t.Errorf("openDir = %q, want the worktree %q", got, worktree)
	}

	// A file where the worktree would be is not a worktree.
	other, err := s.CreateWorktreeParent(root, "ACME-2")
	if err != nil {
		t.Fatalf("CreateWorktreeParent: %v", err)
	}
	if err := os.WriteFile(other, []byte("not a checkout\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := openDir(r, view.Task{ID: "ACME-2", RepoPath: root}, root); got != root {
		t.Errorf("openDir = %q, want the repository %q", got, root)
	}
}

func TestOpenDirFallsBackWhenThereIsNoTask(t *testing.T) {
	fallback := t.TempDir()
	for _, task := range []view.Task{{}, {ID: "ACME-1"}, {RepoPath: fallback}, {ID: "bad/id", RepoPath: fallback}} {
		if got := openDir(nil, task, fallback); got != fallback {
			t.Errorf("openDir(%+v) = %q, want the directory the window suggested", task, got)
		}
	}
}

func TestOpenContextNamesTheTaskAndSaysTheToolsAreThere(t *testing.T) {
	got := openContext(view.Task{
		ID:       "PAY-1",
		Repo:     "payments",
		RepoPath: "/tmp/payments",
		Title:    "Retry the webhook on 5xx",
		Band:     view.NeedsYou,
		Phase:    "review",
	})
	for _, want := range []string{"PAY-1", "payments", "Retry the webhook on 5xx", "needs you", "review", "orbit_inspect_task", "orbit_add_note"} {
		if !strings.Contains(got, want) {
			t.Errorf("the session is not told %q:\n%s", want, got)
		}
	}
	if openContext(view.Task{}) != "" {
		t.Error("a session opened on no task is told about one anyway")
	}
}

func TestOpenPortBuildsTheSessionTheWindowAsksFor(t *testing.T) {
	root, _ := workspace(t)
	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	open := openPort(s, board.NewReader(s, root))
	cmd, err := open(view.Task{ID: "ACME-1", Repo: "acme", RepoPath: root, Band: view.ToDo}, "claude", root)
	if err != nil {
		t.Fatalf("the open port: %v", err)
	}
	if filepath.Base(cmd.Path) != "claude" {
		t.Errorf("the session runs %q, want claude", cmd.Path)
	}
	if cmd.Dir != root {
		t.Errorf("Dir = %q, want the repository %q", cmd.Dir, root)
	}
	if !slices.Contains(cmd.Args, "--mcp-config") {
		t.Errorf("args = %v, want the session handed orbit's own server", cmd.Args)
	}
	if !strings.Contains(cmd.Args[len(cmd.Args)-1], "ACME-1") {
		t.Errorf("the session is not told which task it is about: %v", cmd.Args)
	}
}

// The session is the one thing that changes a task and would otherwise leave
// no trace: a run writes phases, a note writes a note, and an hour at the
// keyboard writes into a worktree while the record shows the same failed
// attempt it showed before.
func TestOpenPortWritesTheSessionIntoTheTasksRecord(t *testing.T) {
	root, _ := workspace(t)
	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	repoPath := filepath.Join(root, "payments")
	if _, err := s.CreateTaskDir(repoPath, "ACME-1"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}
	r := board.NewReader(s, root)
	open := openPort(s, r)
	if _, err := open(view.Task{ID: "ACME-1", Repo: "payments", RepoPath: repoPath}, "claude", repoPath); err != nil {
		t.Fatalf("the open port: %v", err)
	}

	// Read back the way the window reads it, which is the only way this
	// package is allowed to: the fold, not the file.
	entries, err := r.Log(repoPath, "ACME-1")
	if err != nil {
		t.Fatalf("read the record: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("the record has %d entries, want the one session: %v", len(entries), entries)
	}
	// Not a note. The next phase to start is handed every note since the
	// last one, and "the cockpit opened a session" is not an instruction.
	if entries[0].What() != view.EntryDialogue {
		t.Errorf("the record reads as %v, want the session as a dialogue entry", entries[0].What())
	}
	if entries[0].By != "claude" {
		t.Errorf("By = %q, want the engine the terminal went to", entries[0].By)
	}
	if !strings.Contains(entries[0].Text, "interactive session") {
		t.Errorf("the record says %q, want what happened to the task", entries[0].Text)
	}
}

// A record that could not be appended to is not a reason to refuse somebody
// the terminal they asked for — and neither is a gesture made on no task at
// all, which is a cursor on a band header.
func TestOpenPortStillOpensWhenNothingCanBeWritten(t *testing.T) {
	root, _ := workspace(t)
	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	open := openPort(s, board.NewReader(s, root))
	// No task directory was ever made, so there is nowhere to append.
	if _, err := open(view.Task{ID: "ACME-9", RepoPath: root}, "claude", root); err != nil {
		t.Errorf("a session was refused because its record could not be written: %v", err)
	}
	if _, err := open(view.Task{}, "claude", root); err != nil {
		t.Errorf("a session opened on no task was refused: %v", err)
	}
	if _, err := openPort(nil, nil)(view.Task{ID: "ACME-9"}, "claude", root); err != nil {
		t.Errorf("a session was refused for want of a state root: %v", err)
	}
}
