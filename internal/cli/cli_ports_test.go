package cli

// The window's ports, filled the way the cockpit fills them.
//
// learnPort, notePort, the knowledge readers, the engine table and the
// world the verbs reach through: one state root behind all of them, and
// assertions on what each answers rather than on the shape around it.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// portWorld is a state root with one repository worked in it, and the
// reader over it the ports below read through. One home for both: the
// commands open their own store from the environment, so a second root
// would be a task written in one place and read in another.
func portWorld(t *testing.T) (*store.Store, *board.Reader, string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

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

	s, err := store.New(home)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return s, board.NewReader(s, root), repoDir
}

// TestLearningDownWritesFactsDown. A fact from the window's own line goes
// straight to the store it names; a scope nobody knows is refused before
// anything is written.
func TestLearningDownWritesFactsDown(t *testing.T) {
	s, _, _ := portWorld(t)

	if err := learnPort(s)(false, "", "", "never force-push"); err != nil {
		t.Fatalf("learn: %v", err)
	}

	if err := learnPort(s)(true, "go", "", "wrap errors"); err != nil {
		t.Fatalf("learn with a check: %v", err)
	}

	if got := knowsPort(s, ""); len(got()) == 0 {
		t.Error("nothing is known right after learning two facts")
	}

	if err := turnFactPort(s)(knowledge.Fact{
		Phrase: "amounts are cents", Source: knowledge.Human,
		Scope: knowledge.Scope{Kind: knowledge.General},
	}); err != nil {
		t.Fatalf("turn: %v", err)
	}

	was := knowledge.Fact{
		Phrase: "amounts are cents", Source: knowledge.Human,
		Scope: knowledge.Scope{Kind: knowledge.General},
	}
	now := knowledge.Fact{
		Phrase: "amounts are always cents", Source: knowledge.Human,
		Scope: knowledge.Scope{Kind: knowledge.General},
	}

	if err := replaceFactPort(s)(was, now); err != nil {
		t.Fatalf("replace: %v", err)
	}

	// Replacing what is not there still writes the correction: the new
	// sentence stands on its own, and there is nothing to take away.
	if err := replaceFactPort(s)(knowledge.Fact{Phrase: "nobody wrote this"}, now); err != nil {
		t.Fatalf("replace: %v", err)
	}

	if got := knowsPort(s, ""); len(got()) == 0 {
		t.Error("nothing is known after learning, turning and replacing")
	}
}

// TestFactScopeNamesEveryScope. Empty is everywhere, a word is a language,
// and a checkout is a repository.
func TestFactScopeNamesEveryScope(t *testing.T) {
	if got := factScope("", ""); got.Kind != knowledge.General {
		t.Errorf("no scope reads %v", got)
	}

	if got := factScope("go", ""); got.Kind != knowledge.Language || got.Lang != "go" {
		t.Errorf("go reads %v", got)
	}

	if got := factScope("", "/src/acme"); got.Kind != knowledge.Repo {
		t.Errorf("a checkout reads %v", got)
	}
}

// TestOnlyOfKeepsItsOwn. Only the facts about that checkout: the window
// narrows what the model is told to where its work is.
func TestOnlyOfKeepsItsOwn(t *testing.T) {
	facts := []knowledge.Fact{
		{Phrase: "everywhere", Scope: knowledge.Scope{Kind: knowledge.General}},
		{Phrase: "in go", Scope: knowledge.Scope{Kind: knowledge.Language, Lang: "go"}},
		{Phrase: "in acme", Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: "/src/acme"}},
		{Phrase: "in ledger", Scope: knowledge.Scope{Kind: knowledge.Repo, Repo: "/src/ledger"}},
	}

	kept := onlyOf("/src/acme", facts)
	if len(kept) != 1 || kept[0].Phrase != "in acme" {
		t.Errorf("onlyOf kept %v, want only the fact about acme", kept)
	}
}

// TestNotingNeedsATaskOnTheBoard. The window names a row, not a path, so
// the board turns the id into the repository — and an id nobody wrote is
// a refusal, not a note nowhere.
func TestNotingNeedsATaskOnTheBoard(t *testing.T) {
	s, r, repoDir := portWorld(t)

	if err := notePort(r, s)("ACME-404", "hi"); err == nil {
		t.Error("a note on a task nobody wrote was accepted")
	}

	if code, _, errOut := run(t, "board", "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	// The reader was folded before the task was written; a board that
	// forgot would show a reader an empty board a moment after they
	// filled in a form.
	if err := r.Rescan(); err != nil {
		t.Fatalf("rescan: %v", err)
	}

	if err := notePort(r, s)("ACME-1", "cents, not floats"); err != nil {
		t.Fatalf("note: %v", err)
	}
}

// TestEnginesAreNamedAndDrafted. A name nothing answers to is refused with
// the names there are; a fake engine drafts without spending anything.
func TestEnginesAreNamedAndDrafted(t *testing.T) {
	engines := map[string]engine.Engine{"fake": engine.NewFake(`{"name": "x", "phases": []}`)}

	if _, err := engineNamed(engines, "nobody"); err == nil {
		t.Error("an engine nothing answers to was accepted")
	}

	if _, err := draftPort(engines)("nobody", "", "x"); err == nil {
		t.Error("a draft from no engine was accepted")
	}

	out, err := draftPort(engines)("fake", "", "shape this")
	if err != nil {
		t.Fatalf("draft: %v", err)
	}

	if out == "" {
		t.Error("the draft is empty")
	}

	if _, err := askSupervisorPort(nil, engines)("nobody", "", ""); err == nil {
		t.Error("a supervisor nobody drives was accepted")
	}

	if _, err := autoSupervisePort(nil, engines)("nobody", nil); err == nil {
		t.Error("an autopilot nobody drives was accepted")
	}

	if got := enginesPort(engines)(); len(got) != 1 || got[0].Name != "fake" {
		t.Errorf("engines reads %v", got)
	}

	if got := engineNames(engines); len(got) != 1 || got[0] != "fake" {
		t.Errorf("engine names reads %v", got)
	}

	if got := setupGuide("claude")(words.For("en")); len(got) == 0 {
		t.Error("the setup guide is empty")
	}

	if guide := setupGuide("nobody"); guide != nil {
		t.Errorf("an engine nobody ships has a guide: %v", guide(words.For("en")))
	}
}
