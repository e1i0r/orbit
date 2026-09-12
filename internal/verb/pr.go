package verb

// The pull request, and what becomes of it.
//
// One family: opening it is a command, merging and closing stay commands,
// and everything that takes finding out first goes to the supervisor — the
// same words from every door, because the thread they land in is one
// conversation.
//
// The parent keeps its own body. `orbit pr <id>` opens the pull request the
// way it always has, because that is what it has meant for as long as Orbit
// has existed and it is what is written in people's scripts.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/supervisor"
	"github.com/e1i0r/orbit/internal/words"
)

// pr is the family, parent first.
func pr() []Verb {
	return []Verb{
		{
			Name: "pr", OnTask: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.pr", "open the task's pull request")
			},
		},
		{
			Name: "show", Under: "pr", OnTask: true, Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.pr.show", "list the task's pull requests and what became of each")
			},
		},
		{
			Name: "merge", Under: "pr",
			OnTask: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.merge", "merge the pull request and delete its branch")
			},
		},
		{
			Name: "close", Under: "pr",
			OnTask: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.close_pr", "close the pull request without merging")
			},
		},
		{
			Name: "resolve", Under: "pr",
			OnTask: true, Spends: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.pr.resolve", "answer the review threads on the pull request")
			},
		},
		{
			Name: "update", Under: "pr",
			OnTask: true, Spends: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.pr.update", "merge the base branch into the task's branch")
			},
		},
		{
			Name: "checks", Under: "pr",
			OnTask: true, Spends: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.pr.checks", "make the pull request's checks pass")
			},
		},
		{
			Name: "tests", Under: "pr",
			OnTask: true, Spends: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.pr.tests", "add the tests the change is missing")
			},
		},
		{
			Name: "review", Under: "pr",
			OnTask: true, Spends: true, Outward: true,
			About: func(p *words.Printer) string {
				return p.T("verb.pr.review", "leave a senior-style review on the pull request")
			},
		},
	}
}

// erranded passes one of the deliver verbs to the supervisor: the surfaces say
// what is wanted and where, and the supervisor finds out what that takes
// before doing it.
//
// It records and does not answer. What the supervisor makes of it is its
// own loop's, and a verb that quietly called a model would be one that
// spent money without saying so — which is why these say Spends.
func erranded(w World, in In, caption, body string) (Out, error) {
	t, dir, err := checkout(w, in)
	if err != nil {
		return Out{}, err
	}

	if dir == "" {
		return Out{}, errNoCheckout(w, t.ID)
	}

	text := supervisor.Deliver(in.Door, caption, t.ID, dir, body)

	if err := w.Say(text, in.who(), t.ID); err != nil {
		return Out{}, err
	}

	return Out{Said: askedSaid(w, caption, t.ID)}, nil
}

// errNoCheckout is a task with nowhere to work: asked from a door with
// no checkout behind it, where the window would have refused the key.
func errNoCheckout(w World, id string) error {
	return errors.New(w.Words().T("deliver.no_checkout", "{id} has no checkout, so there is nothing to work in",
		words.Arg{Name: "id", Value: id}))
}

// shownPR is every pull request opened for a task, in every repository it
// was worked in, and what became of each one.
func shownPR(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	prs, err := w.Store().PullRequests(t.ID)
	if err != nil {
		return Out{}, err
	}

	if len(prs) == 0 {
		return Out{Said: w.Words().T("verb.pr.show.empty", "{id} has no pull request open anywhere",
			words.Arg{Name: "id", Value: t.ID})}, nil
	}

	var b strings.Builder

	for _, one := range prs {
		fmt.Fprintf(&b, "%s  %s  %s\n", one.Repo, one.State, one.URL)
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: prs}, nil
}

// askedSaid is what the band says once the ask is out, in the words the
// window has always said it in: one conversation in one order means one
// sentence for it, whichever door it came through.
func askedSaid(w World, caption, id string) string {
	p := w.Words()

	switch caption {
	case "CREATE PR":
		return p.T("deliver.pr_asked", "the supervisor was asked to open the pull request for {id}",
			words.Arg{Name: "id", Value: id})
	case "UPDATE PR":
		return p.T("deliver.update_asked",
			"the supervisor was asked to bring {id} up to date with its base branch",
			words.Arg{Name: "id", Value: id})
	case "FIX CHECKS":
		return p.T("deliver.checks_asked", "the supervisor was asked to make {id}'s checks pass",
			words.Arg{Name: "id", Value: id})
	case "MORE TESTS":
		return p.T("deliver.tests_asked", "the supervisor was asked for more tests on {id}",
			words.Arg{Name: "id", Value: id})
	case "RESOLVE COMMENTS":
		return p.T("deliver.resolve_asked", "the supervisor was asked to answer the reviews on {id}",
			words.Arg{Name: "id", Value: id})
	default:
		return p.T("deliver.review_asked", "the supervisor was asked to review {id}",
			words.Arg{Name: "id", Value: id})
	}
}
