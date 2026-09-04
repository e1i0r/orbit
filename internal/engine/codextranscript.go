package engine

// Reading codex's transcript: one file of JSON lines per session, filed
// under the day it ran, with the directory it ran in written inside it.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Transcript is what was said in a session codex was opened in dir for.
func (c Codex) Transcript(dir string, since time.Time) ([]Turn, error) {
	if dir == "" {
		return nil, nil
	}

	turns, err := codexTurns(dir, c.Name(), since)
	if err != nil {
		return nil, err
	}

	return sorted(turns), nil
}

// codexTurns reads the sessions written while this one was open.
//
// The directory is not in the path — the tree is year, month and day — so
// every session of the day the reader came back is opened and asked which
// directory it ran in. What keeps that cheap is the same thing that keeps
// it correct: a file untouched while the terminal was out is not this
// session's.
func codexTurns(dir, engineName string, since time.Time) ([]Turn, error) {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		user, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("reading the session transcript: %w", err)
		}

		home = filepath.Join(user, ".codex")
	}

	files, err := filepath.Glob(filepath.Join(home, "sessions", "*", "*", "*", "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("looking for session transcripts in %s: %w", home, err)
	}

	names := dirNames(dir)

	var turns []Turn

	for _, path := range files {
		fresh, err := writtenSince(path, since)
		if err != nil {
			return nil, err
		}

		if !fresh {
			continue
		}

		read, err := codexFileTurns(path, engineName, names, since)
		if err != nil {
			return nil, err
		}

		turns = append(turns, read...)
	}

	return turns, nil
}

// codexRow is the part of one line of that file this reads: the header row
// that says where the session ran, and the conversation rows.
type codexRow struct {
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// codexPayload is what those rows carry. The header's cwd, and a message's
// role and content — the content of a message being a list of blocks, each
// with the text of one, whatever kind of text it is.
type codexPayload struct {
	Type    string `json:"type"`
	Cwd     string `json:"cwd"`
	Role    string `json:"role"`
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// codexFileTurns is one session file, kept only if it ran in one of the
// names the directory answers to.
//
// The whole file is read before that is known, because the header is the
// first row and a file whose header this cannot read is a file whose
// session ran somewhere it cannot say. Reading the rows and dropping them
// costs a file; guessing costs a task the conversation of another one.
func codexFileTurns(path, engineName string, names []string, since time.Time) ([]Turn, error) {
	var (
		turns []Turn
		here  bool
	)

	read := func(line []byte) {
		var row codexRow

		// A line this cannot read is a row shape it does not know: the file
		// is codex's, and it carries reasoning, token counts and turn
		// contexts beside the conversation.
		if err := json.Unmarshal(line, &row); err != nil {
			return
		}

		var payload codexPayload
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			return
		}

		if row.Type == "session_meta" {
			here = slices.Contains(names, payload.Cwd)
			return
		}

		// response_item and not event_msg: codex writes the same prompt and
		// the same answer under both, and a conversation filed twice is a
		// conversation nobody reads.
		if row.Type != "response_item" || payload.Type != "message" {
			return
		}

		if turn, ok := codexTurn(row.Timestamp, payload, engineName); ok && turn.At.After(since) {
			turns = append(turns, turn)
		}
	}

	if err := eachLine(path, read); err != nil {
		return nil, err
	}

	if !here {
		return nil, nil
	}

	return turns, nil
}

// codexTurn is one message row, and false for one nobody said.
func codexTurn(at time.Time, payload codexPayload, engineName string) (Turn, bool) {
	by := ""

	switch payload.Role {
	case "user":
		by = Operator
	case "assistant":
		by = engineName
	default:
		return Turn{}, false
	}

	var parts []string

	for _, block := range payload.Content {
		if text := strings.TrimSpace(block.Text); text != "" {
			parts = append(parts, text)
		}
	}

	text := strings.Join(parts, "\n\n")
	if text == "" || notSaid(text) {
		return Turn{}, false
	}

	return Turn{At: at, By: by, Text: text}, true
}
