package task

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
)

// maxOutput is how much of an engine's answer is kept in the record.
const maxOutput = 1 << 20

// captured cuts an engine's output down to what the record can hold and says
// in the text when it had to. Truncation that announces itself is honest;
// silent loss is not. The second return is the size of what was said, zero
// when nothing was cut.
func captured(out string) (text string, full int) {
	if len(out) <= maxOutput {
		return out, 0
	}

	n := maxOutput
	// Never cut a rune in half: the record is UTF-8, and a severed tail
	// would come back from the log as a replacement character.
	for n > 0 && !utf8.RuneStart(out[n]) {
		n--
	}

	return out[:n] + fmt.Sprintf("\n…[truncated, full output was %d bytes]", len(out)), len(out)
}

// prepare opens the worktree a run works in, which is the repository the
// task was written against joining it.
//
// It goes through the same verb a repository joined on the fourth phase goes
// through, and that is the point rather than a saving: the first repository
// is not special, and a task whose starting repository was recorded by one
// road and every other by a second would have two accounts of its scope and
// no reason to believe either.
//
// A task with no repository has no worktree to open and gets a directory of
// its own instead. Nothing has joined it yet, so there is nothing to join it
// through: the first repository arrives when the phase runs `orbit join`,
// by the same road the fourth one takes.
func prepare(s *store.Store, t Task) (string, error) {
	if t.Repo.Path == "" {
		return s.CreateWorkDir(t.ID)
	}

	return Join(s, t, t.Repo)
}

// fedOutput is what a phase is handed of the phase before it, which is
// nothing at all unless it asked to be fed.
func fedOutput(p flow.Phase, prev string) string {
	if !p.FeedOutput {
		return ""
	}

	return prev
}

// prompt is what the engine is told for one phase: the task, the phase it is
// running, what the phase before it said, whatever the operator has added
// since, and — on a second attempt at the same phase — what the attempts
// before it tried and why the gate refused them.
//
// tried is variadic because it is empty on the first attempt at every phase,
// which is most of them, and a caller that has nothing to say about earlier
// attempts should not have to say it.
//
// It is written in Markdown because the answer is asked for in Markdown, and
// a prompt that asks in one shape for another is asking twice.
func prompt(t Task, p flow.Phase, notes []string, prevOutput string, others []string, tried ...gateRefusal) string {
	return build(t, p, false, notes, nil, prevOutput, others, tried...)
}

// promptFor is prompt for a phase whose place in the flow is known, which is
// the one thing that decides whether it is asked for the story: the last
// phase is the only one that can tell how the task ended.
func promptFor(t Task, f flow.Flow, n int, notes, reviews []string, prevOutput string, others []string, tried ...gateRefusal) string {
	p := f.Phases[n-1]

	return build(t, p, n == len(f.Phases), notes, reviews, prevOutput, others, tried...)
}

func build(t Task, p flow.Phase, last bool, notes, reviews []string, prevOutput string, others []string, tried ...gateRefusal) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n%s\n\n", t.ID, strings.TrimSpace(t.Text))
	fmt.Fprintf(&b, "## Phase\n\n%s\n", where(t, p))
	b.WriteString(workspace(t, others))

	if p.Prompt != "" {
		fmt.Fprintf(&b, "\n## Phase instructions\n\n%s\n", strings.TrimSpace(p.Prompt))
	}

	// Fenced rather than set as prose: what the phase before wrote is
	// Markdown of its own, and its headings loose under a heading of this
	// prompt would read as sections of the prompt.
	if prevOutput != "" {
		fmt.Fprintf(&b, "\n## Previous phase output\n\n%s\n", engine.Fenced(prevOutput))
	}

	b.WriteString(refusals(tried))

	if len(notes) > 0 {
		b.WriteString("\n## Operator notes\n\n")

		for _, n := range notes {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}

	// After the operator's own notes and before the contract. A reviewer
	// asking for something is the same kind of instruction a note is — it
	// came from a person and it is about this task — but the operator is
	// the one whose word settles a disagreement between the two, so theirs
	// is read first.
	if len(reviews) > 0 {
		b.WriteString("\n## What reviewers asked for\n\n")

		for _, r := range reviews {
			fmt.Fprintf(&b, "- %s\n", r)
		}
	}

	// Before the answer contract, which is last for the reason it says: the
	// last thing said is the thing a model holds on to, and how to write is
	// the thing every phase has to hold on to.
	if isPlan(p.Name) {
		b.WriteString(decisionAsk)
	}

	if last {
		b.WriteString(storyAsk)
	}

	b.WriteString("\n" + engine.AnswerContract)

	return b.String()
}

