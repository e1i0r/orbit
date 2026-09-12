package menu

// What is on the menu, built from the lists the Env carries.
//
// Which menu it is comes from what the pointer was on when it opened. The
// board's carries the commands about no task in particular, refusals
// included: `top` greyed with "you are already in it" teaches the shape of
// the program. A task's carries the verbs, each with its reason. Inside a
// task both are there, under a heading each — a menu that opened on a run
// and offered eleven panes and nothing else said, by omission, that looking
// is all there is to do in here.

import (
	"slices"

	"github.com/e1i0r/orbit/internal/ui/keymap"
)

// entries is the menu as it stands: the submenu drilled into, or one of
// the three tops. Recomputed from the lists rather than remembered — a
// menu frozen at open time would keep offering verbs for a run that
// finished while it was up.
func (s State) entries(e Env) []Entry {
	if s.sub != "" {
		return s.subEntries(e)
	}

	if e.Detail {
		return s.detailEntries(e)
	}

	if s.task == "" {
		return boardEntries(e)
	}

	// A task menu with nothing on it is one whose run left the board while
	// it was up, and the drawing says so.
	return s.taskEntries(e)
}

// subEntries is one family drilled into: the key verbs when it is the
// task's, and the family's commands in every case. No further families:
// they are one level deep and so is the menu.
func (s State) subEntries(e Env) []Entry {
	// Same nothing when the task is gone: the submenu would list verbs
	// about a run that is no longer there. The board's own submenus have
	// no task, so they are never empty for this reason.
	if s.task != "" {
		if _, ok := e.verbs(s.task); !ok {
			return nil
		}
	}

	if s.sub != "task" {
		return s.childEntries(e, s.sub)
	}

	out := s.verbEntries(e)

	// verbEntries already ends in the start dialog; what follows are the
	// family's own commands.
	return append(out, s.childEntries(e, s.sub)...)
}

// verbEntries is what can be done to one task with a keystroke, refusals
// included and each with its reason. It is the same list on the board's
// menu and inside a task, from the same call, because they are the same
// question asked from two places.
func (s State) verbEntries(e Env) []Entry {
	all, ok := e.verbs(s.task)
	if !ok {
		return nil
	}

	out := make([]Entry, 0, len(all))

	for i := range all {
		a := &all[i]

		row := Entry{Glyph: a.Key.Help().Key, Title: a.Key.Help().Desc}
		if e.Says != nil {
			row.Detail = e.Says(a.Key)
		}
		if !a.OK {
			row.Dim = true
			row.Reason = a.Why(e.Words)
		}

		out = append(out, row)
	}

	// And what only a command or a dialog can do to it: the start dialog,
	// which asks which flow before anything runs.
	out = append(out, e.start())

	return out
}

// childEntries is one family's commands on the menu of the task they are
// about, with the repository and the id already filled in.
func (s State) childEntries(e Env, parent string) []Entry {
	args := e.args(s.task)

	out := make([]Entry, 0, len(taskCommands))

	for _, want := range taskCommands {
		if want.name != parent {
			continue
		}

		for _, c := range e.Commands {
			if c.Name != want.name {
				continue
			}

			row := Entry{Command: c.Name, Child: want.child, Args: args, Says: want.says}

			if want.child == "" {
				row.Title, row.Detail = c.Name, c.About
				out = append(out, row)

				continue
			}

			// The title is the child's word alone: the submenu already
			// names the family, and "task note" under "< task" reads
			// the parent twice. Command and Child still carry both
			// words, which is what running it asks for.
			row.Title = want.child
			row.Args = append([]string{want.child}, args...)

			for _, kid := range c.Children {
				if kid.Name == want.child {
					row.Detail, row.NeedsArgs = kid.About, kid.NeedsArgs
				}
			}

			out = append(out, row)
		}
	}

	return out
}

// detailEntries is the menu inside a task: the panes it can be shown in,
// and what can be done to it, under a line naming what each block is.
//
// The panes come first because they are what the keystroke that opens this
// menu has always led to, and a reader who learned the list in that order
// should not have to find it again underneath a block that grew above it.
func (s State) detailEntries(e Env) []Entry {
	out := []Entry{head(e.Words.T("menu.head_panes", "the panes of this task"))}

	for _, p := range e.Panes {
		out = append(out, Entry{
			Glyph:  "[" + p.Key + "]",
			Title:  p.Title,
			Detail: p.Detail,
			Pane:   p.Key,
		})
	}

	verbs := s.taskEntries(e)
	if len(verbs) == 0 {
		return out
	}

	out = append(out, gap(), head(e.Words.T("menu.head_verbs", "what can be done to it")))

	return append(out, verbs...)
}

// head is a line that names a block rather than offering anything. It is
// skipped by the cursor and answers nothing to a click, because a heading a
// reader can land on is a row that looks chosen and does nothing.
func head(title string) Entry { return Entry{Head: true, Title: title} }

// gap is the blank between two blocks. It is an entry rather than a line
// the drawing adds, because the menu is hit-tested by counting rows: a line
// drawn that no entry accounts for puts every click below it on the wrong
// verb.
func gap() Entry { return Entry{Head: true} }

