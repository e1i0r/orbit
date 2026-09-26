package engine

// Reading a session's conversation back out of cline's index and files.

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// clineSchema is the part of cline's sessions index this reads.
const clineSchema = `CREATE TABLE sessions (
	session_id text PRIMARY KEY, cwd text, started_at text, messages_path text);`

// clineStorage builds cline's data directory with one session in dir, whose
// messages are the ones given, started at the given time.
func clineStorage(t *testing.T, dir string, started time.Time, messages []map[string]any) {
	t.Helper()

	data := t.TempDir()
	t.Setenv("CLINE_DATA_DIR", data)

	for _, sub := range []string{"db", "sessions/s1"} {
		if err := os.MkdirAll(filepath.Join(data, sub), 0o755); err != nil {
			t.Fatalf("making %s: %v", sub, err)
		}
	}

	path := filepath.Join(data, "sessions", "s1", "s1.messages.json")

	raw, err := json.Marshal(map[string]any{"version": 1, "messages": messages})
	if err != nil {
		t.Fatalf("encoding the messages: %v", err)
	}

	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("writing the messages: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(data, "db", "sessions.db"))
	if err != nil {
		t.Fatalf("opening the index: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing the index: %v", err)
		}
	}()

	if _, err := db.Exec(clineSchema); err != nil {
		t.Fatalf("building the index: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO sessions VALUES ('s1', ?, ?, ?)`,
		dir, started.UTC().Format(clineTime), path); err != nil {
		t.Fatalf("writing the session: %v", err)
	}
}

// clineMessage is one of cline's, with its content blocks.
func clineMessage(role string, at time.Time, blocks ...map[string]any) map[string]any {
	return map[string]any{"role": role, "ts": at.UnixMilli(), "content": blocks}
}

// TestAClineSessionComesBackWithBothSidesOfIt, and without the blocks that
// are a tool or a thought rather than a sentence.
func TestAClineSessionComesBackWithBothSidesOfIt(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)

	clineStorage(t, dir, start, []map[string]any{
		clineMessage("user", start.Add(time.Second), map[string]any{"type": "text", "text": "look at the review gate"}),
		clineMessage("assistant", start.Add(2*time.Second),
			map[string]any{"type": "thinking", "thinking": "hmm"},
			map[string]any{"type": "tool_use", "name": "read_files", "input": map[string]any{}}),
		clineMessage("assistant", start.Add(3*time.Second), map[string]any{"type": "text", "text": "it is the line ceiling"}),
	})

	turns, err := NewCline().Transcript(dir, start.Add(-time.Minute))
	if err != nil {
		t.Fatalf("Transcript: %v", err)
	}

	if len(turns) != 2 {
		t.Fatalf("turns = %+v, want the question and the answer", turns)
	}

	if turns[0].By != Operator || turns[0].Text != "look at the review gate" {
		t.Errorf("first turn = %+v", turns[0])
	}

	if turns[1].By != "cline" || turns[1].Text != "it is the line ceiling" {
		t.Errorf("second turn = %+v", turns[1])
	}

	// A session started before the question is not asked for, and one in
	// another directory is not this one.
	if turns, err := NewCline().Transcript(dir, start.Add(time.Hour)); err != nil || len(turns) != 0 {
		t.Errorf("a later start read %+v, %v", turns, err)
	}

	elsewhere, err := NewCline().Transcript(t.TempDir(), start.Add(-time.Minute))
	if err != nil || len(elsewhere) != 0 {
		t.Errorf("another directory read %+v, %v", elsewhere, err)
	}
}

// TestNoClineSessionsIsNoTranscript: a machine where cline never ran.
func TestNoClineSessionsIsNoTranscript(t *testing.T) {
	t.Setenv("CLINE_DATA_DIR", t.TempDir())

	if turns, err := NewCline().Transcript(t.TempDir(), time.Time{}); err != nil || turns != nil {
		t.Errorf("Transcript = %+v, %v; want nothing", turns, err)
	}
}
