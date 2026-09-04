package ui

// The verbs about a task that live only in the command table, on the menu of
// the task they are about.
//
// Everything the command line can do to a task, the window can do without
// it. Some of those verbs are keys — pause, resume, skip, cancel and requeue
// through the control port, read through mark-read, show by opening the
// task, pr, merge, close-pr and resolve by their letters on the task's
// screen — and a key is not a menu: the reader who does not already know it
// has nowhere to find out. The rest had nothing at all, because the command
// line is opened on the board, which is the one place in the window where
// there is no task for a verb about one to be about.
//
// Here they all are on the menu of the task itself, with the repository and
// the id already filled in.

// taskMenuCommand is one such verb: the name it has in the table, and
// whether running it needs something typed first.
type taskMenuCommand struct {
	name string
	says bool // takes a message, so the menu opens the box rather than running it
}

// taskMenuCommands is the block, in the order a reader meets them: say
// something to the task, deliver what it did, answer what it asked.
var taskMenuCommands = []taskMenuCommand{
	{name: verbNote, says: true},
	{name: verbDirect, says: true},
	{name: "pr"},
	{name: "resolve"},
	{name: "merge"},
	{name: "close-pr"},
	{name: "approve"},
	{name: "permit"},
	{name: "critical"},
}

// startEntry is the verb that is a screen rather than a command run bare.
// `orbit run` starts a task with the flow it was written for; the window
// asks which flow first, and that question is the start dialog. So the entry
// sends the key that opens it, and there is one way to start a run rather
// than two that answer the flow question differently.
func (m Model) startEntry() menuEntry {
	return menuEntry{
		glyph: m.keys.Start.Help().Key,
		title: m.keys.Start.Help().Desc,
	}
}

// taskCommandEntries is those commands as menu rows about one task.
//
// Whether the task is in a state to be approved, merged or redirected is not
// asked here. The command asks it and answers in the watch, in its own
// words; a second opinion in the menu is a second place for that rule to
// live, and the two would drift.
func (m Model) taskCommandEntries(id string) []menuEntry {
	p := m.opts.Words
	args := repoArgs(m.taskRepoPath(id), id)

	out := make([]menuEntry, 0, len(taskMenuCommands))

	for _, want := range taskMenuCommands {
		for i := range m.opts.Commands {
			c := &m.opts.Commands[i]
			if c.Name != want.name {
				continue
			}

			e := menuEntry{title: c.Name, cmd: c, args: args, says: want.says}
			if c.About != nil {
				e.detail = c.About(p)
			}

			out = append(out, e)
		}
	}

	return out
}