// boardEntries is the commands that are not about one task: one row per
// family, and one row per command that belongs to none.
func boardEntries(e Env) []Entry {
	var out []Entry

	for _, c := range e.Commands {
		// A verb about one task is not on this menu. This one is opened on
		// no row, so there is no task for such a verb to be about:
		// choosing it ran it bare and got its usage back.
		if c.AboutATask {
			continue
		}

		if len(c.Children) > 0 {
			out = append(out, Entry{Title: c.Name, Detail: c.About, Family: c.Name})

			continue
		}

		row := Entry{Title: c.Name, Command: c.Name, Detail: c.About}
		if c.Refused && c.Because != "" {
			row.Dim = true
			row.Detail = ""
			row.Reason = c.Because
		}

		out = append(out, row)
	}

	return out
}

// taskEntries is what can be done to one task: one row per family.
// Choosing one drills into its verbs and commands; the keystrokes keep
// working the way they always have, menu up or not.
//
// It is the same list on the board's menu and inside a task, from the same
// call, because they are the same question asked from two places — and a
// verb offered in one and missing from the other would be read as a verb
// that does not apply here.
func (s State) taskEntries(e Env) []Entry {
	// A task that left the board has nothing on its menu: every verb on
	// it would be a verb about a run that is no longer there.
	if _, ok := e.verbs(s.task); !ok {
		return nil
	}

	var out []Entry

	var seen []string

	for _, want := range taskCommands {
		if slices.Contains(seen, want.name) {
			continue
		}

		seen = append(seen, want.name)

		// The task's own keys need no table behind them: choosing one
		// sends the keystroke, and the window answers it the way a
		// pressed key is answered. Every other family lists what the
		// table carries, so a row never drills into nothing.
		if want.name != "task" {
			supported := false

			for _, c := range e.Commands {
				if c.Name == want.name && len(c.Children) > 0 {
					supported = true
				}
			}

			if !supported {
				continue
			}
		}

		out = append(out, familyRow(e, want.name))
	}

	return out
}

// familyRow is one family on the menu: its name, what it does when the
// table says, and the submenu choosing it drills into.
func familyRow(e Env, name string) Entry {
	row := Entry{Title: name, Family: name}

	for _, c := range e.Commands {
		if c.Name == name {
			row.Detail = c.About
		}
	}

	return row
}

// verbs asks what can be done to a task, and answers no for a window built
// without that port rather than reaching through it.
func (e Env) verbs(id string) ([]keymap.Affordance, bool) {
	if e.Verbs == nil {
		return nil, false
	}

	return e.Verbs(id)
}

// start is the verb that is a screen rather than a command run bare.
// `orbit task start` starts a task with the flow it was written for; the window
// asks which flow first, and that question is the start dialog. So the
// entry sends the key that opens it, and there is one way to start a run
// rather than two that answer the flow question differently.
func (e Env) start() Entry {
	row := Entry{Glyph: e.Keys.Start.Help().Key, Title: e.Keys.Start.Help().Desc}

	if e.Says != nil {
		row.Detail = e.Says(e.Keys.Start)
	}

	return row
}

// saysSomething is a verb that takes a message, so the menu opens the box
// rather than running it.
type saysSomething struct {
	name  string
	child string
	says  bool
}

// taskCommands is the block of verbs that live only in the command table,
// on the menu of the task they are about, in the order a reader meets them:
// say something to the task, deliver what it did, answer what it asked.
//
// Everything the command line can do to a task, the window can do without
// it. Some of those verbs are keys, and a key is not a menu: the reader who
// does not already know it has nowhere to find out. The rest had nothing at
// all, because the command line is opened on the board, which is the one
// place in the window where there is no task for a verb about one to be
// about.
var taskCommands = []saysSomething{
	{name: "task", child: "note", says: true},
	{name: "task", child: "direct", says: true},
	{name: "pr"},
	{name: "pr", child: "show"},
	{name: "pr", child: "resolve"},
	{name: "pr", child: "merge"},
	{name: "pr", child: "close"},
	{name: "pr", child: "update"},
	{name: "pr", child: "checks"},
	{name: "pr", child: "tests"},
	{name: "pr", child: "review"},
	{name: "task", child: "approve"},
	{name: "task", child: "permit"},
	{name: "task", child: "critical"},
}

// args is what a command about the task is run with, and nothing for a
// window built without that port.
func (e Env) args(id string) []string {
	if e.Args == nil {
		return nil
	}

	return e.Args(id)
}

// choice is the entry the cursor may sit on nearest to from, walking in
// direction d and stopping at whichever end it reaches. Heads and gaps are
// passed over, and a list that is nothing but heads has no choice in it,
// which is what -1 says.
func choice(es []Entry, from, d int) int {
	for i := from; i >= 0 && i < len(es); i += d {
		if !es[i].Head {
			return i
		}
	}

	// The end of the list in that direction was a heading. The cursor
	// stays where it can be rather than parking on a line that does
	// nothing.
	for i := from; i >= 0 && i < len(es); i -= d {
		if !es[i].Head {
			return i
		}
	}

	return -1
}
