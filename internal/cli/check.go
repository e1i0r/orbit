package cli

// orbit check: whether the record is still the record — and the same
// question before every command, for a reader who has asked for it.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// checkCommand says in words what SQLite says about the file.
//
// It takes no flags of its own; parse still runs so `-h` shows the same
// shape every other command does, and an unknown flag is refused the same
// way.
func checkCommand(ctx Context, args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	s, err := store.Open()
	if err != nil {
		return err
	}

	d, err := s.Record()
	if err != nil {
		return err
	}

	found, err := d.Check()
	if err != nil {
		return err
	}

	if len(found) > 0 {
		return damaged(ctx, s.DBPath(), found)
	}

	held, err := d.Totals()
	if err != nil {
		return err
	}

	p := ctx.printer()

	fmt.Fprintf(ctx.Out, "%s\n", p.T("check.sound", "{path} is sound, and holds {what}",
		words.Arg{Name: "path", Value: s.DBPath()},
		words.Arg{Name: "what", Value: counted(p, held.Tasks, held.Events, held.Messages)}))

	return nil
}

// damaged prints what was found and answers with the sentence the
// dispatcher prints after it.
//
// The lines themselves are SQLite's own and are not put through the
// printer: they name pages and index entries, and what a reader does with
// one is paste it somewhere it can be read by somebody who knows what a
// page is. Translating half of such a line would make it worse.
func damaged(ctx Context, path string, found []string) error {
	p := ctx.printer()

	fmt.Fprintf(ctx.Out, "%s\n", p.T("check.damaged", "{path} is damaged, and SQLite says:",
		words.Arg{Name: "path", Value: path}))

	for _, line := range found {
		fmt.Fprintf(ctx.Out, "  %s\n", line)
	}

	return errors.New(p.T("check.damaged_error",
		"the record is damaged: copy the file aside, and try `orbit export` before anything writes to it again"))
}

// counted is the record in numbers, for the two commands that say how much
// of it there is.
//
// Three fragments rather than one sentence with three numbers in it: a
// count and its noun have to agree, and they agree differently in every
// language. What is joined here is a list, and a list has the same shape in
// both of the ones Orbit speaks.
func counted(p *words.Printer, tasks, events, turns int) string {
	return strings.Join([]string{
		p.P("record.tasks", tasks, "{n} task", "{n} tasks"),
		p.P("record.events", events, "{n} event", "{n} events"),
		p.P("record.turns", turns, "{n} supervisor turn", "{n} supervisor turns"),
	}, ", ")
}

// checkRecord asks the same question before every command, when the settings
// say to ask it.
//
// It is off until somebody turns it on, and it is a setting rather than a
// flag on one command or another, because both halves of that matter. A
// check costs a full read of the file, which is not a thing to do before
// `orbit list`; and a check somebody has to remember to type is a check that
// gets typed the day after the record stopped being readable, which is the
// day it is of no use. Turning it on is for a machine that has already given
// somebody a reason to doubt its disk.
//
// Damage never stops the command. It is said, on the terminal and in the
// log, and then whatever was typed runs — because what was typed might be
// `orbit export`, which is how the readable part of a damaged record is got
// out, and a guard on the way in would bar that door too.
func checkRecord(ctx Context) {
	s, err := store.Open()
	if err != nil {
		logger.Error("cli/check", "%v", err)
		return
	}

	cfg, err := s.Settings()
	if err != nil {
		logger.Error("cli/check", "%v", err)
		return
	}

	if !cfg.CheckRecord {
		return
	}

	found, err := ask(s)
	if err != nil {
		logger.Error("cli/check", "%v", err)
		fmt.Fprintf(ctx.Err, "orbit: %v\n", err)

		return
	}

	if len(found) == 0 {
		return
	}

	logger.Error("cli/check", "the record is damaged: %s", strings.Join(found, "; "))
	fmt.Fprintf(ctx.Err, "orbit: %s\n", ctx.printer().T("check.warning",
		"the record is damaged — `orbit check` says what SQLite found"))
}

// ask opens the record, asks, and lets go of it again.
//
// The handle is let go of because the command about to run opens its own,
// and the whole point of a handle per process is that there is one.
func ask(s *store.Store) ([]string, error) {
	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	found, err := d.Check()

	return found, errors.Join(err, s.Close())
}
