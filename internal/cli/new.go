package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

func newTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	id := fs.String("id", "", "the identifier of the task")
	// No default here, and the empty string rather than "task": which flow
	// a task walks when nobody says is the user's setting, and a default
	// spelled out on this flag would quietly override it.
	flowName := fs.String("flow", "", "which flow the task walks; the default is the one orbit set flow chose")
	// Written down and run, in one command. The window's compose form has
	// a Save and start button and it saved without starting: it built this
	// command line, and there was nothing on it that said to start. Doing
	// it here rather than in the window is what makes the button honest
	// without a second gesture racing the board's next refresh.
	startNow := fs.Bool("start", false, "start the task as soon as it is written down")

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	text := strings.TrimSpace(strings.Join(fs.Args(), " "))

	if *id == "" {
		return errors.New(ctx.printer().T("new.needs_id", "new needs -id"))
	}

	if text == "" {
		return errors.New(ctx.printer().T("new.needs_text", "new needs the task written out after the flags"))
	}

	// A -repo the reader typed has to open. The default has not been typed:
	// a task written from a directory that is not a checkout is a task
	// against no repository, which is a task all the same, and the first
	// repository joins it in whichever phase the work reaches one.
	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		return fmt.Errorf("open repository %q: %w", *dir, err)
	}

	t, err := task.Create(s, r, *id, text, *flowName)
	if err != nil {
		return fmt.Errorf("create task %q in %q: %w", *id, r.Name, err)
	}

	logger.Info("cli/new", "created task %s in repo %q (flow=%s)", t.ID, r.Name, t.Flow)
	fmt.Fprintf(ctx.Out, "%s\n", written(ctx, t, r))

	if !*startNow {
		return nil
	}

	// The unread cap is what Start weighs a new run against, and a task
	// written this second has been read by nobody — so it is asked for
	// here rather than assumed to be zero.
	unread, err := unreadCount(s, r.Path)
	if err != nil {
		return err
	}

	pid, err := task.Start(s, t, t.Flow, unread)
	if err != nil {
		return fmt.Errorf("start task %q: %w", t.ID, err)
	}

	logger.Info("cli/new", "started task %s as process %d", t.ID, pid)
	fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("new.started",
		"{id} started, process {pid}", words.Arg{Name: "id", Value: t.ID},
		words.Arg{Name: "pid", Value: strconv.Itoa(pid)}))

	return nil
}

// written is the line that says a task was written down: against a
// repository when there is one, and against none when there is not.
//
// Two sentences rather than one with an empty name in it. "ACME-1 written
// against , to walk the review flow" is a line that reads as a bug, and a
// task that starts nowhere is not a bug — it is the thing the reader just
// asked for, and the line says which one they got.
func written(ctx Context, t task.Task, r repo.Repo) string {
	p := ctx.printer()

	if r.Name == "" {
		return p.T("new.written_nowhere",
			"{id} written against no repository yet, to walk the {flow} flow",
			words.Arg{Name: "id", Value: t.ID}, words.Arg{Name: "flow", Value: t.Flow})
	}

	return p.T("new.written",
		"{id} written against {repo}, to walk the {flow} flow",
		words.Arg{Name: "id", Value: t.ID}, words.Arg{Name: "repo", Value: r.Name}, words.Arg{Name: "flow", Value: t.Flow})
}
