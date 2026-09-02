package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// requeueTask takes a task back: whatever is running stops, and the task
// returns to the queue to be started again.
//
// It is beside cancel and it is not cancel. Cancel is for work that is over
// and files the task under done; this is for work that was begun wrongly —
// the brief, the engine, the task itself — and the row belongs back in to do
// where somebody will pick it up once it has been fixed.
func requeueTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("requeue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")

	by := fs.String("by", "operator", "who is taking the task back")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, "requeue")
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/requeue", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/requeue", "load task %q in %q failed: %v", id, r.Name, err)
		return err
	}

	// Ctrl-C reaches the wait rather than being swallowed, for the reason it
	// does in direct -restart: a run that has stopped listening is waited on
	// for half a minute, and that is a terminal the reader may want back.
	signalled, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	why := strings.Join(fs.Args()[1:], " ")
	if err := task.Requeue(signalled, s, t, *by, why); err != nil {
		logger.Error("cli/requeue", "requeue task %q in %q failed: %v", id, r.Name, err)
		return err
	}

	logger.Info("cli/requeue", "task %s in %s taken back to the queue", id, r.Name)
	fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("requeue.back_in_queue", "{id} stopped and back in to do",
		words.Arg{Name: "id", Value: id}))

	return nil
}
