package engine

// Reading claude's transcript: one directory of JSON lines per directory a
// session was opened in, one file per session.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Transcript is what was said in a session claude was opened in dir for.
func (c Claude) Transcript(dir string, since time.Time) ([]Turn, error) {
	if dir == "" {
		return nil, nil
	}

	turns, err := claudeTurns(dir, c.Name(), since)
	if err != nil {
		return nil, err
	}

	return sorted(turns), nil
}

// claudeTurns reads the directories claude may have filed that session
// under.
func claudeTurns(dir, engineName string, since time.Time) ([]Turn, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("reading the session transcript: %w", err)
	}

	var turns []Turn

	// Both names the directory answers to: claude files a session under the
	// path it was actually started in.
	for _, name := range dirNames(dir) {
		read, err := claudeDirTurns(filepath.Join(home, ".claude", "projects", claudeSlug(name)), engineName, since)
		if err != nil {
			return nil, err
		}

		turns = append(turns, read...)
	}

	return turns, nil
}

// claudeSlug is what claude calls a directory: the path with every
// character that is not a letter or a digit turned into a dash. A dot
// becomes one too, which is why /Users/who/.orbit/worktrees/ab12/FRA-62 is
// kept under -Users-who--orbit-worktrees-ab12-FRA-62.
func claudeSlug(dir string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		}

		return '-'
	}, dir)
}

// claudeDirTurns reads every session file in one of those directories.
func claudeDirTurns(dir, engineName string, since time.Time) ([]Turn, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("looking for session transcripts in %s: %w", dir, err)
	}

	var turns []Turn

	for _, path := range files {
		fresh, err := writtenSince(path, since)
		if err != nil {
			return nil, err
		}

		if !fresh {
			continue
		}

		read := func(line []byte) {
			if turn, ok := claudeTurn(line, engineName); ok && turn.At.After(since) {
				turns = append(turns, turn)
			}
		}

		if err := eachLine(path, read); err != nil {
			return nil, err
		}
	}

	return turns, nil
}

// claudeRow is the part of one line of that file this reads. The rest of it
// is claude's bookkeeping: modes, costs, titles, snapshots.
type claudeRow struct {
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	IsMeta      bool      `json:"isMeta"`
	IsSidechain bool      `json:"isSidechain"`
	Message     struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// claudeTurn is what one line says, and false for a line that says nothing
// the reader had a hand in.
//
// A sidechain is a subagent talking to itself, and meta rows are what the
// tool tells itself before the session starts. Neither is the conversation
// somebody had.
func claudeTurn(line []byte, engineName string) (Turn, bool) {
	var row claudeRow

	// A line this cannot read is a row shape it does not know. The file is
	// claude's, not Orbit's, and it gains kinds between releases; the ones
	// this does not recognise are not its to fail on.
	if err := json.Unmarshal(line, &row); err != nil {
		return Turn{}, false
	}

	if row.IsMeta || row.IsSidechain {
		return Turn{}, false
	}

	by := ""

	switch row.Type {
	case "user":
		by = Operator
	case "assistant":
		by = engineName
	default:
		return Turn{}, false
	}

	text := claudeText(row.Message.Content)
	if text == "" || notSaid(text) {
		return Turn{}, false
	}

	return Turn{At: row.Timestamp, By: by, Text: text}, true
}

// claudeText is what was said on a row: the string a typed prompt is, or
// the text blocks of an answer.
//
// Thinking is not text and tool calls are not text, so both fall out here
// by having no block of that kind. A tool's answer comes back as a user row
// made of tool_result blocks, and falls out the same way.
func claudeText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var said string
	if err := json.Unmarshal(raw, &said); err == nil {
		return strings.TrimSpace(said)
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var parts []string

	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, strings.TrimSpace(b.Text))
		}
	}

	return strings.Join(parts, "\n\n")
}
