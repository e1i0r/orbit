package cli

// The one-time move, read from the side the reader stands on.
//
// internal/store/flatten_test.go asks what the move itself does: what came
// up, what stayed where it was, what it refuses to decide. This asks the
// thing around it, which is the only part a person meets — flatten runs
// before every command, so it runs before `orbit top` on the morning after
// an upgrade, and what it owes that reader is a window that lists their
// tasks and a stderr that stays quiet unless something needs them.
//
// A real state root rather than a stand-in, because the move is the
// filesystem's: a store that answered a canned listing would test the
// stand-in. $ORBIT_HOME is the seam, and it is the one store.Open reads.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// filed writes a task the old way — under the repository it was written
// against — and answers the directory it wrote.
//
// The old way is the shape an Orbit from before the flat tree left behind,
// and it is still on disk on every machine that upgraded: repos/<key>/tasks/
// <id>. Nothing writes it any more, so a test that wants to ask about the
// move has to write it by hand.
func filed(t *testing.T, root, repoPath, id string, files map[string]string) string {
	t.Helper()

	s, err := store.New(root)
	if err != nil {
		t.Fatalf("open the state root: %v", err)
	}

	if _, err := s.RegisterRepo(repoPath); err != nil {
		t.Fatalf("register %q: %v", repoPath, err)
	}

	repoDir, err := s.RepoDir(repoPath)
	if err != nil {
		t.Fatalf("where %q is filed: %v", repoPath, err)
	}

	dir := filepath.Join(repoDir, "tasks", id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("create %q: %v", dir, err)
	}

	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	return dir
}

