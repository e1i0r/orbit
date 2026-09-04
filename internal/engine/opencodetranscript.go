package engine

// Reading opencode's transcript: rows in its own database, a session per
// directory, a message per turn and the words in the parts under it.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // the driver the query below is opened with
)

// opencodeSaid is every text part of every message of a session that ran in
// one of the directories asked about, oldest first.
//
// The role lives in the message's JSON and the words in the part's, so both
// come back whole and are read here rather than picked at in SQL: this is
// another program's schema, and a query that reaches inside its documents
// is a query that breaks on an upgrade it has no say in.
const opencodeSaid = `
SELECT m.data, p.data, p.time_created
FROM part p
JOIN message m ON m.id = p.message_id
JOIN session s ON s.id = p.session_id
WHERE s.directory IN (?, ?) AND p.time_created > ?
ORDER BY p.time_created`

// Transcript is what was said in a session opencode was opened in dir for.
func (o OpenCode) Transcript(dir string, since time.Time) ([]Turn, error) {
	if dir == "" {
		return nil, nil
	}

	turns, err := opencodeTurns(dir, o.Name(), since)
	if err != nil {
		return nil, err
	}

	return sorted(turns), nil
}

// opencodeTurns asks that database, and asks nothing of a machine where
// opencode has never run.
func opencodeTurns(dir, engineName string, since time.Time) (turns []Turn, err error) {
	path, err := opencodeDB()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}

	// Read only, and said so in the connection rather than assumed from the
	// queries: this file belongs to another program, which may be running.
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	defer func() {
		if cerr := db.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing the session transcript %s: %w", path, cerr)
		}
	}()

	// Both names the directory answers to, and the same one twice when it
	// only answers to one: opencode writes down the path its process was
	// started in, which on macOS is the resolved one.
	names := dirNames(dir)
	real := names[len(names)-1]

	rows, err := db.Query(opencodeSaid, names[0], real, since.UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("reading the session transcript %s: %w", path, cerr)
		}
	}()

	turns, err = opencodeScan(rows, engineName)
	if err != nil {
		return nil, fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	return turns, nil
}

// opencodeDB is where opencode keeps it, under the data directory the
// desktop convention gives it.
func opencodeDB() (string, error) {
	if data := os.Getenv("XDG_DATA_HOME"); data != "" {
		return filepath.Join(data, "opencode", "opencode.db"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("reading the session transcript: %w", err)
	}

	return filepath.Join(home, ".local", "share", "opencode", "opencode.db"), nil
}

// opencodeScan turns those rows into turns.
//
// A row whose documents this cannot read is skipped rather than fatal, for
// the reason a line of the other transcripts is: the shape is opencode's
// and it gains kinds between releases.
func opencodeScan(rows *sql.Rows, engineName string) ([]Turn, error) {
	var turns []Turn

	for rows.Next() {
		var (
			message, part string
			at            int64
		)

		if err := rows.Scan(&message, &part, &at); err != nil {
			return nil, err
		}

		if turn, ok := opencodeTurn(message, part, at, engineName); ok {
			turns = append(turns, turn)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return turns, nil
}

// opencodeTurn is one part of one message: who the message was from, and
// the words of the part when the part is words at all.
//
// A part is also how a tool call and its output are kept, which is why the
// kind is checked: reasoning and tool parts are the half of the session
// this deliberately leaves behind.
func opencodeTurn(message, part string, at int64, engineName string) (Turn, bool) {
	var m struct {
		Role string `json:"role"`
	}

	if err := json.Unmarshal([]byte(message), &m); err != nil {
		return Turn{}, false
	}

	var p struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	if err := json.Unmarshal([]byte(part), &p); err != nil || p.Type != "text" {
		return Turn{}, false
	}

	by := ""

	switch m.Role {
	case "user":
		by = Operator
	case "assistant":
		by = engineName
	default:
		return Turn{}, false
	}

	text := strings.TrimSpace(p.Text)
	if text == "" || notSaid(text) {
		return Turn{}, false
	}

	return Turn{At: time.UnixMilli(at), By: by, Text: text}, true
}
