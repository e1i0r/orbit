package verb

// The rules you said, waiting to be kept.
//
// The first family: a verb with children, asked for as two words. `orbit
// rules` is the tray, and `orbit rules keep 3` is one row of it — which
// reads as what it is, and which sorts together in the one place somebody
// goes looking for what can be asked for.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// rules is the family, parent first.
func rules() []Verb {
	return []Verb{
		{
			Name: "rules", Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.rules", "the rules you said that Orbit has not been told to keep yet")
			},
		},
		{
			Name: "keep", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.keep", "keep one of them, in your words or in better ones")
			},
			Takes: []Field{
				{Name: "n", Kind: Named, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.rules.n", "which one, by its number in the list")
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
			Name: "drop", Under: "rules",
			About: func(p *words.Printer) string {
				return p.T("verb.rules.drop", "say it was not a rule; the sentence stays where you said it")
			},
			Takes: []Field{{Name: "n", Kind: Named, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.rules.drop.n", "which one, by its number in the list")
			}}},
		},
	}
}

// waiting is the tray, numbered.
//
// The number is the position in this listing and nothing more durable than
// that, the way the supervisor thread's is: keep and drop read it back the
// same way, out of the same order.
func waiting(w World) (Out, error) {
	said, err := learn.Waiting(w.Store())
	if err != nil {
		return Out{}, err
	}

	if len(said) == 0 {
		return Out{Said: w.Words().T("verb.rules.empty",
			"nothing you said is waiting to be kept"), Saw: said}, nil
	}

	// The place is a column only when something has one, because most
	// terminals here are a hundred columns wide and the sentence is what
	// somebody came to read.
	wide := 0

	for _, one := range said {
		if len(one.Path) > 0 {
			wide = max(wide, len(one.Path)+2)
		}
	}

	var b strings.Builder
	for i, one := range said {
		fmt.Fprintf(&b, "%3d  %s  %-12s %-*s%s\n",
			i+1, one.At.Local().Format(time.DateTime), one.From(), wide, one.Path, one.Text)
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: said}, nil
}

// agreed keeps one of them, and Orbit knows it from then on: it goes into
// every phase's prompt, and refuses work at the gate when it was given a
// command that can answer yes or no.
//
// The text is what the reader typed instead, and what they said when they
// typed nothing. Correcting is how most of these are accepted — being asked
// is worth nothing if the only answers are yes and no.
//
// And so is placing it. A rule is said in the middle of one thing and is
// usually true of somewhere narrower than where it was said; this is the
// moment somebody knows which folder, and the only one where they are
// looking at the sentence while they decide.
func agreed(w World, in In) (Out, error) {
	one, err := nth(w, in)
	if err != nil {
		return Out{}, err
	}

	text := strings.TrimSpace(in.Arg("text"))
	if text == "" {
		text = one.Text
	}

	// The checkout the reader named, then the one they are standing in, and
	// nowhere when there is neither — which is what a rule about everything
	// is. openRepo is the same reading every other verb makes of it.
	here, err := openRepo(w, in)
	if err != nil {
		return Out{}, err
	}

	where := learn.Place{Repo: here.Path, Path: in.Arg("in")}
	if err := learn.Keep(w.Store(), one.At, text, in.Arg("check"), where); err != nil {
		return Out{}, err
	}

	// Where it went and not what was typed. Most of these are kept without
	// a path at all, because the folder the work was in came with the
	// sentence — and an answer that left that out would be Orbit filing a
	// rule somewhere and not saying so.
	if named := placed(where.Path, one.Path); named != "" {
		return Out{Said: w.Words().T("verb.rules.kept_in", "Orbit knows it, in {where}: {rule}",
			words.Arg{Name: "where", Value: named},
			words.Arg{Name: "rule", Value: text})}, nil
	}

	return Out{Said: w.Words().T("verb.rules.kept", "Orbit knows it: {rule}",
		words.Arg{Name: "rule", Value: text})}, nil
}

// placed is the folder a kept rule ended up in: what the reader typed, then
// the folder the work was in when the sentence was said.
//
// A dot is the reader saying the whole checkout out loud, which is a place
// with no path — so it answers with nothing, the same as a rule nobody put
// anywhere.
func placed(typed, worked string) string {
	if named := strings.TrimSpace(typed); named != "" {
		if named == "." {
			return ""
		}

		return named
	}

	return worked
}

// dropped says it was not a rule. The sentence stays in the thread where it
// was said, which is where it belonged all along.
func dropped(w World, in In) (Out, error) {
	one, err := nth(w, in)
	if err != nil {
		return Out{}, err
	}

	if err := learn.Drop(w.Store(), one.At); err != nil {
		return Out{}, err
	}

	return Out{Said: w.Words().T("verb.rules.dropped", "left where you said it: {rule}",
		words.Arg{Name: "rule", Value: one.Text})}, nil
}

// nth is the sentence a reader meant by its number in the listing.
func nth(w World, in In) (learn.Said, error) {
	n, err := strconv.Atoi(in.Arg("n"))
	if err != nil {
		return learn.Said{}, errors.New(w.Words().T("verb.rules.not_a_number",
			"a rule is named by its number in the list, and {value} is not a number",
			words.Arg{Name: "value", Value: in.Arg("n")}))
	}

	said, err := learn.Waiting(w.Store())
	if err != nil {
		return learn.Said{}, err
	}

	switch {
	case len(said) == 0:
		return learn.Said{}, errors.New(w.Words().T("verb.rules.empty",
			"nothing you said is waiting to be kept"))
	case n < 1 || n > len(said):
		// {n} is the count, which is what a plural is chosen by, so the
		// number that was asked for goes under a name of its own.
		return learn.Said{}, errors.New(w.Words().P("verb.rules.no_such_rule", len(said),
			"there is no rule {asked}; one rule is waiting",
			"there is no rule {asked}; {n} rules are waiting",
			words.Arg{Name: "asked", Value: in.Arg("n")}))
	}

	return said[n-1], nil
}
