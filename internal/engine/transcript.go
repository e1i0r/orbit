package engine

// What was said in an interactive session, read back after it ended.
//
// A run is watched while it happens: the stream is parsed and the record
// gets a phase out of it. A session the reader was handed the terminal for
// is not — the cockpit suspends itself for the length of it and sees
// nothing. What the session leaves behind is the engine's own transcript on
// disk, and this reads it, so an hour of work in a worktree belongs to the
// task it was done on instead of to a file under a home directory nobody
// opens.
//
// Every engine keeps one and no two keep it alike: claude a file of JSON
// lines per directory, codex a file per session under the day it ran,
// opencode rows in a database. What is shared is the question, so this file
// holds the question and one file per engine holds its answer.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Operator is who the reader is in a transcript, in the word the record
// carries for them everywhere else.
const Operator = "operator"

// Turn is one thing said in a session: who said it, when, and the words.
//
// A tool call is not a turn. What a session did to the files is the
// timeline's business and is already visible in the worktree; this is the
// conversation, which is the half that exists nowhere else once the
// terminal is closed.
type Turn struct {
	At   time.Time
	By   string
	Text string
}

// sorted is the order a conversation is read in, oldest first. Each
// engine's own reading gathers turns per file or per row, and a session
// spread over more than one of either comes back interleaved otherwise.
func sorted(turns []Turn) []Turn {
	sort.SliceStable(turns, func(i, j int) bool { return turns[i].At.Before(turns[j].At) })

	return turns
}

// dirNames is the directory under the name it was given and the name it
// resolves to. An engine writes down the path its process was actually
// started in, and on macOS a worktree under /tmp is /private/tmp by then.
func dirNames(dir string) []string {
	dirs := []string{dir}

	if real, err := filepath.EvalSymlinks(dir); err == nil && real != dir {
		dirs = append(dirs, real)
	}

	return dirs
}

// writtenSince says whether a file was touched while the session was open.
//
// A transcript directory keeps every session ever opened — months of them,
// megabytes each — and a file nothing was written to during this one cannot
// hold a turn from it.
func writtenSince(path string, since time.Time) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	return !info.ModTime().Before(since), nil
}

// eachLine reads a file of one JSON object per line and hands each line to
// read.
//
// The lines are read one at a time rather than scanned, because a line here
// is as long as whatever was said on it: a scanner has a maximum, and one
// pasted file would take the rest of the session down with it.
func eachLine(path string, read func(line []byte)) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("reading the session transcript %s: %w", path, err)
	}

	// The close is reported like any other failure. A file this could not
	// let go of is a file something is wrong with, and a session that came
	// back short would look exactly like a session where little was said.
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing the session transcript %s: %w", path, cerr)
		}
	}()

	r := bufio.NewReader(f)

	for {
		line, lerr := r.ReadBytes('\n')

		if len(line) > 0 {
			read(line)
		}

		if lerr != nil {
			if errors.Is(lerr, io.EOF) {
				return nil
			}

			return fmt.Errorf("reading the session transcript %s: %w", path, lerr)
		}
	}
}

// notSaid is a row the tool wrote into the conversation as if somebody had
// said it: the context an engine opens a session with, the name of a slash
// command and what one printed, the notes a harness injects.
//
// None of it was typed by the reader or answered by the model, and all of
// it is written in the same place their words are.
func notSaid(text string) bool {
	for _, prefix := range []string{
		"<command-name>", "<command-message>", "<local-command-stdout>",
		"<system-reminder>", "<environment_context>", "<user_instructions>",
		"<task-notification>", "Caveat: The messages below",
	} {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}

	return false
}
