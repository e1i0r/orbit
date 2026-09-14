package verb

// What somebody keeps telling runs, listed.
//
// The tray above is the rules a person meant to lay down. This is the ones
// they never said: "add fuzz testing" typed at six tasks in a row is a rule
// nobody ever enunciated, and reading it on its own it is not one.
//
// It offers nothing and decides nothing. What it does is put a person in
// front of what they do without noticing, which is the whole of what they
// need to write the rule themselves.

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// repeating is the listing: the habit said most often first, and under each
// one every time it was said.
//
// The sentences and not a count. The words somebody used are the rule, and
// the number is only the reason to go and look at them — a listing that gave
// the count alone would be asking them to remember what they said.
func repeating(w World) (Out, error) {
	habits, err := learn.Repeated(w.Store())
	if err != nil {
		return Out{}, err
	}

	if len(habits) == 0 {
		return Out{Said: w.Words().T("verb.rules.repeated.none",
			"nothing you have told a run has come up often enough to be a habit"), Saw: habits}, nil
	}

	var b strings.Builder

	for _, h := range habits {
		heading(&b, w.Words(), h)

		for _, one := range h.Said {
			fmt.Fprintf(&b, "     %-10s %s\n", one.Task, one.Text)
		}

		b.WriteString("\n")
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: habits}, nil
}

// heading is the line that names one habit: how often, where, and the words
// that hold it together.
//
// The words are on the heading because they are what somebody checks first.
// A habit grouped by "commit" and "common" is wrong, and the only way to see
// that at a glance is to be shown what it was grouped by.
func heading(b *strings.Builder, p *words.Printer, h learn.Habit) {
	where := filepath.Base(h.Repo)
	if h.Phase != "" {
		where += " · " + h.Phase
	}

	fmt.Fprintf(b, "%s  %s  %s\n",
		p.P("verb.rules.repeated.times", h.Times(), "said once", "said {n} times"),
		where, strings.Join(h.Words, ", "))
}

// drafting asks a model to write down what somebody keeps saying, and puts
// what it answers with in the tray.
//
// The one reading here that costs money, and the only place in Orbit a model
// decides anything. What it produces is an offer like any other: it waits to
// be kept or dropped, and nothing reaches a prompt until somebody says so.
func drafting(ctx context.Context, w World, in In) (Out, error) {
	named := in.Arg("engine")

	ask := func(ctx context.Context, question string) (string, error) {
		return w.Ask(ctx, named, question)
	}

	said, err := learn.Draft(ctx, w.Store(), ask)
	if err != nil {
		return Out{}, err
	}

	if len(said) == 0 {
		return Out{Said: w.Words().T("verb.rules.draft.none",
			"nothing you keep saying amounts to a rule that is not already answered"), Saw: said}, nil
	}

	var b strings.Builder

	for _, one := range said {
		fmt.Fprintf(&b, "%-14s %s\n", one.Topic, one.Text)
	}

	b.WriteString("\n" + w.Words().P("verb.rules.draft.waiting", len(said),
		"it is waiting in the rules for you to keep it or drop it",
		"{n} are waiting in the rules for you to keep them or drop them"))

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: said}, nil
}

// whereAt is the one column that says where a turn happened: the task and
// the phase for the ones that happen inside a run, and whoever did it for the
// ones somebody took from a terminal.
//
// One column and not two, because they never both apply. A gate refusing
// belongs to a run and to nobody; a pause typed at a terminal belongs to a
// person and to no run.
func whereAt(t learn.Turn) string {
	if t.Task == "" {
		return t.By
	}

	if t.Phase == "" {
		return t.Task
	}

	return t.Task + " · " + t.Phase
}

// happened is what one rule has been through since somebody kept it.
//
// The sentence a rule says today is in its file, and that file travels with
// the checkout. What it has been through is of this machine and only ever
// grows, which is why it is asked for separately and why this is the one
// place it can be read.
func happened(w World, in In) (Out, error) {
	rule := strings.TrimSpace(in.Arg("rule"))

	turns, err := learn.History(w.Store(), rule)
	if err != nil {
		return Out{}, err
	}

	if len(turns) == 0 {
		return Out{Said: w.Words().T("verb.rules.history.none",
			"nothing has happened to {rule} since it was kept",
			words.Arg{Name: "rule", Value: rule}), Saw: turns}, nil
	}

	var b strings.Builder

	for _, one := range turns {
		fmt.Fprintf(&b, "%s  %-8s %-16s %s\n",
			one.At.Local().Format(time.DateTime), one.What, whereAt(one), one.Was)
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: turns}, nil
}
