package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
)

// noteTask records a user note for a task.
func noteTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("note", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("note needs the id of a task")
	}

	text := strings.Join(fs.Args()[1:], " ")
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("note needs text for task %s", id)
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/note", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/note", "load task %q in %q failed: %v", id, r.Name, err)
		return err
	}

	if err := task.Note(s, t, text); err != nil {
		logger.Error("cli/note", "append note to %q in %q failed: %v", id, r.Name, err)
		return err
	}

	logger.Info("cli/note", "note added to task %s in repo %s", id, r.Name)
	fmt.Fprintf(ctx.Out, "note recorded for %s — read by the next phase that starts\n", id)

	return nil
}
