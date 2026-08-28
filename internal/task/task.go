// Package task turns a written sentence into a run: a directory, a branch, a
// worktree, and a log of what happened.
package task

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// ErrExists says a task with that id is already written down.
//
// It is a sentinel because the command line has to tell two failures apart
// that read alike in a string: an id already used, which a human fixes by
// choosing another one, and a state root that would not take the file, which
// they cannot. Matching on the words of an error message to tell them apart
// is a test of the message, not of what happened.
var ErrExists = errors.New("already exists")

// Task is one piece of work against one repository.
//
// Flow is which pipeline it walks. It is a property of the task and not of
// whoever typed the command that ran it: two runs of one task that walked
// different pipelines would leave a record that cannot say which was meant,
// and the sentence a person wrote is not the only thing they decided.
type Task struct {
	ID   string
	Repo repo.Repo
	Text string
	Flow string
}

// Create writes the task down and records that it exists.
//
// The written task is the whole interface: everything the engines are told
// comes from this file, and when it is ambiguous the right answer is to stop
// and ask rather than to guess.
//
// flowName is recorded and not resolved. A task written against a flow that
// is later deleted is still a task, and refusing to write it down because a
// JSON file is missing would lose the sentence the user typed — the one
// thing in this system nobody can regenerate. Run is where a name is turned
// into phases, and where a name nothing answers to is refused with the list
// of names that would have worked.
func Create(s *store.Store, r repo.Repo, id, text, flowName string) (Task, error) {
	if id == "" {
		return Task{}, fmt.Errorf("a task needs an id")
	}

	if text == "" {
		return Task{}, fmt.Errorf("task %q has nothing written in it", id)
	}

	flowName, err := chosenFlow(s, flowName)
	if err != nil {
		return Task{}, err
	}

	path, err := s.TaskFilePath(r.Path, id)
	if err != nil {
		return Task{}, err
	}

	if _, statErr := os.Stat(path); statErr == nil {
		return Task{}, fmt.Errorf("task %q in %s %w", id, r.Name, ErrExists)
	}

	dir, err := s.CreateTaskDir(r.Path, id)
	if err != nil {
		return Task{}, err
	}
	// 0600 like everything else under the state root: this file is the
	// verbatim text of a task, and the root is nobody else's business.
	if err := os.WriteFile(path, []byte(text+"\n"), 0o600); err != nil {
		return Task{}, fmt.Errorf("write %q: %w", path, err)
	}

	t := Task{ID: id, Repo: r, Text: text, Flow: flowName}
	if err := emit(s, t, record.Event{Kind: record.TaskCreated, Text: text, Data: map[string]string{"flow": flowName}}); err != nil {
		// The file landed and nothing was recorded. Left alone, that is the
		// worst of both: the task exists to List, `orbit show` has nothing
		// to show, and writing it again is refused as a duplicate — a task
		// that can neither run nor be created. Take the file back out so the
		// retry the user is about to attempt actually works.
		if rmErr := os.Remove(path); rmErr != nil {
			return Task{}, fmt.Errorf("%w; and %q could not be taken back out again: %w", err, path, rmErr)
		}
		// The directory itself is what List keys on, so it has to go too or
		// a task that was never recorded would still show up in it. The
		// removal is deliberately non-recursive, not os.RemoveAll: it
		// succeeds exactly when task.md (just removed above) was the only
		// thing inside, which is the state a failed Create should leave
		// behind, and it fails harmlessly if something else landed in
		// there — a partially written events.jsonl, say — rather than
		// destroying that evidence.
		//
		// Unlike the file's removal above, this error is discarded rather
		// than folded into the returned error. The file's error is folded
		// in because a task.md still on disk blocks the retry the emit
		// error tells the user to attempt — it's a second fact they need
		// to hear. A directory that could not be removed blocks nothing:
		// the duplicate check in Create keys on task.md, already gone, so
		// the retry still succeeds. It would still leave List reporting a
		// task that was never recorded — the very bug this rollback
		// exists to close — but only in the case that removal fails,
		// which happens only when something else is inside the directory,
		// and forcing that onto the caller as a second error would not
		// give them anything to act on beyond what the emit error already
		// says.
		_ = os.Remove(dir)

		return Task{}, fmt.Errorf("%w; task %q was not created", err, id)
	}

	return t, nil
}

