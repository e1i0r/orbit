package engine

// Reading a session's conversation back out of opencode's database.

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// opencodeSchema is the part of opencode's own schema this reads: a session
// knows where it ran, a message who it was from, and a part holds the words.
const opencodeSchema = `
CREATE TABLE session (id text PRIMARY KEY, directory text NOT NULL);
CREATE TABLE message (id text PRIMARY KEY, session_id text NOT NULL, time_created integer NOT NULL, data text NOT NULL);
CREATE TABLE part (id text PRIMARY KEY, message_id text NOT NULL, session_id text NOT NULL, time_created integer NOT NULL, data text NOT NULL);`

// openStorage builds that database where opencode keeps it, and hands back
// a way to put messages in it.
func openStorage(t *testing.T, dir string) func(id, role, part string, at time.Time) {
	t.Helper()

	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)

	if err := os.MkdirAll(filepath.Join(data, "opencode"), 0o755); err != nil {
		t.Fatalf("making the storage directory: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(data, "opencode", "opencode.db"))
	if err != nil {
		t.Fatalf("opening the storage: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing the storage: %v", err)
		}
	})

	if _, err := db.Exec(opencodeSchema); err != nil {
		t.Fatalf("building the storage: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO session VALUES ('ses', ?)`, dir); err != nil {
		t.Fatalf("writing the session: %v", err)
	}

	return func(id, role, part string, at time.Time) {
		t.Helper()

		ms := at.UnixMilli()
		if _, err := db.Exec(`INSERT INTO message VALUES (?, 'ses', ?, ?)`, id, ms, `{"role":"`+role+`"}`); err != nil {
			t.Fatalf("writing the message: %v", err)
		}

		if _, err := db.Exec(`INSERT INTO part VALUES (?, ?, 'ses', ?, ?)`, id+"p", id, ms, part); err != nil {
			t.Fatalf("writing the part: %v", err)
		}
	}
}

// TestAnOpenCodeSessionComesBackWithBothSidesOfIt, and without the parts
// that are a tool rather than a sentence.
func TestAnOpenCodeSessionComesBackWithBothSidesOfIt(t *testing.T) {
	dir := t.TempDir()
	say := openStorage(t, dir)

	start := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	say("m1", "user", `{"type":"text","text":"look at the review gate"}`, start)
	say("m2", "assistant", `{"type":"tool","tool":"bash","state":{}}`, start.Add(time.Second))
	say("m3", "assistant", `{"type":"text","text":"it is the line ceiling"}`, start.Add(2*time.Second))

	turns, err := OpenCode{}.Transcript(dir, start.Add(-time.Hour))
	if err != nil {
		t.Fatalf("reading the transcript: %v", err)
	}

	want := []Turn{
		{By: Operator, Text: "look at the review gate"},
		{By: "opencode", Text: "it is the line ceiling"},
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

// TestAnOpenCodeSessionOlderThanThisOneStaysWhereItIs: the database holds
// every session ever run, in every directory.
func TestAnOpenCodeSessionOlderThanThisOneStaysWhereItIs(t *testing.T) {
	dir := t.TempDir()
	say := openStorage(t, dir)

	start := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	say("m1", "user", `{"type":"text","text":"this morning"}`, start.Add(-2*time.Hour))
	say("m2", "user", `{"type":"text","text":"just now"}`, start)

	turns, err := OpenCode{}.Transcript(dir, start.Add(-time.Hour))
	if err != nil {
		t.Fatalf("reading the transcript: %v", err)
	}

	if len(turns) != 1 || turns[0].Text != "just now" {
		t.Errorf("the session came back as %+v, want only what was said after it started", turns)
	}
}

// TestAMachineWhereOpenCodeHasNeverRunIsNotAFailure. The reader chose an
// engine on a knob; whether they have ever opened it is not this window's
// business to complain about.
func TestAMachineWhereOpenCodeHasNeverRunIsNotAFailure(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	turns, err := OpenCode{}.Transcript(t.TempDir(), time.Time{})
	if err != nil {
		t.Fatalf("asking about a machine with no opencode: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("a machine with no opencode answered %d turns", len(turns))
	}
}
