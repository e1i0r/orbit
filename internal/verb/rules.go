package verb

// The rules you said, waiting to be kept.
//
// The first family: a verb with children, asked for as two words. `orbit
// rules` is the tray, and `orbit rules keep 3` is one row of it — which
// reads as what it is, and which sorts together in the one place somebody
// goes looking for what can be asked for.

import "github.com/e1i0r/orbit/internal/words"

// rules is the family, parent first.
func rules() []Verb {
	return []Verb{
		{
			Name: "rules", Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.rules", "everything about your rules that is waiting for an answer")
			},
			Takes: []Field{
				{Name: "state", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.state",
						"show the rules standing here instead: active, paused, off, or review")
				}},
			},
		},
		{
			Name: "keep", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.keep", "keep one of them, in your words or in better ones")
			},
			Takes: []Field{
				{Name: "n", Kind: Named, Needed: true, Or: "at", About: func(p *words.Printer) string {
					return p.T("verb.rules.n", "which one, by its number in the list")
				}},
				{Name: "at", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.at",
						"which one, by when it was said; for a caller with no numbered list in front of it")
				}},
				{Name: "text", Kind: Words, About: func(p *words.Printer) string {
					return p.T("verb.rules.text", "the rule as you would rather it read; the default is what you said")
				}},
				{Name: "check", Kind: Words, About: func(p *words.Printer) string {
					return p.T("verb.rules.check",
						"a command that fails when the rule is broken, which is what makes it refuse work")
				}},
				{Name: "in", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.in",
						"the folder or file it is about; the default is where the work was, and . is the whole checkout")
				}},
				// Declared rather than left to the caller's own, because a
				// sentence said to the supervisor knows no repository and a
				// browser asking about the board carries none either. What
				// the rule is about is where the reader is, and only the
				// reader can say where that is.
				{Name: "repo", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.repo",
						"the checkout the folder or file is in; the default is the one you are in")
				}},
			},
		},
		{
			Name: "repeated", Under: "rules", Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.rules.repeated",
					"what you keep telling runs, that nobody ever wrote down as a rule")
			},
		},
		{
			Name: "enforced", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.enforced",
					"offer a rule for each thing this checkout already refuses work over")
			},
			Takes: []Field{
				{Name: "repo", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.enforced.repo",
						"the checkout to read; the default is the one you are in")
				}},
			},
		},
		{
			Name: "read", Under: "rules", Spends: true,
			About: func(p *words.Printer) string {
				return p.T("verb.rules.read",
					"have a model read what this project already says about itself, and offer the rules in it")
			},
			Takes: []Field{
				{Name: "repo", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.read.repo",
						"the checkout to read; the default is the one you are in")
				}},
				// `with` and not `engine`, because what this reads is the
				// files each engine keeps — and a rule is Orbit's and every
				// engine is told it, which is the whole point. Named
				// `engine`, the flag read as though the rule were for one.
				{Name: "with", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.read.with",
						"which engine does the reading; the default is the one in your settings")
				}},
			},
		},
		{
			Name: "draft", Under: "rules", Spends: true,
			About: func(p *words.Printer) string {
				return p.T("verb.rules.draft",
					"have a model read what you keep telling runs and write the rule it amounts to")
			},
			Takes: []Field{
				{Name: "with", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.draft.with",
						"which engine does the reading; the default is the one in your settings")
				}},
			},
		},
		{
			Name: "pause", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.pause",
					"stop a rule applying for now, and say why; it goes to be looked at again")
			},
			Takes: []Field{
				{Name: "rule", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.rules.pause.rule", "the rule's name, as orbit knowledge prints it")
				}},
				{Name: "why", Kind: Words, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.rules.pause.why",
						"what you are pausing it for; it is what you will read when you come back")
				}},
				{Name: "task", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.pause.task",
						"the task it was in your way at, if it was; it says where to look later")
				}},
			},
		},
		{
			Name: "resume", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.resume", "have a rule apply again, and stop asking about it")
			},
			Takes: []Field{
				{Name: "rule", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.rules.resume.rule", "the rule's name, as orbit knowledge prints it")
				}},
			},
		},
		{
			Name: "review", Under: "rules", Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.rules.review",
					"everything one rule has put you through, so you can decide about it")
			},
			Takes: []Field{
				{Name: "rule", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.rules.review.rule", "the rule's name, as orbit knowledge prints it")
				}},
			},
		},
		{
			Name: "correct", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.correct",
					"say a rule better, or narrow it to where it was actually true")
			},
			Takes: []Field{
				{Name: "rule", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.rules.correct.rule", "the rule's name, as orbit knowledge prints it")
				}},
				{Name: "in", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.correct.in",
						"the folder or file it is really about, and . for the whole checkout")
				}},
				{Name: "check", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.correct.check",
						"the command that makes it stop the work; empty takes the command away")
				}},
				{Name: "text", Kind: Words, About: func(p *words.Printer) string {
					return p.T("verb.rules.correct.text", "the rule as you would rather it read")
				}},
			},
		},
		{
			Name: "off", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.off",
					"decide against a rule; it stays where it is and nothing is told it")
			},
			Takes: []Field{
				{Name: "rule", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.rules.off.rule", "the rule's name, as orbit knowledge prints it")
				}},
			},
		},
		{
			Name: "history", Under: "rules", Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.rules.history", "what has happened to one rule since it was kept")
			},
			Takes: []Field{
				{Name: "rule", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.rules.history.rule",
						"the rule's name, as orbit knowledge prints it")
				}},
			},
		},
		{
			Name: "drop", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.drop", "say it was not a rule; the sentence stays where you said it")
			},
			Takes: []Field{
				{Name: "n", Kind: Named, Needed: true, Or: "at", About: func(p *words.Printer) string {
					return p.T("verb.rules.drop.n", "which one, by its number in the list")
				}},
				{Name: "at", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.rules.drop.at", "which one, by when it was said")
				}},
			},
		},
	}
}
