package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/task"
)

// readTask writes down that somebody has looked at a finished task.
//
// It is the other half of the unread cap: the cap counts finished work
// nobody has read, and this is the only thing that lowers that count. A
// brake with no release is a brake people disable, so the release is a
// command and not a gesture only the window has.
func readTask(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("read", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(fs, args, out); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("read needs the id of a task")
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		return err
	}
	t, err := task.Load(s, r, id)
	if err != nil {
		return err
	}
	if err := task.MarkRead(s, t); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s marked read\n", id)
	return nil
}