// TestFlattenMovesWhatAnOlderOrbitFiledUnderARepository.
//
// The case the whole function exists for: somebody upgraded, ran the first
// command they always run, and their tasks are there.
func TestFlattenMovesWhatAnOlderOrbitFiledUnderARepository(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	was := filed(t, root, "/w/app", "ACME-1", map[string]string{
		"task.md":      "retry the webhook on 5xx\n",
		"events.jsonl": `{"at":"2026-08-01T10:00:00Z","kind":"task.created","text":"retry the webhook on 5xx"}` + "\n",
	})

	var errOut bytes.Buffer

	flatten(&errOut)

	s, err := store.New(root)
	if err != nil {
		t.Fatalf("open the state root afterwards: %v", err)
	}

	now, err := s.TaskDir("ACME-1")
	if err != nil {
		t.Fatalf("where ACME-1 is now: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(now, "task.md"))
	if err != nil {
		t.Fatalf("the task did not arrive at the flat tree: %v", err)
	}

	if string(body) != "retry the webhook on 5xx\n" {
		t.Errorf("the task that arrived is not the one that was filed: %q", body)
	}

	// Copies and deletes nothing, so an Orbit from before this change still
	// reads its own tree. The record is the only account of what a run did,
	// and the way to be sure a migration did not eat one is to still have it.
	if _, err := os.Stat(filepath.Join(was, "task.md")); err != nil {
		t.Errorf("the old tree was not left where it was: %v", err)
	}

	// Nothing here needs a person, so the reader's terminal stays clean.
	if errOut.Len() > 0 {
		t.Errorf("a move that went through said something to the reader: %q", errOut.String())
	}
}

// TestTheSecondRunFindsEverythingInPlace.
//
// What makes calling this before every command safe rather than merely
// tolerated: a root that has already moved reads two directories and does
// nothing. Were the second run noisy or destructive, the first command after
// every upgrade would be the one that damaged the tree.
func TestTheSecondRunFindsEverythingInPlace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	was := filed(t, root, "/w/app", "ACME-2", map[string]string{"task.md": "raise the ceiling\n"})

	var first, second bytes.Buffer

	flatten(&first)
	flatten(&second)

	if second.Len() > 0 {
		t.Errorf("a root that had already moved said something on the way past: %q", second.String())
	}

	if _, err := os.Stat(filepath.Join(was, "task.md")); err != nil {
		t.Errorf("the second run took the old tree with it: %v", err)
	}

	// And the task is still one task, not two copies of it drifting apart:
	// the flat tree holds it and the link back to the repository survived
	// both runs.
	s, err := store.New(root)
	if err != nil {
		t.Fatalf("open the state root afterwards: %v", err)
	}

	joined, err := s.TaskRepos("ACME-2")
	if err != nil {
		t.Fatalf("which repositories hold ACME-2: %v", err)
	}

	if len(joined) != 1 || joined[0] != "/w/app" {
		t.Errorf("ACME-2 is held by %v, want only /w/app", joined)
	}
}

// TestThePairItWillNotDecideIsSaidOutLoud.
//
// Two repositories each holding a task of the same name cannot both come up
// into one flat tree, and choosing which one wins is not a choice a
// migration gets to make quietly. It reports the pair, names both
// directories because whoever reads it is the one who has to rename one, and
// carries on with the rest.
//
// The line is wiped a moment later when the command is the window, which is
// the same trade the log itself makes; the log keeps it. What matters here
// is that it was said at all — a collision answered with silence is a task
// that quietly stopped being one of the two it was.
func TestThePairItWillNotDecideIsSaidOutLoud(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	filed(t, root, "/w/app", "ACME-3", map[string]string{"task.md": "the one under app\n"})
	filed(t, root, "/w/billing", "ACME-3", map[string]string{"task.md": "the one under billing\n"})

	var errOut bytes.Buffer

	flatten(&errOut)

	said := errOut.String()
	if said == "" {
		t.Fatal("two tasks of the same name in two repositories were answered with silence")
	}

	if !strings.Contains(said, "ACME-3") {
		t.Errorf("the pair was reported without naming the task: %q", said)
	}

	// Two directories, and not the repositories they came from: whoever
	// reads this is the one who has to rename one, and what they rename is a
	// directory. One is the flat tree the first copy landed in; the other is
	// the filing under whichever repository orbit did not move.
	s, err := store.New(root)
	if err != nil {
		t.Fatalf("open the state root afterwards: %v", err)
	}

	if flat := filepath.Join(root, "tasks", "ACME-3"); !strings.Contains(said, flat) {
		t.Errorf("the pair did not name the flat tree the first copy landed in: %q", said)
	}

	var filings []string

	for _, repo := range []string{"/w/app", "/w/billing"} {
		repoDir, err := s.RepoDir(repo)
		if err != nil {
			t.Fatalf("where %q is filed: %v", repo, err)
		}

		filings = append(filings, filepath.Join(repoDir, "tasks", "ACME-3"))
	}

	if !strings.Contains(said, filings[0]) && !strings.Contains(said, filings[1]) {
		t.Errorf("the pair did not name the filing that stayed: %q", said)
	}

	if !strings.Contains(said, "did not move") {
		t.Errorf("the pair did not say which of the two stayed: %q", said)
	}
}

// TestAStateRootThatCannotBeOpenedIsLoggedAndNotPrinted.
//
// Nothing flatten can find is worth refusing to run a command over, and a
// root it cannot open is the least worth it of all: the command that follows
// will fail on its own terms and say why, and a migration failing first
// would bury that under a message about a move the reader never asked for.
// So it goes to the log, which is where a thing nobody has to act on
// belongs, and the terminal is left alone.
func TestAStateRootThatCannotBeOpenedIsLoggedAndNotPrinted(t *testing.T) {
	// Named after a plain file rather than a directory, so MkdirAll has
	// something to refuse. The same shape cli_home_test.go uses for the
	// command that follows.
	blocker := filepath.Join(t.TempDir(), "orbit")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatalf("write the blocker: %v", err)
	}

	t.Setenv("ORBIT_HOME", blocker)

	var errOut bytes.Buffer

	flatten(&errOut)

	if errOut.Len() > 0 {
		t.Errorf("a state root that could not be opened was printed to the reader: %q", errOut.String())
	}
}
