package task

// The control word: one line of text under a task's directory saying what
// the reader wants the run to do at its next phase boundary.
//
//	pause
//
// A file, and not a signal or a socket, for the reason the plan gives: it
// costs a poll interval, and it buys a channel that survives a crash, a
// reboot, and a reader who was not there when the word was written. A word
// left for a task nobody is running is not a mistake — it is waiting for the
// run that starts next.
//
// Both the command line and the window write through Control, so neither has
// a rule the other does not.

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/e1i0r/orbit/internal/store"
)

// The five words a run understands, and the whole of the vocabulary.
//
// pause and resume are the pair a reader flips; cancel ends the run where it
// stands; continue is resume's other half — it lets a phase past the gate
// its flow asked for — and skip lets the run past the phase itself.
const (
	wordPause    = "pause"
	wordResume   = "resume"
	wordCancel   = "cancel"
	wordContinue = "continue"
	wordSkip     = "skip"
)

// controlWords is every word Control accepts, in the order a refusal lists
// them. It is one list so that the door and the gate cannot disagree about
// what a word is: take below reads it too.
var controlWords = []string{wordPause, wordResume, wordCancel, wordContinue, wordSkip}

// Control tells a run what to do at its next phase boundary.
//
// It returns as soon as the word is on disk. Whether the run acted on it is a
// question the record answers, up to one poll later, and the caller is
// already reading it — the same contract Cancel has, for the same reason.
func Control(s *store.Store, t Task, word string) error {
	if !slices.Contains(controlWords, word) {
		return fmt.Errorf("%q is not something a run understands; the words are %s", word, strings.Join(controlWords, ", "))
	}

	path, err := s.ControlPath(t.Repo.Path, t.ID)
	if err != nil {
		return err
	}
	// 0600 like everything else under the state root, and one line, so that
	// `cat control` is a supported way of answering "what did I ask for?".
	if err := os.WriteFile(path, []byte(word+"\n"), 0o600); err != nil {
		return fmt.Errorf("tell task %s to %s: %w", t.ID, word, err)
	}

	return nil
}

// take reads the word a reader left and takes it off, so that one word moves
// a run once. A task with no word waiting is ("", nil): not having been asked
// for anything is an answer, not a fault.
//
// Read and then remove, which loses a word written in the instant between
// the two. The window for that is one file read wide, the loss is visible —
// the run does not do the thing that was asked — and the reader's answer is
// to ask again. What it buys is that nothing but a plain, `cat`-able file
// ever exists under the task directory, which is not true of the atomic
// alternative: a rename leaves a half-taken word behind when the run dies
// between the rename and the read, and nothing would ever clear it.
//
// A word this version does not know is taken off and treated as though there
// had been none. Control refuses every word but the five at the door, and
// that door is the only one the command line and the window use, so the only
// way to get one here is to write the file by hand — and ending somebody's
// run over a typo in a file they were experimenting with is worse than
// ignoring it. This is deliberately the opposite of what readMarker does with
// a damaged run marker (alive.go), because the two mistakes are not
// symmetrical: there, shrugging declares a live run abandoned; here, refusing
// kills a live run.
func take(s *store.Store, t Task) (string, error) {
	path, err := s.ControlPath(t.Repo.Path, t.ID)
	if err != nil {
		return "", err
	}

	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("read the control word of task %s: %w", t.ID, err)
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("clear the control word of task %s: %w", t.ID, err)
	}

	word := strings.TrimSpace(string(body))
	if !slices.Contains(controlWords, word) {
		return "", nil
	}

	return word, nil
}
