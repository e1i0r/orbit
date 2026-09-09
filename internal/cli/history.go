package cli

// orbit history: everything ever said about a task, as markdown.
//
// The same reading the history tab draws and the same file a session is
// handed when the terminal is opened on a task — printed, so a reader can
// pipe it somewhere, paste it into a program Orbit does not drive, or read
// it before deciding who should carry the task on.

import (
	"flag"
	"fmt"
	"io"

	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// taskHistory prints one task's conversation.
func taskHistory(ctx Context, args []string) error {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	keep := fs.Bool("write", false, "leave it in the task's directory as well, and say where")

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, "history")
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		return fmt.Errorf("open repository %q: %w", *dir, err)
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		return fmt.Errorf("load task %q: %w", id, err)
	}

	body, err := task.History(s, t)
	if err != nil {
		return err
	}

	fmt.Fprint(ctx.Out, body)

	if !*keep {
		return nil
	}

	where, err := task.WriteHistory(s, t)
	if err != nil {
		return err
	}

	fmt.Fprintln(ctx.Err, ctx.printer().T("history.written", "also written to {where}",
		words.Arg{Name: "where", Value: where}))

	return nil
}