// Load reads a task's text back off disk.
//
// It exists for callers that only carry an id — the command line, most
// notably — and cannot pass along the Task value Create returned, because
// that happened in a different process.
func Load(s *store.Store, r repo.Repo, id string) (Task, error) {
	path, err := s.TaskFilePath(r.Path, id)
	if err != nil {
		return Task{}, err
	}

	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Task{}, fmt.Errorf("task %q does not exist in %s", id, r.Name)
		}

		return Task{}, fmt.Errorf("read %q: %w", path, err)
	}
	// Create writes text+"\n"; strip exactly that trailing newline and
	// nothing else, so a round trip returns the text unchanged.
	t := Task{ID: id, Repo: r, Text: strings.TrimSuffix(string(body), "\n")}
	t.Flow = writtenFlow(s, t)

	return t, nil
}

// chosenFlow decides which flow a task is written against when the command
// that wrote it named none: the user's default, and failing that the flow
// this program ships.
//
// It is decided here, once, and then written down. A task that looked its
// flow up at run time would walk a different pipeline every time the
// settings moved, which is the property this whole design exists to remove.
func chosenFlow(s *store.Store, name string) (string, error) {
	if name != "" {
		return name, nil
	}

	cfg, err := s.Settings()
	if err != nil {
		return "", err
	}

	if cfg.Flow != "" {
		return cfg.Flow, nil
	}
	// A settings file saved before there was a flow to set — which is every
	// settings file that already exists — says nothing here, and nothing is
	// not a choice.
	return flow.Default, nil
}

// writtenFlow is the flow a task was written against, read back out of the
// one place it is recorded.
//
// task.created and not task.started: a run that overrode the flow once did
// not change what the task was written against, and a second run with no
// -flow should walk the task's own pipeline rather than the last one
// somebody tried.
//
// It is best-effort on purpose. A record that cannot be read is not a task
// that cannot be loaded — the written task is the thing worth having — so
// an unreadable log answers the empty string, and the callers that need a
// flow fall back exactly as Create did.
func writtenFlow(s *store.Store, t Task) string {
	events, err := Events(s, t)
	if err != nil {
		return ""
	}

	name := ""

	for _, e := range events {
		if e.Kind != record.TaskCreated {
			continue
		}

		if recorded, ok := e.Data["flow"]; ok {
			name = recorded
		}
	}

	return name
}

// List returns the ids of every task recorded against a repository, sorted.
func List(s *store.Store, r repo.Repo) ([]string, error) {
	tasksDir, err := s.TasksDir(r.Path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(tasksDir)
	if errors.Is(err, os.ErrNotExist) {
		// A repository nobody has written a task against yet. Reading a
		// path no longer creates it, so this directory is genuinely absent
		// until the first task is created — and "no tasks" is an answer,
		// not a fault.
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("list the tasks of %q: %w", r.Name, err)
	}

	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}

	sort.Strings(ids)

	return ids, nil
}

// Events returns everything recorded about a task, oldest first.
func Events(s *store.Store, t Task) ([]record.Event, error) {
	path, err := s.EventsPath(t.Repo.Path, t.ID)
	if err != nil {
		return nil, err
	}

	return record.Read(path)
}

// emit appends one event to the task's log.
func emit(s *store.Store, t Task, e record.Event) error {
	path, err := s.EventsPath(t.Repo.Path, t.ID)
	if err != nil {
		return err
	}

	if err := record.Append(path, e); err != nil {
		return fmt.Errorf("record %q for task %s: %w", e.Kind, t.ID, err)
	}

	return nil
}
