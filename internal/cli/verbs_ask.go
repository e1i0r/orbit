package cli

// Turning a command line into what a verb was asked with.
//
// Asking is one shape whatever the verb is: flags for the fields the reader
// spelled out, positionals in the order the verb declares them, and the rest
// of the line for the one field of words. What a verb means by what it was
// asked is the verb's, declared once in internal/verb; this is only the
// reading of the line.

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// filled takes what is left on the line and puts it in the fields it is for.
//
// `orbit settings set autopilot on` is what a person writes; `-key autopilot
// -value on` is what a form would have wanted. The order is the order the
// verb declares, one word per field it must have, and a field of words takes
// everything that is left — so the sentence at the end of `orbit note abc it
// needs a test` arrives whole. A flag still wins, because a script that
// spelled it out meant it.
//
// A leading "--" is dropped. It is the shell's way of saying the flags are
// over, and flag stops at the first positional — so a caller that puts it
// after the id is saying so once the flags are over already. Kept, it
// became the first word of every note the window wrote.
func filled(v verb.Verb, args map[string]string, rest []string) []string {
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}

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

		// An optional name is still a name when it is there: `reconcile
		// PAY-1` narrows the sweep to one task, and a reader who typed it
		// named something rather than adding noise. Yes-or-no stays on
		// its flag — a bare "true" on the line is a typo until proven
		// otherwise — and a field of words still takes the rest.
		if !f.Needed && f.Kind != verb.Named {
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
//
// A verb that declares its own repo field is the one carrying the sentence
// that explains what the repository means to it, so the built-in flag is
// not listed twice: one `-repo` on the line, not two that disagree.
func argsOf(v verb.Verb) string {
	out := ""

	if !takes(v, "repo") {
		out = "[-repo <dir>]"
	}

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

// takes says whether the verb declares a field of that name.
func takes(v verb.Verb, name string) bool {
	for _, f := range v.Takes {
		if f.Name == name {
			return true
		}
	}

	return false
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

	in := verb.In{Args: map[string]string{}, By: "operator", Door: "the command line"}
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

	// Ctrl-C reaches the wait rather than being swallowed: a verb that
	// stops a run waits for it to actually be gone, and that is a
	// terminal the reader may want back.
	signalled, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	out, err := verb.Run(signalled, w, v.Path(), in)
	if err != nil {
		return err
	}

	fmt.Fprintln(ctx.Out, out.Said)

	return nil
}
