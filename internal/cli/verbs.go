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
// The hand-written commands stay hand-written. `orbit task start` blocks until a
// phase finishes where a generated one would not, and `orbit pr merge`
// streams as it goes; those are facts about the command line rather than
// about the verb.

import (
	"slices"
	"strings"

	"github.com/e1i0r/orbit/internal/verb"
)

// withVerbs is the hand-written commands, and one for every declared verb
// they do not already carry.
func withVerbs(hand []Command) []Command {
	out := slices.Clone(hand)

	for i, c := range out {
		if v, declared := verb.One(c.Name); declared {
			out[i] = family(c, v)
		}
	}

	for i, c := range out {
		if c.Name == "pr" {
			out[i].Run = streamingPR(c.Run)
		}

		// The task's verbs that stay written by hand: starting blocks
		// until a phase finishes, and merging, killing, joining,
		// directing, permitting and marking run their own way for the
		// same reason merge and close keep theirs.
		if c.Name == "task" {
			out[i].Run = taskHands(c.Run)
		}
	}

	return append(out, fromVerbs(out)...)
}

// taskHands keeps the task verbs that stay written by hand on their own
// bodies. Starting blocks until a phase finishes where asking would not,
// and the rest run their own way for the same reason merge and close keep
// theirs: killing, joining with the run's environment, directing with a
// name on it, and permit and critical answering with their own polarity.
func taskHands(otherwise func(Context, []string) error) func(Context, []string) error {
	hands := map[string]func(Context, []string) error{
		"start":    runTask,
		"cancel":   cancelTask,
		"join":     joinRepo,
		"direct":   directTask,
		"permit":   permitTask,
		"critical": criticalTask,
	}

	return func(ctx Context, args []string) error {
		if len(args) > 0 {
			if run, ok := hands[args[0]]; ok {
				return run(ctx, args[1:])
			}
		}

		return otherwise(ctx, args)
	}
}

// streamingPR keeps merge and close on the bodies that stream as they go
// and put their warnings where warnings belong. The generated asking runs
// the verb and prints the answer after; these two are watched while they
// run, which is what the window's toolbar and every script around them
// were written against.
func streamingPR(otherwise func(Context, []string) error) func(Context, []string) error {
	return func(ctx Context, args []string) error {
		if len(args) > 0 {
			rest := args[1:]

			switch args[0] {
			case "merge":
				return mergePR(ctx, rest)
			case "close":
				return closePR(ctx, rest)
			}
		}

		return otherwise(ctx, args)
	}
}

// fromVerbs is every command the declaration implies that the hand-written
// list does not already carry: one for each verb that belongs to nobody.
func fromVerbs(hand []Command) []Command {
	carried := func(name string) bool {
		return slices.ContainsFunc(hand, func(c Command) bool { return c.Name == name })
	}

	var out []Command

	for _, v := range verb.Every() {
		// A child is not a command of its own. It is reached through its
		// parent, which is the whole point of it having one: `orbit rules
		// keep` and never `orbit keep`, which says nothing about what.
		if v.Under == "" && !carried(v.Name) {
			c := family(commandFor(v), v)

			// The task's verbs that stay written by hand ride the
			// generated parent the same way merge and close ride pr.
			if v.Name == "task" {
				c.Run = taskHands(c.Run)
			}

			out = append(out, c)
		}
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

// family gives a command the dispatch its verb's children need, and hands it
// back untouched when there are none.
//
// A hand-written command keeps what it does on its own: `orbit pr <id>`
// opens the pull request the way it always has, and `orbit pr merge <id>`
// is the child.
func family(c Command, v verb.Verb) Command {
	kids := v.Children()
	if len(kids) == 0 {
		return c
	}

	c.Args, c.Run = familyArgs(c.Args, kids), runFamily(kids, c.Run)

	return c
}

// runFamily reads the first word after the command as the child it names,
// and hands everything after it to that child.
//
// A line with no child at all is the command as it was: `orbit rules` lists
// what `orbit rules keep` acts on, and `orbit pr <id>` opens the pull
// request. The family's front page is the parent itself and needs no word of
// its own.
func runFamily(
	kids []verb.Verb, otherwise func(Context, []string) error,
) func(Context, []string) error {
	return func(ctx Context, args []string) error {
		if len(args) > 0 {
			for _, kid := range kids {
				if args[0] == kid.Name {
					return askFor(ctx, kid, args[1:])
				}
			}
		}

		return otherwise(ctx, args)
	}
}

// familyArgs is the usage line: what the command takes on its own, and then
// the children and what they take.
//
// The repository flag is written once. Every line on this screen begins with
// it, and twice on one line reads as two different flags.
//
// A parent that already takes the task takes no child's: the id is the same
// positional either way, and one child's flags are not the family's —
// `orbit task` lists its children, and each one says its own line elsewhere.
func familyArgs(own string, kids []verb.Verb) string {
	names := make([]string, 0, len(kids))
	for _, kid := range kids {
		names = append(names, kid.Name)
	}

	if strings.Contains(own, "<id>") {
		return own + "  |  " + strings.Join(names, "|")
	}

	return own + "  |  " + strings.Join(names, "|") +
		strings.TrimPrefix(argsOf(kids[0]), "[-repo <dir>]")
}
