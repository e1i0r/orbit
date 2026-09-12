package verb

// The task, and everything one of them can be asked for.
//
// The biggest family: starting work, steering the run that results, saying
// something about it, answering what it asked, and reading what it made.
// The parent keeps its own body — `orbit task <id>` shows the task, the
// way `orbit show <id>` always has.
import "github.com/e1i0r/orbit/internal/words"

// taskFamily is the family, parent first.
func taskFamily() []Verb {
	return []Verb{
		{
			Name: "task", OnTask: true, Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.task", "everything one task can be asked for")
			},
		},
		{
			Under: "task",
			Name:  "start", OnTask: true, Spends: true,
			About: func(p *words.Printer) string {
				return p.T("verb.start", "run the task through its flow")
			},
			Takes: []Field{
				{Name: "flow", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.run.flow", "walk this flow instead of the one it was written against")
				}},
				{Name: "engine", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.run.engine",
						"walk every phase with this engine instead of the ones the flow names")
				}},
			},
		},
		{
			Under: "task",
			Name:  "pause", OnTask: true, About: func(p *words.Printer) string {
				return p.T("verb.pause", "pause the run at its next phase boundary")
			},
		},
		{
			Under: "task",
			Name:  "resume", OnTask: true, About: func(p *words.Printer) string {
				return p.T("verb.resume", "carry on from the pause")
			},
		},
		{
			Under: "task",
			Name:  "continue", OnTask: true, Spends: true, About: func(p *words.Printer) string {
				return p.T("verb.continue", "let the run past the gate it stopped at")
			},
		},
		{
			Under: "task",
			Name:  "skip", OnTask: true, About: func(p *words.Printer) string {
				return p.T("verb.skip", "skip the waiting phase without running it")
			},
		},
		{
			Under: "task",
			Name:  "cancel", OnTask: true, About: func(p *words.Printer) string {
				return p.T("verb.cancel", "stop the run now; what it wrote stays")
			},
		},
		{
			Under: "task",
			Name:  "requeue", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.requeue", "send the task back to to do")
			},
			Takes: []Field{{Name: "why", Kind: Words, About: func(p *words.Printer) string {
				return p.T("verb.requeue.why", "the reason, if you want it on the record")
			}}},
		},
		{
			Under: "task",
			Name:  "note", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.note", "leave a note the next phase reads — the run keeps going")
			},
			Takes: []Field{{Name: "text", Kind: Words, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.note.text", "what you want it to know")
			}}},
		},
		{
			Under: "task",
			Name:  "direct", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.direct", "correct the task and stop the run, so the next run starts corrected")
			},
			Takes: []Field{
				{Name: "text", Kind: Words, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.direct.text", "the correction")
				}},
				{Name: "restart", Kind: YesOrNo, About: func(p *words.Printer) string {
					return p.T("verb.direct.restart", "start the next run now, which spends money")
				}},
			},
		},
		{
			Under: "task",
			Name:  "approve", OnTask: true, About: func(p *words.Printer) string {
				return p.T("verb.approve", "accept the added libraries, so the run goes past the gate")
			},
		},
		{
			Under: "task",
			Name:  "read", OnTask: true, About: func(p *words.Printer) string {
				return p.T("verb.read", "mark a finished task as looked at")
			},
		},
		{
			Under: "task",
			Name:  "history", OnTask: true, Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.history", "everything ever said about a task, in any program")
			},
		},
		{
			Under: "task",
			Name:  "join", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.join", "open another repository's checkout for this task")
			},
			Takes: []Field{{Name: "name", Kind: Named, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.join.name", "the repository to join, by name")
			}}},
		},
		{
			Under: "task",
			Name:  "permit", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.permit", "let the stopped critical action happen — or refuse it")
			},
			Takes: []Field{{Name: "yes", Kind: YesOrNo, About: func(p *words.Printer) string {
				return p.T("verb.permit.yes", "whether to let it happen")
			}}},
		},
		{
			Under: "task",
			Name:  "critical", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.critical", "mark the task critical, so it stops before anything irreversible")
			},
			Takes: []Field{{Name: "on", Kind: YesOrNo, About: func(p *words.Printer) string {
				return p.T("verb.critical.on", "whether it is critical")
			}}},
		},
		{
			Under: "task",
			Name:  "delete", OnTask: true, About: func(p *words.Printer) string {
				return p.T("verb.delete", "remove a task and everything written about it")
			},
		},
		{
			Under: "task",
			Name:  "take", OnTask: true, About: func(p *words.Printer) string {
				return p.T("verb.take", "hand a terminal to an engine, in the task's own checkout")
			},
		},
		{
			Under: "task",
			Name:  "show", OnTask: true, Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.show", "everything the record says about one task")
			},
		},
		{
			Under: "task",
			Name:  "flow", OnTask: true, Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.flow", "the flow a task walks, and how far its run got through it")
			},
		},
		{
			Under: "task",
			Name:  "diff", OnTask: true, Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.diff", "what a task changed in its worktree")
			},
		},
		{
			Under: "task",
			Name:  "compare", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.compare",
					"run the flow's checks on both sides of a task's change, and say what differs")
			},
		},
		{
			Under: "task",
			Name:  "tree", OnTask: true, Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.tree", "the repository as a tree, with what a task changed marked on it")
			},
		},
		{
			Under: "task",
			Name:  "impact", OnTask: true, Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.impact", "what a change reaches beyond the files it touched")
			},
		},
	}
}
