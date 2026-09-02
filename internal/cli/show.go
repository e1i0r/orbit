package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

func show(ctx Context, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	id := fs.Arg(0)
	if id == "" {
		return needsTaskID(ctx, "show")
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		return err
	}

	events, err := task.Events(s, task.Task{ID: id, Repo: r})
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return errors.New(nothingRecorded(ctx, id, r))
	}

	w := tabwriter.NewWriter(ctx.Out, 0, 0, 2, ' ', 0)
	for _, e := range events {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			stamp(e.At), e.Kind, e.Phase, firstLine(detail(e.Text, e.Data)))
	}

	return w.Flush()
}

// nothingRecorded is what show says when the id names no task.
//
// A task can be against no repository, and the refusal for one of those
// names none either: "nothing recorded for ACME-1 in " puts a hole where the
// reader looks for the answer, and the repository is not part of why the id
// found nothing.
func nothingRecorded(ctx Context, id string, r repo.Repo) string {
	p := ctx.printer()
	if r.Name == "" {
		return p.T("show.nothing_recorded_anywhere", "nothing recorded for {id}",
			words.Arg{Name: "id", Value: id})
	}

	return p.T("show.nothing_recorded", "nothing recorded for {id} in {repo}",
		words.Arg{Name: "id", Value: id}, words.Arg{Name: "repo", Value: r.Name})
}

// stamp says when, with the day included: a task that ran last week printed
// as 15:04:05 reads as though it ran this morning.
//
// An event with no time is not a task that happened at the start of the
// Christian era — it is the placeholder record.Read synthesises for a line it
// could not parse. Printing 0001-01-01 there would be a date, and a wrong one.
func stamp(t time.Time) string {
	if t.IsZero() {
		return "—"
	}

	return t.Format("2006-01-02 15:04:05")
}

// detail is what the last cell says about an event, and for an event that
// ended badly that is why it ended.
//
// Text is what the engine printed. The reason a phase failed lives in
// Data["error"], where phaseEnd puts it (task/run.go) so that a log ending
// at phase.failed still says why — and a row that reads "phase.failed" and
// then quotes the last line the engine happened to print reads as though
// that line were the failure. So the reason wins the cell when there is one;
// what the engine printed is in the log, which is a file you can read.
//
// It takes the two fields rather than the event because internal/cli does
// not import internal/record and should not start: that absence is what
// stops a command writing to the log behind internal/task's back
// (arch/imports_test.go).
func detail(text string, data map[string]string) string {
	if reason := data["error"]; reason != "" {
		return reason
	}

	if text == "" {
		// A repository joining is the one event whose whole content is in
		// Data: the name is what the row is about, and without it the line
		// reads "repo.joined" over an empty cell.
		return data["repo"]
	}

	return text
}

// firstLine keeps the table a table. The whole text stays in events.jsonl,
// which is the point of it being a file you can read with cat.
//
// The text is whatever the engine printed, and this table is tab-delimited:
// one tab inside it silently adds a column and every row below drifts, and a
// carriage return would redraw the row over itself on a terminal.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i] + " …"
	}

	return strings.NewReplacer("\t", " ", "\r", " ").Replace(s)
}
