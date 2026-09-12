package cli

// The record ahead of the orbit reading it, and the way out of it.
//
// A record carries the schema version of the orbit that last wrote to it, and
// an orbit that knows less refuses to write into one that knows more — a
// migration cannot be taken back, so the refusal is the safe answer and it
// stays. What was wrong was that the refusal also stopped the whole program:
// every read was barred alongside the writes, so `orbit upgrade` refused to
// run and `orbit list` showed nothing, over a record the reader could still
// have looked at.
//
// So the record opens read-only when it is ahead, and only the writes are
// refused. Reading commands run and writing commands say to upgrade rather
// than writing into a shape this orbit does not know, which is the one
// refusal that stays. That is what this file holds down: both halves,
// because a fix that only opened the door would have taken the refusal
// with it.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite" // the record's own engine, to stamp a version this orbit has never heard of
)

// ahead stamps the record with a schema version this orbit does not know,
// which is what running one newer orbit once and then coming back to this one
// leaves behind.
//
// It writes through a handle of its own for the reason unsound does: this
// package may not import internal/db, and asking the record to declare itself
// newer than its reader is not a thing the record's own package should offer.
//
// The version is a number well past anything rather than this orbit's plus
// one, because that constant is on the other side of a boundary this file
// cannot cross — and a stamp that stopped being ahead would make every
// assertion below pass for the wrong reason.
const ahead = 999

func aheadOfThisOrbit(t *testing.T, orbitHome string) {
	t.Helper()

	if err := os.MkdirAll(orbitHome, 0o755); err != nil {
		t.Fatalf("make the home the record lives in: %v", err)
	}

	d, err := sql.Open("sqlite", filepath.Join(orbitHome, "orbit.db"))
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	// A table first, because SQLite has nowhere to put a version until the
	// file holds a page, and a stamp over an empty file is a stamp nobody
	// wrote.
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS ahead (id INTEGER PRIMARY KEY)`,
		fmt.Sprintf(`PRAGMA user_version = %d`, ahead),
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("stamp the record: %v", err)
		}
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close the record: %v", err)
	}
}

// stampedWith reads back the version the record says it is at.
func stampedWith(t *testing.T, orbitHome string) int {
	t.Helper()

	d, err := sql.Open("sqlite", filepath.Join(orbitHome, "orbit.db"))
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	found := 0

	if err := d.QueryRow(`PRAGMA user_version`).Scan(&found); err != nil {
		t.Fatalf("read the version back: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close the record: %v", err)
	}

	return found
}

// thereIsANewerRelease stands in for the release API answering that there is
// something to go to, so that upgrade has somewhere to go and this is not a
// test about the network being down.
func thereIsANewerRelease(t *testing.T) {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(releaseInfo{TagName: "v2.0.0"}); err != nil {
			t.Errorf("encode release: %v", err)
		}
	}))
	oldEndpoint, oldVersion := updateEndpoint, Version
	updateEndpoint, Version = ts.URL, "1.0.0"

	// The server outlives this helper and not the line that started it: the
	// test it stands in for runs after the helper has returned.
	t.Cleanup(func() {
		updateEndpoint, Version = oldEndpoint, oldVersion

		ts.Close()
	})
}

// TestTheWayOutRunsOverARecordAhead. `orbit upgrade` never reads the
// record: it asks a release API and writes over its own binary. For a command
// that never touches it the record ahead is no verdict at all.
func TestTheWayOutRunsOverARecordAhead(t *testing.T) {
	_, orbitHome := workspace(t)
	aheadOfThisOrbit(t, orbitHome)
	thereIsANewerRelease(t)

	code, out, errOut := run(t, "upgrade", "-check")
	if code != 0 {
		t.Fatalf("upgrade over a record ahead of it exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "v2.0.0") {
		t.Errorf("upgrade printed %q, without the version it found to go to", out)
	}

	if strings.Contains(errOut, "schema version") {
		t.Errorf("the command that is the way out was refused: %s", errOut)
	}
}

// TestTheCommandsThatAskNothingOfTheRecordStillRun. `version` says what this
// orbit is, which is the first thing a reader in this position needs to know,
// and `repos` reads the repositories and their worktrees, which are on disk
// and not in the record.
func TestTheCommandsThatAskNothingOfTheRecordStillRun(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"repos"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, orbitHome := workspace(t)
			aheadOfThisOrbit(t, orbitHome)

			code, out, errOut := run(t, args...)
			if code != 0 {
				t.Fatalf("%v over a record ahead of it exited %d: %s", args, code, errOut)
			}

			if strings.Contains(errOut, "schema version") {
				t.Errorf("%v was refused over a record it never reads: %s", args, errOut)
			}

			if strings.TrimSpace(out) == "" {
				t.Errorf("%v ran and printed nothing", args)
			}
		})
	}
}

// TestAReadingCommandRunsOverARecordAhead. `list` shows nothing the newer
// orbit could have reshaped: the board is folded out of events every orbit
// writes the same way, and a reader locked out of that over a file it could
// have looked at is what "stop the use of it" used to mean.
func TestAReadingCommandRunsOverARecordAhead(t *testing.T) {
	_, orbitHome := workspace(t)
	aheadOfThisOrbit(t, orbitHome)

	code, _, errOut := run(t, "board", "list")
	if code != 0 {
		t.Fatalf("list over a record ahead of it exited %d: %s", code, errOut)
	}

	if strings.Contains(errOut, "schema version") {
		t.Errorf("a reading command was refused over a record it could look at: %s", errOut)
	}
}

// TestAWritingCommandIsRefusedOverARecordAhead. The one refusal, and the part
// a fix aimed only at the dead end would have dropped: `new` writes a task
// into a schema it does not know the shape of, and the answer to that is
// still no — with the way out in the refusal, because telling a reader to
// upgrade has to name the command that does it.
func TestAWritingCommandIsRefusedOverARecordAhead(t *testing.T) {
	root, orbitHome := workspace(t)
	aheadOfThisOrbit(t, orbitHome)

	code, _, errOut := run(t, "board", "new", "-repo", filepath.Join(root, "payments"), "-id", "PAY-1", "a new task")
	if code == 0 {
		t.Fatal("new wrote into a record ahead of this orbit")
	}

	if !strings.Contains(errOut, "schema version") {
		t.Errorf("the refusal did not say what the trouble was: %s", errOut)
	}

	if !strings.Contains(errOut, "upgrade orbit") {
		t.Errorf("the refusal said %q, without the way out of it", errOut)
	}
}

// TestACommandThatSkipsTheRecordDoesNotWriteToItEither. Skipping the record
// is not a licence to touch it: if the way out stamped the file on its way
// past, the refusal standing in front of every other command would be
// standing over a record this orbit had already put its hands on — which is
// exactly the damage the refusal exists to prevent.
func TestACommandThatSkipsTheRecordDoesNotWriteToItEither(t *testing.T) {
	_, orbitHome := workspace(t)
	aheadOfThisOrbit(t, orbitHome)
	thereIsANewerRelease(t)

	code, _, errOut := run(t, "upgrade", "-check")
	if code != 0 {
		t.Fatalf("upgrade over a record ahead of it exited %d: %s", code, errOut)
	}

	if found := stampedWith(t, orbitHome); found != ahead {
		t.Errorf("the record is at %d after a command that was not supposed to touch it, want %d", found, ahead)
	}
}
