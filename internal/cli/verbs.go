package cli

// The commands the command line gets for free.
//
// internal/verb declares every action Orbit can be asked for. Anything in
// that list this package has not written a command for by hand gets one
// built from the declaration: the flags come from what the verb takes, the
// sentence from what it says about itself, and the doing from the one body
// in internal/verb.
//
// Which is the point. A verb added to the declaration turns up here without
// anybody remembering to add it, and the arch test that holds the four ways
// in to the same vocabulary stops being a chore and starts being a promise.
//
// The hand-written commands stay hand-written. `orbit new` has been spelled
// that way in scripts for as long as Orbit has existed, and `orbit run`
// blocks until a phase finishes where a generated one would not; those are
// facts about the command line rather than about the verb.

import (
	"context"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// withVerbs is the hand-written commands, and one for every declared verb
// they do not already carry.
func withVerbs(hand []Command) []Command {
	return append(hand, fromVerbs(hand)...)
}

// fromVerbs is a command for every declared verb the hand-written list does
// not already carry.
func fromVerbs(hand []Command) []Command {
	var out []Command

	for _, v := range verb.Every() {
		if slices.ContainsFunc(hand, func(c Command) bool { return c.Name == v.Name }) {
			continue
		}

		out = append(out, commandFor(v))
	}

	return out
}

// commandFor is one verb as a command.
//
// A reading typed inside the window opens the window's own screen for it
// rather than printing under the cockpit: the reader is already looking at
// the thing they asked for, and a page of text scrolled into the frame is
// how the screen gets corrupted.
func commandFor(v verb.Verb) Command {
	c := Command{
		Name:       v.Name,
		Args:       argsOf(v),
		NeedsArgs:  v.OnTask,
		AboutATask: v.OnTask,
		About:      v.About,
		Run:        func(ctx Context, args []string) error { return askFor(ctx, v, args) },
	}

	if v.Reads {
		c.InWindow = WindowOpens
	}

	return c
}

// filled takes what is left on the line and puts it in the fields it is for.
//
// `orbit set autopilot on` is what a person writes; `-key autopilot -value
// on` is what a form would have wanted. The order is the order the verb
// declares, one word per field it must have, and a field of words takes
// everything that is left — so the sentence at the end of `orbit note abc it
// needs a test` arrives whole. A flag still wins, because a script that
// spelled it out meant it.
func filled(v verb.Verb, args map[string]string, rest []string) []string {
	for _, f := range v.Takes {
		if len(rest) == 0 {
			return nil
		}

		if args[f.Name] != "" {
			continue
		}

		if f.Kind == verb.Words {
			args[f.Name] = strings.Join(rest, " ")

			return nil
		}

		if !f.Needed {
			continue
		}

		args[f.Name], rest = rest[0], rest[1:]
	}

	// Whatever is left has nowhere to go. Printing the settings for a
	// reader who typed `orbit settings autopilot on` and walking away is
	// how a run sat at a gate for ten minutes waiting for an autopilot
	// nobody had turned on.
	return rest
}

// wordsOf is the field a verb takes as typed words, if it has one.
func wordsOf(v verb.Verb) (string, bool) {
	for _, f := range v.Takes {
		if f.Kind == verb.Words {
			return f.Name, true
		}
	}

	return "", false
}

// argsOf is the usage line, built from what the verb takes.
func argsOf(v verb.Verb) string {
	out := "[-repo <dir>]"

	if v.OnTask {
		out += " <id>"
	}

	for _, f := range v.Takes {
		switch {
		case f.Kind == verb.Words:
			continue
		case f.Needed:
			out += " <" + f.Name + ">"
		default:
			out += " [-" + f.Name + " <" + f.Name + ">]"
		}
	}

	if f, ok := wordsOf(v); ok {
		out += " <" + f + ">"
	}

	return out
}

// askFor turns a command line into an In and asks for the verb.
func askFor(ctx Context, v verb.Verb, args []string) error {
	fs := flag.NewFlagSet(v.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")

	// A flag per field, in the words the field says about itself, so
	// `orbit <verb> -h` explains the verb rather than listing its shape.
	said := map[string]*string{}
	for _, f := range v.Takes {
		said[f.Name] = fs.String(f.Name, "", f.About(ctx.printer()))
	}

	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	in := verb.In{Args: map[string]string{}, By: "operator"}
	for name, value := range said {
		in.Args[name] = *value
	}

	rest := fs.Args()
	if v.OnTask {
		if in.Task = fs.Arg(0); in.Task == "" {
			return needsTaskID(ctx, v.Name)
		}

		rest = rest[1:]
	}

	if over := filled(v, in.Args, rest); len(over) > 0 {
		return fmt.Errorf("%s", ctx.printer().T("verb.takes_no_more",
			"{verb} takes nothing after {extra}",
			words.Arg{Name: "verb", Value: v.Name},
			words.Arg{Name: "extra", Value: strings.Join(over, " ")}))
	}

	s, r, err := openMaybe(*dir, given(fs, "repo"))
	if err != nil {
		return fmt.Errorf("open repository %q: %w", *dir, err)
	}

	in.Repo = r.Path

	w := newWorld(s, board.NewReader(s, *dir), ctx.printer())

	out, err := verb.Run(context.Background(), w, v.Name, in)
	if err != nil {
		return err
	}

	fmt.Fprintln(ctx.Out, out.Said)

	return nil
}
