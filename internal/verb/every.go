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
	return slices.Concat(
		theRest(), rules(), pr(), settings(),
		taskFamily(), boardFamily(), supervisorFamily(), knowledgeFamily(),
	)
}

// theRest is every verb that belongs to no family.
//
// Five are left: the record out, the engines and their windows, the
// checkouts, and the flows. Everything else found its family.
func theRest() []Verb {
	return []Verb{
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
		{Name: "flows", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.flows", "every flow a task can be started under")
		}},
		{Name: "engines", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.engines", "what this build can run, and what this machine has installed")
		}},
		{Name: "repos", Reads: true, About: func(p *words.Printer) string {
			return p.T("verb.repos", "the checkouts Orbit is watching, and the work in each")
		}},
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
