package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
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
		return needsTaskID(ctx, "note")
	}

	text := strings.Join(fs.Args()[1:], " ")
	if strings.TrimSpace(text) == "" {
		return errors.New(ctx.printer().T("note.needs_text", "note needs text for task {id}",
			words.Arg{Name: "id", Value: id}))
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
	fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("note.written_down",
		"note recorded for {id} — read by the next phase that starts",
		words.Arg{Name: "id", Value: id}))

	return nil
}
