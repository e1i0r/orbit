package task

// The delta spec: what the change asks of its callers, what it promises
// them, what it assumed, and what it decided not to do.
//
// Asked of the last phase and folded into the answer it was already giving,
// the way the story is — the cheap end of it, and a phase of its own can be
// bought later by a flow that wants one.
//
// It is the engine's claim about its own work. Nothing verifies it and
// nothing can: no command decides whether "assumes UTC timestamps" is true.
// It is kept anyway because the discarded alternatives exist nowhere else —
// they die with the run, and the next person to touch that code pays again
// to find out why the obvious approach was not taken.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// deltaFields are the four, in the order they are asked for and drawn.
var deltaFields = []string{"needs", "guarantees", "assumes", "instead"}

// deltaAsk is what the last phase is told to write besides its answer.
//
// Unlike the story, which is all five or nothing, these four stand alone: a
// change that adds no precondition has nothing to say under needs, and a
// line invented to fill the shape is worse than a shape with a gap. Repeats
// are allowed because one change can ask two things of its callers.
const deltaAsk = "\n## Delta\n\n" +
	"End the answer with a `## Delta` section: what this change asks of the code around it, " +
	"one `field: one short sentence` per line. Repeat a field for a second sentence. " +
	"Leave a field out when there is nothing true to say under it — do not fill the shape.\n\n" +
	"```\n" +
	"needs: something callers must now do that they did not before\n" +
	"guarantees: something this code now holds that it did not before\n" +
	"assumes: something about the world around it that this took for granted\n" +
	"instead: an approach you considered and did not take, and why\n" +
	"```\n\n" +
	"instead is the one nobody else can recover: what you rejected dies with this run unless you write it here.\n"

// deltaIn reads the fields out of an answer, keeping the order the lines
// were written in and every repeat of a field.
func deltaIn(answer string) map[string]string {
	found := map[string][]string{}
	inSection := false

	for _, line := range strings.Split(answer, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			inSection = strings.EqualFold(strings.TrimLeft(trimmed, "# "), "delta")
			continue
		}

		if !inSection {
			continue
		}

		key, value, held := strings.Cut(trimmed, ":")
		if !held {
			continue
		}

		key = strings.ToLower(strings.TrimSpace(strings.Trim(key, "-* `")))
		if value = strings.TrimSpace(value); value != "" && isDeltaField(key) {
			found[key] = append(found[key], value)
		}
	}

	if len(found) == 0 {
		return nil
	}

	// One event, one line per sentence: the record's Data is strings, and
	// the reader that draws these splits them back apart.
	out := make(map[string]string, len(found))
	for key, lines := range found {
		out[key] = strings.Join(lines, "\n")
	}

	return out
}

// isDeltaField reports whether a key is one of the four.
func isDeltaField(key string) bool {
	for _, f := range deltaFields {
		if f == key {
			return true
		}
	}

	return false
}

// noteDelta writes down whatever a phase said about its own change.
//
// Every phase, and not the last alone: a flow that ends at a human gate
// leaves its last phase unfinished until somebody resumes it, and until then
// the pane had nothing to show about work that was already done. The pane
// reads the newest, so a later phase's account replaces an earlier one — and
// a phase that changed nothing writes nothing, which is what leaves the one
// that did standing.
func noteDelta(s *store.Store, t Task, _ flow.Flow, _ int, out engine.Result) {
	fields := deltaIn(out.Output)
	if fields == nil {
		return
	}

	if err := emit(s, t, record.Event{Kind: record.TaskDelta, Data: fields}); err != nil {
		logger.Warn("task/run", "%s: the delta was not written down: %v", t.ID, err)
	}
}
