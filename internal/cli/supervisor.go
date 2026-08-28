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

// supervisorCommand reads or appends to the persistent supervisor conversation thread.
func supervisorCommand(ctx Context, args []string) error {
	fs := flag.NewFlagSet("supervisor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	by := fs.String("by", "operator", "who is speaking in the thread")
	channel := fs.String("channel", "cli", "channel where the message originated")
	taskID := fs.String("task", "", "task id if this message refers to a specific task")
	repoName := fs.String("repo", "", "repository name if this message refers to a specific repo")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	s, err := store.Open()
	if err != nil {
		logger.Error("cli/supervisor", "open state store failed: %v", err)
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		rdr := board.NewReader(s, "")
		lines, err := rdr.SupervisorLog()
		if err != nil {
			logger.Error("cli/supervisor", "read supervisor history failed: %v", err)
			return err
		}
		if len(lines) == 0 {
			fmt.Fprintln(ctx.Out, "the supervisor thread is empty")
			return nil
		}
		for _, l := range lines {
			timeStr := l.At.Format("15:04:05")
			tag := fmt.Sprintf("[%s via %s]", l.By, l.Channel)
			if l.TaskID != "" {
				tag += fmt.Sprintf(" (%s)", l.TaskID)
			}
			fmt.Fprintf(ctx.Out, "%s %s %s\n", timeStr, tag, l.Text)
		}
		return nil
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
