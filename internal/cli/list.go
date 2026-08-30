package cli

import (
	"flag"
	"fmt"
	"io"

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

	s, r, err := openBoth(*dir)
	if err != nil {
		return err
	}

	ids, err := task.List(s, r)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("list.no_tasks", "no tasks against {repo}",
			words.Arg{Name: "repo", Value: r.Name}))

		return nil
	}

	for _, id := range ids {
		fmt.Fprintln(ctx.Out, id)
	}

	return nil
}
