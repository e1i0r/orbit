package verb

// The verbs that put something on a task's record rather than steering the
// run on it: a word for the next phase, a correction, an answer to what a
// gate asked, and the readings of what is already written there.

import (
	"context"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/task"
)

// noted leaves a word for the phase that starts next.
func noted(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	if err := task.Note(w.Store(), t, in.Arg("text")); err != nil {
		return Out{}, err
	}

	return Out{Said: "noted on " + t.ID}, nil
}

// directed corrects a task and stops the run so the next one reads it.
func directed(ctx context.Context, w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	if !in.Yes("restart") {
		if err := task.Direct(w.Store(), t, in.who(), in.Arg("text")); err != nil {
			return Out{}, err
		}

		return Out{Said: t.ID + " redirected — the next run reads it"}, nil
	}

	unread, err := w.Unread(in.Repo)
	if err != nil {
		return Out{}, err
	}

	pid, err := task.Reopen(ctx, w.Store(), t, in.who(), in.Arg("text"), t.Flow, unread)
	if err != nil {
		return Out{}, err
	}

	return Out{Said: t.ID + " redirected and started again", Pid: pid}, nil
}

// approved says yes to what the dependency gate stopped the run for. What is
// approved is what is pending: a reader answers the question the record
// asked them, and a list from outside would be a yes to a question nobody
// put.
func approved(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	f, err := flowOf(w, t)
	if err != nil {
		return Out{}, err
	}

	names := task.Pending(w.Store(), t, f)
	if len(names) == 0 {
		// Nothing waiting is an answer, not a refusal: the reader asked
		// and was told. Refusing would read as though asking were wrong,
		// and "approved" would read as a decision nobody made.
		return Out{Said: t.ID + " has added no dependency waiting on you"}, nil
	}

	if err := task.Approve(w.Store(), t, names); err != nil {
		return Out{}, err
	}

	return Out{Said: "approved for " + t.ID + ": " + strings.Join(names, ", "), Of: names}, nil
}

// marked files a finished task as looked at.
func marked(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	if err := task.MarkRead(w.Store(), t); err != nil {
		return Out{}, err
	}

	return Out{Said: t.ID + " marked as read"}, nil
}

// told is everything ever said about a task, in any program.
func told(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	body, err := task.History(w.Store(), t)
	if err != nil {
		return Out{}, err
	}

	return Out{Said: body}, nil
}

// given hands a task's work to the world.
func given(ctx context.Context, w World, in In, verb string) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	body, err := w.Deliver(ctx, t, verb)
	if err != nil {
		return Out{}, err
	}

	return Out{Said: strings.TrimSpace(body)}, nil
}

// spoken says something in the supervisor's thread.
//
// It records and does not answer. The supervisor reads the thread on its own
// schedule, and a verb that called a model here would spend money in a place
// nobody was told to expect it.
func spoken(_ context.Context, w World, in In) (Out, error) {
	if err := w.Say(in.Arg("text"), in.who(), in.Task); err != nil {
		return Out{}, err
	}

	return Out{Said: w.Words().T("verb.say.done", "said in the supervisor thread")}, nil
}

// learnt writes down something true about the code, so the next run against
// it is told before it starts.
func learnt(w World, in In) (Out, error) {
	// The repository the reader named, then the one they are in. A surface
	// with no working directory — a browser tab, a tool call — has only the
	// first, and a fact about no repository at all is refused by knowledge
	// itself rather than filed somewhere nobody will read it.
	where := in.Arg("repo")
	if where == "" {
		where = in.Repo
	}

	fact := knowledge.Fact{
		Phrase: strings.TrimSpace(in.Arg("text")),
		Source: knowledge.Human,
		Scope:  knowledge.Scope{Kind: knowledge.Repo, Repo: where},
	}

	if err := w.Learn(fact); err != nil {
		return Out{}, err
	}

	return Out{Said: "written down: " + fact.Phrase}, nil
}

