package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
)

// directTask interrupts an in-flight run while preserving memory, records the
// directive and note, and optionally restarts the task.
func directTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("direct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	by := fs.String("by", "operator", "who is giving the directive")

	restart := fs.Bool("restart", false, "immediately restart the task after directing it")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("direct needs the id of a task")
	}

	text := strings.Join(fs.Args()[1:], " ")
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("direct needs a message for task %s", id)
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/direct", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/direct", "load task %q in %q failed: %v", id, r.Name, err)
		return err
	}

	if *restart {
		unread := 0

		rdr := board.NewReader(s, *dir)
		if b, _, rErr := rdr.Refresh(); rErr == nil {
			unread = board.Unread(b)
		}

		pid, err := task.Reopen(s, t, *by, text, t.Flow, unread)
		if err != nil {
			logger.Error("cli/direct", "reopen task %q failed: %v", id, err)
			return err
		}

		logger.Info("cli/direct", "task %s redirected and restarted (pid %d)", id, pid)
		fmt.Fprintf(ctx.Out, "%s redirected and restarted (pid %d)\n", id, pid)

		return nil
	}

	if err := task.Direct(s, t, *by, text); err != nil {
		logger.Error("cli/direct", "direct task %q failed: %v", id, err)
		return err
	}

	logger.Info("cli/direct", "directive recorded for task %s", id)
	fmt.Fprintf(ctx.Out, "%s redirected: directive recorded\n", id)

	return nil
}
