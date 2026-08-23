package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/task"
)

func newTask(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("repo", ".", "the repository the task is against")
	id := fs.String("id", "", "the identifier of the task")
	// No default here, and the empty string rather than "task": which flow
	// a task walks when nobody says is the user's setting, and a default
	// spelled out on this flag would quietly override it.
	flowName := fs.String("flow", "", "which flow the task walks; the default is the one orbit set flow chose")
	if err := parse(fs, args, out); err != nil {
		return err
	}
	text := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if *id == "" {
		return fmt.Errorf("new needs -id")
	}
	if text == "" {
		return fmt.Errorf("new needs the task written out after the flags")
	}

	s, r, err := openBoth(*dir)
	if err != nil {
		return err
	}
	t, err := task.Create(s, r, *id, text, *flowName)
	if err != nil {
		return err
	}
	// The flow is echoed even when it was not typed. It was still chosen —
	// by the settings, or by what this program ships — and a decision the
	// user did not make is exactly the one worth showing them.
	fmt.Fprintf(out, "%s written against %s, to walk the %s flow\n", t.ID, r.Name, t.Flow)
	return nil
}
