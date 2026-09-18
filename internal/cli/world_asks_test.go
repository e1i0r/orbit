package cli

// The two ports that hand a terminal, or a bill, to something else.
//
// Ask read zero and Take read the first line. Between them they are the only
// two places the command line starts a program and waits: one spends money
// with no task behind it, the other gives away the terminal until the reader
// comes back. Both are worth being sure about.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// installed puts a stand-in for that engine on PATH, printing those lines.
func installed(t *testing.T, name, lines string) {
	t.Helper()

	dir := t.TempDir()

	script := "#!/bin/sh\ncat <<'EOF'\n" + lines + "\nEOF\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("write the stand-in %s: %v", name, err)
	}

	t.Setenv("PATH", dir+":/bin:/usr/bin")
}

// TestWhatSomebodyKeepsSayingIsReadByTheEngineTheySet. Named or set, and
// never whichever happens to be installed: this is the one reading that
// spends money with no task behind it.
func TestWhatSomebodyKeepsSayingIsReadByTheEngineTheySet(t *testing.T) {
	w, s, _ := worldOf(t)

	installed(t, "claude", `{"type":"result","result":"you keep asking for tests","session_id":"s1"}`)

	setEngine(t, s, "claude")

	said, err := w.Ask(context.Background(), "", "what do I keep saying")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}

	if said != "you keep asking for tests" {
		t.Errorf("it answered %q, want what the engine said", said)
	}
}

// TestTheEngineNamedAtTheCallWinsOverTheOneThatIsSet, because a reader who
// typed -engine meant that one for this reading.
func TestTheEngineNamedAtTheCallWinsOverTheOneThatIsSet(t *testing.T) {
	w, s, _ := worldOf(t)

	dir := t.TempDir()

	for name, said := range map[string]string{
		"claude": `{"type":"result","result":"from the one that is set","session_id":"s1"}`,
		"codex": `{"type":"thread.started","thread_id":"s2"}
{"type":"item.completed","item":{"id":"i0","type":"agent_message","text":"from the one that was named"}}
{"type":"turn.completed","usage":{"output_tokens":3}}`,
	} {
		script := "#!/bin/sh\ncat <<'EOF'\n" + said + "\nEOF\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatalf("write the stand-in %s: %v", name, err)
		}
	}

	t.Setenv("PATH", dir+":/bin:/usr/bin")

	setEngine(t, s, "claude")

	said, err := w.Ask(context.Background(), "codex", "what do I keep saying")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}

	if said != "from the one that was named" {
		t.Errorf("it answered %q, want the engine named at the call", said)
	}
}

// TestAReadingThatBrokeNamesTheEngineThatBroke. A reader told only that
// something failed has to guess which of four programs it was.
func TestAReadingThatBrokeNamesTheEngineThatBroke(t *testing.T) {
	w, s, _ := worldOf(t)

	dir := t.TempDir()

	script := "#!/bin/sh\nexit 3\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write the stand-in: %v", err)
	}

	t.Setenv("PATH", dir+":/bin:/usr/bin")

	setEngine(t, s, "claude")

	_, err := w.Ask(context.Background(), "", "what do I keep saying")
	if err == nil {
		t.Fatal("a reading whose engine exited 3 answered as if it worked")
	}

	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("the failure is %q, want it to name the engine", err)
	}
}

// TestTakingATaskNothingKnowsAboutIsRefusedByName. The terminal is about to
// be handed over, so a mistyped id has to stop here rather than opening a
// session about something else.
func TestTakingATaskNothingKnowsAboutIsRefusedByName(t *testing.T) {
	w, _, _ := worldOf(t)

	_, err := w.Take("ACME-404", "")
	if err == nil {
		t.Fatal("a task nothing knows about was taken")
	}

	if !strings.Contains(err.Error(), "ACME-404") {
		t.Errorf("the refusal is %q, want it to say back the id", err)
	}
}

// TestATaskNoEngineHasWalkedHasNoSessionToCarryOn. There is nothing to hand
// the terminal to, and that is a fact about the task rather than a failure:
// the sentence says so, in those words, because the window prints it.
func TestATaskNoEngineHasWalkedHasNoSessionToCarryOn(t *testing.T) {
	w, _, repoDir := worldOf(t)

	if code, _, errs := run(t, "board", "new", "-repo", repoDir, "-id", "ACME-1", "pay the thing"); code != 0 {
		t.Fatalf("write the task down: %s", errs)
	}

	_, err := w.Take("ACME-1", repoDir)
	if err == nil {
		t.Fatal("a task nothing has run was taken")
	}

	if !strings.Contains(err.Error(), "no session to carry on") {
		t.Errorf("the refusal is %q, want it to say there is no session", err)
	}
}

// setEngine is the standing choice, written the way `orbit settings set`
// writes it.
func setEngine(t *testing.T, s *store.Store, name string) {
	t.Helper()

	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("read the settings: %v", err)
	}

	cfg.Engine = name

	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("write the settings: %v", err)
	}
}
