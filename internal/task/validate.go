package task

// The last word on a run: does what came out answer what was asked, and if
// not, whose problem is it.

import (
	"context"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// validateMarker says a prompt is the last word on a run rather than work.
const validateMarker = "## Validation"

// validateAsk is what the validating model is told, and it offers three
// answers because there are three: it is done, it needs another run, or it
// needs a person.
//
// "Another run" and "a person" are the two the phase is really about. A run
// that missed half the task can be told what it missed and sent round again
// for the price of a run; a run that is wrong about what the task meant
// cannot, because the thing that misread it would be doing the rereading.
// Which of those it is, is the judgement being asked for.
const validateAsk = validateMarker + "\n\n" +
	"Below is what a task asked for and what the run that answered it did. " +
	"Say which of these three is true, on the first line and nothing else on it:\n\n" +
	"```\n" +
	"verdict: done\n" +
	"verdict: again <one line saying what is missing>\n" +
	"verdict: human <one line saying what a person has to decide>\n" +
	"```\n\n" +
	"`again` is for work that is incomplete in a way another run could " +
	"finish, and the line is what that run will be told. `human` is for " +
	"anything a run cannot settle: the task is ambiguous, the approach is " +
	"wrong, or what it would take is a decision somebody has to make. " +
	"Prefer `done` when the work answers the task, even if you would have " +
	"written it differently.\n"

// verdict is the three answers, as this package acts on them.
type verdict int

const (
	verdictDone verdict = iota
	verdictAgain
	verdictHuman
)

// validate asks the flow's own engine whether the run answered the task, and
// acts on what it says.
//
// The flow's engine and not a second one somebody configures: it is already
// running, it is already paid for, and a validator on a different model is a
// second opinion nobody asked for about work the first one understands.
//
// A model that cannot be reached, or an answer this cannot read, is done.
// The run finished, its gates passed, and refusing to call that finished
// because a judge was unavailable would report an outage as the work's
// fault — the same rule the consistency gate keeps, for the same reason.
func validate(ctx context.Context, s *store.Store, t Task, f flow.Flow, eng engine.Engine, wt string) error {
	if !f.Validate_ || eng == nil {
		return nil
	}

	out, err := eng.Run(ctx, engine.Request{
		Prompt:      validatePrompt(t, s),
		Dir:         wt,
		Permissions: []string{flow.PermissionRead},
		Env:         childEnv(t),
	})
	if err != nil {
		logger.Warn("task/run", "%s: the run could not be validated: %v", t.ID, err)
		return nil
	}

	kind, why := verdictOf(out.Output)
	switch kind {
	case verdictDone:
		return nil
	case verdictAgain:
		return sendBack(s, t, why)
	case verdictHuman:
		return handOver(s, t, why)
	}

	return nil
}

// validatePrompt is the task as it was written, and the story of what became
// of it.
//
// The story and not the diff: the diff is in the worktree the validator is
// running in and it can read as much of it as it needs, where a diff pasted
// into a prompt is a copy that can be stale and is usually too long to be
// read at all.
func validatePrompt(t Task, s *store.Store) string {
	var b strings.Builder

	b.WriteString(validateAsk)
	fmt.Fprintf(&b, "\n## What was asked\n\n%s\n", strings.TrimSpace(t.Text))

	if story := StoryOf(s, t); story != nil {
		fmt.Fprintf(&b, "\n## What the run says it did\n\n- the way in: %s\n- what it is for: %s\n"+
			"- what went wrong: %s\n- why: %s\n- what was done: %s\n",
			story.Entry, story.Purpose, story.Symptom, story.Cause, story.Fix)
	}

	b.WriteString("\nThe worktree you are running in holds the change itself.\n")

	return b.String()
}

// verdictOf reads the judge's first line.
//
// Anything it cannot read is done, for the reason validate's own doc gives:
// a run that finished with its gates green is finished, and a gate that
// fired on its own misreadings is one nobody leaves on.
func verdictOf(answer string) (verdict, string) {
	lines := strings.Split(strings.TrimSpace(answer), "\n")
	if len(lines) == 0 {
		return verdictDone, ""
	}

	first := strings.TrimSpace(lines[0])

	rest, found := strings.CutPrefix(strings.ToLower(first), "verdict:")
	if !found {
		return verdictDone, ""
	}

	rest = strings.TrimSpace(rest)
	switch {
	case strings.HasPrefix(rest, "again"):
		return verdictAgain, said(first, "again")
	case strings.HasPrefix(rest, "human"):
		return verdictHuman, said(first, "human")
	}

	return verdictDone, ""
}

// said is what the verdict line carries after the word, taken from the line
// as it was written rather than from the lowered copy.
func said(line, word string) string {
	lower := strings.ToLower(line)

	at := strings.Index(lower, word)
	if at < 0 {
		return ""
	}

	return strings.TrimSpace(line[at+len(word):])
}

// sendBack puts the task in the queue again with what it is missing.
//
// The reason is written as a note as well as into the requeue, because the
// note is what the next run reads: a task sent round again without being
// told why is the same run a second time, which is the mistake the attempt
// cap already exists to stop.
func sendBack(s *store.Store, t Task, why string) error {
	if why == "" {
		why = "the supervisor asked for another run and said no more than that"
	}

	if err := Note(s, t, why); err != nil {
		return err
	}

	return emit(s, t, record.Event{
		Kind: record.TaskRequeued,
		Text: why,
		Data: map[string]string{"by": "supervisor"},
	})
}

// handOver stops the task and says what a person has to decide.
//
// task.stuck and not task.failed: nothing broke. The run did what it was
// asked and what is left is a judgement, which is exactly the band that word
// exists for.
func handOver(s *store.Store, t Task, why string) error {
	if why == "" {
		why = "the supervisor could not settle this one and did not say why"
	}

	return emit(s, t, record.Event{
		Kind: record.TaskStuck,
		Text: why,
		Data: map[string]string{"by": "supervisor", "attempts": "1"},
	})
}

// lastEngine is the engine of the flow's last phase, which is the one that
// validates.
//
// A loop at the end names none of its own, so the engine of the last phase
// inside it is the answer: what matters is that the model doing the reading
// is one this run has already been using.
func lastEngine(f flow.Flow) string {
	if len(f.Phases) == 0 {
		return ""
	}

	last := f.Phases[len(f.Phases)-1]
	if last.Loop == nil || len(last.Loop.Phases) == 0 {
		return last.Engine
	}

	return last.Loop.Phases[len(last.Loop.Phases)-1].Engine
}

// requeued says the validator has already ended this run in a word of its
// own, so nothing more is written about it.
//
// It is read back from the record rather than returned from validate,
// because what ends a run is the event and not the function that wrote it —
// a reader folding this log later sees the same thing this does.
func requeued(s *store.Store, t Task) bool {
	events, err := Events(s, t)
	if err != nil {
		return false
	}

	for i := len(events) - 1; i >= 0; i-- {
		switch events[i].Kind {
		case record.TaskRequeued, record.TaskStuck:
			return events[i].Data["by"] == "supervisor"
		case record.TaskStarted:
			return false
		}
	}

	return false
}
