package verb

// The tray: the sentences nobody has answered yet, and the two answers.
//
// The number is the position in the listing and nothing more durable than
// that, the way the supervisor thread's is: keep and drop read it back the
// same way, out of the same order.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// waiting is what the reader asked about: the rules standing in one state,
// or — with nothing asked — everything wanting a decision from them.
func waiting(w World, in In) (Out, error) {
	if asked := strings.TrimSpace(in.Arg("state")); asked != "" {
		return where(w, asked)
	}

	return unanswered(w)
}

// numbered is the tray: the sentences nobody has answered, by their position
// in this listing and nothing more durable than that. Keep and drop read the
// number back the same way, out of the same order.
func numbered(said []learn.Said) string {
	// Both columns are as wide as their widest row and no wider, and the
	// place is not a column at all when nothing has one: most terminals
	// here are a hundred columns across, and the sentence is what somebody
	// came to read.
	//
	// Counted in runes and not in bytes, because a column width is what a
	// reader sees: the mark between a task and the model that found
	// something is one character and three bytes.
	from, place := 0, 0

	for _, one := range said {
		from = max(from, utf8.RuneCountInString(one.From()))

		if one.Path != "" {
			place = max(place, utf8.RuneCountInString(one.Path)+2)
		}
	}

	var b strings.Builder
	for i, one := range said {
		fmt.Fprintf(&b, "%3d  %s  %-*s  %-*s%s\n",
			i+1, one.At.Local().Format(time.DateTime),
			from, one.From(), place, one.Path, one.Text)
	}

	return strings.TrimRight(b.String(), "\n")
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

	// The command the sentence arrived with, unless the reader named one.
	// A rule read off what the checkout already refuses work over is a
	// command that has been running for years; asking whoever keeps it to
	// type that command again is asking them to copy it out of a file
	// Orbit already read.
	check := in.Arg("check")
	if _, said := in.Args["check"]; !said {
		check = one.Gate
	}

	where := learn.Place{Repo: here.Path, Path: in.Arg("in")}
	if err := learn.Keep(w.Store(), one.At, text, check, where); err != nil {
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

// nth is the sentence a caller meant: by the instant it was said, or by its
// number in the listing.
//
// Two ways because two callers. A person reads a numbered list and types the
// number, and asking them for a timestamp would be asking them to copy one.
// A screen has no list in front of a reader to count down, and a number is
// not a name anyway: between reading the tray and answering it, another
// sentence can arrive and push the one that was meant along. The instant is
// the sentence's own name — the thread is append-only and no two turns share
// one — so a caller that has it says it.
func nth(w World, in In) (learn.Said, error) {
	if at := strings.TrimSpace(in.Arg("at")); at != "" {
		return saidAt(w, at)
	}

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

// saidAt is the sentence said at that instant, and a refusal when the tray
// no longer holds one — which is what somebody else having answered it
// first looks like from here.
func saidAt(w World, at string) (learn.Said, error) {
	when, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return learn.Said{}, errors.New(w.Words().T("verb.rules.not_an_instant",
			"a sentence is named by when it was said, and {value} is not an instant",
			words.Arg{Name: "value", Value: at}))
	}

	said, err := learn.Waiting(w.Store())
	if err != nil {
		return learn.Said{}, err
	}

	for _, one := range said {
		if one.At.Equal(when) {
			return one, nil
		}
	}

	return learn.Said{}, errors.New(w.Words().T("verb.rules.gone",
		"nothing said at {when} is waiting to be kept; somebody may have answered it already",
		words.Arg{Name: "when", Value: at}))
}
