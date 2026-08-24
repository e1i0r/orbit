package cli

// The four things the window is allowed to do, as closures, and the one
// sweep that runs before its first frame.
//
// Every one of them is here rather than in internal/ui for the same reason:
// they all take a *store.Store, and internal/ui cannot name that type.
// That is arch.layers working, not a gap in it — a window able to name the
// store is a window one line away from writing to it. This file is the only
// place in Orbit where a gesture on a screen and a function that appends to
// a record are in the same scope.
//
// Nothing here interprets a refusal. Every error comes back exactly as the
// verb phrased it, because an interpretation is a second copy of that verb's
// rules and the two disagree the first time the verb changes its mind.

import (
	"errors"
	"os/exec"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/view"
)

// subject is the task a gesture is about, in the shape internal/task takes.
//
// Only the id and the repository's path and name are filled, and they are
// every field the verbs below reach for: the store hashes the path, and the
// name is what an error message says. The alternative — repo.Open on the
// row's path — would run git twice per keystroke to learn a remote and a
// base branch that nothing on this path reads, and would fail the gesture
// outright for a repository somebody had moved.
func subject(t view.Task) task.Task {
	return task.Task{ID: t.ID, Repo: repo.Repo{Path: t.RepoPath, Name: t.Repo}}
}

// controlPort is the five words a run understands, reaching the same
// function `orbit pause` calls.
func controlPort(s *store.Store) func(view.Task, string) error {
	return func(t view.Task, word string) error {
		return task.Control(s, subject(t), word)
	}
}

// markReadPort is the gesture that moves the unread brake back by one.
func markReadPort(s *store.Store) func(view.Task) error {
	return func(t view.Task) error {
		return task.MarkRead(s, subject(t))
	}
}

// startPort is the only thing in this program that a window can do which
// spends money, and it is a load away from being refused.
//
// task.Load is not ceremony. A row on the board is a fold of a record, and a
// record outlives the task file it describes — a state root somebody tidied
// by hand still draws rows. Starting one of those would spawn a run whose
// first act is to read a file that is not there, after the pid has been
// reported to the window and written into the band. Loading first turns that
// into one sentence naming the id, before anything is spawned.
//
// flowName is passed through exactly as the dialog chose it, empty included:
// which flow an unnamed run walks is `orbit run`'s rule — the task's own,
// then the built-in default — and answering it a second time here is how the
// window and the command line start disagreeing about what was run.
func startPort(s *store.Store) func(view.Task, string, int) (int, error) {
	return func(t view.Task, flowName string, unread int) (int, error) {
		loaded, err := task.Load(s, subject(t).Repo, t.ID)
		if err != nil {
			return 0, err
		}
		return task.Start(s, loaded, flowName, unread)
	}
}

// takePort builds the interactive session the window suspends itself for,
// and runs nothing.
//
// A task that has never run has no engine and no session, and that is not a
// failure: it is answered with no command and no error, which is what makes
// the window say so in the reader's own language rather than in this
// package's English. An engine this build does not have is a different
// thing, and it is named — that one is a fact about the binary, and a reader
// who is told which engine is missing can do something about it.
func takePort(r *board.Reader, engines map[string]engine.Engine) func(view.Task) (*exec.Cmd, error) {
	return func(t view.Task) (*exec.Cmd, error) {
		if t.Engine == "" {
			return nil, nil
		}
		eng, ok := engines[t.Engine]
		if !ok {
			return nil, &unknownEngineError{Name: t.Engine, ID: t.ID}
		}
		session, err := lastSession(r, t)
		if err != nil {
			return nil, err
		}
		dir, err := r.Worktree(t.RepoPath, t.ID)
		if err != nil {
			return nil, err
		}
		// A session of "" is takeCommand's rule and not this function's:
		// it answers with no command, and the window has one sentence for
		// a task there is nothing to carry on for.
		return takeCommand(eng, session, dir)
	}
}

// unknownEngineError is a record naming an engine this build cannot run.
//
// It is a type rather than fmt.Errorf because the window puts it in the
// activity band verbatim, and a sentence a reader is expected to act on is
// worth being able to test by name.
type unknownEngineError struct {
	Name string // the engine the record names
	ID   string // the task whose record names it
}

