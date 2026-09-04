package engine

// Reading a session's conversation back off claude's own transcript.

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTranscript puts one session file where claude keeps them for dir,
// under a home directory this test owns.
func writeTranscript(t *testing.T, home, dir, name, body string) {
	t.Helper()

	d := filepath.Join(home, ".claude", "projects", claudeSlug(dir))
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatalf("making %s: %v", d, err)
	}

	if err := os.WriteFile(filepath.Join(d, name), []byte(body), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// row is one line of that file, in the shape claude writes.
func row(kind, at, content string) string {
	return `{"type":"` + kind + `","timestamp":"` + at + `","message":{"role":"user","content":` + content + `}}` + "\n"
}

// TestASessionComesBackWithBothSidesOfIt.
//
// The reader's prompts and the model's answers, in the order they were
// said, and nothing else: thinking is not something that was said out loud,
// a tool call is the timeline's business, and its result is a file, not a
// sentence.
func TestASessionComesBackWithBothSidesOfIt(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()

	writeTranscript(t, home, dir, "s.jsonl",
		row("user", "2026-09-03T10:00:00Z", `"look at the review gate"`)+
			row("assistant", "2026-09-03T10:00:05Z", `[{"type":"thinking","thinking":"hm"},{"type":"text","text":"reading it now"}]`)+
			row("assistant", "2026-09-03T10:00:06Z", `[{"type":"tool_use","name":"Bash","input":{}}]`)+
			row("user", "2026-09-03T10:00:07Z", `[{"type":"tool_result","content":"ok"}]`)+
			row("user", "2026-09-03T10:00:08Z", `"and the ceiling?"`)+
			`{"type":"cost-state","costUSD":0.2}`+"\n")

	t.Setenv("HOME", home)

	turns, err := Claude{}.Transcript(dir, time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("reading the transcript: %v", err)
	}

	want := []Turn{
		{By: Operator, Text: "look at the review gate"},
		{By: "claude", Text: "reading it now"},
		{By: Operator, Text: "and the ceiling?"},
	}

	if len(turns) != len(want) {
		t.Fatalf("the session came back as %d turns, want %d: %+v", len(turns), len(want), turns)
	}

	for i, w := range want {
		if turns[i].By != w.By || turns[i].Text != w.Text {
			t.Errorf("turn %d is %q by %q, want %q by %q", i, turns[i].Text, turns[i].By, w.Text, w.By)
		}
	}
}

// TestWhatWasSaidBeforeTheSessionStaysWhereItIs.
//
// One directory keeps every session ever opened in it. Without the moment
// the terminal was handed over, coming back from ten minutes of work would
// file every conversation ever had on that task, again.
func TestWhatWasSaidBeforeTheSessionStaysWhereItIs(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()

	writeTranscript(t, home, dir, "s.jsonl",
		row("user", "2026-09-01T10:00:00Z", `"last week's session"`)+
			row("user", "2026-09-03T10:00:00Z", `"today's"`))

	t.Setenv("HOME", home)

	turns, err := Claude{}.Transcript(dir, time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("reading the transcript: %v", err)
	}

	if len(turns) != 1 || turns[0].Text != "today's" {
		t.Errorf("the session came back as %+v, want only what was said after it started", turns)
	}
}

// TestOneEngineDoesNotReadAnothersTranscript: the file is claude's, and
// codex asked about the same directory answers about its own sessions.
func TestOneEngineDoesNotReadAnothersTranscript(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()

	writeTranscript(t, home, dir, "s.jsonl", row("user", "2026-09-03T10:00:00Z", `"said in a session"`))
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", t.TempDir())

	turns, err := Codex{}.Transcript(dir, time.Time{})
	if err != nil {
		t.Fatalf("asking about an engine with no transcript: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("codex answered %d turns off claude's transcript", len(turns))
	}
}

// TestNothingWasSaidInADirectoryWithNoSessions: a task nobody opened a
// terminal on is not an error.
func TestNothingWasSaidInADirectoryWithNoSessions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	turns, err := Claude{}.Transcript(t.TempDir(), time.Time{})
	if err != nil {
		t.Fatalf("asking about a directory with no sessions: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("a directory with no sessions answered %d turns", len(turns))
	}
}

// TestTheSlugIsThePathWithEveryOtherCharacterDashed, which is the whole of
// how a directory is found again.
func TestTheSlugIsThePathWithEveryOtherCharacterDashed(t *testing.T) {
	if got := claudeSlug("/Users/who/.orbit/worktrees/ab12/FRA-62"); got != "-Users-who--orbit-worktrees-ab12-FRA-62" {
		t.Errorf("the slug is %q, want the one claude writes", got)
	}
}
