package cli

// What the readings are made of, and the two verbs that need this terminal.
//
// internal/verb reads the settings, the engines and their windows for
// itself: those are facts about this build and this machine, and it can ask
// them directly. What it cannot ask is which repositories are in view — that
// is the board, and whose board it is depends on who is asking — nor where a
// way in is allowed to write files, nor whether it has a terminal to give
// away.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/export"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/view"
)

// Facts is everything Orbit has been told, across every repository the board
// knows.
func (w world) Facts() ([]knowledge.Fact, error) {
	return knowsAllPort(w.reader, w.store)(), nil
}

// Board is every task under the root this command line was pointed at.
func (w world) Board() (board.Board, error) {
	b, _, err := w.reader.Refresh()
	if err != nil {
		return board.Board{}, fmt.Errorf("read the board: %w", err)
	}

	return b, nil
}

// Log is one task's record, folded by the package that owns the format.
func (w world) Log(repoPath, id string) ([]view.Entry, error) {
	return w.reader.Log(repoPath, id)
}

// Export writes the record back out as JSON lines, one file per task.
func (w world) Export(into, only string) (string, error) {
	out, err := export.Run(w.store, into, only)
	// Only when nothing came out and something went wrong. A record that is
	// simply empty exported successfully, and refusing on it reads as a
	// command that failed to run at all.
	if err != nil && out.Tasks == 0 {
		return "", err
	}

	said := fmt.Sprintf("%d tasks, %d events and %d messages written to %s",
		out.Tasks, out.Events, out.Messages, into)

	if err != nil {
		said += fmt.Sprintf(" (%v)", err)
	}

	return said, nil
}

// Take hands this terminal to the engine that last walked the task, in the
// task's own checkout.
//
// The process keeps the terminal for as long as the engine holds it: this is
// the reader sitting down in the worktree, not a command that starts
// something and returns. What comes back is the sentence said afterwards.
func (w world) Take(id, repoPath string) (string, error) {
	b, err := w.Board()
	if err != nil {
		return "", err
	}

	var row view.Task

	for _, t := range b.Tasks {
		if t.ID == id {
			row = t

			break
		}
	}

	if row.ID == "" {
		return "", fmt.Errorf("no task %q on the board", id)
	}

	cmd, err := takePort(w.reader, w.engines)(row)
	if err != nil {
		return "", err
	}

	if cmd == nil {
		return "", errors.New("no engine has walked " + id + " yet, so there is no session to carry on")
	}

	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return "", err
		}
	}

	return "back from " + row.Engine + " in " + id, nil
}
