package cli

// orbit top: one window over every repository, and one frame you can pipe.
//
// This file is the composition root. Everything the window is made of is
// built here — the store, the board reader, the settings adapter, the
// printer, the four ports — and handed over as values on ui.Options. The
// window itself is given no path, no store and no engine, which is the whole
// of why internal/ui can be tested without any of them.
//
// There are two ways out of this command and they draw the same board
// through the same functions: a full-screen program for a terminal, and a
// single frame of plain text for anything that is not one. The second is not
// a debugging aid. It is what makes a window testable at all — the program
// this replaces could only be checked by looking at it — and it is what a
// pipe, a log and a CI job get instead of a screenful of escape codes.

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/env"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/quota"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/supervisor"
	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/words"
)

// top opens the window, or draws one frame of it.
//
// The sweep runs before either. Its errors, and the log's, are carried past
// the frame and past the window rather than returned on the spot: a sentence
// printed immediately before a full-screen program starts is a sentence the
// alt screen wipes before anybody has read it, and one printed instead of the
// frame would replace the thing that was asked for with a complaint about a
// record on the side.
func top(ctx Context, args []string) error {
	// The flags are read by a function of their own so that what they were
	// set to is something a test can ask for. See topflags.go: every test in
	// this package hands the command a buffer and therefore takes the plain
	// branch below through interactive(ctx.Out), so -once cannot be the deciding
	// term in any of them, and asking parseTop directly is the only way this
	// flag has a test that can fail.
	dir, once, lang, err := parseTop(ctx, args)
	if err != nil {
		return err
	}

	opts, s, err := window(ctx, dir, lang)
	if err != nil {
		return err
	}

	logger.Info("cli/top", "orbit top started on %q (once=%v, lang=%q, %d descriptors open)",
		dir, once, lang, logger.OpenFiles())

	trouble := reconcileAll(s)

	if drawsOneFrame(once, interactive(ctx.Out)) {
		opts.Quota = quotaPort(quota.FromEnv(), true)

		frame, err := ui.Plain(opts)
		if err != nil {
			return err
		}

		fmt.Fprintln(ctx.Out, frame)

		return trouble
	}

	if _, err := tea.NewProgram(fullScreen{ui.New(opts)}).Run(); err != nil {
		return fmt.Errorf("%s: %w", ctx.printer().T("top.window_failed", "the window"), err)
	}
	// The same count as the line above, at the other end of a window that
	// may have been open for a day. The two together are the whole of what
	// a leak in here would look like from outside.
	logger.Info("cli/top", "orbit top closed (%d descriptors open)", logger.OpenFiles())

	return trouble
}

