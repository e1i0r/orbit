package task

// The task story: how this prompt became this diff, in five fields.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// storyFields are the five, in the order they are asked for and drawn: the
// route in, what it is for, what went wrong, why, and what was done.
//
// Five and not four. "The primary key collided" is the cause and "upsert
// instead of insert" is the fix, and a report that gives one of them without
// the other is the sentence this whole thing exists to replace — true at the
// depth of the code, with no way back up to the product.
var storyFields = []string{"entry", "purpose", "symptom", "cause", "fix"}

// storyAsk is what the last phase of a flow is told to write besides its
// answer.
//
// Asked of the last phase and folded into the answer it was already giving,
// rather than paid for as a phase of its own. The spec left that open; this
// is the cheap end of it, and the expensive end can be bought later by a
// flow that adds a phase for it.
//
// Fields and not prose, because prose is what the report tab already is. The
// five are separate so that each can have its evidence hung under it, which
// is the whole reason the story is a structure Orbit assembled rather than a
// paragraph a model wrote.
const storyAsk = "\n## Story\n\n" +
	"End the answer with a `## Story` section: five lines, in this order, " +
	"each `field: one short sentence`.\n\n" +
	"```\n" +
	"entry: the outermost thing a user or a caller reaches — a route, a command, a job\n" +
	"purpose: what that entry point is for, in the words of the product and not of the code\n" +
	"symptom: what was going wrong, as somebody using it would have noticed\n" +
	"cause: why it was going wrong\n" +
	"fix: what was changed about it\n" +
	"```\n\n" +
	"Write all five or none. Four of five is a shape that looks complete " +
	"with a field missing, and a reader cannot tell which one is gone.\n"

// Story is how a prompt became a diff, as everything outside this package
// reads it.
//
// It is a type and not the map the record carries because a caller writing
// a diagram or a pane needs the five by name, and a map hands them the
// chance to ask for a sixth that will always be empty.
type Story struct {
	Entry   string
	Purpose string
	Symptom string
	Cause   string
	Fix     string
}

// StoryOf is the story of the attempt that stands, and nothing when the task
// has none.
//
// The last one written, for the reason the pane draws the last one: a task
// run three times told its story three times, and the two before it are
// about work that was thrown away.
func StoryOf(s *store.Store, t Task) *Story {
	events, err := Events(s, t)
	if err != nil {
		return nil
	}

	var found *Story

	for _, e := range events {
		if e.Kind != record.TaskStory {
			continue
		}

		story := Story{
			Entry:   e.Data["entry"],
			Purpose: e.Data["purpose"],
			Symptom: e.Data["symptom"],
			Cause:   e.Data["cause"],
			Fix:     e.Data["fix"],
		}

		if story.whole() {
			found = &story
		}
	}

	return found
}

// whole is the same rule the parser and the window both keep: five fields or
// none. A story with a link missing draws as a chain that still looks whole.
func (s Story) whole() bool {
	return s.Entry != "" && s.Purpose != "" && s.Symptom != "" && s.Cause != "" && s.Fix != ""
}

// storyIn reads the five fields out of an answer, and says whether it found
// a whole story.
//
// All five or nothing. A story with a field missing draws as a chain with a
// link out of it — it still looks authoritative, and the reader has no way
// to see which claim was never made.
func storyIn(answer string) (map[string]string, bool) {
	found := map[string]string{}
	inSection := false

	for _, line := range strings.Split(answer, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			inSection = strings.EqualFold(strings.TrimLeft(trimmed, "# "), "story")
			continue
		}

		if !inSection {
			continue
		}

		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}

		key = strings.ToLower(strings.TrimSpace(strings.Trim(key, "-* `")))
		if value = strings.TrimSpace(value); value != "" && known(key) {
			found[key] = value
		}
	}

	if len(found) != len(storyFields) {
		return nil, false
	}

	return found, true
}

// known reports whether a key is one of the five.
func known(key string) bool {
	for _, f := range storyFields {
		if f == key {
			return true
		}
	}

	return false
}

// noteStory writes the story down, once, off the last phase of the flow.
//
// The last phase because the story is about the task and a phase in the
// middle of one does not know how it ends. Best-effort and logged, for the
// reason the decisions are: what it writes is derived from an answer the
// record already holds whole.
func noteStory(s *store.Store, t Task, f flow.Flow, n int, out engine.Result) {
	if n != len(f.Phases) {
		return
	}

	fields, ok := storyIn(out.Output)
	if !ok {
		return
	}

	if err := emit(s, t, record.Event{Kind: record.TaskStory, Data: fields}); err != nil {
		logger.Warn("task/run", "%s: the story was not written down: %v", t.ID, err)
	}
}
