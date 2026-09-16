package verb

// What the history says this project does, offered as rules.
//
// The documents say what somebody wanted. The commits say what the team kept
// doing. If the CONTRIBUTING says every change comes with tests and half of
// last year's commits brought none, that is not a rule — it is something
// somebody wrote once and nobody held to, and offering it starts lying to the
// agent.
//
// The other way round too, and it is the better half: there are things nobody
// ever wrote down that the repository does without fail. That is knowledge in
// no document at all.
//
// It reads git and spends nothing. Every offer carries the count that backs
// it, which is what lets somebody say yes without going to look.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/words"
)

// fromTheHistory offers what the commits say about how this project is
// worked on.
//
// A repository with less history than a reading needs offers nothing, and
// that is an answer: there is not enough yet, which is different from having
// looked and found none.
func fromTheHistory(
	w World, here repo.Repo, customs []repo.Custom, commits int,
) ([]learn.Said, error) {
	d := w.Store()
	at := time.Now().UTC()

	var out []learn.Said

	for _, one := range customs {
		if !one.Holds() {
			continue
		}

		said := learn.Said{
			At: at, Text: asARule(w.Words(), one), By: learn.FromTheHistory,
			Repo: here.Path, Path: alongsideIt(one), Topic: topicOf(one),
			About: w.Words().P("verb.rules.read.commits", commits,
				"the last commit", "the last {n} commits"),
			Habit: learn.FromTheHistory + ":" + one.Kind + ":" + one.Where,
		}

		already, err := learn.Answered(d, said.Habit)
		if err != nil {
			return out, err
		}

		if already {
			continue
		}

		if err := learn.Propose(d, said); err != nil {
			return out, err
		}

		out = append(out, said)
		at = at.Add(time.Nanosecond)
	}

	return out, nil
}

// whatItActuallyDoes is every reading, held or not, in one sentence each, for
// the model that is about to read what the project says about itself.
//
// Both halves on purpose. A measurement that held backs a written rule; one
// that was taken and did not hold is what says a written rule is no longer
// true — and that is the thing a document can never tell you about itself.
func whatItActuallyDoes(p *words.Printer, customs []repo.Custom) []string {
	out := make([]string, 0, len(customs))
	for _, one := range customs {
		out = append(out, asARule(p, one))
	}

	return out
}

// asARule is one reading of the history, in the words somebody would say it
// in, with the count that backs it inside the sentence.
//
// The count is in the rule and not beside it, because the rule is what the
// agent reads: "changes under internal/db come with a test, as 37 of the
// last 40 did" tells it both what to do and that the project means it.
func asARule(p *words.Printer, one repo.Custom) string {
	held := words.Arg{Name: "times", Value: strconv.Itoa(one.Times)}
	of := words.Arg{Name: "of", Value: strconv.Itoa(one.Of)}

	if one.Kind == repo.MessagesKeepAShape {
		return p.T("verb.rules.read.shape",
			"commit messages here are written with {shape}, as {times} of the last {of} were",
			words.Arg{Name: "shape", Value: one.Where}, held, of)
	}

	return p.T("verb.rules.read.tests",
		"a change under {where} comes with a change to a test, as {times} of the last {of} did",
		words.Arg{Name: "where", Value: one.Where}, held, of)
}

// topicOf is the kind of thing one reading is about, out of the same list
// everything else chooses from.
func topicOf(one repo.Custom) string {
	if one.Kind == repo.MessagesKeepAShape {
		return "process"
	}

	return "testing"
}

// alongsideIt is where a reading applies: the folder it is about, and the
// whole checkout for one about how the project is worked on rather than about
// any part of it.
func alongsideIt(one repo.Custom) string {
	if one.Kind == repo.TestsTravel {
		return one.Where
	}

	return ""
}

// coldRows is the two halves of a cold reading as one listing.
func coldRows(said []learn.Said) string {
	var b strings.Builder

	for _, one := range said {
		fmt.Fprintf(&b, "%-14s %-22s %s\n", one.Topic, one.About, one.Text)
	}

	return strings.TrimRight(b.String(), "\n")
}

// enforcing offers a rule for each thing the checkout already refuses work
// over: the commands its own pull requests have to pass.
//
// It spends nothing and asks no model, which is why it takes no engine and
// is not marked as spending. A workflow either parses or it does not — there
// is no sentence here that needs verifying against a citation, because the
// sentence is the command.
func enforcing(w World, in In) (Out, error) {
	here, err := openRepo(w, in)
	if err != nil {
		return Out{}, err
	}

	var gates []learn.Gate

	for _, g := range repo.Gates(here.Path) {
		gates = append(gates, learn.Gate{Command: g.Command, Where: g.Where})
	}

	said, err := learn.Enforced(w.Store(), here.Path, gates)
	if err != nil {
		return Out{}, err
	}

	if len(said) == 0 {
		return Out{
			Said: w.Words().T("verb.rules.enforced.none",
				"nothing this checkout refuses work over is unanswered; "+
					"a rule here comes from what a pull request has to pass"),
			Saw: said,
		}, nil
	}

	return Out{Said: gateRows(said) + "\n\n" + w.Words().P("verb.rules.enforced.waiting", len(said),
		"it is waiting in the rules for you to keep it or drop it",
		"{n} are waiting in the rules for you to keep them or drop them",
		words.Arg{Name: "n", Value: strconv.Itoa(len(said))}), Saw: said}, nil
}

// gateRows is the listing: the command a rule would refuse work with, and
// the sentence beside it.
//
// The command first, because it is the whole of why this reading is worth
// anything — every other source leaves a reader deciding what the gate would
// be, and here it is already written and already running.
func gateRows(said []learn.Said) string {
	var b strings.Builder

	for _, one := range said {
		fmt.Fprintf(&b, "%-34s %s\n", one.Gate, one.Text)
	}

	return strings.TrimRight(b.String(), "\n")
}