// window builds everything one window is made of.
//
// The order is deliberate. The directory is checked first, because it is the
// part a person typed and the part most likely to be wrong, and store.Open
// creates $ORBIT_HOME as a side effect of merely opening it — the same
// reasoning openBoth is written with. The settings come next, and they are
// allowed to refuse: see newSettings, where a file that cannot be read is a
// sentence rather than a window running without its brake.
//
// The store is handed back beside the options, and only to this file. It is
// what the sweep needs, and it is deliberately not reachable from anything
// the window is given.
func window(ctx Context, dir, lang string) (ui.Options, *store.Store, error) {
	// The reader's language, before the store this check stands in front of
	// is open. The Context's printer is the saved setting and nothing else
	// (cli.go says so), so the flag and then $ORBIT_LANG are weighed here on
	// top of it — the same order the window below is given. Left to the
	// printer alone, `orbit top /nowhere` refused in English in front of a
	// window that would have drawn in Spanish.
	p := ctx.printer()

	switch spoken := env.Read(env.Lang); {
	case lang != "":
		p = words.For(words.Resolve(lang, "", ""))
	case spoken != "":
		p = words.For(words.Resolve(spoken, "", ""))
	}

	if err := mustBeDirectory(p, dir); err != nil {
		return ui.Options{}, nil, err
	}

	s, err := store.Open()
	if err != nil {
		return ui.Options{}, nil, err
	}

	cfg, err := newSettings(s)
	if err != nil {
		return ui.Options{}, nil, err
	}
	// The directory is what the board is of, and not only what the header
	// says. The reader walks it for repositories and asks the store what has
	// been written against each; a reader given only the store would answer
	// the same board for every directory on the machine.
	r := board.NewReader(s, dir)
	// The engines this build can run, by the name a record carries. It is a
	// map of one today, and it is a map because the record already names its
	// engine and a task run by something else has to be answered by name
	// rather than by assumption.
	engines := newEngines()

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}

	// The reader's language, weighed once and handed to everything that
	// speaks: the flag beats $ORBIT_LANG, which beats the saved setting,
	// which beats the locale the process was started in. The window is
	// given the answer, not the question, and so are its ports.
	spoken := words.For(words.Resolve(lang, env.Read(env.Lang), cfg.Language()))

	return ui.Options{
		Root: underHome(dir, home),
		// What this build calls itself, so the header can tell a release
		// that is news from the one already running. cli/upgrade.go asks the
		// same question of the same variable; the window has to be handed it
		// because internal/ui cannot reach in here for it.
		Version: Version,
		// The board reader the window is handed carries the settings file on
		// its clock: see poll. The settings adapter answers from memory, and
		// this is what keeps what it holds in step with the file.
		Reader:   poll{Reader: r, cfg: cfg},
		Settings: cfg,
		Words:    spoken,
		Control:  controlPort(s),
		Start:    startPort(s),
		MarkRead: markReadPort(s),
		Requeue:  requeuePort(s),
		RecordSupervisor: func(conversation, by, channel, message string) error {
			return supervisor.RecordIn(s, conversation, "", by, channel, "", "", message)
		},
		NewConversation:    func() string { return supervisor.NewID(time.Now().UTC()) },
		RemoveConversation: func(id string) error { return supervisor.Remove(s, id) },
		Knows:              knowsPort(s, dir),
		KnowsAll:           knowsAllPort(r, s),
		RepoGates:          gatesPort(s),
		RepoFolders:        repo.Folders,
		RepoChecks:         repo.Checks,
		TurnFact:           turnFactPort(s),
		ReplaceFact:        windowReplacePort(s),
		RuleStory:          ruleStoryPort(s, spoken),
		Waiting:            waitingPort(s),
		KeepRule:           keepRulePort(s, dir),
		DropRule:           dropRulePort(s),
		ForgetRule:         forgetRulePort(s, spoken),
		Learn:              learnPort(s),
		NoteTask:           notePort(r, s),
		RetractSupervisor: func(at time.Time) error {
			return supervisor.Retract(s, at)
		},
		RecordDeliver: deliverPort(s),
		AskSupervisor: askSupervisorPort(s, engines),
		Draft:         draftPort(engines),
		AutoSupervise: autoSupervisePort(s, engines),
		DeleteTask:    deleteTaskPort(s),
		Take:          takePort(r, engines),
		Open:          openPort(s, r),
		FileSession:   fileSessionPort(s, r, engines),
		Flows:         s,
		// canResume is asked per task rather than once for the build: the
		// engine a task ran under is the one that decides whether its
		// session can be carried on, and that name lives on the task.
		CanResume: func(name string) bool { return canResume(engines, name) },
		// The palette's two halves: the list it shows, read off the table,
		// and the way it runs one, which is the table's own Run with the
		// settings adapter answering what language the refusal is in.
		Commands: commandTable(),
		Do:       doPort(cfg),
		// The id rule the compose form types against: the store's own, the
		// one every write goes through, and nobody's second copy of it.
		ValidID: store.ValidTaskID,
		Quota:   quotaPort(quota.FromEnv(), false),
		Engines: enginesPort(engines),
	}, s, nil
}

// quotaPort adapts quota.Meter to ui.Options.Quota.
//
// Money is carried across as the answer quota.Mode gave rather than as the
// mode itself, for the reason every port here carries answers: the window
// draws what it is told, and a mode it could read is a mode it could
// interpret. The one place that decides what a number about an engine means
// is the package that knows how the engine is paid for.
func quotaPort(m *quota.Meter, syncWait bool) func(string) roster.Reading {
	if m == nil {
		return nil
	}

	return func(engine string) roster.Reading {
		reading := m.Read(engine, syncWait)

		out := roster.Reading{
			Engine:  reading.Engine,
			Money:   reading.Mode.Spends(),
			Sourced: reading.Sourced,
		}

		for _, w := range reading.Windows {
			out.Windows = append(out.Windows, roster.Window{
				Key:      w.Key,
				Label:    w.Label,
				Pct:      w.Pct,
				ResetsIn: w.ResetsIn,
			})
		}

		return out
	}
}

// mustBeDirectory refuses a root that is not one.
//
// board.Reader would refuse it too — its walk of a path that is not there
// fails, and Refresh carries that up — but the sentence it produces is about
// a walk, and this one is about what was typed. A person who wrote /wrok
// wants to be told that, on the spot, and not to read a window's refusal to
// enumerate.
func mustBeDirectory(p *words.Printer, dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("%s: %w", p.T("top.look_at", "look at {dir}",
			words.Arg{Name: "dir", Value: strconv.Quote(dir)}), err)
	}

	if !info.IsDir() {
		return errors.New(p.T("top.not_a_directory", "{dir} is not a directory",
			words.Arg{Name: "dir", Value: strconv.Quote(dir)}))
	}

	return nil
}
