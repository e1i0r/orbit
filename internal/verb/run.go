package verb

// What each verb actually does.
//
// One body per verb, and every way in reaches this one. What used to be
// four switch statements over four vocabularies is a switch over the
// vocabulary declared next door — so a verb behaves the same however it was
// asked for, including the parts nobody thinks about until they differ: who
// the record says did it, what happens to a run already in flight, and
// whether the board is told to look again.

import (
	"context"
	"fmt"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/supervisor"
	"github.com/e1i0r/orbit/internal/task"
)

// do is the verb, done.
func (v Verb) do(ctx context.Context, w World, in In) (Out, error) {
	switch v.Path() {
	case "new":
		return wrote(w, in)
	case "run":
		return started(w, in)
	case "pause", "resume", "continue", "skip":
		return controlled(w, in, v.Name)
	case "cancel":
		return cancelled(w, in)
	case "requeue":
		return requeued(ctx, w, in)
	case "note":
		return noted(w, in)
	case "direct":
		return directed(ctx, w, in)
	case "approve":
		return approved(w, in)
	case "read":
		return marked(w, in)
	case "history":
		return told(w, in)
	case "pr", "pr merge", "pr close":
		return given(ctx, w, in, v.Path())
	case "pr show":
		return shownPR(w, in)
	case "pr resolve":
		return erranded(w, in, "RESOLVE COMMENTS", supervisor.ResolveComments)
	case "pr update":
		return erranded(w, in, "UPDATE PR", supervisor.UpdatePR)
	case "pr checks":
		return erranded(w, in, "FIX CHECKS", supervisor.FixChecks)
	case "pr tests":
		return erranded(w, in, "MORE TESTS", supervisor.MoreTests)
	case "pr review":
		return erranded(w, in, "DEEP REVIEW", supervisor.Review)
	case "say":
		return spoken(ctx, w, in)
	case "learn":
		return learnt(w, in)
	case "join":
		return joined(w, in)
	case "permit":
		return permitted(w, in)
	case "critical":
		return marked_(w, in)
	case "thread":
		return heard(w)
	case "retract":
		return unsaid(w, in)
	case "settings":
		return kept(w)
	case "settings set":
		return changed(w, in)
	case "list":
		return listed(w, in)
	case "show":
		return shown(w, in)
	case "flows":
		return shapes(w)
	case "repos":
		return checkouts(w)
	case "export":
		return written(w, in)
	case "take":
		return handed(w, in)
	case "knowledge":
		return known(w)
	case "engines":
		return running()
	case "quota":
		return left()
	case "flow":
		return shaped(w, in)
	case "diff":
		return diffed(w, in)
	case "compare":
		return weighed(w, in)
	case "tree":
		return mapped(w, in)
	case "impact":
		return reaches(w, in)
	case "reconcile":
		return reconciled(w, in)
	case "delete":
		return deleted(w, in)
	case "rules":
		return waiting(w)
	case "rules keep":
		return agreed(w, in)
	case "rules drop":
		return dropped(w, in)
	}

	return Out{}, fmt.Errorf("%q is declared and not done", v.Path())
}

// wrote puts a task on the board, and starts it in the same breath when the
// reader asked for that: two gestures raced the board's next refresh.
func wrote(w World, in In) (Out, error) {
	one, t, err := writeDown(w, in)
	if err != nil {
		return Out{}, err
	}

	// The board is asked to look again here rather than by the caller: the
	// set of tasks changed, and a surface that forgot showed a reader an
	// empty board a moment after they filled in a form.
	if err := w.Looked(); err != nil {
		return Out{}, err
	}

	if !in.Yes("run") {
		return Out{Said: writtenDown(t, one), Of: []string{t.ID}}, nil
	}

	if _, err := started(w, In{Task: t.ID, Repo: in.Arg("repo"), Args: in.Args}); err != nil {
		// The task is written and that stands. A run that would not start
		// is a second thing to tell the reader, not a reason to pretend
		// the task is not there.
		return Out{Said: writtenDown(t, one), Of: []string{t.ID}}, err
	}

	return Out{Said: writtenDown(t, one) + " and started", Of: []string{t.ID}}, nil
}

// writeDown finds the repository and writes the task down: against one when
// there is one, and against none when the reader is standing nowhere.
func writeDown(w World, in In) (repo.Repo, task.Task, error) {
	one, err := openRepo(w, in)
	if err != nil {
		return repo.Repo{}, task.Task{}, err
	}

	t, err := task.Create(w.Store(), one, in.Arg("id"), in.Arg("text"), in.Arg("flow"))
	if err != nil {
		return repo.Repo{}, task.Task{}, err
	}

	return one, t, nil
}

// openRepo is the checkout a task is written against, and none when the
// reader named none and is standing in none either. A -repo the reader
// typed has to open; the default is allowed to come back empty.
func openRepo(w World, in In) (repo.Repo, error) {
	at := in.Arg("repo")
	if at == "" {
		at = in.Repo
	}

	if at == "" {
		return repo.Repo{}, nil
	}

	one, err := repo.Open(at)
	if err != nil {
		return repo.Repo{}, err
	}

	return one, nil
}

// writtenDown is the task on the board: what it is, where it will be
// worked, and what it will walk.
//
// Two sentences rather than one with an empty name in it. "ACME-1 written
// against , to walk the review flow" is a line that reads as a bug, and a
// task that starts nowhere is not a bug — it is the thing the reader just
// asked for, and the line says which one they got.
func writtenDown(t task.Task, r repo.Repo) string {
	walking := ""
	if t.Flow != "" {
		walking = " to walk " + t.Flow
	}

	if r.Name == "" {
		return t.ID + " written down against no repository yet" + walking
	}

	return t.ID + " written down against " + r.Name + walking
}

// started runs a task in a process of its own.
func started(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	unread, err := w.Unread(in.Repo)
	if err != nil {
		return Out{}, err
	}

	walking := in.Arg("flow")
	if walking == "" {
		walking = t.Flow
	}

	if _, err := task.StartWith(w.Store(), t, walking, in.Arg("engine"), unread); err != nil {
		return Out{}, err
	}

	return Out{Said: t.ID + " started"}, nil
}

// controlled leaves one of the words a run understands where it will find
// it, at its next phase boundary.
func controlled(w World, in In, word string) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	if err := task.Control(w.Store(), t, word); err != nil {
		return Out{}, err
	}

	return Out{Said: said[word](t.ID)}, nil
}

// said is what each control word tells the reader it did. The sentences are
// here rather than in the surfaces because the act is the same act, and a
// browser saying "paused" while a terminal says "asked to pause" is two
// accounts of one thing.
var said = map[string]func(id string) string{
	"pause":    func(id string) string { return id + " asked to pause at its next phase" },
	"resume":   func(id string) string { return id + " asked to carry on" },
	"continue": func(id string) string { return id + " let past the gate it was waiting at" },
	"skip":     func(id string) string { return id + " skipped past the phase it was in" },
}

// cancelled asks the run to stop where it stands, and to write down that it
// was stopped — the signal rather than a word, because a cancel should not
// wait for a phase to end.
func cancelled(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	if err := task.Cancel(w.Store(), t); err != nil {
		return Out{}, err
	}

	return Out{Said: "asked the run of " + t.ID + " to stop"}, nil
}

// requeued takes a task back, stopping whatever holds it and waiting for it
// to be gone first.
func requeued(ctx context.Context, w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	if err := task.Requeue(ctx, w.Store(), t, in.who(), in.Arg("why")); err != nil {
		return Out{}, err
	}

	return Out{Said: t.ID + " is back in the queue"}, nil
}
