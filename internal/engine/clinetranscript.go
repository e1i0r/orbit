package engine

// What was said in a cline session, read back from where cline keeps it.

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

// clineSessions is the sessions cline opened in a directory since a time.
// cline writes its times as ISO text in UTC, which sorts as it reads.
const clineSessions = `
SELECT messages_path FROM sessions
WHERE cwd IN (?, ?) AND started_at > ? AND messages_path IS NOT NULL
ORDER BY started_at`

// clineTime is how cline writes a time: UTC, to the millisecond, always the
// same width, so that comparing two of them as text compares the times.
const clineTime = "2006-01-02T15:04:05.000Z"

// Transcript is what was said in the sessions cline was opened in dir for.
//
// cline keeps an index of its sessions in SQLite, with the directory each
// was started in, and each session's messages in a JSON file of its own
// beside it. The index says which files; the files say what was said.
func (c Cline) Transcript(dir string, since time.Time) ([]Turn, error) {
	if dir == "" {
		return nil, nil
	}

	paths, err := clineSessionFiles(dir, since)
	if err != nil {
		return nil, err
	}

	var turns []Turn

	for _, path := range paths {
		said, err := clineTurns(path, c.Name(), since)
		if err != nil {
			return nil, err
		}

		turns = append(turns, said...)
	}

	return sorted(turns), nil
}

// clineData is cline's data directory, as cline itself resolves it.
func clineData() (string, error) {
	if data := os.Getenv("CLINE_DATA_DIR"); data != "" {
		return data, nil
	}

	if root := os.Getenv("CLINE_DIR"); root != "" {
		return filepath.Join(root, "data"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("reading the session transcript: %w", err)
	}

	return filepath.Join(home, ".cline", "data"), nil
}

// clineSessionFiles is the messages file of every session cline opened in
// dir since the given time.
func clineSessionFiles(dir string, since time.Time) (paths []string, err error) {
	data, err := clineData()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(data, "db", "sessions.db")
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}

	// Read only, and said so in the connection rather than assumed from the
	// query: this file belongs to another program, which may be running.
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	defer func() {
		if cerr := db.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing the session transcript %s: %w", path, cerr)
		}
	}()

	names := dirNames(dir)

	rows, err := db.Query(clineSessions, names[0], names[len(names)-1], since.UTC().Format(clineTime))
	if err != nil {
		return nil, fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("reading the session transcript %s: %w", path, cerr)
		}
	}()

	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("reading the session transcript %s: %w", path, err)
		}

		paths = append(paths, p)
	}

	return paths, rows.Err()
}

// clineMessages is a session's messages file, as far as a transcript needs.
type clineMessages struct {
	Messages []struct {
		Role    string `json:"role"`
		TS      int64  `json:"ts"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"messages"`
}

// clineTurns is what one messages file says was said since a time: the
// text of each message, by the operator or by the engine. Tool calls and
// thinking are the record's to show, not the conversation's.
func clineTurns(path, engineName string, since time.Time) ([]Turn, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	var file clineMessages
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	var turns []Turn

	for _, m := range file.Messages {
		at := time.UnixMilli(m.TS)
		if !at.After(since) {
			continue
		}

		var text []string

		for _, c := range m.Content {
			if c.Type == "text" && strings.TrimSpace(c.Text) != "" {
				text = append(text, strings.TrimSpace(c.Text))
			}
		}

		if len(text) == 0 {
			continue
		}

		by := engineName
		if m.Role == "user" {
			by = Operator
		}

		turns = append(turns, Turn{At: at, By: by, Text: strings.Join(text, "\n\n")})
	}

	return turns, nil
}
