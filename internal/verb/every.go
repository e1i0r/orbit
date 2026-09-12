package verb

// Every verb, in the order a reader meets them: starting work, steering the
// run that results, saying something about it, answering what it asked, and
// handing what it made to the world.

import (
	"slices"

	"github.com/e1i0r/orbit/internal/words"
)

// Every is the whole vocabulary.
//
// A verb added here and nowhere else fails the build, which is the point:
// the list is what the four ways in are checked against, and a new action
// that reached only one of them was the thing this package was written to
// stop.
func Every() []Verb {
	return slices.Concat(theRest(), rules(), pr(), settings())
}

// theRest is every verb that belongs to no family.
//
// A family lives in a file of its own, named after it — rules.go, pr.go, and
// the settings beside the table of what they are — so that what a family is
// can be read in one sitting. This is what is left over, and it shrinks
// every time another one is named.
func theRest() []Verb {
	return []Verb{
		{
			Name:   "new",
			OnTask: false,
			About: func(p *words.Printer) string {
				return p.T("verb.new", "write a task down on the board")
			},
			Takes: []Field{
				{Name: "id", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.new.id", "how you and Orbit will both refer to it")
				}},
				{Name: "text", Kind: Words, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.new.text", "what the work is, written for whoever does it")
				}},
				{Name: "repo", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.new.repo", "which checkout it is against, if it is against one yet")
				}},
				{Name: "flow", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.new.flow", "the shape of work it walks; the default is the one you set")
				}},
				{Name: "run", Kind: YesOrNo, About: func(p *words.Printer) string {
					return p.T("verb.new.run", "start it as soon as it is written")
				}},
			},
		},
		{
			Name: "run", OnTask: true, Spends: true,
			About: func(p *words.Printer) string {
				return p.T("verb.run", "run a task that is not running")
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
		{Name: "pause", OnTask: true, About: func(p *words.Printer) string {
			return p.T("verb.pause", "ask a run to stop at its next phase boundary")
		}},
		{Name: "resume", OnTask: true, About: func(p *words.Printer) string {
			return p.T("verb.resume", "let a run carry on from a pause")
		}},
		{Name: "continue", OnTask: true, Spends: true, About: func(p *words.Printer) string {
			return p.T("verb.continue", "let a phase past the gate its flow stopped it at")
		}},
		{Name: "skip", OnTask: true, About: func(p *words.Printer) string {
			return p.T("verb.skip", "let a run past the phase it is in, without running it")
		}},
		{Name: "cancel", OnTask: true, About: func(p *words.Printer) string {
			return p.T("verb.cancel", "stop a run where it stands")
		}},
		{
			Name: "requeue", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.requeue", "stop whatever holds a task and put it back in to do")
			},
			Takes: []Field{{Name: "why", Kind: Words, About: func(p *words.Printer) string {
				return p.T("verb.requeue.why", "the reason, if you want it on the record")
			}}},
		},
		{
			Name: "note", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.note", "leave a word for the phase that starts next")
			},
			Takes: []Field{{Name: "text", Kind: Words, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.note.text", "what you want it to know")
			}}},
		},
		{
			Name: "direct", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.direct", "correct a task, stopping the run so the next one reads it")
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
		{Name: "approve", OnTask: true, About: func(p *words.Printer) string {
			return p.T("verb.approve", "say yes to the libraries a task added")
		}},
		{Name: "read", OnTask: true, About: func(p *words.Printer) string {
			return p.T("verb.read", "mark a finished task as looked at")
		}},
		{Name: "history", OnTask: true, Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.history", "everything ever said about a task, in any program")
		}},
		{
			Name: "say",
			About: func(p *words.Printer) string {
				return p.T("verb.say", "say something in the supervisor thread")
			},
			Takes: []Field{{Name: "text", Kind: Words, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.say.text", "what to say")
			}}},
		},
		{
			Name: "join", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.join", "open a checkout of another repository for a task")
			},
			Takes: []Field{{Name: "name", Kind: Named, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.join.name", "the repository to join, by name")
			}}},
		},
		{
			Name: "permit", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.permit", "answer the question a critical action stopped for")
			},
			Takes: []Field{{Name: "yes", Kind: YesOrNo, About: func(p *words.Printer) string {
				return p.T("verb.permit.yes", "whether to let it happen")
			}}},
		},
		{
			Name: "critical", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.critical", "mark a task as one whose changes need answering for")
			},
			Takes: []Field{{Name: "on", Kind: YesOrNo, About: func(p *words.Printer) string {
				return p.T("verb.critical.on", "whether it is critical")
			}}},
		},
		{
			Name: "reconcile",
			About: func(p *words.Printer) string {
				return p.T("verb.reconcile", "close the records of runs whose processes are gone")
			},
			Takes: []Field{{Name: "task", Kind: Named, About: func(p *words.Printer) string {
				return p.T("verb.reconcile.task", "just this one, rather than every task here")
			}}},
		},
		{Name: "delete", OnTask: true, About: func(p *words.Printer) string {
			return p.T("verb.delete", "remove a task and everything written about it")
		}},
		{Name: "take", OnTask: true, About: func(p *words.Printer) string {
			return p.T("verb.take", "hand a terminal to an engine, in the task's own checkout")
		}},
		{
			Name: "export",
			About: func(p *words.Printer) string {
				return p.T("verb.export", "write the record out as JSON lines")
			},
			Takes: []Field{{Name: "into", Kind: Named, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.export.into", "a directory that is empty or does not exist yet")
			}}},
		},
		{Name: "quota", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.quota", "what is left of each engine's windows")
		}},
		{Name: "list", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.list", "every task under this root, in the band that says what it waits for")
		}},
		{Name: "show", OnTask: true, Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.show", "everything the record says about one task")
		}},
		{Name: "flow", OnTask: true, Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.flow", "the flow a task walks, and how far its run got through it")
		}},
		{Name: "diff", OnTask: true, Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.diff", "what a task changed in its worktree")
		}},
		{
			Name: "compare", OnTask: true,
			About: func(p *words.Printer) string {
				return p.T("verb.compare",
					"run the flow's checks on both sides of a task's change, and say what differs")
			},
		},
		{Name: "tree", OnTask: true, Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.tree", "the repository as a tree, with what a task changed marked on it")
		}},
		{Name: "impact", OnTask: true, Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.impact", "what a change reaches beyond the files it touched")
		}},
		{Name: "knowledge", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.knowledge", "everything Orbit has been told about this code")
		}},
		{Name: "flows", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.flows", "every flow a task can be started under")
		}},
		{Name: "engines", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.engines", "what this build can run, and what this machine has installed")
		}},
		{Name: "repos", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.repos", "the checkouts Orbit is watching, and the work in each")
		}},
		{Name: "thread", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.thread", "everything said in the supervisor thread")
		}},
		{
			Name: "retract",
			About: func(p *words.Printer) string {
				return p.T("verb.retract", "take back a line of the supervisor thread")
			},
			Takes: []Field{{Name: "line", Kind: Named, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.retract.line", "which line, as `orbit thread` numbers them")
			}}},
		},
		{
			Name: "learn",
			About: func(p *words.Printer) string {
				return p.T("verb.learn", "write down something true about this code")
			},
			Takes: []Field{
				{Name: "text", Kind: Words, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.learn.text", "the fact, in a sentence")
				}},
				{Name: "repo", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.learn.repo", "which checkout it is about; the default is the one you are in")
				}},
			},
		},
	}
}

// One is the verb by that name, and whether there is one.
//
// By Path, so that a child is asked for the way it is written: "rules keep"
// and not "keep", which two families could both answer to.
func One(name string) (Verb, bool) {
	for _, v := range Every() {
		if v.Path() == name {
			return v, true
		}
	}

	return Verb{}, false
}
