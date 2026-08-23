package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/task"
)

func newTask(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	id := fs.String("id", "", "the identifier of the task")
	if err := parse(fs, args, out); err != nil {
		return err
	}
	text := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if *id == "" {
		return fmt.Errorf("new needs -id")
	}
	if text == "" {
		return fmt.Errorf("new needs the task written out after the flags")
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		return err
	}
	if _, err := task.Create(s, r, *id, text); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s written against %s\n", *id, r.Name)
	return nil
}
