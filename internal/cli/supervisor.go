package cli

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// supervisorCommand reads or appends to the persistent supervisor conversation thread.
func supervisorCommand(ctx Context, args []string) error {
	fs := flag.NewFlagSet("supervisor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	by := fs.String("by", "operator", "who is speaking in the thread")
	channel := fs.String("channel", "cli", "channel where the message originated")
	taskID := fs.String("task", "", "task id if this message refers to a specific task")
	repoName := fs.String("repo", "", "repository name if this message refers to a specific repo")

	retract := fs.String("retract", "", "take back the line with this `number`, as `orbit supervisor` numbers them")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	s, err := store.Open()
	if err != nil {
		logger.Error("cli/supervisor", "open state store failed: %v", err)
		return err
	}

	// A string rather than an int, so that a typed -retract 0 is refused
	// instead of being indistinguishable from not having asked at all.
	if *retract != "" {
		n, err := strconv.Atoi(*retract)
		if err != nil {
			return fmt.Errorf("-retract takes the number of a line in the thread, not %q", *retract)
		}

		return retractLine(ctx, s, n)
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return printThread(ctx, s)
	}

	text := strings.Join(remaining, " ")
	if err := task.RecordSupervisor(s, "", *by, *channel, *taskID, *repoName, text); err != nil {
		logger.Error("cli/supervisor", "record supervisor message failed: %v", err)
		return err
	}

	logger.Info("cli/supervisor", "recorded supervisor message by %s", *by)
	fmt.Fprintf(ctx.Out, "recorded in supervisor thread: %s\n", text)

	return nil
}

// printThread lists the thread, numbered, so a line can be pointed at.
//
// The number is the position in this listing and nothing more durable than
// that: an event has no id, and the thread only grows at the end, so what is
// line 3 today is line 3 tomorrow. -retract reads it back the same way.
func printThread(ctx Context, s *store.Store) error {
	lines, err := thread(s)
	if err != nil {
		return err
	}

	if len(lines) == 0 {
		fmt.Fprintln(ctx.Out, "the supervisor thread is empty")
		return nil
	}

	for i, l := range lines {
		fmt.Fprintf(ctx.Out, "%3d  %s\n", i+1, line(l))
	}

	return nil
}

// line is one thread entry as it reads on a terminal.
func line(l view.SupervisorLine) string {
	tag := fmt.Sprintf("[%s via %s]", l.By, l.Channel)
	if l.TaskID != "" {
		tag += fmt.Sprintf(" (%s)", l.TaskID)
	}
	// A withdrawn line is still shown. It was said, and hiding it would
	// leave the rest of the thread answering something that is not there.
	if l.Retracted {
		tag += " (retracted)"
	}

	return fmt.Sprintf("%s %s %s", l.At.Format("15:04:05"), tag, l.Text)
}

// retractLine takes back the line at the given position in the listing.
func retractLine(ctx Context, s *store.Store, n int) error {
	lines, err := thread(s)
	if err != nil {
		return err
	}

	if n < 1 || n > len(lines) {
		return fmt.Errorf("there is no line %d in the supervisor thread; it has %d", n, len(lines))
	}

	l := lines[n-1]
	if l.Retracted {
		return fmt.Errorf("line %d was already taken back", n)
	}

	if err := task.RetractSupervisor(s, l.At); err != nil {
		logger.Error("cli/supervisor", "retract supervisor message failed: %v", err)
		return err
	}

	logger.Info("cli/supervisor", "retracted supervisor line %d", n)
	fmt.Fprintf(ctx.Out, "took back line %d: %s\n", n, l.Text)

	return nil
}

func thread(s *store.Store) ([]view.SupervisorLine, error) {
	lines, err := board.NewReader(s, "").SupervisorLog()
	if err != nil {
		logger.Error("cli/supervisor", "read supervisor history failed: %v", err)
		return nil, err
	}

	return lines, nil
}
