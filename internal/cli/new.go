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

func newTask(ctx Context, args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	id := fs.String("id", "", "the identifier of the task")
	// No default here, and the empty string rather than "task": which flow
	// a task walks when nobody says is the user's setting, and a default
	// spelled out on this flag would quietly override it.
	flowName := fs.String("flow", "", "which flow the task walks; the default is the one orbit set flow chose")
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

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/new", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Create(s, r, *id, text, *flowName)
	if err != nil {
		logger.Error("cli/new", "create task %q in %q failed: %v", *id, r.Name, err)
		return err
	}

	logger.Info("cli/new", "created task %s in repo %s (flow=%s)", t.ID, r.Name, t.Flow)
	fmt.Fprintf(ctx.Out, "%s\n", ctx.printer().T("new.written",
		"{id} written against {repo}, to walk the {flow} flow",
		words.Arg{Name: "id", Value: t.ID}, words.Arg{Name: "repo", Value: r.Name}, words.Arg{Name: "flow", Value: t.Flow}))

	return nil
}
