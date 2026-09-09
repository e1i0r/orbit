package task

// Everything ever said about a task, in one file.
//
// The reason this exists is a person blocked: the engine they were working
// with has run out of quota for three hours, and everything it knew feels
// stuck inside it. It is not. What was said is already in the record —
// internal/cli files every turn of an interactive session back into it — and
// what was written is in the worktree. What was missing was a way to hand
// all of it to whichever program still has budget.
//
// So this is a rendering and not a second store. The record is the truth,
// this is it read out as prose, and it is rewritten rather than appended to:
// a file that accumulated its own copy would drift from the record the first
// time anything was retracted.
//
// It is under the task's own directory and not in the worktree. A worktree
// is removed when somebody prunes it, and the conversation would go with it;
// the task's directory is Orbit's and outlives every checkout.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// historyFile is what the rendering is called inside the task's directory.
const historyFile = "history.md"

// History is the conversation about a task, as markdown.
//
// Every turn anybody had with any program, in the order it was said, each
// one carrying who said it. Which program it was said in is a property of
// the turn and not of the file: it is one conversation about one task that
// happened to be had with more than one tool.
func History(s *store.Store, t Task) (string, error) {
	// A window built without a store is one that cannot read the record,
	// and saying so is the answer. It reaches here because the session
	// gesture is offered before anything is known about what is behind it.
	if s == nil {
		return "", fmt.Errorf("task %s: there is no store to read the record from", t.ID)
	}

	events, err := Events(s, t)
	if err != nil {
		return "", fmt.Errorf("read the record of %s: %w", t.ID, err)
	}

	return historyOf(t, events), nil
}

// WriteHistory renders it and leaves it in the task's directory, answering
// where it put it.
//
// Written rather than generated on demand because what reads it is another
// program: an engine handed a terminal can open a file, and asking it to run
// a command to produce one is a step it may not take.
func WriteHistory(s *store.Store, t Task) (string, error) {
	body, err := History(s, t)
	if err != nil {
		return "", err
	}

	dir, err := s.TaskDir(t.ID)
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, historyFile)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", fmt.Errorf("write the history of %s: %w", t.ID, err)
	}

	return path, nil
}

// historyOf is the rendering itself, kept apart from the disk so it can be
// asserted without one.
func historyOf(t Task, events []record.Event) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", t.ID)

	if title := firstLine(t.Text); title != "" {
		fmt.Fprintf(&b, "%s\n\n", title)
	}

	b.WriteString("Everything said about this task, oldest first, whichever program it was ")
	b.WriteString("said in. Written by Orbit from the task's record.\n")

	turns := 0

	for _, e := range events {
		line := turnOf(e)
		if line == "" {
			continue
		}

		if turns == 0 {
			b.WriteString("\n---\n")
		}

		turns++

		// Bold and not a heading. What a turn carries is somebody else's
		// markdown, headings and all, and a heading of ours above it would
		// be a document whose outline is the engine's prose rather than
		// the conversation.
		fmt.Fprintf(&b, "\n**%s** · %s\n\n%s\n", spokeBy(e), when(e.At), line)
	}

	if turns == 0 {
		b.WriteString("\nNothing has been said about it yet.\n")
	}

	return b.String()
}

// turnOf is what one event contributes to the conversation, and nothing for
// the events that are not somebody speaking.
//
// The task's own text is not a turn: it is the heading above all of them,
// and repeating it as the first thing said would put the brief in the
// conversation twice.
func turnOf(e record.Event) string {
	switch e.Kind {
	case record.TaskDialogue, record.TaskNoted:
		return strings.TrimSpace(e.Text)
	case record.PhaseFinished, record.PhaseFailed:
		return strings.TrimSpace(e.Text)
	}

	return ""
}

// spokeBy is who spoke: the engine when one did, the person when they did,
// and Orbit itself for the rest.
func spokeBy(e record.Event) string {
	for _, key := range []string{"by", "engine"} {
		if who := e.Data[key]; who != "" {
			return who
		}
	}

	if e.Phase != "" {
		return "the " + e.Phase + " phase"
	}

	return "orbit"
}

// when is a timestamp a person reads, and nothing for a clock that never
// answered.
func when(at time.Time) string {
	if at.IsZero() {
		return "at no recorded time"
	}

	return at.Format("2006-01-02 15:04")
}

// firstLine is a task's title: the first line of what was written in it.
func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")

	return line
}
