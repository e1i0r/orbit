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
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/words"
)

// top opens the window, or draws one frame of it.
//
// The sweep runs before either. Its errors are carried past the frame and
// past the window rather than returned on the spot: a sentence printed
// immediately before a full-screen program starts is a sentence the alt
// screen wipes before anybody has read it, and one printed instead of the
// frame would replace the thing that was asked for with a complaint about a
// record on the side.
func top(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("top", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	once := fs.Bool("once", false, "draw one frame as plain text and exit — what a pipe, a log or CI reads")
	lang := fs.String("lang", "", "draw in this language, e.g. es; otherwise $ORBIT_LANG, then the saved setting, then $LANG")
	dir, err := parseTop(fs, args, out)
	if err != nil {
		return err
	}

	opts, s, err := window(dir, *lang)
	if err != nil {
		return err
	}
	swept := reconcileAll(s)

	if *once || !interactive(out) {
		frame, err := ui.Plain(opts)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, frame)
		return swept
	}
	if _, err := tea.NewProgram(fullScreen{ui.New(opts)}).Run(); err != nil {
		return fmt.Errorf("the window: %w", err)
	}
	return swept
}

// parseTop reads top's flags with the directory allowed on either side of
// them.
//
// flag stops at the first argument that is not a flag, so `orbit top ~/work
// -once` would leave -once unread — and the failure would not be a message
// but a full-screen window where a frame was asked for, in a pipe, in CI.
// Every other command in this program takes its directory as -repo and never
// meets this; top's directory is the thing a person types most often, and
// making them type a flag for it to be read is the wrong half of the trade.
//
// So: parse, take one positional, parse what is left, until nothing is left.
// A second directory is refused rather than one of them being chosen
// silently — two roots is a person meaning something this command cannot do,
// and picking the first is how they find out an hour later.
func parseTop(fs *flag.FlagSet, args []string, out io.Writer) (string, error) {
	var dirs []string
	rest := args
	for {
		if err := parse(fs, rest, out); err != nil {
			return "", err
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		dirs = append(dirs, rest[0])
		rest = rest[1:]
	}
	if len(dirs) > 1 {
		return "", fmt.Errorf("top takes one directory, and was given %d: %s", len(dirs), strings.Join(dirs, " "))
	}
	if len(dirs) == 1 {
		return dirs[0], nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("locate the working directory: %w", err)
	}
	return dir, nil
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
func window(dir, lang string) (ui.Options, *store.Store, error) {
	if err := mustBeDirectory(dir); err != nil {
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
	r := board.NewReader(s)
	// The engines this build can run, by the name a record carries. It is a
	// map of one today, and it is a map because the record already names its
	// engine and a task run by something else has to be answered by name
	// rather than by assumption.
	engines := map[string]engine.Engine{"claude": engine.NewClaude()}
	return ui.Options{
		Root:     dir,
		Reader:   r,
		Settings: cfg,
		// Four sources, weighed once, here: the flag beats $ORBIT_LANG,
		// which beats the saved setting, which beats the locale the process
		// was started in. The window is handed the answer, not the question.
		Words:     words.For(words.Resolve(lang, os.Getenv("ORBIT_LANG"), cfg.Language())),
		Control:   controlPort(s),
		Start:     startPort(s),
		MarkRead:  markReadPort(s),
		Take:      takePort(r, engines),
		Flows:     s,
		CanResume: canResume(engines),
	}, s, nil
}

// mustBeDirectory refuses a root that is not one.
//
// The board's repositories come from the state root rather than from this
// path — what it is for is the header and the empty state — so a typo would
// otherwise draw a perfectly good window saying "No repositories under
// /wrok", which reads as "you have no work" rather than as "you typed that
// wrong".
func mustBeDirectory(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("look at %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", dir)
	}
	return nil
}

// interactive is whether opening a full-screen program over this writer is
// something a person asked for.
//
// Both halves are load-bearing. A writer that is not the process's own
// stdout is a caller that collected the output — a test, another command —
// and a program that seized the terminal there would take a keyboard nobody
// offered it; every test in this package passes a buffer, so this is the
// line that makes `orbit top` impossible to open by accident from inside one.
// A stdout that is not a character device is a pipe or a file, where a frame
// is what was wanted whether or not -once was typed.
func interactive(out io.Writer) bool {
	if out != io.Writer(os.Stdout) {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// fullScreen draws a model on the alternate screen.
//
// bubbletea v2 has no WithAltScreen option: whether a frame takes the whole
// terminal is a field on the View a model returns, so asking for one is
// something only a model can do. internal/ui's View does not set it, and it
// is right not to — a window that always seized the screen could not be
// rendered into a golden, and where a frame is drawn is this layer's
// decision rather than the layout's.
//
// It is a wrapper of four lines rather than a flag on ui.Options for the
// same reason: Options is what the window is allowed to reach, and how the
// terminal is put into a mode is not part of what it draws. Re-wrapping in
// Update is what keeps the alt screen through every state the model moves
// to — a bubbletea model answers with the next model, and an unwrapped one
// would drop the terminal back to the scrollback on the first keystroke.
type fullScreen struct{ tea.Model }

// Update passes the message down and keeps the wrapper on what comes back.
func (f fullScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := f.Model.Update(msg)
	return fullScreen{next}, cmd
}

// View is the model's own frame, on the alternate screen.
func (f fullScreen) View() tea.View {
	v := f.Model.View()
	v.AltScreen = true
	return v
}
