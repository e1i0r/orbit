package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/task"
)

func list(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository to list")
	if err := parse(fs, args, out); err != nil {
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
		fmt.Fprintf(out, "no tasks against %s\n", r.Name)
		return nil
	}
	for _, id := range ids {
		fmt.Fprintln(out, id)
	}
	return nil
}