// where is the line that says which phase is running and in what.
//
// A task with no repository is told so plainly rather than being handed an
// empty name in a sentence shaped for one — "in repository “" is a fact
// about a bug, not about the task. What to do about it is in the workspace
// section below, next to the names: a phase told it may join a repository
// on one line and given the list on another is a phase that guesses.
func where(t Task, p flow.Phase) string {
	if t.Repo.Name == "" {
		return fmt.Sprintf("`%s`. This task is not being worked in any repository yet.", p.Name)
	}

	return fmt.Sprintf("`%s`, in repository `%s`.", p.Name, t.Repo.Name)
}

// workspace is what the phase is told about the repositories it is not in.
//
// The names, and the one command that opens a checkout of one — nothing
// else. Orbit does not read the task and suggest which of them it thinks the
// work needs: a scope guessed up front is the same forgetting with a
// different author, and the whole of the many-repos design is that the set
// stays open until the work closes it. A workspace with nothing else in it
// says nothing at all, because a list of no repositories and an instruction
// for reaching them is a paragraph asking to be misread.
//
// For a task that is in none of them the same list is the whole of where the
// work can go, and the paragraph says so: the phase is not being invited to
// widen a scope it already has, it is being told how to get one.
func workspace(t Task, others []string) string {
	if len(others) == 0 {
		return ""
	}

	var b strings.Builder

	if t.Repo.Name == "" {
		b.WriteString("\n## Repositories in this workspace\n\n")
	} else {
		b.WriteString("\n## Other repositories in this workspace\n\n")
	}

	for _, name := range others {
		fmt.Fprintf(&b, "- `%s`\n", name)
	}

	if t.Repo.Name == "" {
		fmt.Fprintf(&b, "\nThis task has no checkout of its own yet. Run "+
			"`orbit join <name>` for whichever of these the work turns out to be "+
			"in: it prints a checkout to work in, and the repository becomes part "+
			"of the task. Do this in any phase, as many times as the work calls "+
			"for; nothing had to be declared in advance.\n")

		return b.String()
	}

	fmt.Fprintf(&b, "\nIf this task turns out to need a change in one of them, run "+
		"`orbit join <name>`. It prints a checkout to work in, and the repository "+
		"becomes part of the task. Do this whenever the work calls for it, in any "+
		"phase; nothing had to be declared in advance.\n")

	return b.String()
}

// elsewhere is the repositories a phase is told it can reach, by name.
//
// A walk that fails answers nothing rather than stopping the run. The
// listing is a convenience for the engine — a name it would otherwise have
// to be told — and a task that cannot see its neighbours is a task that runs
// in the one repository it already has, which is what every run did before
// any of this. A task with none runs in its own directory and is told
// nothing, which is the same trade one step further along.
func elsewhere(s *store.Store, t Task) []string {
	found, err := reachable(s, t.Repo)
	if err != nil {
		logger.Error("task/run", "list the repositories task %q can reach failed: %v", t.ID, err)
		return nil
	}

	var names []string

	for _, r := range found {
		if r.Path != t.Repo.Path {
			names = append(names, r.Name)
		}
	}

	return names
}