// joined opens a checkout of another repository for a task, so one task can
// reach into as many as the work needs.
func joined(w World, in In) (Out, error) {
	t, against, err := w.Find(in.Task, in.Repo)
	if err != nil {
		return Out{}, err
	}

	one, err := task.Joinable(w.Store(), against, in.Arg("name"))
	if err != nil {
		return Out{}, err
	}

	where, err := task.Join(w.Store(), t, one)
	if err != nil {
		return Out{}, err
	}

	return Out{Said: t.ID + " joined " + one.Name + " at " + where}, nil
}

// permitted answers the question a critical action stopped for.
func permitted(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	act, waiting := task.Waiting(w.Store(), t)
	if !waiting {
		return Out{}, fmt.Errorf("%s is not waiting to be permitted anything", t.ID)
	}

	yes := in.Yes("yes")
	if err := task.Answer(w.Store(), t, act, yes, in.who()); err != nil {
		return Out{}, err
	}

	if yes {
		return Out{Said: t.ID + " may go ahead"}, nil
	}

	return Out{Said: t.ID + " was refused"}, nil
}

// marked_ marks a task as one whose changes need answering for, or stops.
func marked_(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	on := in.Yes("on")
	if err := task.Mark(w.Store(), t, on, in.who()); err != nil {
		return Out{}, err
	}

	if on {
		return Out{Said: t.ID + " is critical"}, nil
	}

	return Out{Said: t.ID + " is no longer critical"}, nil
}

// reconciled closes the record of a run whose process is gone, so a task
// does not read as running for ever because something was killed.
func reconciled(w World, in In) (Out, error) {
	// A task by name, or the one the caller was already about. Reconcile is
	// a repository's question — `orbit reconcile` sweeps every task under a
	// root — and naming one is the narrowing, not the whole of it.
	if id := in.Arg("task"); id != "" {
		in.Task = id
	}

	if in.Task == "" {
		return swept(w, in)
	}

	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	changed, err := task.Reconcile(w.Store(), t)
	if err != nil {
		return Out{}, err
	}

	if !changed {
		return Out{Said: t.ID + " needed nothing"}, nil
	}

	return Out{Said: t.ID + " was left open by a run that is gone, and is closed now"}, nil
}

// swept closes every record in the repository whose run is gone.
//
// One damaged task does not stop the sweep: the reader asked what is still
// open, and answering for the eleven that could be read is worth more than
// refusing because the twelfth could not.
func swept(w World, in In) (Out, error) {
	_, r, err := w.Find("", in.Repo)
	if err != nil {
		return Out{}, err
	}

	ids, err := task.List(w.Store(), r)
	if err != nil {
		return Out{}, err
	}

	var closed []string

	for _, id := range ids {
		t, err := task.Load(w.Store(), r, id)
		if err != nil {
			continue
		}

		if changed, err := task.Reconcile(w.Store(), t); err == nil && changed {
			closed = append(closed, id)
		}
	}

	if len(closed) == 0 {
		return Out{Said: "every run here is accounted for"}, nil
	}

	return Out{
		Said: strings.Join(closed, ", ") + " were left open by runs that are gone, and are closed now",
		Of:   closed,
	}, nil
}

// deleted removes a task and everything written about it.
func deleted(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	if err := task.Delete(w.Store(), t); err != nil {
		return Out{}, err
	}

	if err := w.Looked(); err != nil {
		return Out{}, err
	}

	return Out{Said: t.ID + " is gone"}, nil
}

// flowOf is the flow a task walks, by the same reading `orbit run` makes of
// it: the task's own, then the one Orbit ships. Not the settings default,
// which is what the next task written gets.
func flowOf(w World, t task.Task) (flow.Flow, error) {
	chosen := t.Flow
	if chosen == "" {
		chosen = flow.Default
	}

	return flow.Resolve(w.Store(), chosen)
}

// found is the task a verb is about.
func found(w World, in In) (task.Task, error) {
	t, _, err := w.Find(in.Task, in.Repo)

	return t, err
}

// who the record says asked. A person at any of the controls is the
// operator; a model says its own name.
func (i In) who() string {
	if i.By == "" {
		return "operator"
	}

	return i.By
}
