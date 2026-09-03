package task

// The consistency gate: a change that goes against something the task
// already decided stops and says which decision it went against.

import (
	"context"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// checkMarker is the line that says a prompt is this check and not work.
//
// It is in the prompt rather than kept as a flag beside it because the only
// thing that can tell a caller which of the two a model was asked is the
// text it was sent — a run reading its own record back sees prompts, not
// intentions.
const checkMarker = "## Consistency check"

// checkAsk is the whole of what the judging model is told, and its shape is
// what makes the answer readable.
//
// It asks for a verdict on the first line and the reason under it, because a
// model asked to explain first will explain for a paragraph and bury the
// word that decides whether a run stops.
const checkAsk = checkMarker + "\n\n" +
	"Below are decisions this task recorded earlier, and the change it has " +
	"just made to the files those decisions govern. Say whether the change " +
	"goes against any of them.\n\n" +
	"Answer with one of these as the first line, and nothing else on it:\n\n" +
	"```\n" +
	"verdict: consistent\n" +
	"verdict: contradicts <the id of the decision it goes against>\n" +
	"```\n\n" +
	"Then one or two sentences saying why. Judge only what the decision " +
	"actually says: a change the decision does not speak to is consistent " +
	"with it, and so is a change that follows it in a way the decision did " +
	"not anticipate.\n"

// contradiction is a change that went against a decision, and why.
type contradiction struct {
	Decision string
	Why      string
}

// governed is the decisions this task recorded whose scope reaches the files
// it has changed.
//
// From the record rather than from the files under .orbit/: the event is the
// decision and the file is a copy of it, so a copy somebody edited by hand
// is not what a gate is entitled to act on.
func governed(s *store.Store, t Task, changed []repo.Change) []record.Event {
	events, err := Events(s, t)
	if err != nil {
		return nil
	}

	var out []record.Event

	for _, e := range events {
		if e.Kind != record.DecisionMade {
			continue
		}

		if reaches(e.Data["scope"], changed) {
			out = append(out, e)
		}
	}

	return out
}

// reaches says whether any changed file is under any path of a scope.
func reaches(scope string, changed []repo.Change) bool {
	for _, path := range strings.Split(scope, ",") {
		for _, c := range changed {
			if governs(strings.TrimSpace(path), c.Path) {
				return true
			}
		}
	}

	return false
}

// governs is one scope entry against one changed file: the file itself, or
// anything under it when the entry names a directory.
//
// An empty scope governs nothing. A decision that named no paths was not
// making a claim about any particular code, and reading it as a claim about
// all of it would stop every run in the repository.
func governs(scope, path string) bool {
	if scope == "" {
		return false
	}

	scope = strings.TrimSuffix(scope, "/")

	return path == scope || strings.HasPrefix(path, scope+"/")
}

// contradicts asks the phase's own engine whether what was just changed goes
// against what was decided, and answers with the first contradiction it is
// told about.
//
// It is asked at all only when a decision reaches the changed files, which
// is what keeps it from costing anything in a repository that has decided
// nothing about the code being worked on. One call and not one per decision:
// the decisions that govern a change are read together, the way a person
// reviewing it would read them.
//
// A model that cannot be reached, or that answers something this cannot
// read, is not a contradiction. The gate exists to catch a change that goes
// against a decision, and a run stopped because a judge was unavailable
// would be a gate reporting its own outage as the work's fault.
func contradicts(ctx context.Context, s *store.Store, t Task, f flow.Flow, p flow.Phase, eng engine.Engine, wt string) *contradiction {
	if f.AllowContradictions || eng == nil {
		return nil
	}

	changed := taskWrote(changesOf(s, t))

	decisions := governed(s, t, changed)
	if len(decisions) == 0 {
		return nil
	}

	out, err := eng.Run(ctx, engine.Request{
		Prompt:      checkPrompt(decisions, changed),
		Dir:         wt,
		Permissions: []string{flow.PermissionRead},
		Env:         childEnv(t),
	})
	if err != nil {
		logger.Warn("task/run", "%s: the consistency check of phase %q could not be made: %v", t.ID, p.Name, err)
		return nil
	}

	id, why := verdictIn(out.Output)
	if id == "" {
		return nil
	}

	return &contradiction{Decision: id, Why: why}
}

// checkPrompt is what the judging model reads: the decisions, then the files
// the change touched.
//
// The paths and not the diff itself. The model is run in the worktree and
// may read it, which is the posture this asks for — and a diff pasted into a
// prompt is the one copy of the change that can be stale by the time it is
// judged.
func checkPrompt(decisions []record.Event, changed []repo.Change) string {
	var b strings.Builder

	b.WriteString(checkAsk)
	b.WriteString("\n## Decisions\n\n")

	for _, d := range decisions {
		fmt.Fprintf(&b, "- id: %s\n  scope: %s\n  decision: %s\n",
			d.Data["id"], d.Data["scope"], strings.ReplaceAll(strings.TrimSpace(d.Text), "\n", " "))
	}

	b.WriteString("\n## Files this task changed\n\n")

	for _, c := range changed {
		fmt.Fprintf(&b, "- %s\n", c.Path)
	}

	return b.String()
}

// verdictIn reads the judge's answer: the decision it says was contradicted,
// and the reason it gave.
//
// Anything it cannot read is consistent. The alternative — treating an
// unreadable answer as a contradiction — would stop runs whenever a model
// phrased its verdict differently, and a gate that fires on its own
// misreadings is one nobody leaves on.
func verdictIn(answer string) (id, why string) {
	lines := strings.Split(strings.TrimSpace(answer), "\n")
	if len(lines) == 0 {
		return "", ""
	}

	first := strings.TrimSpace(lines[0])

	rest, found := strings.CutPrefix(strings.ToLower(first), "verdict:")
	if !found {
		return "", ""
	}

	name, ok := strings.CutPrefix(strings.TrimSpace(rest), "contradicts")
	if !ok {
		return "", ""
	}

	// The id is taken from the original line and not the lowered one: a
	// decision called Keep-It is not the same file as keep-it.
	id = strings.TrimSpace(first[len(first)-len(strings.TrimSpace(name)):])
	if id == "" {
		return "", ""
	}

	return id, strings.TrimSpace(strings.Join(lines[1:], "\n"))
}

// stopContradicting ends a run that went against what it had decided.
//
// It names the decision and repeats the judge's reason, because the reader's
// answer is one of two things — change the code back, or supersede the
// decision — and neither can be chosen from the fact that something was
// inconsistent.
func stopContradicting(s *store.Store, t Task, p flow.Phase, c contradiction) error {
	text := fmt.Sprintf("The change goes against the decision %q, so phase %q did not run.\n%s",
		c.Decision, p.Name, c.Why)

	_ = emit(s, t, record.Event{ //nolint:errcheck // best-effort: the run is ending either way
		Kind: record.TaskContradicts,
		Text: text,
		Data: map[string]string{"decision": c.Decision, "phase": p.Name},
	})

	return fmt.Errorf("task %s: %s", t.ID, text)
}
