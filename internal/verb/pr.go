package verb

// The pull request, and what becomes of it.
//
// Three things about one pull request, and they used to be three unrelated
// words: pr, merge, close-pr. Nobody reading that list learned that the
// second and third are what you do to what the first made — and close-pr was
// the only name in the whole vocabulary with a dash in it, which is what
// happens when a flat namespace runs out of words.
//
// The parent keeps its own body. `orbit pr <id>` opens the pull request the
// way it always has, because that is what it has meant for as long as Orbit
// has existed and it is what is written in people's scripts.

import "github.com/e1i0r/orbit/internal/words"

// pr is the family, parent first.
func pr() []Verb {
	return []Verb{
		{
			Name: "pr", OnTask: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.pr", "open a pull request from a task's worktree")
			},
		},
		{
			Name: "merge", Under: "pr",
			OnTask: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.merge", "merge a task's pull request and delete its branch")
			},
		},
		{
			Name: "close", Under: "pr",
			OnTask: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.close_pr", "close a task's pull request without merging it")
			},
		},
	}
}
