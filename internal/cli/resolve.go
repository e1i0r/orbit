package cli

// orbit resolve: what reviewers asked for, brought back for a phase to
// answer.

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// resolveComments reads the reviews on a task's pull requests into its
// record, so the next run answers them.
//
// It reads and records; it does not run. Answering a review costs money and
// changes code, and the reader who typed this asked for the comments to be
// picked up — `orbit run` is how they say to go and answer them. The two
// verbs stay apart for the reason cancel and requeue do: one of them spends,
// and a command that quietly does both is a command somebody runs twice.
func resolveComments(ctx Context, args []string) error {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, "resolve")
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/resolve", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/resolve", "load task %q failed: %v", id, err)
		return err
	}

	where, err := worked(s, r, id)
	if err != nil {
		logger.Error("cli/resolve", "read the repositories of task %q failed: %v", id, err)
		return err
	}

	return gather(ctx, s, t, where)
}

// gather reads every repository the task was worked in, and says what it
// found.
//
// A repository whose pull request cannot be read is reported and not fatal.
// A task with three of them and one unreachable remote is a task whose other
// two reviews are still worth answering, and refusing all three would make a
// network fault look like a task with nothing to do.
func gather(ctx Context, s *store.Store, t task.Task, where []repo.Repo) error {
	p := ctx.printer()
	total := 0

	for _, r := range where {
		wtDir, err := s.WorktreeDir(r.Path, t.ID)
		if err != nil {
			return err
		}

		comments, err := r.ReviewComments(wtDir, branchOf(t))
		if err != nil {
			logger.Warn("cli/resolve", "read the reviews of task %q in %q: %v", t.ID, r.Name, err)
			fmt.Fprintln(ctx.Err, p.T("resolve.unread", "{repo}: the pull request could not be read",
				words.Arg{Name: "repo", Value: r.Name}))

			continue
		}

		n, err := task.Review(s, t, r, comments)
		if err != nil {
			return err
		}

		total += n
	}

	if total == 0 {
		fmt.Fprintln(ctx.Out, p.T("resolve.nothing", "nobody has asked {id} for anything",
			words.Arg{Name: "id", Value: t.ID}))

		return nil
	}

	fmt.Fprintln(ctx.Out, p.P("resolve.found", total,
		"{n} comment is now on {id}; `orbit run {id}` answers it",
		"{n} comments are now on {id}; `orbit run {id}` answers them",
		words.Arg{Name: "id", Value: t.ID}))

	return nil
}
