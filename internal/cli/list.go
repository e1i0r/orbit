package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

func list(ctx Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository to list")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	// Orbit is opened on the directory above the repositories, so a reader
	// running `orbit list` there is asking about the workspace and not
	// about a checkout. Requiring one made the answer to "what is on the
	// board" a sentence about git.
	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		return err
	}

	ids, err := tasksListed(s, r, given(fs, "repo"))
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		if !given(fs, "repo") {
			fmt.Fprintln(ctx.Out, ctx.printer().T("list.none", "no tasks yet"))

			return nil
		}

		fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("list.no_tasks", "no tasks against {repo}",
			words.Arg{Name: "repo", Value: r.Name}))

		return nil
	}

	for _, id := range ids {
		fmt.Fprintln(ctx.Out, id)
	}

	return nil
}

// tasksListed is the tasks a reader asked for: the ones that touched a
// repository they named, or every task there is when they named none.
//
// Named, and not "happens to be standing in". Orbit is opened on the
// directory above the repositories and a reader can also be inside one of
// them; a listing that filtered by where the shell happens to be would
// answer two different questions from two directories with nobody having
// asked differently. -repo is the ask.
//
// Every task and not none for the unnamed case. A workspace holds tasks that
// reach into three repositories and tasks that reach into no repository at
// all, and a listing that answered "nothing" would be wrong about both.
func tasksListed(s *store.Store, r repo.Repo, named bool) ([]string, error) {
	if !named || r.Path == "" {
		return s.TaskIDs()
	}

	return task.List(s, r)
}
