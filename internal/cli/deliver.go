package cli

// Delivering a task that was worked in more than one repository.
//
// A task is not filed under a repository any more: repositories join it by
// being worked in, and by the time it is delivered there may be three. Each
// one gets a pull request, and each body names the others — a reviewer
// looking at the frontend change has to be able to see that a backend change
// goes with it, because merging one half of a pair is how a broken pair
// ships.
//
// The order they merge in is not Orbit's to say. It opens them all, in the
// order the repositories joined, and leaves the sequence to the reader. The
// dependency runs whichever way the work runs, and nothing this program can
// read says which way that is.

import (
	"fmt"
	"os"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// aPullRequest is what the delivery did in one repository of a task.
type aPullRequest struct {
	repo  repo.Repo
	wtDir string
	url   string // empty when the work left this repository as it found it
}

// worked is every repository the task was worked in, in the order they
// joined it.
//
// A task whose record says nothing about repositories was written before one
// could join, or by a hand: the repository the command was run from is the
// whole of its scope, and answering nothing at all would deliver a task that
// has work in it as a task with nowhere to deliver to.
func worked(s *store.Store, r repo.Repo, taskID string) ([]repo.Repo, error) {
	paths, err := s.TaskRepos(taskID)
	if err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return []repo.Repo{r}, nil
	}

	repos := make([]repo.Repo, 0, len(paths))

	for _, path := range paths {
		one, err := repo.Open(path)
		if err != nil {
			return nil, err
		}

		repos = append(repos, one)
	}

	return repos, nil
}

// deliverTask opens a pull request in every repository that has something to
// show for the task, and then tells each of them about the others.
//
// Two passes, because the first cannot link: the second pull request has no
// URL while the first body is being written. So the bodies are written once
// with the task in them and once more with the siblings.
func deliverTask(ctx Context, s *store.Store, t task.Task, where []repo.Repo) error {
	opened, err := checkouts(ctx, s, t, where)
	if err != nil {
		return err
	}

	// The story is read once, here, and carried into every body. Reading it
	// per repository would ask the record the same question N times for one
	// answer, and a story that changed between two of those reads would put
	// two accounts of one task in two pull requests.
	story := task.StoryOf(s, t)

	for i, one := range opened {
		if opened[i].url, err = openPR(ctx, t, one, story); err != nil {
			return err
		}
	}

	report(ctx, opened)

	return crossLink(ctx, t, opened, story)
}

// checkouts is where the task's work is in each of its repositories, and a
// refusal when one of them has none.
//
// Every repository is looked at before any of them is pushed. A checkout
// that is not there is a delivery that cannot be made — the work it was
// supposed to carry is not anywhere this command can reach — and stopping
// halfway through would leave a task with a pull request in one repository,
// nothing in the next, and no way to tell from either that the other was
// meant to exist.
func checkouts(ctx Context, s *store.Store, t task.Task, where []repo.Repo) ([]aPullRequest, error) {
	found := make([]aPullRequest, 0, len(where))

	for _, r := range where {
		wtDir, err := s.WorktreeDir(r.Path, t.ID)
		if err != nil {
			return nil, err
		}

		if _, err := os.Stat(wtDir); err != nil {
			logger.Error("cli/pr", "task %s has no worktree in %s: %v", t.ID, r.Name, err)

			return nil, fmt.Errorf("%s: %w", ctx.printer().T("pr.no_checkout",
				"{repo} has no checkout of this task to deliver from; `orbit join -task {id} {repo}` opens one again",
				words.Arg{Name: "repo", Value: r.Name},
				words.Arg{Name: "id", Value: t.ID}), err)
		}

		found = append(found, aPullRequest{repo: r, wtDir: wtDir})
	}

	return found, nil
}

// openPR delivers the task's work in one repository.
//
// It answers an empty URL when the repository has nothing to deliver, which
// is not a failure: a repository the work opened, read and left as it was
// joined the task and is worth saying so, and a pull request for an
// unchanged branch is a review request for nothing.
func openPR(ctx Context, t task.Task, one aPullRequest, story *task.Story) (string, error) {
	r, wtDir := one.repo, one.wtDir

	if err := r.SyncBaseBranch(wtDir, r.Base); err != nil {
		logger.Warn("cli/pr", "sync base branch %q into %q: %v", r.Base, wtDir, err)
	}

	if err := r.CommitWorktree(wtDir, subjectOf(t)); err != nil {
		logger.Error("cli/pr", "commit worktree %q failed: %v", wtDir, err)

		return "", err
	}

	delivers, err := r.WorktreeAhead(wtDir, r.Base)
	if err != nil {
		logger.Error("cli/pr", "read the work in %q failed: %v", wtDir, err)

		return "", err
	}

	if !delivers {
		return "", nil
	}

	return pushAndOpen(ctx, t, r, wtDir, story)
}

