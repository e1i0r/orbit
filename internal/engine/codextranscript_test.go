package engine

// Reading a session's conversation back off codex's own transcript.

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeRollout puts one session file where codex keeps them, under the day
// it ran, and says which directory that session was opened in.
func writeRollout(t *testing.T, home, cwd, name string, rows ...string) {
	t.Helper()

	dir := filepath.Join(home, "sessions", "2026", "09", "03")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("making %s: %v", dir, err)
	}

	body := `{"timestamp":"2026-09-03T10:00:00Z","type":"session_meta","payload":{"cwd":"` + cwd + `"}}` + "\n"
	for _, r := range rows {
		body += r
	}

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// message is one conversation row of that file.
func message(at, role, content string) string {
	return `{"timestamp":"` + at + `","type":"response_item","payload":{"type":"message","role":"` + role + `","content":` + content + `}}` + "\n"
}

// TestACodexSessionComesBackWithBothSidesOfIt, and without the environment
// codex opens every session by reciting to itself.
func TestACodexSessionComesBackWithBothSidesOfIt(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()

	writeRollout(t, home, dir, "rollout.jsonl",
		message("2026-09-03T10:00:01Z", "user", `[{"type":"input_text","text":"<environment_context>\n  <cwd>/x</cwd>"}]`),
		message("2026-09-03T10:00:02Z", "user", `[{"type":"input_text","text":"look at the review gate"}]`),
		`{"timestamp":"2026-09-03T10:00:03Z","type":"response_item","payload":{"type":"function_call","name":"shell"}}`+"\n",
		message("2026-09-03T10:00:04Z", "assistant", `[{"type":"output_text","text":"it is the line ceiling"}]`),
		`{"timestamp":"2026-09-03T10:00:05Z","type":"event_msg","payload":{"type":"agent_message","message":"it is the line ceiling"}}`+"\n")

	t.Setenv("CODEX_HOME", home)

	turns, err := Codex{}.Transcript(dir, time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("reading the transcript: %v", err)
	}

	want := []Turn{
		{By: Operator, Text: "look at the review gate"},
		{By: "codex", Text: "it is the line ceiling"},
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

// TestASessionRunSomewhereElseIsNotThisTasks.
//
// codex files its sessions by the day rather than by the directory, so
// every session of the day is opened to find this one. The header says
// where each ran, and a task's record is the wrong place to learn what was
// said about another one.
func TestASessionRunSomewhereElseIsNotThisTasks(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()

	writeRollout(t, home, filepath.Join(dir, "elsewhere"), "other.jsonl",
		message("2026-09-03T10:00:02Z", "user", `[{"type":"input_text","text":"said in another worktree"}]`))

	t.Setenv("CODEX_HOME", home)

	turns, err := Codex{}.Transcript(dir, time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("reading the transcript: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("a session run elsewhere came back as %+v", turns)
	}
}

// TestACodexSessionOlderThanThisOneStaysWhereItIs, for the reason claude's
// does: the tree keeps every session, and coming back from ten minutes of
// work is not a reason to file the whole day again.
func TestACodexSessionOlderThanThisOneStaysWhereItIs(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()

	writeRollout(t, home, dir, "rollout.jsonl",
		message("2026-09-03T08:00:00Z", "user", `[{"type":"input_text","text":"this morning"}]`),
		message("2026-09-03T10:00:00Z", "user", `[{"type":"input_text","text":"just now"}]`))

	t.Setenv("CODEX_HOME", home)

	turns, err := Codex{}.Transcript(dir, time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("reading the transcript: %v", err)
	}

	if len(turns) != 1 || turns[0].Text != "just now" {
		t.Errorf("the session came back as %+v, want only what was said after it started", turns)
	}
}
