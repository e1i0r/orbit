package cli

// orbit approve: saying yes to the libraries a task reached for.

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// approveTask writes down that a reader accepted what the dependency gate
// stopped a run for.
//
// It approves what is pending now rather than taking names as arguments. The
// reader is answering the question the record asked them — these libraries,
// this task — and a command that took its own list would let somebody
// approve a name the task never added, which is a yes to a question nobody
// put.
func approveTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("approve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, "approve")
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		logger.Error("cli/approve", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, id)
	if err != nil {
		logger.Error("cli/approve", "load task %q failed: %v", id, err)
		return err
	}

	f, err := flowOfTask(s, t)
	if err != nil {
		logger.Error("cli/approve", "resolve the flow of task %q failed: %v", id, err)
		return err
	}

	pending := task.Pending(s, t, f)
	if len(pending) == 0 {
		fmt.Fprintln(ctx.Out, ctx.Words.T("approve.nothing", "{id} has added no dependency waiting on you",
			words.Arg{Name: "id", Value: t.ID}))

		return nil
	}

	if err := task.Approve(s, t, pending); err != nil {
		logger.Error("cli/approve", "approve the dependencies of task %q failed: %v", id, err)
		return err
	}

	fmt.Fprintln(ctx.Out, ctx.Words.T("approve.done", "approved for {id}: {names}",
		words.Arg{Name: "id", Value: t.ID},
		words.Arg{Name: "names", Value: strings.Join(pending, ", ")}))

	return nil
}

// flowOfTask is the flow a task walks, by the same reading `orbit run` makes
// of it: the task's own, then the one Orbit ships. Not the settings default,
// which is what the next task written gets.
func flowOfTask(s flow.Source, t task.Task) (flow.Flow, error) {
	chosen := t.Flow
	if chosen == "" {
		chosen = flow.Default
	}

	return flow.Resolve(s, chosen)
}