// pushAndOpen puts the branch on the remote and asks gh for a pull request.
func pushAndOpen(ctx Context, t task.Task, r repo.Repo, wtDir string, story *task.Story) (string, error) {
	branch := branchOf(t)
	if err := r.PushBranch(wtDir, branch); err != nil {
		logger.Error("cli/pr", "push branch %q failed: %v", branch, err)

		return "", err
	}

	url, err := r.CreatePR(wtDir, titleOf(t), bodyOf(t, nil, story), branch, r.Base)
	if err != nil {
		logger.Error("cli/pr", "gh pr create failed: %v", err)

		return "", fmt.Errorf("%s: %w", ctx.printer().T("pr.pushed_but_not_opened",
			"branch {branch} was pushed and gh pr create refused",
			words.Arg{Name: "branch", Value: branch}), err)
	}

	logger.Info("cli/pr", "created pull request %s for task %s in %s (base=%s)", url, t.ID, r.Name, r.Base)

	return url, nil
}

// crossLink tells every pull request of the task about the others.
//
// One repository is left alone: a task delivered where it was written shells
// out to gh once, the way it always has, and there is nothing for a body to
// say about siblings it does not have.
func crossLink(ctx Context, t task.Task, opened []aPullRequest, story *task.Story) error {
	if len(opened) < 2 {
		return nil
	}

	for i, one := range opened {
		if one.url == "" {
			continue
		}

		if err := one.repo.EditPR(one.wtDir, branchOf(t), bodyOf(t, siblings(opened, i), story)); err != nil {
			logger.Error("cli/pr", "gh pr edit failed in %s: %v", one.repo.Name, err)

			return fmt.Errorf("%s: %w", ctx.printer().T("pr.opened_but_not_linked",
				"the pull requests are open and gh pr edit refused, so none of them names the others"), err)
		}
	}

	return nil
}

// siblings is every repository of the task except the one at i.
func siblings(opened []aPullRequest, i int) []aPullRequest {
	rest := make([]aPullRequest, 0, len(opened)-1)
	rest = append(rest, opened[:i]...)

	return append(rest, opened[i+1:]...)
}

// report says what the delivery did, a line per repository.
func report(ctx Context, opened []aPullRequest) {
	p := ctx.printer()

	for _, one := range opened {
		if one.url == "" {
			fmt.Fprintf(ctx.Out, "%s\n", p.T("pr.nothing_to_deliver",
				"{repo}: nothing changed here, so there is no pull request",
				words.Arg{Name: "repo", Value: one.repo.Name}))

			continue
		}

		fmt.Fprintf(ctx.Out, "%s\n", p.T("pr.created", "pull request created for {repo}: {url}",
			words.Arg{Name: "repo", Value: one.repo.Name},
			words.Arg{Name: "url", Value: one.url}))
	}
}

// bodyOf is what the pull request says: the task, and where else it is being
// delivered.
//
// A repository that changed nothing is named too, without a link. It is the
// answer to a question a reviewer would otherwise have to ask — whether the
// task looked at that repository at all — and "I looked here and it needed
// nothing" is worth reading.
func bodyOf(t task.Task, others []aPullRequest, story *task.Story) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## Orbit Task: %s\n\n%s\n", t.ID, t.Text)
	b.WriteString(storySection(story))

	if len(others) > 0 {
		b.WriteString("\n### The rest of this task\n\n")

		for _, one := range others {
			fmt.Fprintf(&b, "- %s\n", one.line())
		}
	}

	b.WriteString("\nGenerated automatically by Orbit.")

	return b.String()
}

// line is how one repository of a task reads in another's pull request.
func (a aPullRequest) line() string {
	if a.url == "" {
		return "`" + a.repo.Name + "` — joined, nothing to change"
	}

	return "`" + a.repo.Name + "` — " + a.url
}

// opening is the first line of the task, which is what a commit subject and
// a pull request title are made of.
func opening(t task.Task) string {
	return strings.SplitN(strings.TrimSpace(t.Text), "\n", 2)[0]
}

// branchOf is the branch a task's work is on, and it is the same name in
// every repository the task joined: three checkouts of orbit/PAY-1 are three
// halves of one task, and a reviewer reading any of them reads the same name.
func branchOf(t task.Task) string {
	return "orbit/" + t.ID
}

// subjectOf is the commit subject the delivery writes.
func subjectOf(t task.Task) string {
	return clipWords(fmt.Sprintf("feat(%s): %s", t.ID, opening(t)), 72)
}

// titleOf is the pull request title.
func titleOf(t task.Task) string {
	title := fmt.Sprintf("%s: %s", t.ID, opening(t))
	// 87 and not 90, because the three dots are part of what a reader sees:
	// the branch that cuts at a space appends them, and 90 plus three dots
	// is 93 characters in a field meant to hold 90.
	if short := clipWords(title, 87); short != title {
		return short + "..."
	}

	return title
}
