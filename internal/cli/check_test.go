package cli

// orbit check, and the same question asked before every command by whoever
// has turned it on.

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite" // the record's own engine, to damage it the way a half-applied migration would
)

// unsound leaves the record readable and wrong: a column declared NOT NULL
// over rows that are null, which is what a half-applied migration leaves
// behind. Every query still answers — that is the point of it — so the
// command runs, the migration in front of it finishes, and the only thing
// that knows anything is wrong is the check.
//
// It writes through a handle of its own rather than through internal/db,
// which this package may not import and which has no door for a statement
// like this anyway. Damaging a record is not something the record's own
// package should offer.
func unsound(t *testing.T, orbitHome string) {
	t.Helper()

	d, err := sql.Open("sqlite", filepath.Join(orbitHome, "orbit.db"))
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	for _, stmt := range []string{
		`PRAGMA writable_schema=ON`,
		`UPDATE sqlite_schema SET sql = replace(sql, 'run_id   INTEGER REFERENCES', 'run_id   INTEGER NOT NULL REFERENCES')
		  WHERE type = 'table' AND name = 'event'`,
		`PRAGMA writable_schema=RESET`,
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("rewrite the schema: %v", err)
		}
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close the record: %v", err)
	}
}

// corrupt scribbles over the head of every page of the record but the first,
// which holds the schema. What is left is a file that opens, answers some
// questions and is broken anyway — the damage a reader has no other way of
// finding out about. Its twin lives in internal/db's own tests, where the
// same shape proves that SQLite's refusal comes back as an answer.
func corrupt(t *testing.T, orbitHome string) {
	t.Helper()

	path := filepath.Join(orbitHome, "orbit.db")

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open %s to break it: %v", path, err)
	}

	const page = 4096

	for at := page; at < len(body); at += page {
		if _, err := f.WriteAt([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, int64(at)); err != nil {
			t.Fatalf("scribble at %d: %v", at, err)
		}
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}

func TestCheckSaysASoundRecordIsSoundAndHowMuchIsInIt(t *testing.T) {
	root, _ := workspace(t)
	writeTask(t, root)

	code, out, errOut := run(t, "check")
	if code != 0 {
		t.Fatalf("check of a sound record exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "is sound") {
		t.Errorf("check said %q, which does not say the record is sound", out)
	}

	if !strings.Contains(out, "1 task") || !strings.Contains(out, "supervisor turn") {
		t.Errorf("check said %q, which does not say how much the record holds", out)
	}
}

// TestCheckSaysDamageInWords. The point of the command: what a reader gets
// is a sentence saying the record is damaged and where it is, with SQLite's
// own line under it — rather than the raw `database disk image is malformed
// (11)` that asking SQLite directly would answer with.
func TestCheckSaysDamageInWords(t *testing.T) {
	root, orbitHome := workspace(t)
	writeTask(t, root)
	corrupt(t, orbitHome)

	code, out, errOut := run(t, "check")
	if code == 0 {
		t.Fatal("check of a damaged record exited 0")
	}

	if !strings.Contains(out, "is damaged") || !strings.Contains(out, "SQLite says") {
		t.Errorf("check said %q, which does not say the record is damaged", out)
	}

	if !strings.Contains(out, "malformed") {
		t.Errorf("check said %q, without the line SQLite answered with", out)
	}

	if !strings.Contains(errOut, "orbit export") {
		t.Errorf("check failed with %q, which does not say what to do next", errOut)
	}

	// The migration in front of every command cannot finish on this file
	// either, and its five lines of SQLite would print above the sentence
	// this command exists to print.
	if strings.Contains(errOut, "malformed") {
		t.Errorf("check failed with %q, which is the dump it was written to replace", errOut)
	}
}

// TestNothingIsCheckedBeforeACommandUntilItIsAskedFor. The check costs a
// full read of the file, which is not a thing to do before every `orbit
// list` on the strength of nobody having chosen it.
func TestNothingIsCheckedBeforeACommandUntilItIsAskedFor(t *testing.T) {
	root, orbitHome := workspace(t)
	writeTask(t, root)
	unsound(t, orbitHome)

	code, _, errOut := run(t, "version")
	if code != 0 {
		t.Fatalf("version exited %d: %s", code, errOut)
	}

	if strings.Contains(errOut, "damaged") {
		t.Errorf("a command nobody asked to check the record said %q", errOut)
	}
}

// TestTheCheckBeforeACommandWarnsAndLetsItRun. Damage never stops the
// command, because the command might be the export that gets what is left
// out of the file.
func TestTheCheckBeforeACommandWarnsAndLetsItRun(t *testing.T) {
	root, orbitHome := workspace(t)
	writeTask(t, root)

	if code, _, errOut := run(t, "settings", "check-record", "on"); code != 0 {
		t.Fatalf("turning the check on exited %d: %s", code, errOut)
	}

	unsound(t, orbitHome)

	code, out, errOut := run(t, "version")
	if code != 0 {
		t.Fatalf("a command over a damaged record exited %d: %s", code, errOut)
	}

	if !strings.Contains(errOut, "damaged") {
		t.Errorf("the command said %q, without a word about the record being damaged", errOut)
	}

	if !strings.Contains(out, "orbit") {
		t.Errorf("the command printed %q, so the warning stopped it running", out)
	}
}

func TestCheckRecordIsASettingLikeAnyOther(t *testing.T) {
	_, orbitHome := workspace(t)

	code, out, errOut := run(t, "settings", "check-record", "on")
	if code != 0 {
		t.Fatalf("set check-record on exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "check-record is now on") {
		t.Errorf("set said %q, which does not say what the setting now is", out)
	}

	if cfg := settings(t, orbitHome); !cfg.CheckRecord {
		t.Error("check-record is off on disk after being turned on")
	}

	if code, _, errOut := run(t, "settings", "check-record", "off"); code != 0 {
		t.Fatalf("set check-record off exited %d: %s", code, errOut)
	}

	if cfg := settings(t, orbitHome); cfg.CheckRecord {
		t.Error("check-record is on on disk after being turned off")
	}
}

// TestCheckRefusesAFlagItDoesNotHave. The command takes none, and parse
// still runs so that a mistyped one is answered with the same shape every
// other command answers with.
func TestCheckRefusesAFlagItDoesNotHave(t *testing.T) {
	root, _ := workspace(t)
	writeTask(t, root)

	code, _, errOut := run(t, "check", "-everything")
	if code == 0 {
		t.Fatal("check accepted a flag it does not have")
	}

	if !strings.Contains(errOut, "everything") {
		t.Errorf("check said %q, without the flag it did not understand", errOut)
	}
}
