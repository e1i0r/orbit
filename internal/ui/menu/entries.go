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

import "github.com/e1i0r/orbit/internal/ui/keymap"

// entries is the menu as it stands, one of three lists.
func (s State) entries(e Env) []Entry {
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

// boardEntries is the commands that are not about one task.
func boardEntries(e Env) []Entry {
	out := make([]Entry, 0, len(e.Commands))

	for _, c := range e.Commands {
		// A verb about one task is not on this menu. This one is opened on
		// no row, so there is no task for such a verb to be about:
		// choosing it ran it bare and got its usage back.
		if c.AboutATask {
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

// taskEntries is what can be done to one task, refusals included and each
// with its reason.
//
// It is the same list on the board's menu and inside a task, from the same
// call, because they are the same question asked from two places — and a
// verb offered in one and missing from the other would be read as a verb
// that does not apply here.
func (s State) taskEntries(e Env) []Entry {
	all, ok := e.verbs(s.task)
	if !ok {
		return nil
	}

	out := make([]Entry, 0, len(all))

	for i := range all {
		a := &all[i]

		row := Entry{Glyph: a.Key.Help().Key, Title: a.Key.Help().Desc}
		if !a.OK {
			row.Dim = true
			row.Reason = a.Why(e.Words)
		}

		out = append(out, row)
	}

	// And what only a command or a dialog can do to it. They are in the
	// same block rather than under a heading of their own: a reader asking
	// what can be done to a task is not asking which of the answers is a
	// key.
	out = append(out, e.start())

	return append(out, s.commandEntries(e)...)
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
// `orbit run` starts a task with the flow it was written for; the window
// asks which flow first, and that question is the start dialog. So the
// entry sends the key that opens it, and there is one way to start a run
// rather than two that answer the flow question differently.
func (e Env) start() Entry {
	return Entry{Glyph: e.Keys.Start.Help().Key, Title: e.Keys.Start.Help().Desc}
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
	{name: "note", says: true},
	{name: "direct", says: true},
	{name: "pr"},
	{name: "resolve"},
	{name: "pr", child: "merge"},
	{name: "pr", child: "close"},
	{name: "approve"},
	{name: "permit"},
	{name: "critical"},
}

// commandEntries is those commands as menu rows about one task, with the
// repository and the id already filled in.
//
// Whether the task is in a state to be approved, merged or redirected is
// not asked here. The command asks it and answers in the watch, in its own
// words; a second opinion in the menu is a second place for that rule to
// live, and the two would drift.
func (s State) commandEntries(e Env) []Entry {
	out := make([]Entry, 0, len(taskCommands))

	for _, want := range taskCommands {
		for _, c := range e.Commands {
			if c.Name != want.name {
				continue
			}

			title, detail, args := c.Name, c.About, e.args(s.task)
			if want.child != "" {
				title = c.Name + " " + want.child
				args = append([]string{want.child}, args...)

				for _, kid := range c.Children {
					if kid.Name == want.child {
						detail = kid.About
					}
				}
			}

			out = append(out, Entry{
				Title:   title,
				Detail:  detail,
				Command: c.Name,
				Args:    args,
				Says:    want.says,
			})
		}
	}

	return out
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
