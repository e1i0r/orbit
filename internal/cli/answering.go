package cli

// The two verbs a reader reaches for while a pull request is open.
//
// They are a file of their own because the table met the size ceiling, and
// they are the pair that belongs together anywhere: one brings back what a
// reviewer asked for, the other says yes to what the task asked for. Both
// are answers to a question somebody else put.

import "github.com/e1i0r/orbit/internal/words"

// answering is the tail of the table: what a reader does about a task that
// is waiting on them.
func answering() []Command {
	return []Command{{
		Name: "resolve", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.resolve", "read what reviewers asked on the pull requests into the task, for the next run to answer")
		},
		Run: resolveComments,
	}, {
		Name: "approve", Args: "-repo <dir> <id>", NeedsArgs: true, AboutATask: true,
		About: func(p *words.Printer) string {
			return p.T("cmd.approve", "say yes to the libraries a task added, so its next run goes past the gate")
		},
		Run: approveTask,
	}}
}
