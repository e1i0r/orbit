package cli

// What the browser's buttons actually do.
//
// They ask internal/verb, which is the same thing `orbit run` and the
// cockpit's [s] ask: a task started from a browser tab and one started from
// a terminal are the same run, written down the same way, refused for the
// same reasons and reported in the same words.
//
// This file used to be a method per verb — Start, Pause, Resume, Continue,
// Skip, Cancel, Note, Direct, Requeue, Approve, Write, Deliver — each one a
// short call into internal/task, and each one a chance to differ from the
// command line's version of the same word. They did differ: the browser's
// resume was the cockpit's continue, and a note written here recorded a
// different author than a note written there. What is left is the adapter,
// and the two readings that are not verbs.
//
// A task is found by its id and the repository it is against, because that
// pair is what internal/web has: it reads the board, and a row carries both.

import (
	"context"
	"fmt"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/web"
	"github.com/e1i0r/orbit/internal/words"
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
	words *words.Printer
}

// Ask does one verb, by the name the declaration gives it.
//
// The whole of the browser's doing, in one method, because the browser is
// not where it is decided what a verb means. A name nothing answers to is a
// refusal with a sentence in it, which is what the page shows.
func (h hands) Ask(name string, in web.Asked) (web.Answered, error) {
	out, err := verb.Run(context.Background(), newWorld(h.store, h.board, h.words), name, verb.In{
		Task: in.Task,
		Repo: in.Repo,
		Args: in.Args,
		By:   by,
		Door: "the browser",
	})
	if err != nil {
		return web.Answered{}, err
	}

	return web.Answered{Said: out.Said, Of: out.Of, Saw: out.Saw}, nil
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

// flowOfTask is the flow a task walks, by the same reading `orbit run` makes
// of it: the task's own, then the one Orbit ships. Not the settings default,
// which is what the next task written gets.
func flowOfTask(s flow.Source, t task.Task) (flow.Flow, error) {
	chosen := t.Flow
	if chosen == "" {
		chosen = flow.Default
	}

	return flow.Resolve(s, chosen)
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
