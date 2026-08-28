package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
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
		unread, err := unreadCount(s, *dir)
		if err != nil {
			logger.Error("cli/direct", "count what is unread before restarting %q: %v", id, err)
			return err
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

// unreadCount is how many finished tasks nobody has looked at yet, which is
// the brake `orbit run` and this command are both refused by.
//
// A board that could not be read is a refusal and not a zero. Zero is the one
// number task.Start never stops for — atCap is written as limit > 0 && unread
// >= limit — so a state root that failed to walk used to take the brake off
// silently, at the moment it was least worth trusting. The reader is told
// instead, and can restart the task again once the count can be taken.
//
// The count is of the repository this command was pointed at, and not of
// every repository the state root knows about. The window counts over the
// root it was opened on; here the only root anybody named is -repo, and
// widening it to the whole machine would be this command inventing a root
// nobody typed. It is the narrower number, and it is the honest one to take
// from what was said.
func unreadCount(s *store.Store, dir string) (int, error) {
	b, _, err := board.NewReader(s, dir).Refresh()
	if err != nil {
		return 0, err
	}

	return board.Unread(b), nil
}
