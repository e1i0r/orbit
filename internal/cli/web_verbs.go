package cli

// What the browser's buttons actually do.
//
// Each one is the same call the command line makes — `orbit run`, `orbit
// pause`, `orbit cancel`, `orbit approve` — so a task started from a browser
// tab and one started from a terminal are the same run, written down the
// same way. Nothing here is a second implementation of a verb; it is the
// adapter that lets internal/web ask for one without naming internal/task.
//
// A task is found by its id and the repository it is against, because that
// pair is what internal/web has: it reads the board, and a row carries both.

import (
	"context"
	"fmt"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/web"
)

// by is who the record says did it. "operator" is the word `orbit requeue`
// and the cockpit both use for a person at the controls, and a browser tab
// is the same person: a second word would split one reader into two in the
// record.
const by = "operator"

// hands is the verbs port, over one store and the board behind it.
type hands struct {
	store *store.Store
	board *board.Reader
}

// Start runs a task in a process of its own.
//
// The unread count comes off a fresh read of the board, because that is
// where the cap is measured and task.Start will not work it out for itself:
// internal/board is the reader of that format, and internal/task must not
// become a second one.
func (h hands) Start(id, at string) error {
	t, err := h.find(id, at)
	if err != nil {
		return err
	}

	unread := 0

	if b, _, readErr := h.board.Refresh(); readErr == nil {
		unread = board.Unread(b)
	}

	if _, err := task.Start(h.store, t, t.Flow, unread); err != nil {
		return err
	}

	return nil
}

// Pause, Resume and Cancel are the three words a run understands, written
// where the run will look for them.
func (h hands) Pause(id, at string) error  { return h.control(id, at, "pause") }
func (h hands) Resume(id, at string) error { return h.control(id, at, "resume") }

// Continue is the word that lets a phase past the gate its flow asked it to
// stop at — what the cockpit's [h] sends, and not what [r] sends.
func (h hands) Continue(id, at string) error { return h.control(id, at, "continue") }

// Skip lets the run past the phase itself rather than past its gate.
func (h hands) Skip(id, at string) error { return h.control(id, at, "skip") }

// Cancel asks the process to stop rather than writing a word for it: a
// cancel a reader pressed should not wait for the next phase boundary, and
// `orbit cancel` does not either. The run turns the signal into a cancelled
// context and writes down that it was stopped, which is why this asks
// instead of killing.
func (h hands) Cancel(id, at string) error {
	t, err := h.find(id, at)
	if err != nil {
		return err
	}

	return task.Cancel(h.store, t)
}

func (h hands) control(id, at, word string) error {
	t, err := h.find(id, at)
	if err != nil {
		return err
	}

	return task.Control(h.store, t, word)
}

// Note leaves a word for the phase that starts next. A note written while
// nothing is running is still recorded, and read when a run begins.
func (h hands) Note(id, at, text string) error {
	t, err := h.find(id, at)
	if err != nil {
		return err
	}

	return task.Note(h.store, t, text)
}

// Direct records a correction and stops the run in flight so that the next
// one starts having read it. With restart, it starts that next one too,
// waiting for the old one to actually be gone first — which is why it can
// take half a minute to answer.
func (h hands) Direct(id, at, text string, restart bool) error {
	t, err := h.find(id, at)
	if err != nil {
		return err
	}

	if !restart {
		return task.Direct(h.store, t, by, text)
	}

	unread := 0

	if b, _, readErr := h.board.Refresh(); readErr == nil {
		unread = board.Unread(b)
	}

	if _, err := task.Reopen(context.Background(), h.store, t, by, text, t.Flow, unread); err != nil {
		return err
	}

	return nil
}

// Requeue takes a task back to the queue, stopping whatever holds it and
// waiting for it to be gone — otherwise the run's own task.cancelled lands
// after the requeue and the fold files the task under Done anyway.
func (h hands) Requeue(id, at, why string) error {
	t, err := h.find(id, at)
	if err != nil {
		return err
	}

	return task.Requeue(context.Background(), h.store, t, by, why)
}

// Approve says yes to what the dependency gate stopped the run for.
//
// What is approved is what is pending, read here and not taken from the
// caller: a reader approves the question they were shown, and a list from
// outside would be a yes to a question nobody put.
func (h hands) Approve(id, at string) ([]string, error) {
	t, err := h.find(id, at)
	if err != nil {
		return nil, err
	}

	f, err := flowOfTask(h.store, t)
	if err != nil {
		return nil, err
	}

	names := task.Pending(h.store, t, f)
	if len(names) == 0 {
		return nil, fmt.Errorf("%s has added no dependency waiting on you", t.ID)
	}

	if err := task.Approve(h.store, t, names); err != nil {
		return nil, err
	}

	return names, nil
}

// History is everything ever said about the task, as markdown — the same
// rendering `orbit history` prints and the window's history tab draws.
func (h hands) History(id, at string) (string, error) {
	t, err := h.find(id, at)
	if err != nil {
		return "", err
	}

	return task.History(h.store, t)
}

// Standing is what the task can be asked for right now: whether a process
// holds it, and what is waiting to be approved.
//
// Held is asked of the run marker and not of the band. A run parked at a
// gate sits in "needs you" and is alive, so a page choosing its buttons off
// the band offered Start on a task something was already running — a button
// whose whole job was to be refused.
//
// It answers an empty standing rather than an error for a task that cannot
// be loaded: this is read beside a task on every look at it, and a screen
// that failed because one of its buttons could not be worked out is a screen
// nobody can use.
func (h hands) Standing(id, at string) web.Standing {
	t, err := h.find(id, at)
	if err != nil {
		return web.Standing{}
	}

	var now web.Standing

	if _, alive, err := task.Alive(h.store, t); err == nil {
		now.Held = alive
	}

	if f, err := flowOfTask(h.store, t); err == nil {
		now.Pending = task.Pending(h.store, t, f)
	}

	return now
}

// find is the task one id in one repository means.
//
// A task with no repository is loaded against the zero one, which is what it
// was written against: repo.Open on an empty path would open whatever
// directory the server happens to be running in.
func (h hands) find(id, at string) (task.Task, error) {
	var one repo.Repo

	if at != "" {
		opened, err := repo.Open(at)
		if err != nil {
			return task.Task{}, fmt.Errorf("open repository %q: %w", at, err)
		}

		one = opened
	}

	t, err := task.Load(h.store, one, id)
	if err != nil {
		return task.Task{}, fmt.Errorf("load task %q: %w", id, err)
	}

	return t, nil
}
