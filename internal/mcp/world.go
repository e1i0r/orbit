package mcp

// What a verb reaches the machine through, on this way in.
//
// internal/verb's ports, filled for a tool call. They are the same ports the
// command line fills, so a task started by a model and one started at a
// terminal are the same run — refused for the same reasons, recorded with
// the same words, and read back the same way.

import (
	"context"
	"errors"
	"fmt"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/export"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/supervisor"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// world is internal/verb's ports over one tool call's store and board.
type world struct {
	sb *storeAndBoard
}

// world is the one every tool call builds.
func (sn Session) world(sb *storeAndBoard) world { return world{sb: sb} }

// context is what a verb that reaches the network is given. A tool call has
// no deadline of its own — the client's transport carries one — so this is
// the background and says so rather than inventing a timeout.
func (sn Session) context() context.Context { return context.Background() }

// Store is where a task is written down and read back.
func (w world) Store() *store.Store { return w.sb.store }

// Find is the task one id in one repository means.
func (w world) Find(id, repoPath string) (task.Task, repo.Repo, error) {
	var one repo.Repo

	if repoPath != "" {
		opened, err := repo.Open(repoPath)
		if err != nil {
			return task.Task{}, repo.Repo{}, err
		}

		one = opened
	}

	if id == "" {
		return task.Task{}, one, nil
	}

	t, err := task.Load(w.sb.store, one, id)
	if err != nil {
		return task.Task{}, one, err
	}

	return t, one, nil
}

// Unread is how many finished tasks nobody has looked at, off the board this
// call already folded rather than a second walk of the disk.
func (w world) Unread(string) (int, error) { return board.Unread(w.sb.board), nil }

// Looked is nothing here. The board was folded when this call began and is
// thrown away when it ends; there is no reader holding one open to tell.
func (w world) Looked() error { return nil }

// Facts is everything Orbit has been told, across every repository this
// session looks in.
func (w world) Facts() ([]knowledge.Fact, error) {
	ks := knowledge.NewStore(w.sb.store.Root())

	// Beside the error: Load answers with the facts it could read, so a file
	// with a typo in its header costs that file and leaves the rest.
	facts, err := ks.Load("")
	if err != nil {
		return knowledge.Every(facts), nil //nolint:nilerr // see above
	}

	for _, r := range w.sb.board.RepoList {
		own, err := ks.LoadRepo(r.Path)
		if err != nil {
			continue
		}

		facts = append(facts, own...)
	}

	return knowledge.Every(facts), nil
}

// Board is every task under the roots this session looks in, folded when
// this call began.
func (w world) Board() (board.Board, error) { return w.sb.board, nil }

// Log is one task's record, folded by the package that owns the format.
//
// A reader of its own, over the repository the task is in rather than over
// the session's roots: this is one task's log and not a walk of the board,
// and the board this call folded is already in hand for everything else.
func (w world) Log(repoPath, id string) ([]view.Entry, error) {
	return board.NewReader(w.sb.store, repoPath).Log(repoPath, id)
}

// Export writes the record back out as JSON lines, one file per task.
func (w world) Export(into, only string) (string, error) {
	out, err := export.Run(w.sb.store, into, only)
	if err != nil && out.Tasks == 0 {
		return "", err
	}

	return fmt.Sprintf("%d tasks, %d events and %d messages written to %s",
		out.Tasks, out.Events, out.Messages, into), nil
}

// Take is refused. It hands a terminal to an engine, and a tool call has no
// terminal to hand over.
func (w world) Take(string, string) (string, error) {
	return "", errors.New("taking the keyboard needs a terminal; run orbit take, or press t in the window")
}

// Deliver is refused. Opening, merging and closing a pull request are a
// person's decisions, and this is the server a model speaks to.
func (w world) Deliver(context.Context, task.Task, string) (string, error) {
	return "", errors.New("a pull request is a person's decision; ask them to run orbit pr")
}

// Say puts something in the supervisor's thread, as the model.
func (w world) Say(text, by, about string) error {
	return supervisor.Record(w.sb.store, "", by, "mcp", about, "", text)
}

// Learn writes down something true about the code.
func (w world) Learn(fact knowledge.Fact) error {
	if err := fact.Validate(); err != nil {
		return err
	}

	if _, err := knowledge.NewStore(w.sb.store.Root()).Save(fact); err != nil {
		return err
	}

	return nil
}

// Words is English. A tool call is read by a model and then quoted back to
// whoever is watching, and the language that reaches them is the client's to
// choose — not this server's, which has no reader of its own.
func (w world) Words() *words.Printer { return words.For("") }
