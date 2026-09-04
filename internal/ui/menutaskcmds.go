package ui

// The three commands about a task that no key in this window offers.
//
// Every other verb about a task is reached some other way: pause, resume,
// skip, cancel and requeue go through the control port, read through
// mark-read, show by opening the task, pr, merge, close-pr and resolve by
// their letters on the task's screen, and note by the note line. These
// three were reached only from the command line, which is opened on the
// board — the one place in the window where there is no task for them to be
// about. So they were listed where they could not be run, and nowhere else.
//
// Here they are on the menu of the task itself, with the repository and the
// id already filled in, which is the whole of what they take.

// taskOnlyByCommand is those three, in the order the table has them.
//
// It is a list and not a rule because the rule has an exception the list
// does not: `direct` is also about a task and also has no key, and it takes
// a message as well — there is nothing for a menu to fill that in with, so
// it stays off until the window has somewhere to type it.
var taskOnlyByCommand = []string{"approve", "permit", "critical"}

// taskCommandEntries is those commands as menu rows about one task.
//
// Whether the task is in a state to be approved, permitted or marked is not
// asked here. The command asks it and answers in the watch, in its own
// words; a second opinion in the menu is a second place for that rule to
// live, and the two would drift.
func (m Model) taskCommandEntries(id string) []menuEntry {
	p := m.opts.Words
	args := repoArgs(m.taskRepoPath(id), id)

	out := make([]menuEntry, 0, len(taskOnlyByCommand))

	for _, name := range taskOnlyByCommand {
		for i := range m.opts.Commands {
			c := &m.opts.Commands[i]
			if c.Name != name {
				continue
			}

			e := menuEntry{title: c.Name, cmd: c, args: args}
			if c.About != nil {
				e.detail = c.About(p)
			}

			out = append(out, e)
		}
	}

	return out
}
