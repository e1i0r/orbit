package verb

// The board, and what is about no task in particular.
//
// Writing a task down, listing them, and closing what their runs left
// behind. The parent keeps its own body: `orbit board` lists, the way
// `orbit list` always has.
import "github.com/e1i0r/orbit/internal/words"

// boardFamily is the family, parent first.
func boardFamily() []Verb {
	return []Verb{
		{
			Name: "board", Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.board", "everything about the board rather than about one task")
			},
		},
		{
			Under:  "board",
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
			Under: "board",
			Name:  "list", Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.list", "every task written down, in the band that says what it waits for")
			},
			Takes: []Field{{Name: "repo", Kind: Named, About: func(p *words.Printer) string {
				return p.T("verb.list.repo", "only the tasks worked in this checkout")
			}}},
		},
		{
			Under: "board",
			Name:  "reconcile",
			About: func(p *words.Printer) string {
				return p.T("verb.reconcile", "close the records of runs whose processes are gone")
			},
			Takes: []Field{{Name: "task", Kind: Named, About: func(p *words.Printer) string {
				return p.T("verb.reconcile.task", "just this one, rather than every task here")
			}}},
		},
	}
}
