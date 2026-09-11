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
		// A child is not a command of its own. It is reached through its
		// parent, which is the whole point of it having one: `orbit rules
		// keep` and never `orbit keep`, which says nothing about what.
		if v.Under != "" {
			continue
		}

		if slices.ContainsFunc(hand, func(c Command) bool { return c.Name == v.Name }) {
			continue
		}

		c := commandFor(v)
		if kids := v.Children(); len(kids) > 0 {
			c.Args, c.Run = familyArgs(kids), runFamily(v, kids)
		}

		out = append(out, c)
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

// runFamily reads the first word after the command as the child it names,
// and hands everything after it to that child.
//
// A line with no child at all is the parent itself, which is how `orbit
// rules` lists what `orbit rules keep` acts on: the listing is the family's
// front page and needs no word of its own.
func runFamily(parent verb.Verb, kids []verb.Verb) func(Context, []string) error {
	return func(ctx Context, args []string) error {
		if len(args) > 0 {
			for _, kid := range kids {
				if args[0] == kid.Name {
					return askFor(ctx, kid, args[1:])
				}
			}
		}

		return askFor(ctx, parent, args)
	}
}

// familyArgs is the usage line: the children, and then what the first of
// them takes, because they mostly take the same thing.
func familyArgs(kids []verb.Verb) string {
	names := make([]string, 0, len(kids))
	for _, kid := range kids {
		names = append(names, kid.Name)
	}

	return "[" + strings.Join(names, "|") + "] " + argsOf(kids[0])
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

	// Only the first field of words is written at the end of the line: that
	// is the one filled takes the rest of the line for, and a second would
	// swallow it. The others are reachable by their flag and are shown as
	// what they are.
	trailing, _ := wordsOf(v)

	for _, f := range v.Takes {
		switch {
		case f.Name == trailing:
			continue
		case f.Kind == verb.Words || !f.Needed:
			out += " [-" + f.Name + " <" + f.Name + ">]"
		default:
			out += " <" + f.Name + ">"
		}
	}

	if trailing != "" {
		out += " <" + trailing + ">"
	}

	return out
}

// askFor turns a command line into an In and asks for the verb.
func askFor(ctx Context, v verb.Verb, args []string) error {
	fs := flag.NewFlagSet(v.Path(), flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	// A flag per field, in the words the field says about itself, so
	// `orbit <verb> -h` explains the verb rather than listing its shape.
	said := map[string]*string{}
	for _, f := range v.Takes {
		said[f.Name] = fs.String(f.Name, "", f.About(ctx.printer()))
	}

	// And -repo, unless the verb declares one of its own. Registering it
	// twice is what flag panics on, so a verb that named a field `repo`
	// took every invocation of itself down with it: `orbit learn` did
	// exactly that, and shipped green because nothing ran it. The verb's
	// own wins, because it is the one carrying the sentence that explains
	// what the repository means to it.
	dir := said["repo"]
	if dir == nil {
		dir = fs.String("repo", ".", "the repository the task is against")
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
			return needsTaskID(ctx, v.Path())
		}

		rest = rest[1:]
	}

	if over := filled(v, in.Args, rest); len(over) > 0 {
		return fmt.Errorf("%s", ctx.printer().T("verb.takes_no_more",
			"{verb} takes nothing after {extra}",
			words.Arg{Name: "verb", Value: v.Path()},
			words.Arg{Name: "extra", Value: strings.Join(over, " ")}))
	}

	// A declared repo left empty is the caller standing where they are,
	// which is what the built-in flag defaults to.
	where := *dir
	if where == "" {
		where = "."
	}

	s, r, err := openMaybe(where, given(fs, "repo"))
	if err != nil {
		return fmt.Errorf("open repository %q: %w", where, err)
	}

	in.Repo = r.Path

	w := newWorld(s, board.NewReader(s, *dir), ctx.printer())

	out, err := verb.Run(context.Background(), w, v.Path(), in)
	if err != nil {
		return err
	}

	fmt.Fprintln(ctx.Out, out.Said)

	return nil
}
