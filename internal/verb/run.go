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

	"github.com/e1i0r/orbit/internal/task"
)

// do is the verb, done.
func (v Verb) do(ctx context.Context, w World, in In) (Out, error) {
	switch v.Name {
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
	case "pr", "merge", "close-pr":
		return given(ctx, w, in, v.Name)
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
	case "set":
		return changed(w, in)
	case "list":
		return listed(w)
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
	}

	return Out{}, fmt.Errorf("%q is declared and not done", v.Name)
}

// wrote puts a task on the board, and starts it in the same breath when the
// reader asked for that: two gestures raced the board's next refresh.
func wrote(w World, in In) (Out, error) {
	_, r, err := w.Find("", in.Arg("repo"))
	if err != nil {
		return Out{}, err
	}

	t, err := task.Create(w.Store(), r, in.Arg("id"), in.Arg("text"), in.Arg("flow"))
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
		return Out{Said: t.ID + " written down", Of: []string{t.ID}}, nil
	}

	if _, err := started(w, In{Task: t.ID, Repo: in.Arg("repo"), Args: in.Args}); err != nil {
		// The task is written and that stands. A run that would not start
		// is a second thing to tell the reader, not a reason to pretend
		// the task is not there.
		return Out{Said: t.ID + " written down", Of: []string{t.ID}}, err
	}

	return Out{Said: t.ID + " written down and started", Of: []string{t.ID}}, nil
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
	"skip":     func(id string) string { return id + " let past the phase it was in" },
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
