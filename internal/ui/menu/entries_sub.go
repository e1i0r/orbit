package menu

// One family drilled into: its key verbs, and its commands.
//
// Apart from entries.go rather than in it because that file met the
// size ceiling, and this is the seam the program already draws: the
// top names families, and everything here answers what is inside one.

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
