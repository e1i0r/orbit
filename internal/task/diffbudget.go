package task

// The diff budget: how much a task may change, and where.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// diffVerdict is a diff that went past what the flow allowed: how big it
// was, what it was allowed to be, and the files nobody planned for.
type diffVerdict struct {
	Lines     int
	Budget    int
	Unplanned []string
}

// overDiff measures what the task has changed in every repository it has
// joined, and answers with a verdict when that is more than the flow allows.
//
// Two questions, one measurement. How much was changed is a number the flow
// carries; where it was changed is compared against the plan phase's own
// output, which is re-derived from this run's record every time rather than
// declared once — a scope somebody wrote down in advance is a scope that
// stops being true the moment the work turns out to be somewhere else, and
// the point of asking at all is to notice exactly that.
//
// It answers nothing at all when the flow sets no budget. A repository that
// cannot be read is skipped rather than fatal: a count that could not be
// taken is not a diff that was too big, and stopping a run over it would
// refuse work for a reason that has nothing to do with the work.
func overDiff(s *store.Store, t Task, f flow.Flow) *diffVerdict {
	if f.DiffBudget <= 0 {
		return nil
	}

	changed := changesOf(s, t)
	if len(changed) == 0 {
		return nil
	}

	lines := 0
	for _, c := range changed {
		lines += c.Lines()
	}

	unplanned := outsidePlan(s, t, changed)
	if lines <= f.DiffBudget && len(unplanned) == 0 {
		return nil
	}

	return &diffVerdict{Lines: lines, Budget: f.DiffBudget, Unplanned: unplanned}
}

// changesOf is every file the task has changed, across every repository it
// has joined.
//
// All of them and not only the one the run started in: a task that spans
// three repositories is one piece of work, and a budget that counted one of
// them would be three budgets nobody set.
func changesOf(s *store.Store, t Task) []repo.Change {
	var all []repo.Change

	for _, path := range reposOf(s, t) {
		r, err := repo.Open(path)
		if err != nil {
			continue
		}

		wt, err := s.WorktreeDir(r.Path, t.ID)
		if err != nil {
			continue
		}

		changes, err := r.WorktreeChanges(wt)
		if err != nil {
			continue
		}

		all = append(all, taskWrote(changes)...)
	}

	return all
}

// taskWrote leaves out what Orbit itself put in the worktree.
//
// A decision's copy under .orbit/ is written by Orbit, in the same checkout,
// between one phase and the next. Counted, it would spend a task's diff
// budget on Orbit's own writing and report a file the plan never named — and
// both would be true, and neither would be the reader's problem. What these
// two gates are about is what the agent changed.
func taskWrote(changes []repo.Change) []repo.Change {
	kept := make([]repo.Change, 0, len(changes))

	for _, c := range changes {
		if c.Path == OrbitDir || strings.HasPrefix(c.Path, OrbitDir+"/") {
			continue
		}

		kept = append(kept, c)
	}

	return kept
}

// reposOf is where the task has worked, and the repository it was written
// against when it has worked nowhere yet.
func reposOf(s *store.Store, t Task) []string {
	paths, err := s.TaskRepos(t.ID)
	if err == nil && len(paths) > 0 {
		return paths
	}

	if t.Repo.Path == "" {
		return nil
	}

	return []string{t.Repo.Path}
}

// outsidePlan is the changed files the plan did not mention.
//
// A file is planned for when the plan phase's own output names its path.
// That is a plain reading of the text and not an interpretation of it: Orbit
// does not decide what the model meant by a paragraph, and a plan that names
// `internal/task/run.go` has named it whatever else it says around it.
//
// A run with no plan phase, or one whose plan said nothing, has no scope —
// and no scope is not an empty scope. Every file is planned for then, and
// the line budget is the whole of the gate: a flow without a plan phase is a
// flow that never agreed where the work would be, so there is nothing to
// have gone outside of.
func outsidePlan(s *store.Store, t Task, changed []repo.Change) []string {
	plan := planText(s, t)
	if strings.TrimSpace(plan) == "" {
		return nil
	}

	var out []string

	for _, c := range changed {
		if !strings.Contains(plan, c.Path) {
			out = append(out, c.Path)
		}
	}

	sort.Strings(out)

	return out
}

// planText is what the plan phase of this run answered.
//
// This run and not the last one: the record is read from the newest
// task.started, so a second attempt is measured against the plan the second
// attempt made. A phase counts as the plan when its name says so, which is
// the same reading a person makes of a flow file.
func planText(s *store.Store, t Task) string {
	events, err := Events(s, t)
	if err != nil {
		return ""
	}

	text := ""

	for _, e := range events {
		if e.Kind == record.TaskStarted {
			text = ""
			continue
		}

		if e.Kind == record.PhaseFinished && isPlan(e.Phase) {
			text = e.Text
		}
	}

	return text
}

// stopChanging ends a run whose diff went past what the flow allowed.
//
// It stops rather than trying again. A phase told to do the same work in
// fewer lines is a phase that will write the same work again — what the
// number is for is a person deciding whether this change is still the change
// they asked for, and that decision cannot be delegated back to the thing
// that made it.
func stopChanging(s *store.Store, t Task, p flow.Phase, v diffVerdict) error {
	text := fmt.Sprintf("%d lines changed against a budget of %d, so phase %q did not run.",
		v.Lines, v.Budget, p.Name)
	if len(v.Unplanned) > 0 {
		text += fmt.Sprintf("\nChanged and not in the plan: %s.", strings.Join(v.Unplanned, ", "))
	}

	data := map[string]string{
		"lines":  strconv.Itoa(v.Lines),
		"budget": strconv.Itoa(v.Budget),
		"phase":  p.Name,
	}
	if len(v.Unplanned) > 0 {
		data["unplanned"] = strings.Join(v.Unplanned, ",")
	}

	_ = emit(s, t, record.Event{ //nolint:errcheck // best-effort: the run is ending either way
		Kind: record.TaskOverDiff,
		Text: text,
		Data: data,
	})

	return fmt.Errorf("task %s: %s", t.ID, text)
}
