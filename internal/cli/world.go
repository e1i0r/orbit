package cli

// What the verbs reach the machine through.
//
// internal/verb declares every action Orbit can be asked for and does each
// of them once; what it cannot do is open a repository, count what is
// unread, or run `gh`. Those are this package's, and they arrive there
// through the ports declared beside the verbs.
//
// One implementation, filled here, used by all four ways in. A second one
// would be the thing the verb package was written to stop.

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/supervisor"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// world is the machine, as the verbs reach it.
type world struct {
	store   *store.Store
	reader  *board.Reader
	engines map[string]engine.Engine
	words   *words.Printer
}

// newWorld is the one every way in builds.
func newWorld(s *store.Store, r *board.Reader, p *words.Printer) world {
	return world{store: s, reader: r, engines: newEngines(), words: p}
}

func (w world) Store() *store.Store { return w.store }

// Find is the task one id in one repository means.
//
// An empty id answers the repository alone, which is what writing a task
// needs: there is no task yet, and the repository is what it will be
// against.
func (w world) Find(id, repoPath string) (task.Task, repo.Repo, error) {
	var one repo.Repo

	if repoPath != "" {
		opened, err := repo.Open(repoPath)
		if err != nil {
			return task.Task{}, repo.Repo{}, fmt.Errorf("open repository %q: %w", repoPath, err)
		}

		one = opened
	}

	if id == "" {
		return task.Task{}, one, nil
	}

	t, err := task.Load(w.store, one, id)
	if err != nil {
		return task.Task{}, one, fmt.Errorf("load task %q: %w", id, err)
	}

	return t, one, nil
}

// Unread is how many finished tasks nobody has looked at, which is the one
// number task.Start will not work out for itself: internal/board is the
// reader of that format and internal/task must not become a second one.
func (w world) Unread(repoPath string) (int, error) {
	if w.reader != nil {
		b, _, err := w.reader.Refresh()
		if err != nil {
			return 0, err
		}

		return board.Unread(b), nil
	}

	return unreadCount(w.store, repoPath)
}

// Looked tells the board to look again, because the set of tasks changed.
//
// Refresh re-reads the records of the tasks it already knows; a task written
// a moment ago is not one of those. A reader with no board — the command
// line, which builds one per command — has nothing to tell.
func (w world) Looked() error {
	if w.reader == nil {
		return nil
	}

	return w.reader.Rescan()
}

// Learn writes down something true about the code.
func (w world) Learn(fact knowledge.Fact) error {
	if err := fact.Validate(); err != nil {
		return err
	}

	if _, err := knowledge.NewStore(w.store.Root()).Save(fact); err != nil {
		return err
	}

	return nil
}

// Say puts something in the supervisor's thread, on whichever channel the
// caller is: the same call `orbit supervisor` makes.
func (w world) Say(text, by, about string) error {
	if err := supervisor.Record(w.store, "", by, "orbit", about, "", text); err != nil {
		return fmt.Errorf("record supervisor message: %w", err)
	}

	return nil
}

// Deliver hands a task's work to the world by running the command line's own
// verb, with a buffer where the terminal would be.
//
// The command itself and not a copy of it: what opening a pull request means
// — syncing the base branch, writing the body from the task's story, which
// remote, what to do when gh is missing — is a page of decisions, and a
// second copy would be a second answer to every one of them.
func (w world) Deliver(_ context.Context, t task.Task, verb string) (string, error) {
	run, known := map[string]func(Context, []string) error{
		"pr":       createPR,
		"pr merge": mergePR,
		"pr close": closePR,
	}[verb]
	if !known {
		return "", fmt.Errorf("%q is not something a task can be delivered by", verb)
	}

	args := []string{}
	if t.Repo.Path != "" {
		args = append(args, "-repo", t.Repo.Path)
	}

	var out, warned bytes.Buffer

	// Both writers, joined: these commands say the outcome on one and the
	// warnings on the other, and a reader who is not at a terminal has one
	// place to read them.
	err := run(Context{Out: &out, Err: &warned, Words: w.words}, append(args, t.ID))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String() + "\n" + warned.String()), nil
}

// Words is the reader's own language, so that a verb's sentence comes back
// in it rather than in the one this package happens to be written in.
func (w world) Words() *words.Printer { return w.words }