func (e *unknownEngineError) Error() string {
	return "task " + e.ID + " ran on " + e.Name + ", which this build of orbit cannot run"
}

// lastSession is the newest session id one task's record carries, or "" for
// a task whose engine never reported one.
//
// The newest and not the first: a task that has been run three times has
// three conversations behind it, and the one a reader means by "carry this
// on" is the one that produced what is on screen now. It reads through the
// same port the window polls, so a log the reader is already watching costs
// nothing to ask again.
func lastSession(r *board.Reader, t view.Task) (string, error) {
	entries, err := r.Log(t.RepoPath, t.ID)
	if err != nil {
		return "", err
	}
	var session string
	for _, e := range entries {
		if e.Session != "" {
			session = e.Session
		}
	}
	return session, nil
}

// reconcileAll closes the records of runs whose processes are gone, once,
// before the window draws anything.
//
// This is the single write the window's whole existence is responsible for,
// and it is here rather than behind a gesture because the alternative is a
// board that says RUNNING about a process that died when a laptop closed —
// for ever, since the only thing that could have said otherwise was the
// process itself. It is the same function `orbit reconcile` calls, over
// every repository the state root knows of rather than one.
//
// It runs before the plain frame as well as before the window. A frame that
// disagreed with the window about the same state would be two answers to one
// question, which is the defect this codebase spends the most effort not
// having.
//
// One damaged task does not stop the sweep, and nothing it finds stops the
// window opening: the errors are joined and handed back to be reported after
// the screen is given up, because a sentence printed before a full-screen
// program starts is a sentence nobody reads.
func reconcileAll(s *store.Store) error {
	refs, err := s.Repos()
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}
	for _, ref := range refs {
		// Path only: everything below it hashes the path, and the name a
		// row displays comes from the board's own read of the repository
		// rather than from here.
		r := repo.Repo{Path: ref.Path}
		ids, listErr := task.List(s, r)
		if listErr != nil {
			errs = append(errs, listErr)
			continue
		}
		for _, id := range ids {
			if _, wroteErr := task.Reconcile(s, task.Task{ID: id, Repo: r}); wroteErr != nil {
				errs = append(errs, wroteErr)
			}
		}
	}
	return errors.Join(errs...)
}

// enginesPort adapts the engine map and declared engines to ui.Options.Engines.
func enginesPort(engines map[string]engine.Engine) func() []ui.EngineInfo {
	return func() []ui.EngineInfo {
		var list []ui.EngineInfo
		names := []string{"claude", "codex", "opencode"}
		setupGuides := map[string][]string{
			"claude": {
				"1. Install Claude Code: npm install -g @anthropic-ai/claude-code",
				"2. Run 'claude' in a terminal to authenticate",
				"3. Ensure 'claude' is in your PATH",
			},
			"codex": {
				"1. Install Codex CLI: npm install -g @openai/codex",
				"2. Export OPENAI_API_KEY in your environment",
				"3. Ensure 'codex' is in your PATH",
			},
			"opencode": {
				"1. Install OpenCode CLI binary",
				"2. Configure local model endpoint or API keys",
				"3. Ensure 'opencode' is in your PATH",
			},
		}

		for _, name := range names {
			eng, hasEng := engines[name]
			_, pathErr := exec.LookPath(name)
			available := hasEng && pathErr == nil
			if available {
				var models []ui.ChoiceInfo
				for _, m := range eng.Models() {
					models = append(models, ui.ChoiceInfo{ID: m.ID, Label: m.Label})
				}
				var efforts []ui.ChoiceInfo
				for _, e := range eng.Efforts() {
					efforts = append(efforts, ui.ChoiceInfo{ID: e.ID, Label: e.Label})
				}
				list = append(list, ui.EngineInfo{
					Name:      name,
					Available: true,
					Models:    models,
					Efforts:   efforts,
					CanThink:  eng.CanThink(),
				})
			} else {
				list = append(list, ui.EngineInfo{
					Name:      name,
					Available: false,
					Setup:     setupGuides[name],
				})
			}
		}
		return list
	}
}
